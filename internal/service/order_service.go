package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/fsdevblog/groph-loyal/internal/repository/repoargs"
	"github.com/shopspring/decimal"

	"github.com/fsdevblog/groph-loyal/pkg/uow"

	"github.com/fsdevblog/groph-loyal/internal/domain"
)

type OrderService struct {
	uow       uow.UOW
	orderRepo OrderRepository
}

func NewOrderService(u uow.UOW) (*OrderService, error) {
	orderRepo, err := uow.GetRepositoryAs[OrderRepository](u, uow.RepositoryName(repoargs.OrderRepoName))
	if err != nil {
		return nil, err
	}
	return &OrderService{
		uow:       u,
		orderRepo: orderRepo,
	}, nil
}

// OrdersForAccrualMonitoring возвращает заказы подлежащие мониторингу начисления баллов лояльности.
func (o *OrderService) OrdersForAccrualMonitoring(ctx context.Context, limit uint) ([]domain.Order, error) {
	orders, err := o.orderRepo.GetForMonitoring(ctx, limit)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	return orders, nil
}

type UpdateAccrualArgs struct {
	Error   error
	OrderID int64
	Status  domain.OrderStatusType
	Accrual decimal.Decimal
}

type updateAccrualData struct {
	OrderID int64
	Status  domain.OrderStatusType
	Accrual decimal.Decimal
}

// UpdateAccrual обновляет данные заказов с новыми статусами и суммами начисленных баллов.
//
// Параметры:
//   - ctx: контекст для управления жизненным циклом
//   - updates: срез структур для обновления заказов.
//
// Алгоритм работы:
//  1. Обновляет данные заказов в базе данных
//  2. Создает транзакции баланса для юзеров, которым предполагается начисление баллов лояльности.
func (o *OrderService) UpdateAccrual(
	ctx context.Context,
	updates []UpdateAccrualArgs,
) error {
	txErr := o.uow.Do(ctx, func(c context.Context, tx uow.TX) error {
		successData, failureIDs := o.splitSuccessFailureUpdates(updates)

		if err := o.updateSuccessOrdersWithAccrual(c, tx, successData); err != nil {
			return err //nolint:wrapcheck
		}

		if err := o.incrementErrAttempts(c, tx, failureIDs); err != nil {
			return err //nolint:wrapcheck
		}

		return nil
	})

	if txErr != nil {
		return fmt.Errorf("updating orders with accrual: %w", txErr)
	}
	return nil
}

func (o *OrderService) incrementErrAttempts(ctx context.Context, tx uow.TX, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	repo, repoErr := uow.GetAs[OrderRepository](tx, uow.RepositoryName(repoargs.OrderRepoName))
	if repoErr != nil {
		return repoErr
	}
	return repo.IncrementErrAttempts(ctx, ids) //nolint:wrapcheck
}

func (o *OrderService) updateSuccessOrdersWithAccrual(ctx context.Context, tx uow.TX, data []updateAccrualData) error {
	if len(data) == 0 {
		return nil
	}
	orders, updOrdersErr := o.updateOrdersWithAccrual(ctx, tx, data)
	if updOrdersErr != nil {
		return updOrdersErr
	}

	if bTransErr := o.createBalanceTransactionsForOrders(ctx, tx, orders); bTransErr != nil {
		return bTransErr
	}
	return nil
}

// splitSuccessFailureUpdates разбивает срез структур на 2 логические части. Одну для обновления в репозитории
// а вторую - срез id, которые нужно пометить как ошибочные.
func (o *OrderService) splitSuccessFailureUpdates(updates []UpdateAccrualArgs) ([]updateAccrualData, []int64) {
	var successData = make([]updateAccrualData, 0, len(updates))
	var failureIDs = make([]int64, 0, len(updates))
	for _, update := range updates {
		if update.Error == nil {
			successData = append(successData, updateAccrualData{
				OrderID: update.OrderID,
				Status:  update.Status,
				Accrual: update.Accrual,
			})
		} else {
			failureIDs = append(failureIDs, update.OrderID)
		}
	}
	return successData, failureIDs
}

// createBalanceTransactionsForOrders создает записи в таблице balance_transactions для заказов со статусом
// OrderStatusProcessed
//
// Параметры:
//   - ctx: контекст для управления жизненным циклом
//   - tx: UnitOfWork транзакция
//   - orders: список заказов для обработки
//
// Алгоритм работы:
//  1. Фильтрует срез ордеров по вышеупомянутому статусу.
//  2. Формирует и отправляет батч запрос для репозитория транзакций баланса.
//  3. Анализирует полученный результат, игнорируя ошибки дубликата так как дубликат может быть только по ID заказа,
//     а заказ обрабатывается лишь единожды.
//
// Возвращает ошибку. Если при батч запросе произошло несколько ошибок, вернется последняя ошибка.
func (o *OrderService) createBalanceTransactionsForOrders(ctx context.Context, tx uow.TX, orders []domain.Order) error {
	var transDTO = make([]repoargs.BalanceTransactionCreate, 0, len(orders))
	for _, order := range orders {
		if order.Status == domain.OrderStatusProcessed {
			transDTO = append(transDTO, repoargs.BalanceTransactionCreate{
				UserID:    order.UserID,
				OrderID:   order.ID,
				OrderCode: order.OrderCode,
				Amount:    order.Accrual,
				Direction: domain.DirectionDebit,
			})
		}
	}
	if len(transDTO) == 0 {
		// если балансов для обновления нет, выходим.
		return nil
	}

	balanceRepo, balanceRepoErr :=
		uow.GetAs[BalanceTransactionRepository](tx, uow.RepositoryName(repoargs.BalanceTransactionRepoName))

	if balanceRepoErr != nil {
		return balanceRepoErr //nolint:wrapcheck
	}

	var balanceTransactionErr error

	balanceRepo.BatchCreate(ctx, transDTO, func(_ int, err error) {
		if err != nil {
			if errors.Is(err, domain.ErrDuplicateKey) {
				return
			}
			balanceTransactionErr = err
		}
	})
	return balanceTransactionErr
}

// updateOrdersWithAccrual вспомогательный метод, выполняющий батч запрос на обновление заказов с данными начисления
// баллов.
func (o *OrderService) updateOrdersWithAccrual(
	ctx context.Context,
	tx uow.TX,
	updates []updateAccrualData,
) ([]domain.Order, error) {
	repo, repoErr := uow.GetAs[OrderRepository](tx, uow.RepositoryName(repoargs.OrderRepoName))
	if repoErr != nil {
		return nil, repoErr //nolint:wrapcheck
	}

	var orders = make([]domain.Order, len(updates))

	var repoArgs = make([]repoargs.BatchUpdateWithAccrualData, len(updates))
	for i, update := range updates {
		repoArgs[i] = repoargs.BatchUpdateWithAccrualData{
			ID:      update.OrderID,
			Status:  update.Status,
			Accrual: update.Accrual,
		}
	}
	// updOrderErr будет хранить последнюю ошибку результата батч вставки. Мне кажется нет смысла ошибки объединять.
	var updOrderErr error
	repo.BatchUpdateWithAccrualData(ctx, repoArgs, func(i int, dbOrder *domain.Order, err error) {
		if err != nil {
			updOrderErr = err
			return
		}
		orders[i] = *dbOrder
	})
	return orders, updOrderErr
}

// Create создает новый заказ в БД. Возвращает 2 значения, созданный заказ и ошибку. Если заказ уже присутствует
// в БД вернется ошибка *domain.DuplicateOrderError, во всех других случаях - domain.ErrUnknown.
func (o *OrderService) Create(ctx context.Context, userID int64, orderCode string) (*domain.Order, error) {
	var order *domain.Order

	order, createErr := o.orderRepo.CreateOrder(ctx, userID, orderCode)
	if createErr != nil {
		// Если запись присутствует в БД. Получаем её и передаем в domain.DuplicateOrderError.
		if errors.Is(createErr, domain.ErrDuplicateKey) {
			existingOrder, existingOrderErr := o.orderRepo.FindByOrderCode(ctx, orderCode)
			if existingOrderErr != nil {
				return nil, fmt.Errorf("creating order: %w", existingOrderErr)
			}
			return nil, &domain.DuplicateOrderError{Order: existingOrder}
		}

		return nil, fmt.Errorf("creating order: %w", createErr)
	}

	return order, nil
}

// GetByUserID Возвращает заказы от userID отсортированные по дате создания по убыванию.
func (o *OrderService) GetByUserID(ctx context.Context, userID int64) ([]domain.Order, error) {
	orders, err := o.orderRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	return orders, nil
}

package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/fsdevblog/groph-loyal/pkg/uow"

	"github.com/fsdevblog/groph-loyal/internal/domain"
)

type OrderService struct {
	uow       uow.UOW
	orderRepo domain.OrderRepository
}

func NewOrderService(u uow.UOW) (*OrderService, error) {
	orderRepo, err := uow.GetRepositoryAs[domain.OrderRepository](u, uow.RepositoryName(domain.OrderRepoName))
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
	statuses := []domain.OrderStatusType{
		domain.OrderStatusNew,
		domain.OrderStatusProcessing,
		domain.OrderStatusProcessing,
	}

	orders, err := o.orderRepo.GetByStatuses(ctx, limit, statuses)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	return orders, nil
}

// UpdateOrdersWithAccrual обновляет данные заказов с новыми статусами и суммами начисленных баллов.
//
// Параметры:
//   - ctx: контекст для управления жизненным циклом
//   - updates: срез структур для обновления заказов.
//
// Алгоритм работы:
//  1. Обновляет данные заказов в базе данных
//  2. Создает транзакции баланса для юзеров, которым предполагается начисление баллов лояльности.
func (o *OrderService) UpdateOrdersWithAccrual(ctx context.Context, updates []domain.OrderAccrualUpdateDTO) error {
	txErr := o.uow.Do(ctx, func(c context.Context, tx uow.TX) error {
		orders, updOrdersErr := o.updateOrdersWithAccrual(c, tx, updates)
		if updOrdersErr != nil {
			return updOrdersErr //nolint:wrapcheck
		}

		if bTransErr := o.createBalanceTransactionsForOrders(c, tx, orders); bTransErr != nil {
			return bTransErr
		}
		return nil
	})

	if txErr != nil {
		return fmt.Errorf("updating orders with accrual: %w", txErr)
	}
	return nil
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
	var transDTO = make([]domain.BalanceTransactionCreateDTO, 0, len(orders))
	for _, order := range orders {
		if order.Status == domain.OrderStatusProcessed {
			transDTO = append(transDTO, domain.BalanceTransactionCreateDTO{
				UserID:    order.UserID,
				OrderID:   &order.ID,
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
		uow.GetAs[domain.BalanceTransactionRepository](tx, uow.RepositoryName(domain.BalanceTransactionRepoName))

	if balanceRepoErr != nil {
		return balanceRepoErr //nolint:wrapcheck
	}

	var balanceTransactionErr error

	balanceRepo.BatchCreate(ctx, transDTO, func(i int, err error) {
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
	updates []domain.OrderAccrualUpdateDTO,
) ([]domain.Order, error) {
	repo, repoErr := uow.GetAs[domain.OrderRepository](tx, uow.RepositoryName(domain.OrderRepoName))
	if repoErr != nil {
		return nil, repoErr //nolint:wrapcheck
	}

	var orders = make([]domain.Order, len(updates))

	// updOrderErr будет хранить последнюю ошибку результата батч вставки. Мне кажется нет смысла ошибки объединять.
	var updOrderErr error
	repo.BatchUpdateWithAccrualData(ctx, updates, func(i int, dbOrder *domain.Order, err error) {
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
	txErr := o.uow.Do(ctx, func(c context.Context, tx uow.TX) error {
		repo, repoErr := uow.GetAs[domain.OrderRepository](tx, uow.RepositoryName(domain.OrderRepoName))
		if repoErr != nil {
			return repoErr
		}

		var createErr error
		order, createErr = repo.CreateOrder(c, userID, orderCode)

		if createErr != nil {
			// Если запись присутствует в БД. Получаем её и передаем в domain.DuplicateOrderError.
			if errors.Is(createErr, domain.ErrDuplicateKey) {
				existingOrder, existingOrderErr := repo.FindByOrderCode(c, orderCode)
				if existingOrderErr != nil {
					return existingOrderErr //nolint:wrapcheck
				}
				return &domain.DuplicateOrderError{Order: existingOrder}
			}

			return createErr //nolint:wrapcheck
		}
		return nil
	})

	if txErr != nil {
		return nil, fmt.Errorf("creating order: %w", txErr)
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

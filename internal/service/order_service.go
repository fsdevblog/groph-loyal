package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/fsdevblog/groph-loyal/internal/repository/repoargs"
	"github.com/shopspring/decimal"

	"github.com/fsdevblog/groph-loyal/pkg/uow"

	"github.com/fsdevblog/groph-loyal/internal/domain"
)

const attemptCoefficient float64 = 1.1 // Коэффициент для экспоненциального увеличения задержки попыток проверки заказа

// OrderService реализует бизнес-логику работы с заказами.
type OrderService struct {
	uow       uow.UOW         // Unit of Work для транзакционных операций
	orderRepo OrderRepository // Репозиторий заказов
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

// OrdersForAccrualMonitoring возвращает заказы, которые необходимо мониторить для начисления баллов лояльности.
func (o *OrderService) OrdersForAccrualMonitoring(ctx context.Context, limit uint) ([]domain.Order, error) {
	orders, err := o.orderRepo.GetForMonitoring(ctx, limit)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	return orders, nil
}

// UpdateAccrualArgs — структура, описывающая аргумент для обновления начисления у одного заказа.
type UpdateAccrualArgs struct {
	Error   error                  // ошибка, возникшая при обработке заказа (если есть)
	OrderID int64                  // ID заказа
	Attempt uint                   // кол-во текущих попыток
	Status  domain.OrderStatusType // новый статус заказа (значение по умолчанию если есть ошибка)
	Accrual decimal.Decimal        // сумма начисленных баллов (значение по умолчанию если есть ошибка)
}

// successUpdatesAccrual — структура для успешных обновлений начислений по заказу.
type successUpdatesAccrual struct {
	OrderID int64
	Status  domain.OrderStatusType
	Accrual decimal.Decimal
}

// failureUpdatesAccrual — структура для неудачных (ошибочных) обновлений, где требуется инкрементировать счетчик
// попыток.
type failureUpdatesAccrual struct {
	OrderID        int64
	CurrentAttempt uint
}

// UpdateAccrual обновляет статусы заказов и начисления баллов согласно переданному срезу обновлений.
// Сначала сохраняет новые статусы и баллы по успешным заказам, затем увеличивает число попыток у неуспешных.
func (o *OrderService) UpdateAccrual(
	ctx context.Context,
	updates []UpdateAccrualArgs,
) error {
	txErr := o.uow.Do(ctx, func(c context.Context, tx uow.TX) error {
		successData, failureIDs := o.splitSuccessFailureUpdates(updates)

		// тут очень хотелось запустить апдейт параллельно через errgroup, но
		// к сожалению tx не потокобезопасен.. Облом..
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

// incrementErrAttempts вычисляет время следующей попытки. Инкремент самой попытки лежит на плечах репозитория.
// Делает батч обновление, в случае ошибок возвращает последнюю.
func (o *OrderService) incrementErrAttempts(ctx context.Context, tx uow.TX, fails []failureUpdatesAccrual) error {
	if len(fails) == 0 {
		return nil
	}
	repo, repoErr := uow.GetAs[OrderRepository](tx, uow.RepositoryName(repoargs.OrderRepoName))
	if repoErr != nil {
		return repoErr
	}

	// конвертируем данные в аргументы для репозитория.
	var args = make([]repoargs.OrderBatchIncrementAttempts, len(fails))
	for i, fail := range fails {
		// высчитываем время следующей проверки.
		seconds := math.Pow(attemptCoefficient, float64(fail.CurrentAttempt))
		delay := jitter(seconds, 0.25, 0.25) //nolint:mnd
		nextAttempt := time.Now().Add(time.Duration(delay) * time.Second)

		args[i] = repoargs.OrderBatchIncrementAttempts{
			ID:            fail.OrderID,
			NextAttemptAt: nextAttempt,
		}
	}
	var incrementErr error
	repo.IncrementErrAttempts(ctx, args, func(_ int, err error) {
		if err != nil {
			incrementErr = err
		}
	})

	return incrementErr
}

// updateSuccessOrdersWithAccrual обновляет статусы заказов с успешным начислением
// и создает транзакции начислений баллов.
func (o *OrderService) updateSuccessOrdersWithAccrual(
	ctx context.Context,
	tx uow.TX,
	data []successUpdatesAccrual,
) error {
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

// splitSuccessFailureUpdates разбивает срез обновлений на две группы:
// - успешные (без ошибок)
// - неудачные (с ошибками).
func (o *OrderService) splitSuccessFailureUpdates(updates []UpdateAccrualArgs) (
	[]successUpdatesAccrual,
	[]failureUpdatesAccrual,
) {
	var successData = make([]successUpdatesAccrual, 0, len(updates))
	var failureIDs = make([]failureUpdatesAccrual, 0, len(updates))
	for _, update := range updates {
		if update.Error == nil {
			successData = append(successData, successUpdatesAccrual{
				OrderID: update.OrderID,
				Status:  update.Status,
				Accrual: update.Accrual,
			})
		} else {
			failureIDs = append(failureIDs, failureUpdatesAccrual{
				OrderID:        update.OrderID,
				CurrentAttempt: update.Attempt,
			})
		}
	}
	return successData, failureIDs
}

// createBalanceTransactionsForOrders создает записи в таблице транзакций баланса для заказов со статусом PROCESSED.
//
// 1. Фильтрует заказы по статусу PROCESSED.
// 2. Формирует батч и вызывает репозиторий транзакций баланса.
// 3. Игнорирует ошибки дубликата (обработка заказа допускается один раз).
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
	updates []successUpdatesAccrual,
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

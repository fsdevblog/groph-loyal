package sqlc

import (
	"context"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/repository/repoargs"
	"github.com/fsdevblog/groph-loyal/internal/repository/sqlc/sqlcgen"
)

type OrderRepository struct {
	q *sqlcgen.Queries
}

func NewOrderRepository(conn sqlcgen.DBTX) *OrderRepository {
	return &OrderRepository{q: sqlcgen.New(conn)}
}

func (o *OrderRepository) BatchUpdateWithAccrualData(
	ctx context.Context,
	updates []repoargs.BatchUpdateWithAccrualData,
	fn repoargs.OrderBatchQueryRow,
) {
	var data = make([]sqlcgen.Orders_UpdateWithAccrualDataParams, len(updates))
	for i, update := range updates {
		data[i] = sqlcgen.Orders_UpdateWithAccrualDataParams{
			Status:  sqlcgen.OrderStatusType(update.Status),
			Accrual: update.Accrual,
			ID:      update.ID,
		}
	}
	r := o.q.Orders_UpdateWithAccrualData(ctx, data)

	r.QueryRow(func(i int, dbOrder sqlcgen.Order, err error) {
		fn(i, convertOrderModel(dbOrder), convertErr(err, "updating order with id %d", updates[i].ID))
	})
}

func (o *OrderRepository) GetForMonitoring(
	ctx context.Context,
	limit uint,
) ([]domain.Order, error) {
	safeLimit, safeLimitErr := safeConvertUintToInt32(limit)
	if safeLimitErr != nil {
		return nil, convertErr(safeLimitErr, "converting limit to int32")
	}

	dbOrders, err := o.q.Orders_GetForMonitoring(ctx, safeLimit)
	if err != nil {
		return nil, convertErr(err, "getting orders for monitoring")
	}
	var orders = make([]domain.Order, len(dbOrders))
	for i, order := range dbOrders {
		orders[i] = *convertOrderModel(order)
	}
	return orders, nil
}

func (o *OrderRepository) IncrementErrAttempts(ctx context.Context, orderIDs []int64) error {
	if err := o.q.Orders_IncrementAttempts(ctx, orderIDs); err != nil {
		return convertErr(err, "incrementing err attempts for orders with ids `%v`", orderIDs)
	}
	return nil
}

func (o *OrderRepository) CreateOrder(ctx context.Context, userID int64, orderCode string) (*domain.Order, error) {
	dbOrder, err := o.q.Orders_Create(ctx, sqlcgen.Orders_CreateParams{
		UserID:    userID,
		OrderCode: orderCode,
		Status:    sqlcgen.OrderStatusTypeNEW,
	})
	if err != nil {
		return nil, convertErr(err, "creating order with code `%s`", orderCode)
	}

	return convertOrderModel(dbOrder), nil
}

func (o *OrderRepository) FindByOrderCode(ctx context.Context, orderCode string) (*domain.Order, error) {
	dbOrder, err := o.q.Orders_FindByOrderCode(ctx, orderCode)
	if err != nil {
		return nil, convertErr(err, "finding order by code `%s`", orderCode)
	}
	return convertOrderModel(dbOrder), nil
}

// GetByUserID Возвращает список заказов по id юзера, отсортированный по дате создания по убыванию.
func (o *OrderRepository) GetByUserID(ctx context.Context, userID int64) ([]domain.Order, error) {
	dbOrders, err := o.q.Orders_GetByUserID(ctx, userID)
	if err != nil {
		return nil, convertErr(err, "getting orders by userID `%d`", userID)
	}
	var orders = make([]domain.Order, len(dbOrders))
	for i, order := range dbOrders {
		orders[i] = *convertOrderModel(order)
	}
	return orders, nil
}

func convertOrderModel(dbModel sqlcgen.Order) *domain.Order {
	return &domain.Order{
		ID:        dbModel.ID,
		CreatedAt: dbModel.CreatedAt.Time,
		UpdatedAt: dbModel.UpdatedAt.Time,
		UserID:    dbModel.UserID,
		OrderCode: dbModel.OrderCode,
		Status:    domain.OrderStatusType(dbModel.Status),
		Accrual:   dbModel.Accrual,
	}
}

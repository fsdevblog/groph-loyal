package sqlc

import (
	"context"
	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/repository/sqlc/sqlcgen"
)

type orderRepository struct {
	q *sqlcgen.Queries
}

func NewOrderRepository(conn sqlcgen.DBTX) domain.OrderRepository {
	return &orderRepository{q: sqlcgen.New(conn)}
}

func (o *orderRepository) CreateOrder(ctx context.Context, userID int64, orderCode string) (*domain.Order, error) {
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

func (o *orderRepository) FindByOrderCode(ctx context.Context, orderCode string) (*domain.Order, error) {
	dbOrder, err := o.q.Orders_FindByOrderCode(ctx, orderCode)
	if err != nil {
		return nil, convertErr(err, "finding order by code `%s`", orderCode)
	}
	return convertOrderModel(dbOrder), nil
}

func convertOrderModel(dbModel sqlcgen.Order) *domain.Order {
	accrual, _ := safeConvertInt32ToUint(dbModel.Accrual)

	return &domain.Order{
		ID:        dbModel.ID,
		CreatedAt: dbModel.CreatedAt.Time,
		UpdatedAt: dbModel.UpdatedAt.Time,
		UserID:    dbModel.UserID,
		OrderCode: dbModel.OrderCode,
		Status:    domain.OrderStatus(dbModel.Status),
		Accrual:   accrual,
	}
}

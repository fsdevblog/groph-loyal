package accrual

//go:generate mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks

import (
	"context"

	"github.com/fsdevblog/groph-loyal/internal/transport/accrual/dto"

	"github.com/fsdevblog/groph-loyal/internal/domain"
)

type Client interface {
	GetOrderAccrual(ctx context.Context, orderCode string) (*dto.OrderAccrualResponse, error)
}

type Servicer interface {
	OrdersForAccrualMonitoring(ctx context.Context, limit uint) ([]domain.Order, error)
	UpdateOrdersWithAccrual(ctx context.Context, updates []domain.OrderAccrualUpdateDTO) error
}

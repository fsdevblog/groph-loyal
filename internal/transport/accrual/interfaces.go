package accrual

//go:generate mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks

import (
	"context"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/service"
	"github.com/fsdevblog/groph-loyal/internal/transport/accrual/client"
)

type Client interface {
	GetOrderAccrual(ctx context.Context, orderCode string) (*client.Response, error)
}

type Servicer interface {
	OrdersForAccrualMonitoring(ctx context.Context, limit uint) ([]domain.Order, error)
	UpdateAccrual(ctx context.Context, updates []service.UpdateAccrualArgs) error
}

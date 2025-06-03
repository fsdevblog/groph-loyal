package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/uow"
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

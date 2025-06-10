package service

import (
	"fmt"

	"github.com/fsdevblog/groph-loyal/pkg/uow"
)

type AppServices struct {
	UserService  *UserService
	OrderService *OrderService
	BlService    *BalanceTransactionService
}

func Factory(unitOfWork uow.UOW, jwtSecret []byte) (*AppServices, error) {
	userService, userServiceErr := NewUserService(unitOfWork, jwtSecret)

	if userServiceErr != nil {
		return nil, fmt.Errorf("service factory: %s", userServiceErr.Error())
	}

	orderService, orderServiceErr := NewOrderService(unitOfWork)
	if orderServiceErr != nil {
		return nil, fmt.Errorf("service factory: %s", orderServiceErr.Error())
	}

	blService, blServiceErr := NewBalanceTransactionService(unitOfWork)
	if blServiceErr != nil {
		return nil, fmt.Errorf("service factory: %s", blServiceErr.Error())
	}

	return &AppServices{
		UserService:  userService,
		OrderService: orderService,
		BlService:    blService,
	}, nil
}

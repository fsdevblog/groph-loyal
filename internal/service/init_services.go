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

// NewAppServices инициализирует и возвращает структуру AppServices, содержащую все сервисы приложения.
// Метод создает экземпляры UserService, OrderService и BalanceTransactionService, используя предоставленный unitOfWork
// и секретный ключ для JWT.
//
// Если при создании какого-либо из сервисов возникает ошибка, метод немедленно прерывает выполнение
// и возвращает эту ошибку.
func NewAppServices(unitOfWork uow.UOW, jwtSecret []byte) (*AppServices, error) {
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

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
	userService, errUserService := NewUserService(unitOfWork, jwtSecret)

	if errUserService != nil {
		return nil, fmt.Errorf("service factory: %s", errUserService.Error())
	}

	orderService, errOrderService := NewOrderService(unitOfWork)
	if errOrderService != nil {
		return nil, fmt.Errorf("service factory: %s", errOrderService.Error())
	}

	blService, errBlService := NewBalanceTransactionService(unitOfWork)
	if errBlService != nil {
		return nil, fmt.Errorf("service factory: %s", errBlService.Error())
	}

	return &AppServices{
		UserService:  userService,
		OrderService: orderService,
		BlService:    blService,
	}, nil
}

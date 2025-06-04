package service

import (
	"context"
	"testing"
	"time"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	repomocks "github.com/fsdevblog/groph-loyal/internal/domain/mocks"
	"github.com/fsdevblog/groph-loyal/internal/uow"
	uowmocks "github.com/fsdevblog/groph-loyal/internal/uow/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

type OrderServiceTestSuite struct {
	suite.Suite
	mockUOW       *uowmocks.MockUOW
	mockTX        *uowmocks.MockTX
	mockOrderRepo *repomocks.MockOrderRepository
	orderService  *OrderService
}

func TestOrderServiceSuite(t *testing.T) {
	suite.Run(t, new(OrderServiceTestSuite))
}

func (s *OrderServiceTestSuite) SetupTest() {
	mockCtrl := gomock.NewController(s.T())
	s.mockUOW = uowmocks.NewMockUOW(mockCtrl)
	s.mockOrderRepo = repomocks.NewMockOrderRepository(mockCtrl)
	s.mockTX = uowmocks.NewMockTX(mockCtrl)

	// Мок получения репозитория из uow. Выполняется в инициализации сервиса.
	s.mockUOW.EXPECT().GetRepository(uow.RepositoryName(domain.OrderRepoName)).
		Return(s.mockOrderRepo, nil).AnyTimes()

	// Инициализация сервиса.
	orderService, servErr := NewOrderService(s.mockUOW)
	s.Require().NoError(servErr)
	s.orderService = orderService
}

func (s *OrderServiceTestSuite) TestCreate() {
	var userID int64 = 1
	validOrderCode := "12345678903"
	existingOrderCode := "79927398713"

	createdOrder := domain.Order{
		ID:        1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    userID,
		OrderCode: validOrderCode,
		Status:    domain.OrderStatusNew,
		Accrual:   0,
	}

	existingOrder := domain.Order{
		ID:        2,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    userID,
		OrderCode: existingOrderCode,
		Status:    domain.OrderStatusNew,
		Accrual:   0,
	}

	// Мок транзакции uow.
	s.mockTX.EXPECT().Get(uow.RepositoryName(domain.OrderRepoName)).
		Return(s.mockOrderRepo, nil).MinTimes(1)

	// Мок репозитория для валидного кода.
	s.mockOrderRepo.EXPECT().
		CreateOrder(gomock.Any(), userID, validOrderCode).
		Return(&createdOrder, nil)

	// Мок репозитория для существующего кода.
	s.mockOrderRepo.EXPECT().
		CreateOrder(gomock.Any(), userID, existingOrderCode).
		Return(nil, domain.ErrDuplicateKey)

	// Мок репозитория поиска существующего кода.
	s.mockOrderRepo.EXPECT().FindByOrderCode(gomock.Any(), existingOrderCode).
		Return(&existingOrder, nil)

	// Мок uow.
	s.mockUOW.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context, uow.TX) error) error {
			return fn(ctx, s.mockTX)
		}).MinTimes(1)

	cases := []struct {
		name        string
		userID      int64
		orderCode   string
		wantErrType error
		wantOrder   *domain.Order
	}{
		{
			name:      "ok",
			userID:    userID,
			orderCode: validOrderCode,
			wantOrder: &createdOrder,
		},
		{
			name:        "duplicate order",
			userID:      userID,
			orderCode:   existingOrderCode,
			wantErrType: new(domain.DuplicateOrderError),
		},
	}

	for _, t := range cases {
		s.Run(t.name, func() {
			order, err := s.orderService.Create(s.T().Context(), t.userID, t.orderCode)

			if t.wantErrType != nil {
				s.Require().Error(err)
				s.Require().ErrorAs(err, &t.wantErrType)
				return
			}

			s.Equal(t.wantOrder, order)
		})
	}
}

func (s *OrderServiceTestSuite) TestGetByUserID() {
	var userID int64 = 1
	var emptyUserID int64 = 2

	orders := []domain.Order{
		{
			ID:        1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			UserID:    userID,
			OrderCode: "12345678903",
			Status:    domain.OrderStatusNew,
			Accrual:   0,
		},
		{
			ID:        2,
			CreatedAt: time.Now().Add(-time.Hour),
			UpdatedAt: time.Now(),
			UserID:    userID,
			OrderCode: "12345678904",
			Status:    domain.OrderStatusProcessed,
			Accrual:   100,
		},
	}

	s.mockOrderRepo.EXPECT().
		GetByUserID(gomock.Any(), userID).
		Return(orders, nil)

	s.mockOrderRepo.EXPECT().
		GetByUserID(gomock.Any(), emptyUserID).
		Return([]domain.Order{}, nil)

	cases := []struct {
		name      string
		userID    int64
		wantEmpty bool
	}{
		{
			name:   "ok",
			userID: userID,
		},
		{
			name:      "empty result",
			userID:    emptyUserID,
			wantEmpty: true,
		},
	}

	for _, t := range cases {
		s.Run(t.name, func() {
			result, err := s.orderService.GetByUserID(s.T().Context(), t.userID)

			s.Require().NoError(err)
			if t.wantEmpty {
				s.Require().Empty(result)
			} else {
				s.Require().Len(result, 2)
				s.Equal(userID, result[0].UserID)
				s.Equal(userID, result[1].UserID)
			}
		})
	}
}

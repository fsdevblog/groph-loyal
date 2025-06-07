package service

import (
	"context"
	"testing"
	"time"

	"github.com/fsdevblog/groph-loyal/internal/repository/repoargs"
	"github.com/fsdevblog/groph-loyal/internal/service/mocks"

	"github.com/fsdevblog/groph-loyal/pkg/uow"
	uowmocks "github.com/fsdevblog/groph-loyal/pkg/uow/mocks"
	"github.com/shopspring/decimal"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

type OrderServiceTestSuite struct {
	suite.Suite
	mockCtrl        *gomock.Controller
	mockUOW         *uowmocks.MockUOW
	mockTX          *uowmocks.MockTX
	mockBalanceRepo *mocks.MockBalanceTransactionRepository
	mockOrderRepo   *mocks.MockOrderRepository
	orderService    *OrderService
}

func TestOrderServiceSuite(t *testing.T) {
	suite.Run(t, new(OrderServiceTestSuite))
}

func (s *OrderServiceTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockUOW = uowmocks.NewMockUOW(s.mockCtrl)
	s.mockOrderRepo = mocks.NewMockOrderRepository(s.mockCtrl)
	s.mockTX = uowmocks.NewMockTX(s.mockCtrl)
	s.mockBalanceRepo = mocks.NewMockBalanceTransactionRepository(s.mockCtrl)

	// Мок получения репозитория из uow. Выполняется в инициализации сервиса.
	s.mockUOW.EXPECT().GetRepository(uow.RepositoryName(repoargs.OrderRepoName)).
		Return(s.mockOrderRepo, nil).AnyTimes()

	// Инициализация сервиса.
	orderService, servErr := NewOrderService(s.mockUOW)
	s.Require().NoError(servErr)
	s.orderService = orderService
}

func (s *OrderServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *OrderServiceTestSuite) TestUpdateAccrual() {
	// Подготовка тестовых данных для репозитория
	updates := []repoargs.BatchUpdateWithAccrualData{
		{
			ID:      1,
			Status:  domain.OrderStatusProcessed,
			Accrual: decimal.NewFromInt(500),
		},
		{
			ID:      2,
			Status:  domain.OrderStatusProcessing,
			Accrual: decimal.Zero,
		},
	}

	var serviceUpdates = make([]UpdateAccrualArgs, len(updates))
	// конвертирование тестовых данных для сервисного слоя
	for i, update := range updates {
		serviceUpdates[i] = UpdateAccrualArgs{
			OrderID: update.ID,
			Status:  update.Status,
			Accrual: update.Accrual,
		}
	}

	updatedOrders := []domain.Order{
		{
			ID:        1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			UserID:    100,
			OrderCode: "ORDER-001",
			Status:    domain.OrderStatusProcessed,
			Accrual:   decimal.NewFromInt(500),
		},
		{
			ID:        2,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			UserID:    101,
			OrderCode: "ORDER-002",
			Status:    domain.OrderStatusProcessing,
			Accrual:   decimal.Zero,
		},
	}

	// Настраиваем мок для получения репозитория из транзакции
	s.mockTX.EXPECT().
		Get(uow.RepositoryName(repoargs.OrderRepoName)).
		Return(s.mockOrderRepo, nil)

	s.mockTX.EXPECT().
		Get(uow.RepositoryName(repoargs.BalanceTransactionRepoName)).
		Return(s.mockBalanceRepo, nil)

	// Настраиваем мок для batch обновления заказов
	s.mockOrderRepo.EXPECT().
		BatchUpdateWithAccrualData(gomock.Any(), updates, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ []repoargs.BatchUpdateWithAccrualData, fn repoargs.OrderBatchQueryRow) {
			for i, order := range updatedOrders {
				fn(i, &order, nil)
			}
		})

	// Настраиваем мок для создания транзакций баланса
	s.mockBalanceRepo.EXPECT().
		BatchCreate(gomock.Any(), gomock.Any(), gomock.Any()).
		Do(func(
			_ context.Context,
			btDTO []repoargs.BalanceTransactionCreate,
			_ repoargs.BalanceTransBatchQueryRow,
		) {
			// проверяем что в мок попали нужные данные.
			s.Len(btDTO, 1) // только одна запись с нужным статусом.
			s.Equal(domain.DirectionDebit, btDTO[0].Direction)
			s.NotNil(btDTO[0].OrderID)
			s.Equal(int64(1), btDTO[0].OrderID) // и id этой записи - 1.
		})

	// Настраиваем мок для выполнения транзакции
	s.mockUOW.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context, uow.TX) error) error {
			return fn(ctx, s.mockTX)
		})

	// Выполняем тестируемый метод
	err := s.orderService.UpdateAccrual(context.Background(), serviceUpdates)

	// Проверяем результат
	s.NoError(err)
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
		Accrual:   decimal.NewFromInt(0),
	}

	existingOrder := domain.Order{
		ID:        2,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    userID,
		OrderCode: existingOrderCode,
		Status:    domain.OrderStatusNew,
		Accrual:   decimal.NewFromInt(0),
	}

	// Мок транзакции uow.
	s.mockTX.EXPECT().Get(uow.RepositoryName(repoargs.OrderRepoName)).
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
				s.Require().ErrorAs(err, &t.wantErrType) //nolint:testifylint
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
			Accrual:   decimal.NewFromInt(0),
		},
		{
			ID:        2,
			CreatedAt: time.Now().Add(-time.Hour),
			UpdatedAt: time.Now(),
			UserID:    userID,
			OrderCode: "12345678904",
			Status:    domain.OrderStatusProcessed,
			Accrual:   decimal.NewFromInt(100),
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

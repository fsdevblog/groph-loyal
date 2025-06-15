package service

import (
	"context"
	"testing"
	"time"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/repository/repoargs"
	"github.com/fsdevblog/groph-loyal/internal/service/mocks"
	"github.com/fsdevblog/groph-loyal/pkg/uow"
	uowmocks "github.com/fsdevblog/groph-loyal/pkg/uow/mocks"
	"github.com/golang/mock/gomock"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type BalanceTransactionServiceTestSuite struct {
	suite.Suite
	mockCtrl      *gomock.Controller
	mockUOW       *uowmocks.MockUOW
	mockTX        *uowmocks.MockTX
	mockBlRepo    *mocks.MockBalanceTransactionRepository
	mockOrderRepo *mocks.MockOrderRepository
	service       *BalanceTransactionService
}

func TestBalanceTransactionServiceSuite(t *testing.T) {
	suite.Run(t, new(BalanceTransactionServiceTestSuite))
}

func (s *BalanceTransactionServiceTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockUOW = uowmocks.NewMockUOW(s.mockCtrl)
	s.mockTX = uowmocks.NewMockTX(s.mockCtrl)
	s.mockBlRepo = mocks.NewMockBalanceTransactionRepository(s.mockCtrl)
	s.mockOrderRepo = mocks.NewMockOrderRepository(s.mockCtrl)

	// Настроить возврат BalanceTransactionRepository в сервисе при инициализации
	s.mockUOW.EXPECT().
		GetRepository(uow.RepositoryName(repoargs.BalanceTransactionRepoName)).
		Return(s.mockBlRepo, nil).AnyTimes()

	var err error
	s.service, err = NewBalanceTransactionService(s.mockUOW)
	s.Require().NoError(err)
}

func (s *BalanceTransactionServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *BalanceTransactionServiceTestSuite) TestGetUserBalance() {
	debitAmount := decimal.NewFromInt(150) // всего начислений
	creditAmount := decimal.NewFromInt(20) // потрачено

	expected := &UserBalance{
		UserID:    123,
		Current:   debitAmount.Sub(creditAmount),
		Withdrawn: creditAmount,
	}

	s.mockBlRepo.EXPECT().
		GetUserBalance(gomock.Any(), expected.UserID).
		Return(&repoargs.BalanceAggregation{
			DebitAmount:  debitAmount,
			CreditAmount: creditAmount,
		}, nil)

	balance, err := s.service.GetUserBalance(s.T().Context(), expected.UserID)
	s.Require().NoError(err)

	// убеждаемся что баланс возвращается верный.
	s.Equal(expected.Current, balance.Current)
	s.Equal(expected.Withdrawn, balance.Withdrawn)
}

func (s *BalanceTransactionServiceTestSuite) TestWithdraw_EnoughBalance() {
	availableBalance := decimal.NewFromInt(10)

	// на баланса 10 баллов.
	balanceAgr := repoargs.BalanceAggregation{
		DebitAmount:  decimal.NewFromInt(100),
		CreditAmount: decimal.NewFromInt(90),
	}
	order := domain.Order{
		ID:        1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    123,
		OrderCode: "ORDER-001",
		Status:    domain.OrderStatusNew,
	}
	blTransaction := domain.BalanceTransaction{
		ID:        1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    order.UserID,
		OrderID:   order.ID,
		OrderCode: order.OrderCode,
		Amount:    availableBalance,
	}
	// Настраиваем мок для получения репозитория из транзакции
	s.mockTX.EXPECT().
		Get(uow.RepositoryName(repoargs.OrderRepoName)).
		Return(s.mockOrderRepo, nil).Times(2)

	s.mockTX.EXPECT().
		Get(uow.RepositoryName(repoargs.BalanceTransactionRepoName)).
		Return(s.mockBlRepo, nil).Times(2)

	// настраиваем мок для создания заказа
	s.mockOrderRepo.EXPECT().CreateOrder(gomock.Any(), order.UserID, order.OrderCode).
		Return(&order, nil).Times(2)

	// настраиваем мок, возвращающий баланс юзера
	s.mockBlRepo.EXPECT().GetUserBalance(gomock.Any(), order.UserID).
		Return(&balanceAgr, nil).Times(2)

	// настраиваем мок, создающий транзакцию баланса
	s.mockBlRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, args repoargs.BalanceTransactionCreate) (*domain.BalanceTransaction, error) {
			// убеждаемся что мок вызван с правильными данными.
			s.Equal(order.UserID, args.UserID)
			s.Equal(order.ID, args.OrderID)
			s.Equal(order.OrderCode, args.OrderCode)
			s.Equal(domain.DirectionCredit, args.Direction)
			s.Equal(availableBalance, args.Amount)
			return &blTransaction, nil
		}).Times(1)

	// настраиваем мок UOW обертку.
	s.mockUOW.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, fn func(context.Context, uow.TX) error) error {
			return fn(s.T().Context(), s.mockTX)
		},
	).Times(2)

	cases := []struct {
		accrual decimal.Decimal
		name    string
		wantErr error
	}{
		{name: "ok", accrual: availableBalance, wantErr: nil},
		{
			name:    "not enough balance",
			accrual: availableBalance.Add(decimal.NewFromFloat(0.001)),
			wantErr: domain.ErrNotEnoughBalance,
		},
	}

	for _, t := range cases {
		s.Run(t.name, func() {
			result, err := s.service.Withdraw(s.T().Context(), order.UserID, order.OrderCode, t.accrual)
			if t.wantErr != nil {
				s.Require().ErrorIs(err, t.wantErr)
				return
			}
			s.Require().NoError(err)
			s.NotNil(result)
		})
	}
}

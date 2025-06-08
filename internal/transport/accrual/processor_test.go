package accrual

import (
	"context"

	"github.com/fsdevblog/groph-loyal/internal/service"
	"github.com/fsdevblog/groph-loyal/internal/transport/accrual/client"

	"github.com/shopspring/decimal"

	"net/http"
	"testing"
	"time"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/transport/accrual/mocks"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

type ProcessorTestSuite struct {
	suite.Suite
	processor      *Processor
	mockHTTPClient *mocks.MockClient
	mockService    *mocks.MockServicer
	ctrl           *gomock.Controller
}

func (s *ProcessorTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())

	s.mockHTTPClient = mocks.NewMockClient(s.ctrl)
	s.mockService = mocks.NewMockServicer(s.ctrl)

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	s.processor = NewProcessor(s.mockService, "", logger)
	s.processor.client = s.mockHTTPClient
}

func (s *ProcessorTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func TestProcessorSuite(t *testing.T) {
	suite.Run(t, new(ProcessorTestSuite))
}

// TestProcess_NoOrders Тест на случай, когда нет заказов для обработки.
func (s *ProcessorTestSuite) TestProcess_NoOrders() {
	s.mockService.EXPECT().
		OrdersForAccrualMonitoring(gomock.Any(), s.processor.limitPerIteration).
		Return([]domain.Order{}, ErrNoOrders)

	err := s.processor.process(s.T().Context())

	s.ErrorIs(err, ErrNoOrders)
}

// TestProcess_ErrorAccrualReq Тест на случай, когда есть заказы, но ошибка при получении информации о начислениях.
func (s *ProcessorTestSuite) TestProcess_ErrorAccrualReq() {
	// Создаем тестовые данные
	testOrders := []domain.Order{
		{ID: 1, OrderCode: "ORDER-001", UserID: 100, Status: domain.OrderStatusNew},
		{ID: 2, OrderCode: "ORDER-002", UserID: 101, Status: domain.OrderStatusNew},
	}

	// Настраиваем мок-сервис для возврата тестовых заказов.
	s.mockService.EXPECT().
		OrdersForAccrualMonitoring(gomock.Any(), s.processor.limitPerIteration).
		Return(testOrders, nil)

	// Настраиваем мок-хттп-клиент для имитации ошибок при получении информации о начислениях.
	internalError := client.NewStatusCodeError(http.StatusInternalServerError)
	noContentError := client.NewStatusCodeError(http.StatusNoContent)

	s.mockHTTPClient.EXPECT().
		GetOrderAccrual(gomock.Any(), "ORDER-001").
		Return(nil, internalError)
	s.mockHTTPClient.EXPECT().
		GetOrderAccrual(gomock.Any(), "ORDER-002").
		Return(nil, noContentError)

	// Настраиваем мок-сервис для обновления статуса заказа.
	s.mockService.EXPECT().
		UpdateAccrual(gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, updates []service.UpdateAccrualArgs) {
			// Убеждаемся что ошибки были отправлены в сервис
			s.Require().Len(updates, 2)
			s.Error(updates[0].Error) //nolint:testifylint
			s.Error(updates[1].Error) //nolint:testifylint
		}).Return(nil)

	ctx, cancel := context.WithTimeout(s.T().Context(), time.Second)
	defer cancel()
	err := s.processor.process(ctx)

	// Проверяем результаты
	s.NoError(err)
}

// TestProcess_Success Тест на успешную обработку заказов.
func (s *ProcessorTestSuite) TestProcess_Success() {
	// Создаем тестовые данные
	testOrders := []domain.Order{
		{ID: 1, OrderCode: "ORDER-001", UserID: 100, Status: domain.OrderStatusNew},
		{ID: 2, OrderCode: "ORDER-002", UserID: 101, Status: domain.OrderStatusNew},
	}

	accrualResponses := []*client.Response{
		{OrderCode: "ORDER-001", Status: "PROCESSED", Accrual: decimal.NewFromInt(500)},
		{OrderCode: "ORDER-002", Status: "PROCESSING"},
	}

	// Настраиваем мок-сервис для возврата тестовых заказов.
	s.mockService.EXPECT().
		OrdersForAccrualMonitoring(gomock.Any(), s.processor.limitPerIteration).
		Return(testOrders, nil)

	// Настраиваем мок-хттп-клиент для возврата тестовых ответов.
	s.mockHTTPClient.EXPECT().
		GetOrderAccrual(gomock.Any(), "ORDER-001").
		Return(accrualResponses[0], nil)
	s.mockHTTPClient.EXPECT().
		GetOrderAccrual(gomock.Any(), "ORDER-002").
		Return(accrualResponses[1], nil)

	// Ожидаем вызов обновления с правильными данными.
	s.mockService.EXPECT().
		UpdateAccrual(gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, updates []service.UpdateAccrualArgs) {
			s.Require().Len(updates, 2)

			// Проверяем обновления.
			var foundFirstUpdate bool
			var foundSecondUpdate bool

			for _, update := range updates {
				if update.OrderID == 1 {
					s.Equal(domain.OrderStatusProcessed, update.Status)
					s.Equal(decimal.NewFromInt(500), update.Accrual)
					foundFirstUpdate = true
				}

				if update.OrderID == 2 {
					s.Equal(domain.OrderStatusProcessing, update.Status)
					s.True(update.Accrual.IsZero())
					foundSecondUpdate = true
				}
			}

			s.Truef(foundFirstUpdate, "Не найдено обновление для заказа с ID=%d", 1)
			s.Truef(foundSecondUpdate, "Не найдено обновление для заказа с ID=%d", 2)
		}).
		Return(nil)

	ctx, cancel := context.WithTimeout(s.T().Context(), time.Second)
	defer cancel()
	err := s.processor.process(ctx)
	s.NoError(err)
}

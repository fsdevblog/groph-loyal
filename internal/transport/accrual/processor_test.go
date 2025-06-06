package accrual

import (
	"context"

	"github.com/shopspring/decimal"

	"net/http"
	"testing"
	"time"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/transport/accrual/dto"
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
	logger.SetLevel(logrus.FatalLevel)

	s.processor = NewProcessor(s.mockService, logger)
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

	s.NoError(err)
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
	testError := NewStatusCodeError(http.StatusInternalServerError)
	s.mockHTTPClient.EXPECT().
		GetOrderAccrual(gomock.Any(), "ORDER-001").
		Return(nil, testError)
	s.mockHTTPClient.EXPECT().
		GetOrderAccrual(gomock.Any(), "ORDER-002").
		Return(nil, testError)

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

	accrualResponses := []*dto.OrderAccrualResponse{
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
		UpdateOrdersWithAccrual(gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, updates []domain.OrderAccrualUpdateDTO) {
			s.Require().Len(updates, 2)

			// Проверяем обновления.
			var foundFirstUpdate bool
			var foundSecondUpdate bool

			for _, update := range updates {
				if update.ID == 1 {
					s.Equal(domain.OrderStatusProcessed, update.Status)
					s.Equal(decimal.NewFromInt(500), update.Accrual)
					foundFirstUpdate = true
				}

				if update.ID == 2 {
					s.Equal(domain.OrderStatusProcessing, update.Status)
					s.True(update.Accrual.IsZero())
					foundSecondUpdate = true
				}
			}

			s.True(foundFirstUpdate, "Не найдено обновление для заказа с ID=1")
			s.True(foundSecondUpdate, "Не найдено обновление для заказа с ID=2")
		}).
		Return(nil)

	ctx, cancel := context.WithTimeout(s.T().Context(), time.Second)
	defer cancel()
	err := s.processor.process(ctx)
	s.NoError(err)
}

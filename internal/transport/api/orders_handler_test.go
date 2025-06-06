package api

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/fsdevblog/groph-loyal/internal/config"
	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/logger"
	"github.com/fsdevblog/groph-loyal/internal/transport/api/mocks"
	"github.com/fsdevblog/groph-loyal/internal/transport/api/testutils"
	"github.com/fsdevblog/groph-loyal/internal/transport/api/tokens"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

type OrderHandlerTestSuite struct {
	suite.Suite
	router           *gin.Engine
	config           *config.Config
	mockOrderService *mocks.MockOrderServicer
	jwtSecret        []byte
}

func TestOrderHandlerSuite(t *testing.T) {
	suite.Run(t, new(OrderHandlerTestSuite))
}

func (s *OrderHandlerTestSuite) SetupTest() {
	mockCtrl := gomock.NewController(s.T())
	defer mockCtrl.Finish()

	s.config = &config.Config{
		RunAddress: "localhost:80",
	}

	s.mockOrderService = mocks.NewMockOrderServicer(mockCtrl)
	s.jwtSecret = []byte("super secret key")

	s.router = New(RouterArgs{
		Logger:       logger.New(os.Stdout),
		OrderService: s.mockOrderService,
		JWTSecretKey: s.jwtSecret,
	})
}

func (s *OrderHandlerTestSuite) TestCreateOrder() {
	var currentUserID int64 = 1
	var anotherUserID int64 = 2

	currentUserJWTToken, cJWTTokenErr := tokens.GenerateUserJWT(currentUserID, time.Hour, s.jwtSecret)
	s.Require().NoError(cJWTTokenErr)

	anotherUserJWTToken, aJWTTokenErr := tokens.GenerateUserJWT(anotherUserID, time.Hour, s.jwtSecret)
	s.Require().NoError(aJWTTokenErr)

	validPayload := []byte("12345678903")
	existingPayload := []byte("79927398713")
	invalidPayload := []byte("12345678")

	// Моки
	// Валидный запрос
	s.mockOrderService.EXPECT().
		Create(gomock.Any(), currentUserID, string(validPayload)).
		Return(&domain.Order{}, nil).Times(1)
	// Текущий юзер уже загрузил данный ордер.
	s.mockOrderService.EXPECT().
		Create(gomock.Any(), currentUserID, string(existingPayload)).
		Return(nil, domain.NewDuplicateOrderError(&domain.Order{UserID: currentUserID})).Times(1)
	// Кто-то другой уже загрузил данный ордер.
	s.mockOrderService.EXPECT().
		Create(gomock.Any(), anotherUserID, string(existingPayload)).
		Return(nil, domain.NewDuplicateOrderError(&domain.Order{UserID: 123123})).Times(1)
	// Ожидаем что мок не будет вызван.
	s.mockOrderService.EXPECT().
		Create(gomock.Any(), currentUserID, string(invalidPayload)).
		Times(0)

	cases := []struct {
		name        string
		payload     []byte
		wantStatus  int
		jwtToken    string
		contentType string
	}{
		{
			name:        "all ok",
			payload:     validPayload,
			wantStatus:  http.StatusAccepted,
			jwtToken:    currentUserJWTToken,
			contentType: "text/plain; charset=utf-8",
		}, {
			name:        "present by current user",
			payload:     existingPayload,
			wantStatus:  http.StatusOK,
			jwtToken:    currentUserJWTToken,
			contentType: "text/plain; charset=utf-8",
		}, {
			name:        "present by another user",
			payload:     existingPayload,
			wantStatus:  http.StatusConflict,
			jwtToken:    anotherUserJWTToken,
			contentType: "text/plain; charset=utf-8",
		}, {
			name:        "not authorized",
			payload:     validPayload,
			wantStatus:  http.StatusUnauthorized,
			contentType: "text/plain; charset=utf-8",
		}, {
			name:        "invalid payload",
			payload:     invalidPayload,
			wantStatus:  http.StatusUnprocessableEntity,
			jwtToken:    currentUserJWTToken,
			contentType: "text/plain; charset=utf-8",
		}, {
			name:        "bad request",
			payload:     []byte(""),
			wantStatus:  http.StatusBadRequest,
			jwtToken:    currentUserJWTToken,
			contentType: "application/json; charset=utf-8",
		},
	}
	for _, t := range cases {
		s.Run(t.name, func() {
			args := testutils.RequestArgs{
				Router: s.router,
				Method: http.MethodPost,
				URL:    RouteGroup + OrdersRoute,
				Body:   bytes.NewReader(t.payload),
			}
			var reqOpts []func(*testutils.RequestOptions)
			if t.jwtToken != "" {
				authHeader := fmt.Sprintf("Bearer %s", t.jwtToken)
				reqOpts = append(reqOpts, testutils.WithHeader("Authorization", authHeader))
			}
			reqOpts = append(reqOpts, testutils.WithHeader("Content-Type", t.contentType))
			res, err := testutils.MakeRequest(args, reqOpts...)

			defer func() {
				closeErr := res.Body.Close()
				s.Require().NoError(closeErr)
			}()

			s.Require().NoError(err)
			s.Equal(t.wantStatus, res.StatusCode)
		})
	}
}

func (s *OrderHandlerTestSuite) TestIndex() {
	var userID int64 = 1
	var noOrdersUserID int64 = 2

	userJWTToken, uJWTErr := tokens.GenerateUserJWT(userID, time.Hour, s.jwtSecret)
	s.Require().NoError(uJWTErr)
	userNoOrdersJWTToken, uNoOrdersJWTErr := tokens.GenerateUserJWT(noOrdersUserID, time.Hour, s.jwtSecret)
	s.Require().NoError(uNoOrdersJWTErr)

	orders := []domain.Order{
		{
			ID:        1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			UserID:    userID,
			OrderCode: "11111111",
			Status:    domain.OrderStatusNew,
			Accrual:   decimal.NewFromInt(0),
		},
	}
	s.mockOrderService.EXPECT().GetByUserID(gomock.Any(), userID).Return(orders, nil)
	s.mockOrderService.EXPECT().GetByUserID(gomock.Any(), noOrdersUserID).Return([]domain.Order{}, nil)

	cases := []struct {
		name       string
		jwtToken   string
		wantStatus int
	}{
		{
			name:       "all ok",
			jwtToken:   userJWTToken,
			wantStatus: http.StatusOK,
		}, {
			name:       "not authorized",
			jwtToken:   "",
			wantStatus: http.StatusUnauthorized,
		}, {
			name:       "no orders",
			jwtToken:   userNoOrdersJWTToken,
			wantStatus: http.StatusNoContent,
		},
	}
	for _, t := range cases {
		s.Run(t.name, func() {
			args := testutils.RequestArgs{
				Router: s.router,
				Method: http.MethodGet,
				URL:    RouteGroup + OrdersRoute,
			}
			var reqOpts []func(*testutils.RequestOptions)
			if t.jwtToken != "" {
				authHeader := fmt.Sprintf("Bearer %s", t.jwtToken)
				reqOpts = append(reqOpts, testutils.WithHeader("Authorization", authHeader))
			}
			res, err := testutils.MakeRequest(args, reqOpts...)
			defer func() {
				closeErr := res.Body.Close()
				s.Require().NoError(closeErr)
			}()

			s.Require().NoError(err)
			s.Equal(t.wantStatus, res.StatusCode)
		})
	}
}

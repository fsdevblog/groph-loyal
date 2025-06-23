package client

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/stretchr/testify/suite"
)

type ClientTestSuite struct {
	suite.Suite
	server *httptest.Server
}

func TestClientSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (s *ClientTestSuite) TearDownTest() {
	if s.server != nil {
		s.server.Close()
	}
}

// TestGetOrderAccrual_Success Тест на успешный ответ с начисленными баллами.
func (s *ClientTestSuite) TestGetOrderAccrual() { //nolint:gocognit
	type tcase struct {
		name         string
		orderCode    string
		httpStatus   int
		wantResponse *Response
		wantErrType  error
	}

	cases := []tcase{
		{
			name:       "valid request",
			orderCode:  "11111111",
			httpStatus: http.StatusOK,
			wantResponse: &Response{
				OrderCode: "11111111",
				Status:    StatusProcessed,
				Accrual:   decimal.NewFromInt(500),
			},
		}, {
			name:         "no content",
			orderCode:    "11111112",
			httpStatus:   http.StatusNoContent,
			wantResponse: nil,
			wantErrType:  new(StatusCodeError),
		}, {
			name:         "too many requests",
			orderCode:    "11111113",
			httpStatus:   http.StatusTooManyRequests,
			wantResponse: nil,
			wantErrType:  new(TooManyRequestError),
		}, {
			name:         "internal error",
			orderCode:    "11111114",
			httpStatus:   http.StatusInternalServerError,
			wantResponse: nil,
			wantErrType:  new(StatusCodeError),
		},
	}

	// хендлер для тестового сервера. В зависимости от пути запроса определяет тот или иной кейс и выдает
	// тот или иной ответ.
	serverHandler := func() func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			// подбираем кейс, чтоб выдать ожидаемый ответ.
			var rc *tcase
			for _, c := range cases {
				code, exist := strings.CutPrefix(r.URL.Path, "/api/orders/")
				s.Require().True(exist) //nolint:testifylint
				if code == c.orderCode {
					rc = &c
					break
				}
			}
			s.Require().NotNilf(rc, "тест для пути %s не найден", r.URL.Path) //nolint:testifylint

			var body []byte
			if rc.httpStatus == http.StatusOK {
				w.Header().Set("Content-Type", "application/json")
				var bErr error
				body, bErr = json.Marshal(rc.wantResponse)
				s.NoError(bErr)
			}
			w.WriteHeader(rc.httpStatus)

			if body != nil {
				_, wErr := w.Write(body)
				s.NoError(wErr)
			}
		}
	}

	s.server = httptest.NewServer(http.HandlerFunc(serverHandler()))

	var statusCodeError *StatusCodeError
	var tooManyRequestError *TooManyRequestError

	for _, t := range cases {
		s.Run(t.name, func() {
			client := New(s.server.URL)
			response, err := client.GetOrderAccrual(s.T().Context(), t.orderCode)

			if t.wantErrType != nil {
				s.Require().Error(err)
				switch {
				case errors.As(t.wantErrType, &statusCodeError):
					s.Require().ErrorAs(err, &statusCodeError)
				case errors.As(t.wantErrType, &tooManyRequestError):
					s.Require().ErrorAs(err, &tooManyRequestError)
				default:
					s.FailNow("unexpected err type")
				}
				return
			}

			s.Require().NoError(err)
			s.NotNil(response)
			s.Equal(t.wantResponse, response)
		})
	}
}

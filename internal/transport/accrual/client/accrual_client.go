package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"io"
	"net/http"
)

const RouteOrderAccrual = "/api/orders/%s"

// Константы минимального и максимально значения в заголовке Retry-After.
const (
	minRetryAfter = 1
	maxRetryAfter = 120
)

type StatusType string

const (
	StatusRegistered StatusType = "REGISTERED"
	StatusInvalid    StatusType = "INVALID"
	StatusProcessing StatusType = "PROCESSING"
	StatusProcessed  StatusType = "PROCESSED"
)

type Response struct {
	Status    StatusType      `json:"status"`
	OrderCode string          `json:"order"`
	Accrual   decimal.Decimal `json:"accrual,omitempty"`
}

// HTTPClient является реализацией интерфейса Client для HTTP запросов к accrual.
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) HTTPClient {
	return HTTPClient{
		baseURL:    baseURL,
		httpClient: http.DefaultClient,
	}
}

// GetOrderAccrual получает информацию о начислении баллов для заказа.
// При ответе сервера со статусом отличным от http.StatusOK, возвращает ошибку StatusCodeErr, или
// TooManyRequestError в случае http.StatusTooManyRequests.
//
//nolint:nonamedreturns
func (c HTTPClient) GetOrderAccrual(
	ctx context.Context,
	orderCode string,
) (response *Response, err error) {
	// Формируем URL запроса.
	url := c.baseURL + fmt.Sprintf(RouteOrderAccrual, orderCode)

	// Создаем запрос.
	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if reqErr != nil {
		return nil, fmt.Errorf("create request: %s", reqErr.Error())
	}

	// Выполняем запрос.
	resp, doErr := c.httpClient.Do(req)
	if doErr != nil {
		return nil, fmt.Errorf("do request: %s", doErr.Error())
	}

	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()

	if resp.StatusCode == http.StatusTooManyRequests {
		minValue := decimal.NewFromInt(minRetryAfter)
		maxValue := decimal.NewFromInt(maxRetryAfter)

		retryAfterStr := resp.Header.Get("Retry-After")

		retryAfter, parseErr := decimal.NewFromString(retryAfterStr)
		if parseErr != nil || retryAfter.LessThan(minValue) || retryAfter.GreaterThan(maxValue) {
			// в случае ошибки или неверных данных ставим 60 секунд
			retryAfter = decimal.NewFromInt(60) //nolint:mnd
		}

		ra := time.Duration(retryAfter.IntPart()) * time.Second
		return nil, NewTooManyRequestError(ra)
	}

	// Статус отличный от http.StatusOK нас не интересует.
	if resp.StatusCode != http.StatusOK {
		err = NewStatusCodeError(resp.StatusCode)
		return nil, err
	}

	// Парсим успешный ответ.
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		err = fmt.Errorf("read response: %s", readErr.Error())
		return nil, err
	}

	if jsonErr := json.Unmarshal(body, &response); jsonErr != nil {
		err = fmt.Errorf("parse response: %s", jsonErr.Error())
		return nil, err
	}

	return response, nil
}

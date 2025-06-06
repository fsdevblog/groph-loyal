package accrual

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/fsdevblog/groph-loyal/internal/transport/accrual/dto"

	"io"
	"net/http"
)

const RouteOrderAccrual = "/api/orders/%s"

// HTTPClient является реализацией интерфейса Client для HTTP запросов к accrual.
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHTTPClient(baseURL string) HTTPClient {
	return HTTPClient{
		baseURL:    baseURL,
		httpClient: http.DefaultClient,
	}
}

// GetOrderAccrual получает информацию о начислении баллов для заказа. В случае ошибки возвращает или StatusCodeErr
// или не типизированную ошибку.
//
//nolint:nonamedreturns
func (c HTTPClient) GetOrderAccrual(
	ctx context.Context,
	orderCode string,
) (response *dto.OrderAccrualResponse, err error) {
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

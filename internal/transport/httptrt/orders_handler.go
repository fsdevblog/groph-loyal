package httptrt

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/gin-gonic/gin"
)

const (
	// maxOrderCodeLength при увеличении значения константы, нужно выполнить миграцию на увеличение
	// максимальной длины поля order_code.
	maxOrderCodeLength = 20
)

type OrdersHandler struct {
	orderSvs OrderServicer
}

func NewOrdersHandler(orderSvs OrderServicer) *OrdersHandler {
	return &OrdersHandler{
		orderSvs: orderSvs,
	}
}

func (o *OrdersHandler) Create(c *gin.Context) {
	currentUserID := getUserIDFromContext(c)

	if !strings.Contains(c.ContentType(), "text/plain") {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	body, err := c.GetRawData()
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err).SetType(gin.ErrorTypePrivate)
		return
	}
	orderCode := string(body)

	if len(orderCode) > maxOrderCodeLength || !isValidLuhn(orderCode) {
		c.AbortWithStatus(http.StatusUnprocessableEntity)
		return
	}

	reqCtx, cancel := context.WithTimeout(c, DefaultServiceTimeout)
	defer cancel()

	order, createErr := o.orderSvs.Create(reqCtx, currentUserID, orderCode)
	if createErr != nil {
		var duplicateErr *domain.DuplicateOrderError

		if errors.As(createErr, &duplicateErr) {
			// В зависимости от принадлежности Order'а текущему юзеру, возвращаем тот или иной http статус.
			var statusCode = http.StatusConflict
			if duplicateErr.Order.UserID == currentUserID {
				statusCode = http.StatusOK
			}
			c.AbortWithStatus(statusCode)
			return
		}
		_ = c.AbortWithError(http.StatusInternalServerError, createErr).
			SetType(gin.ErrorTypePrivate)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"order": order})
}

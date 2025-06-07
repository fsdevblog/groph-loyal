package api

import (
	"context"
	"errors"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"net/http"
)

type BalanceHandler struct {
	svs BalanceServicer
}

func NewBalanceHandler(svs BalanceServicer) *BalanceHandler {
	return &BalanceHandler{
		svs: svs,
	}
}

type BalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

func (b *BalanceHandler) Index(c *gin.Context) {
	currentUserID := getUserIDFromContext(c)

	reqCtx, cancel := context.WithTimeout(c, DefaultServiceTimeout)
	defer cancel()

	balance, err := b.svs.GetUserBalance(reqCtx, currentUserID)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err).SetType(gin.ErrorTypePrivate)
		return
	}

	c.JSON(http.StatusOK, &BalanceResponse{
		Current:   balance.Current.InexactFloat64(),
		Withdrawn: balance.Withdrawn.InexactFloat64(),
	})
}

type WithdrawParams struct {
	OrderCode string          `json:"order"`
	Amount    decimal.Decimal `json:"sum"`
}

func (b *BalanceHandler) Withdraw(c *gin.Context) {
	currentUserID := getUserIDFromContext(c)

	var params WithdrawParams
	if bindErr := c.ShouldBindJSON(&params); bindErr != nil {
		_ = c.AbortWithError(http.StatusBadRequest, bindErr).SetType(gin.ErrorTypeBind)
		return
	}

	if len(params.OrderCode) > maxOrderCodeLength || !isValidLuhn(params.OrderCode) {
		c.AbortWithStatus(http.StatusUnprocessableEntity)
		return
	}

	reqCtx, cancel := context.WithTimeout(c, DefaultServiceTimeout)
	defer cancel()

	_, err := b.svs.Withdraw(reqCtx, currentUserID, params.OrderCode, params.Amount)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotEnoughBalance):
			c.AbortWithStatus(http.StatusPaymentRequired)
		default:
			_ = c.AbortWithError(http.StatusInternalServerError, err).SetType(gin.ErrorTypePrivate)
		}
		return
	}

	c.AbortWithStatus(http.StatusOK)
}

package api

import (
	"github.com/gin-gonic/gin"
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
	Current   float64 `json:"balance"`
	Withdrawn float64 `json:"withdrawn"`
}

func (b *BalanceHandler) Index(c *gin.Context) {
	currentUserID := getUserIDFromContext(c)

	balance, err := b.svs.GetUserBalance(c, currentUserID)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err).SetType(gin.ErrorTypePrivate)
		return
	}

	c.JSON(http.StatusOK, BalanceResponse{
		Current:   balance.DebitAmount.InexactFloat64(),
		Withdrawn: balance.CreditAmount.InexactFloat64(),
	})
}

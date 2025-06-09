package repoargs

import "github.com/shopspring/decimal"

type CreateUser struct {
	Username string
	Password string
}

type BalanceAggregation struct {
	DebitAmount  decimal.Decimal
	CreditAmount decimal.Decimal
}

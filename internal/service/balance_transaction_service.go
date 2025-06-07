package service

import (
	"context"
	"fmt"

	"github.com/fsdevblog/groph-loyal/internal/repository/repoargs"

	"github.com/shopspring/decimal"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/pkg/uow"
)

type BalanceTransactionService struct {
	uow    uow.UOW
	blRepo BalanceTransactionRepository
}

func NewBalanceTransactionService(u uow.UOW) (*BalanceTransactionService, error) {
	rName := uow.RepositoryName(repoargs.BalanceTransactionRepoName)
	blRepo, blRepoErr := uow.GetRepositoryAs[BalanceTransactionRepository](u, rName)
	if blRepoErr != nil {
		return nil, blRepoErr
	}
	return &BalanceTransactionService{
		uow:    u,
		blRepo: blRepo,
	}, nil
}

type UserBalance struct {
	UserID    int64
	Current   decimal.Decimal
	Withdrawn decimal.Decimal
}

func (b *BalanceTransactionService) GetUserBalance(
	ctx context.Context,
	userID int64,
) (*UserBalance, error) {
	balance, err := b.blRepo.GetUserBalance(ctx, userID)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	return &UserBalance{
		UserID:    userID,
		Current:   balance.DebitAmount.Sub(balance.CreditAmount),
		Withdrawn: balance.CreditAmount,
	}, nil
}

// Withdraw создает ордер и в счет оплаты которого списываются балы. Затем собственно списывает.
func (b *BalanceTransactionService) Withdraw(
	ctx context.Context,
	userID int64,
	orderCode string,
	amount decimal.Decimal,
) (*domain.BalanceTransaction, error) {
	var bl *domain.BalanceTransaction
	txErr := b.uow.Do(ctx, func(c context.Context, tx uow.TX) error {
		orderRepo, orderRepoErr := uow.GetAs[OrderRepository](tx, uow.RepositoryName(repoargs.OrderRepoName))

		if orderRepoErr != nil {
			return orderRepoErr //nolint:wrapcheck
		}
		order, orderErr := orderRepo.CreateOrder(c, userID, orderCode)
		if orderErr != nil {
			return orderErr //nolint:wrapcheck
		}

		// проверяем баланс юзера
		blRName := uow.RepositoryName(repoargs.BalanceTransactionRepoName)
		blRepo, blRepoErr := uow.GetAs[BalanceTransactionRepository](tx, blRName)

		if blRepoErr != nil {
			return blRepoErr //nolint:wrapcheck
		}

		balance, balanceErr := blRepo.GetUserBalance(ctx, userID)
		if balanceErr != nil {
			return balanceErr //nolint:wrapcheck
		}
		balanceSum := balance.DebitAmount.Sub(balance.CreditAmount)
		if balanceSum.LessThan(amount) {
			return domain.ErrNotEnoughBalance
		}

		// создаем транзакцию credit.
		var createErr error
		bl, createErr = blRepo.Create(c, repoargs.BalanceTransactionCreate{
			UserID:    userID,
			OrderID:   order.ID,
			Direction: domain.DirectionCredit,
			Amount:    amount,
		})
		if createErr != nil {
			return createErr //nolint:wrapcheck
		}
		return nil
	})

	if txErr != nil {
		return nil, fmt.Errorf("creating balance transaction: %w", txErr)
	}

	return bl, nil
}

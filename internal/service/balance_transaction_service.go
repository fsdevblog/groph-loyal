package service

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/pkg/uow"
)

type BalanceTransactionService struct {
	uow    uow.UOW
	blRepo domain.BalanceTransactionRepository
}

func NewBalanceTransactionService(u uow.UOW) (*BalanceTransactionService, error) {
	rName := uow.RepositoryName(domain.BalanceTransactionRepoName)
	blRepo, blRepoErr := uow.GetRepositoryAs[domain.BalanceTransactionRepository](u, rName)
	if blRepoErr != nil {
		return nil, blRepoErr
	}
	return &BalanceTransactionService{
		uow:    u,
		blRepo: blRepo,
	}, nil
}

func (b *BalanceTransactionService) GetUserBalance(
	ctx context.Context,
	userID int64,
) (*domain.UserBalanceSumDTO, error) {
	balance, err := b.blRepo.GetUserBalance(ctx, userID)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	balance.DebitAmount = balance.DebitAmount.Sub(balance.CreditAmount)
	return balance, nil
}

func (b *BalanceTransactionService) Withdraw(
	ctx context.Context,
	userID int64,
	orderCode string,
	amount decimal.Decimal,
) (*domain.BalanceTransaction, error) {

	var bl *domain.BalanceTransaction
	txErr := b.uow.Do(ctx, func(c context.Context, tx uow.TX) error {
		// проверяем баланс юзера

		blRepo, blRepoErr :=
			uow.GetAs[domain.BalanceTransactionRepository](tx, uow.RepositoryName(domain.BalanceTransactionRepoName))

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

		// получаем ID ордера
		orderRepo, orderRepoErr := uow.GetAs[domain.OrderRepository](tx, uow.RepositoryName(domain.OrderRepoName))
		if orderRepoErr != nil {
			return orderRepoErr //nolint:wrapcheck
		}
		order, orderErr := orderRepo.FindByOrderCode(c, orderCode)
		if orderErr != nil {
			return orderErr //nolint:wrapcheck
		}

		if order.UserID != userID {
			return domain.ErrOwnerConflict
		}

		// создаем транзакцию credit.
		var createErr error
		bl, createErr = blRepo.Create(c, domain.BalanceTransactionCreateDTO{
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

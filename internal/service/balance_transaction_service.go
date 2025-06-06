package service

import (
	"context"

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

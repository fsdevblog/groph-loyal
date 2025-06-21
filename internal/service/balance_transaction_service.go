package service

import (
	"context"
	"fmt"

	"github.com/fsdevblog/groph-loyal/internal/repository/repoargs"

	"github.com/shopspring/decimal"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/pkg/uow"
)

// BalanceTransactionService управляет операциями над балансом пользователя.
type BalanceTransactionService struct {
	uow    uow.UOW
	blRepo BalanceTransactionRepository
}

// NewBalanceTransactionService создает новый сервис для транзакций баланса.
func NewBalanceTransactionService(u uow.UOW) (*BalanceTransactionService, error) {
	rName := uow.RepositoryName(repoargs.BalanceTransactionRepoName)
	blRepo, errBlRepo := uow.GetRepositoryAs[BalanceTransactionRepository](u, rName)
	if errBlRepo != nil {
		return nil, errBlRepo
	}
	return &BalanceTransactionService{
		uow:    u,
		blRepo: blRepo,
	}, nil
}

// UserBalance отражает текущее состояние баланса пользователя:
// сколько всего начислено, сколько списано, и итоговая сумма на счету.
type UserBalance struct {
	UserID    int64
	Current   decimal.Decimal
	Withdrawn decimal.Decimal
}

// GetUserBalance возвращает агрегированную информацию о балансе пользователя:
//   - текущий баланс (Current),
//   - сумма всех списаний (Withdrawn).
//
// В случае неудачи возвращает обернутую ошибку.
// Если пользователь не найден, вернется ошибка domain.ErrRecordNotFound.
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

// Withdraw реализует процедуру списания баллов пользователя через создание заказа в счет которого списываются баллы:
// - Сначала создаётся новый заказ на списание,
// - Затем проверяется, хватает ли средств на счету,
// - Если хвататет — регистрируется транзакция на списание (credit).
//
// Операция проводятся в рамках транзакции.
// В случае недостаточного баланса возвращает ошибку domain.ErrNotEnoughBalance.
// При успешном исполнении возвращает созданную транзакцию баланса.
func (b *BalanceTransactionService) Withdraw(
	ctx context.Context,
	userID int64,
	orderCode string,
	amount decimal.Decimal,
) (*domain.BalanceTransaction, error) {
	var bl *domain.BalanceTransaction
	transFn := func(c context.Context, tx uow.TX) error {
		orderRepo, errOrderRepo := uow.GetAs[OrderRepository](tx, uow.RepositoryName(repoargs.OrderRepoName))
		if errOrderRepo != nil {
			return errOrderRepo //nolint:wrapcheck
		}

		// Создаем заказ, в счет которого будут списываться баллы.
		order, errOrder := orderRepo.CreateOrder(c, userID, orderCode)
		if errOrder != nil {
			return errOrder //nolint:wrapcheck
		}

		blRName := uow.RepositoryName(repoargs.BalanceTransactionRepoName)
		blRepo, errBlRepo := uow.GetAs[BalanceTransactionRepository](tx, blRName)
		if errBlRepo != nil {
			return errBlRepo //nolint:wrapcheck
		}

		// проверяем баланс юзера
		balance, errBalance := blRepo.GetUserBalance(ctx, userID)
		if errBalance != nil {
			return errBalance //nolint:wrapcheck
		}
		balanceSum := balance.DebitAmount.Sub(balance.CreditAmount)
		if balanceSum.LessThan(amount) {
			return domain.ErrNotEnoughBalance
		}

		// создаем транзакцию credit.
		var errCreate error
		bl, errCreate = blRepo.Create(c, repoargs.BalanceTransactionCreate{
			UserID:    userID,
			OrderID:   order.ID,
			OrderCode: orderCode,
			Direction: domain.DirectionCredit,
			Amount:    amount,
		})
		if errCreate != nil {
			return errCreate //nolint:wrapcheck
		}
		return nil
	}

	var maxRetriesWithIsolation uint = 3
	errTx := b.uow.DoWithIsolation(ctx, transFn, uow.IsolationSerializable, maxRetriesWithIsolation)
	if errTx != nil {
		return nil, fmt.Errorf("creating balance transaction: %w", errTx)
	}

	return bl, nil
}

// GetByDirection возвращает все транзакции пользователя по выбранному направлению:
// дебетовые (начисления) или кредитовые (списания).
// В случае ошибки — возвращает обернутую ошибку.
func (b *BalanceTransactionService) GetByDirection(
	ctx context.Context,
	userID int64,
	direction domain.DirectionType,
) ([]domain.BalanceTransaction, error) {
	t, err := b.blRepo.GetByDirection(ctx, userID, direction)
	if err != nil {
		return nil, fmt.Errorf("getting balance transactions by direction: %w", err)
	}
	return t, nil
}

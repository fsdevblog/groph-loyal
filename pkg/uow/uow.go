package uow

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryName string
type Repository any
type RepositoryFactory func(DBTX) Repository

type UnitOfWork struct {
	conn         *pgxpool.Pool
	repositories map[RepositoryName]RepositoryFactory
}

func NewUnitOfWork(conn *pgxpool.Pool) *UnitOfWork {
	return &UnitOfWork{
		conn:         conn,
		repositories: make(map[RepositoryName]RepositoryFactory),
	}
}

// Register регистрирует репозиторий у себя в мапе. Если репозиторий уже зарегистрирован, возвращает
// ошибку ErrRepositoryAlreadyRegistered.
func (u *UnitOfWork) Register(name RepositoryName, factory RepositoryFactory) error {
	if _, ok := u.repositories[name]; ok {
		return ErrRepositoryAlreadyRegistered
	}
	u.repositories[name] = factory
	return nil
}

// Do выполняет функцию fn внутри транзакции.
func (u *UnitOfWork) Do(ctx context.Context, fn func(context.Context, TX) error) (err error) {
	tx, txErr := u.conn.BeginTx(ctx, pgx.TxOptions{})
	if txErr != nil {
		return txErr //nolint:wrapcheck
	}
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			if err == nil {
				err = rollbackErr
			} else {
				err = errors.Join(err, rollbackErr)
			}
		}
	}()

	transErr := fn(ctx, NewTransaction(tx, u.repositories))
	if transErr != nil {
		return transErr
	}
	err = tx.Commit(ctx)
	return
}

// GetRepository возвращает репозиторий или ошибку ErrRepositoryNotRegistered.
func (u *UnitOfWork) GetRepository(name RepositoryName) (Repository, error) {
	if repoFactory, ok := u.repositories[name]; ok {
		return repoFactory(u.conn), nil
	}
	return nil, ErrRepositoryNotRegistered
}

// GetRepositoryAs возвращает репозиторий по имени name и приводит его к типу T. Возвращает ошибки
// ErrRepositoryNotRegistered и ErrInvalidRepositoryType.
func GetRepositoryAs[T any](u UOW, name RepositoryName) (T, error) {
	var res T
	repo, err := u.GetRepository(name)
	if err != nil {
		return res, err //nolint:wrapcheck
	}
	r, ok := repo.(T)

	if !ok {
		return res, ErrInvalidRepositoryType
	}

	return r, nil
}

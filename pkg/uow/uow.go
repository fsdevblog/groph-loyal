package uow

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

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

func New(conn *pgxpool.Pool) *UnitOfWork {
	return &UnitOfWork{
		conn:         conn,
		repositories: make(map[RepositoryName]RepositoryFactory),
	}
}

// MassRegister массовая регистрация репозиториев.
func (u *UnitOfWork) MassRegister(repos map[RepositoryName]RepositoryFactory) error {
	var massErr error
	for name, factory := range repos {
		if err := u.Register(name, factory); err != nil {
			massErr = errors.Join(massErr, err)
		}
	}
	return massErr
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

type IsolationLevel string

const (
	IsolationDefault        IsolationLevel = "read_committed"
	IsolationRepeatableRead IsolationLevel = "repeatable_read"
	IsolationSerializable   IsolationLevel = "serializable"
)

// TransactionOptions структура настроек для обертки транзакции.
type TransactionOptions struct {
	IsolationLevel IsolationLevel // Уровень изоляции транзакции.
}

func WithIsolationLevel(isoLevel IsolationLevel) func(*TransactionOptions) {
	return func(opts *TransactionOptions) {
		opts.IsolationLevel = isoLevel
	}
}

const pgIsolationCode = "40001"

// DoWithIsolation выполняет транзакцию с заданным уровнем изоляции и автоматическими повторами.
// Принимает функцию fn, которая будет выполнена в рамках транзакции с уровнем изоляции isoLevel.
// При возникновении ошибок изоляции (код Postgres 40001) автоматически повторяет транзакцию
// до maxRetries раз. При превышении лимита повторов возвращает ErrIsolationRetryLimitReached.
// При возникновении других ошибок немедленно прерывает выполнение и возвращает ошибку.
func (u *UnitOfWork) DoWithIsolation(
	ctx context.Context,
	fn func(context.Context, TX) error,
	isoLevel IsolationLevel,
	maxRetries uint,
) error {
	for range maxRetries {
		err := u.Do(ctx, fn, WithIsolationLevel(isoLevel))
		if err == nil {
			return nil
		}
		var errPg *pgconn.PgError
		if errors.As(err, &errPg) && errPg.Code == pgIsolationCode {
			continue
		}
		return err
	}
	return ErrIsolationRetryLimitReached
}

// Do выполняет функцию fn внутри транзакции.
func (u *UnitOfWork) Do(
	ctx context.Context,
	fn func(context.Context, TX) error,
	opts ...func(options *TransactionOptions),
) (err error) {
	options := TransactionOptions{
		IsolationLevel: IsolationDefault,
	}
	if len(opts) > 0 {
		for _, opt := range opts {
			opt(&options)
		}
	}

	tx, txErr := u.conn.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: convertIsolationLevelType(options.IsolationLevel),
	})
	if txErr != nil {
		return txErr //nolint:wrapcheck
	}
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = errors.Join(err, rollbackErr)
		}
	}()

	transErr := fn(ctx, NewTransaction(tx, u.repositories))
	if transErr != nil {
		return transErr
	}
	err = tx.Commit(ctx)
	return err //nolint:wrapcheck
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

// convertIsolationLevelType конвертирует типы для изоляции транзакции в pgx.TxIsoLevel.
// Возвращает pgx.ReadCommitted при некорректном IsolationLevel.
func convertIsolationLevelType(isolationLevel IsolationLevel) pgx.TxIsoLevel {
	switch isolationLevel {
	case IsolationRepeatableRead:
		return pgx.RepeatableRead
	case IsolationSerializable:
		return pgx.Serializable
	case IsolationDefault:
		return pgx.ReadCommitted
	default:
		return pgx.ReadCommitted
	}
}

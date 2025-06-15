package uow

import (
	"github.com/jackc/pgx/v5"
)

type Transaction struct {
	repositories map[RepositoryName]RepositoryFactory
	tx           pgx.Tx
}

func NewTransaction(tx pgx.Tx, repositories map[RepositoryName]RepositoryFactory) *Transaction {
	return &Transaction{
		repositories: repositories,
		tx:           tx,
	}
}

// Get возвращает репозиторий или ошибку ErrRepositoryNotRegistered.
func (t *Transaction) Get(name RepositoryName) (Repository, error) {
	if repo, ok := t.repositories[name]; ok {
		return repo(t.tx), nil
	}
	return nil, ErrRepositoryNotRegistered
}

// GetAs возвращает зарегистрированный репозиторий с именем name приведенный к типу T
// или ошибки ErrRepositoryNotRegistered в случае не найденного репозитория с указанным name, ErrInvalidRepositoryType

func GetAs[T any](t TX, name RepositoryName) (T, error) {
	repo, err := t.Get(name)
	var res T
	if err != nil {
		return res, err //nolint:wrapcheck
	}
	res, ok := repo.(T)
	if !ok {
		return res, ErrInvalidRepositoryType
	}
	return res, nil
}

package uow

//go:generate mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks
import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type TX interface {
	Get(name RepositoryName) (Repository, error)
}

type DBTX interface {
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
	QueryRow(context.Context, string, ...interface{}) pgx.Row
	SendBatch(context.Context, *pgx.Batch) pgx.BatchResults
}

type UOW interface {
	Register(name RepositoryName, factory RepositoryFactory) error
	Do(ctx context.Context, fn func(ctx context.Context, tx TX) error) error
	GetRepository(name RepositoryName) (Repository, error)
}

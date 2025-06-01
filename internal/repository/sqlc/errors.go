package sqlc

import (
	"errors"
	"fmt"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	uniqueViolationCode = "23505"
)

func convertErr(err error, msg string) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		errType := domain.ErrUnknown
		if isUniqueViolationErr(pgErr) {
			errType = domain.ErrDuplicateKey
		}
		return fmt.Errorf("[repository/%s] %w: %s", msg, errType, pgErr.Message)
	}

	return fmt.Errorf("[repository/%s] %w: %s", msg, domain.ErrUnknown, err.Error())
}

func isUniqueViolationErr(err *pgconn.PgError) bool {
	return err.Code == uniqueViolationCode
}

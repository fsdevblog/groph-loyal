package sqlc

import (
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	uniqueViolationCode = "23505"
)

func convertErr(err error, format string, formatArgs ...any) error {
	if err == nil {
		return nil
	}

	msg := fmt.Sprintf(format, formatArgs...)

	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("[repository/%s] %w", msg, domain.ErrRecordNotFound)
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

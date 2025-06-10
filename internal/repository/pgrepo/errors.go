package pgrepo

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

// convertErr преобразует ошибку к стандартному виду для слоя репозитория.
// Добавляет форматированное сообщение контекста, тип бизнес-ошибки и оригинальное сообщение.
// Особенности:
//   - Для ошибок отсутствия данных (pgx.ErrNoRows) возвращает ErrRecordNotFound из domain.
//   - Для ошибок базы Postgres определяет дубликаты ключей (uniqueViolationCode) как ErrDuplicateKey из domain.
//   - Все остальные ошибки возвращаются как ErrUnknown с оригинальным сообщением.
//
// Используется для единообразной обработки и возврата ошибок из репозитория.
func convertErr(err error, format string, formatArgs ...any) error {
	if err == nil {
		return nil
	}

	msg := fmt.Sprintf(format, formatArgs...)

	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("[repository/%s] %w", msg, domain.ErrRecordNotFound)
	}

	var pgErr *pgconn.PgError
	errType := domain.ErrUnknown

	if errors.As(err, &pgErr) {
		if isUniqueViolationErr(pgErr) {
			errType = domain.ErrDuplicateKey
		}
	}

	return fmt.Errorf("[repository/%s] %w: %s", msg, errType, err.Error())
}

func isUniqueViolationErr(err *pgconn.PgError) bool {
	return err.Code == uniqueViolationCode
}

package pgrepo

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

func Connect(ctx context.Context, migrationsDir, dsn string, l *logrus.Logger) (*pgxpool.Pool, error) {
	type connResult struct {
		conn *pgxpool.Pool
		err  error
	}
	connChan := make(chan connResult, 1)
	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		var attempts uint
		var maxAttempts uint = 30
		var retryInterval = 3 * time.Second

		for {
			select {
			case <-ctx.Done():
				connChan <- connResult{err: ctx.Err()}
				return
			default:
				conn, connErr := newPostgresConnection(ctx, dsn)
				if connErr != nil {
					attempts++
					if attempts > maxAttempts {
						connChan <- connResult{
							err: fmt.Errorf("init postgres connection after %d attempts: %w", maxAttempts, connErr),
						}
					}
					l.WithError(connErr).
						WithField("CurrentAttempt", fmt.Sprintf("#%d / %d", attempts, maxAttempts)).
						Warnf("init postgres connection error, retrying in %.f seconds", retryInterval.Seconds())
					time.Sleep(retryInterval)
					continue
				}
				connChan <- connResult{conn: conn}
				return
			}
		}
	}(wg)

	wg.Wait()
	close(connChan)

	res := <-connChan
	if res.err != nil {
		return nil, fmt.Errorf("init postgres connection: %s", res.err.Error())
	}

	if err := postgresMigrate(migrationsDir, dsn); err != nil {
		return nil, err
	}
	return res.conn, nil
}

func newPostgresConnection(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	poolConfig, confErr := pgxpool.ParseConfig(dsn)
	if confErr != nil {
		return nil, fmt.Errorf("parse postgres config: %s", confErr.Error())
	}
	pool, poolErr := pgxpool.NewWithConfig(ctx, poolConfig)
	if poolErr != nil {
		return nil, fmt.Errorf("failed to create pool: %s", poolErr.Error())
	}

	// Проверяем, что соединение работает (Ping)
	if pingErr := pool.Ping(ctx); pingErr != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %s", pingErr.Error())
	}

	return pool, nil
}

func postgresMigrate(dir string, dsn string) error {
	m, mErr := migrate.New("file://"+dir, dsn)
	if mErr != nil {
		return fmt.Errorf("failed to create migrate instance: %w", mErr)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to migrate schema: %w", err)
	}
	return nil
}

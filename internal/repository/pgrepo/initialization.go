package pgrepo

import (
	"context"
	"errors"
	"fmt"

	"sync"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
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
				conn, errConn := newPostgresConnection(ctx, dsn)
				if errConn != nil {
					attempts++
					if attempts > maxAttempts {
						connChan <- connResult{
							err: fmt.Errorf("init postgres connection after %d attempts: %w", maxAttempts, errConn),
						}
						return
					}
					l.WithError(errConn).
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
	poolConfig, errConf := pgxpool.ParseConfig(dsn)
	if errConf != nil {
		return nil, fmt.Errorf("parse postgres config: %s", errConf.Error())
	}
	pool, errPool := pgxpool.NewWithConfig(ctx, poolConfig)
	if errPool != nil {
		return nil, fmt.Errorf("failed to create pool: %s", errPool.Error())
	}

	// Проверяем, что соединение работает (Ping)
	if errPing := pool.Ping(ctx); errPing != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %s", errPing.Error())
	}

	return pool, nil
}

func postgresMigrate(dir string, dsn string) error {
	m, errM := migrate.New("file://"+dir, dsn)
	if errM != nil {
		return fmt.Errorf("failed to create migrate instance: %w", errM)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to migrate schema: %w", err)
	}
	return nil
}

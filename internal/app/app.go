package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/fsdevblog/groph-loyal/internal/config"
	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/repository/sqlc"
	"github.com/fsdevblog/groph-loyal/internal/service"
	"github.com/fsdevblog/groph-loyal/internal/transport/httptrt"
	"github.com/fsdevblog/groph-loyal/internal/uow"
	"github.com/golang-migrate/migrate/v4"
	"github.com/sirupsen/logrus"

	// driver for migration applying postgres.
	_ "github.com/golang-migrate/migrate/v4/database/postgres" //nolint:revive
	// driver to get migrations from files (*.sql in our case).
	"os/signal"
	"syscall"

	_ "github.com/golang-migrate/migrate/v4/source/file" //nolint:revive
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	Config *config.Config
	Logger *logrus.Logger
}

func New(conf *config.Config, l *logrus.Logger) *App {
	return &App{
		Config: conf,
		Logger: l,
	}
}

func (a *App) Run() error {
	notifyCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	conn, connErr := initPostgres(notifyCtx, a.Config.MigrationsDir, a.Config.DatabaseDSN, a.Logger)
	if connErr != nil {
		return connErr
	}

	unitOfWork, uowErr := initUOW(conn)
	if uowErr != nil {
		return fmt.Errorf("app run: %s", uowErr.Error())
	}
	userService, userServiceErr := service.NewUserService(unitOfWork, []byte(a.Config.JWTUserSecret))

	if userServiceErr != nil {
		return fmt.Errorf("app run: %s", userServiceErr.Error())
	}

	router := httptrt.New(httptrt.RouterArgs{
		Logger:      a.Logger,
		UserService: userService,
	})

	errChan := make(chan error, 1)

	go func() {
		if runErr := router.Run(a.Config.RunAddress); runErr != nil {
			errChan <- runErr
		}
	}()

	select {
	case <-notifyCtx.Done():
		return notifyCtx.Err() //nolint:wrapcheck
	case err := <-errChan:
		return err
	}
}

func initPostgres(ctx context.Context, migrationsDir, dsn string, l *logrus.Logger) (*pgxpool.Pool, error) {
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
						WithField("Attempt", fmt.Sprintf("#%d / %d", attempts, maxAttempts)).
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

func initUOW(conn *pgxpool.Pool) (*uow.UnitOfWork, error) {
	unitOfWork := uow.NewUnitOfWork(conn)

	userRepoFactoryFn := func(dbtx uow.DBTX) uow.Repository {
		return sqlc.NewUserRepository(dbtx)
	}
	if regErr := unitOfWork.Register(uow.RepositoryName(domain.UserRepoName), userRepoFactoryFn); regErr != nil {
		return nil, fmt.Errorf("init UOW: %s", regErr.Error())
	}

	return unitOfWork, nil
}

package app

import (
	"context"
	"fmt"

	"github.com/fsdevblog/groph-loyal/internal/repository/repoargs"

	"github.com/fsdevblog/groph-loyal/internal/transport/accrual"

	"github.com/fsdevblog/groph-loyal/pkg/uow"

	"github.com/fsdevblog/groph-loyal/internal/config"
	"github.com/fsdevblog/groph-loyal/internal/repository/pgrepo"
	"github.com/fsdevblog/groph-loyal/internal/service"
	"github.com/fsdevblog/groph-loyal/internal/transport/api"
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

	a.Logger.Infof("Starting app with config: %+v", a.Config)
	conn, connErr := pgrepo.Connect(notifyCtx, a.Config.MigrationsDir, a.Config.DatabaseDSN, a.Logger)
	if connErr != nil {
		return fmt.Errorf("app run: %s", connErr.Error())
	}
	defer conn.Close()

	unitOfWork, uowErr := initUOW(conn)
	if uowErr != nil {
		return fmt.Errorf("app run: %s", uowErr.Error())
	}

	services, sErr := service.Factory(unitOfWork, []byte(a.Config.JWTUserSecret))

	if sErr != nil {
		return fmt.Errorf("app run: %s", sErr.Error())
	}

	router := api.New(api.RouterArgs{
		Logger:       a.Logger,
		UserService:  services.UserService,
		OrderService: services.OrderService,
		BlService:    services.BlService,
		JWTSecretKey: []byte(a.Config.JWTUserSecret),
	})

	errChan := make(chan error, 1)

	go func() {
		if runErr := router.Run(a.Config.RunAddress); runErr != nil {
			errChan <- runErr
		}
	}()

	processor := accrual.New(services.OrderService, a.Config.AccrualSystemAddress, a.Logger).
		SetAccrualWorkers(5).    //nolint:mnd
		SetLimitPerIteration(50) //nolint:mnd

	go processor.Run(notifyCtx)

	select {
	case <-notifyCtx.Done():
		return notifyCtx.Err() //nolint:wrapcheck
	case err := <-errChan:
		return err
	}
}

func initUOW(conn *pgxpool.Pool) (*uow.UnitOfWork, error) {
	unitOfWork := uow.NewUnitOfWork(conn)

	// user repo
	userRepoFactoryFn := func(dbtx uow.DBTX) uow.Repository {
		return pgrepo.NewUserRepository(dbtx)
	}
	if regErr := unitOfWork.Register(uow.RepositoryName(repoargs.UserRepoName), userRepoFactoryFn); regErr != nil {
		return nil, fmt.Errorf("init UOW: %s", regErr.Error())
	}

	// order repo
	orderRepoFactoryFn := func(dbtx uow.DBTX) uow.Repository {
		return pgrepo.NewOrderRepository(dbtx)
	}
	if regErr := unitOfWork.Register(uow.RepositoryName(repoargs.OrderRepoName), orderRepoFactoryFn); regErr != nil {
		return nil, fmt.Errorf("init UOW: %s", regErr.Error())
	}

	// balance transaction repo
	balanceTransactionRepoFactoryFn := func(dbtx uow.DBTX) uow.Repository {
		return pgrepo.NewBalanceTransactionRepository(dbtx)
	}
	if regErr := unitOfWork.Register(
		uow.RepositoryName(repoargs.BalanceTransactionRepoName),
		balanceTransactionRepoFactoryFn,
	); regErr != nil {
		return nil, fmt.Errorf("init UOW: %s", regErr.Error())
	}

	return unitOfWork, nil
}

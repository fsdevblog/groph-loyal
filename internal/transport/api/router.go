package api

import (
	"fmt"
	"time"

	"github.com/fsdevblog/groph-loyal/internal/transport/api/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const (
	DefaultServiceTimeout = 3 * time.Second
)

const (
	RouteGroup           = "/api"
	RegisterRoute        = "/user/register"
	LoginRoute           = "/user/login"
	OrdersRoute          = "/user/orders"
	BalanceRoute         = "/user/balance"
	BalanceWithdrawRoute = "/user/balance/withdraw"
	WithdrawalsRoute     = "/user/withdrawals"
)

type RouterArgs struct {
	Logger       *logrus.Logger
	UserService  UserServicer
	OrderService OrderServicer
	BlService    BalanceServicer
	JWTSecretKey []byte
}

func MustNew(args RouterArgs) *gin.Engine {
	r, err := New(args)
	if err != nil {
		panic(err)
	}
	return r
}

func New(args RouterArgs) (*gin.Engine, error) {
	if err := registerValidators(); err != nil {
		return nil, fmt.Errorf("initialize router: %s", err.Error())
	}
	r := gin.New()
	r.Use(gin.Recovery())
	if args.Logger != nil {
		r.Use(middlewares.Logger(args.Logger))
	}
	r.Use(middlewares.Errors())

	authHandler := NewAuthHandler(args.UserService)
	ordersHandler := NewOrdersHandler(args.OrderService)
	balanceHandler := NewBalanceHandler(args.BlService)

	api := r.Group(RouteGroup)

	api.POST(RegisterRoute, middlewares.NonAuthRequired(args.JWTSecretKey), authHandler.Register)
	api.POST(LoginRoute, middlewares.NonAuthRequired(args.JWTSecretKey), authHandler.Login)

	api.Use(middlewares.AuthRequired(args.JWTSecretKey))
	// ниже все роуты группы требуют авторизованного пользователя.
	api.POST(OrdersRoute, ordersHandler.Create)
	api.GET(OrdersRoute, ordersHandler.Index)

	api.GET(BalanceRoute, balanceHandler.Index)
	api.POST(BalanceWithdrawRoute, balanceHandler.Withdraw)
	api.GET(WithdrawalsRoute, balanceHandler.Withdrawals)
	return r, nil
}

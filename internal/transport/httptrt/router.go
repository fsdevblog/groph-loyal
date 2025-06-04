package httptrt

import (
	"time"

	"github.com/fsdevblog/groph-loyal/internal/transport/httptrt/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const (
	DefaultServiceTimeout = 3 * time.Second
)

const (
	APIRouteGroup    = "/api"
	APIRegisterRoute = "/user/register"
	APILoginRoute    = "/user/login"
	APIOrdersRoute   = "/user/orders"
)

type RouterArgs struct {
	Logger       *logrus.Logger
	UserService  UserServicer
	OrderService OrderServicer
	JWTSecretKey []byte
}

func New(args RouterArgs) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	if args.Logger != nil {
		r.Use(middlewares.LoggerMiddleware(args.Logger))
	}

	authHandler := NewAuthHandler(args.UserService)
	ordersHandler := NewOrdersHandler(args.OrderService)

	api := r.Group(APIRouteGroup)

	api.POST(APIRegisterRoute, middlewares.NonAuthRequiredMiddleware(args.JWTSecretKey), authHandler.Register)
	api.POST(APILoginRoute, middlewares.NonAuthRequiredMiddleware(args.JWTSecretKey), authHandler.Login)

	api.Use(middlewares.AuthRequiredMiddleware(args.JWTSecretKey))
	// ниже все роуты группы требуют авторизованного пользователя.
	api.POST(APIOrdersRoute, ordersHandler.Create)
	api.GET(APIOrdersRoute, ordersHandler.Index)
	return r
}

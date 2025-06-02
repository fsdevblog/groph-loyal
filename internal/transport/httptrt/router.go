package httptrt

import (
	"github.com/fsdevblog/groph-loyal/internal/transport/httptrt/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"time"
)

const (
	DefaultServiceTimeout = 3 * time.Second
)

const (
	APIRouteGroup    = "/api"
	APIRegisterRoute = "/user/register"
)

type RouterArgs struct {
	Logger       *logrus.Logger
	UserService  UserServicer
	JWTSecretKey []byte
}

func New(args RouterArgs) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	if args.Logger != nil {
		r.Use(middlewares.LoggerMiddleware(args.Logger))
	}

	authHandler := NewAuthHandler(args.UserService)
	api := r.Group(APIRouteGroup)

	api.POST(APIRegisterRoute, authHandler.Register)
	return r
}

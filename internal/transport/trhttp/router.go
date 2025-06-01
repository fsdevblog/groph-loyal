package trhttp

import (
	"github.com/fsdevblog/groph-loyal/internal/transport/trhttp/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type RouterArgs struct {
	Logger      *logrus.Logger
	UserService UserServicer
}

func New(args RouterArgs) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	if args.Logger != nil {
		r.Use(middlewares.LoggerMiddleware(args.Logger))
	}

	authHandler := NewAuthHandler(args.UserService)
	api := r.Group("/api")

	api.POST("/user/register", authHandler.Register)
	return r
}

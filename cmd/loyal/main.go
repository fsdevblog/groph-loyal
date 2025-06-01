package main

import (
	"context"
	"errors"
	"github.com/fsdevblog/groph-loyal/internal/logger"
	"os"

	"github.com/fsdevblog/groph-loyal/internal/app"
	"github.com/fsdevblog/groph-loyal/internal/config"
)

func main() {
	conf := config.MustLoadConfig()
	l := logger.New(os.Stdout)

	if err := app.New(conf, l).Run(); err != nil {
		if errors.Is(err, context.Canceled) {
			l.Info("graceful shutdown")
			os.Exit(0)
		}
		panic(err)
	}
}

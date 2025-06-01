package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/fsdevblog/groph-loyal/internal/app"
	"github.com/fsdevblog/groph-loyal/internal/config"
	"os"
)

func main() {
	conf := config.MustLoadConfig()

	if err := app.New(conf).Run(); err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Println("shutdown signal received")
			os.Exit(0)
		} else {
			panic(err)
		}
	}
}

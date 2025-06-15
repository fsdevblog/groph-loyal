package config

import (
	"errors"
	"flag"
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	RunAddress           string `env:"RUN_ADDRESS"`
	AccrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	DatabaseDSN          string `env:"DATABASE_URI"`
	MigrationsDir        string `env:"MIGRATIONS_DIR"`
	JWTUserSecret        string `env:"JWT_USER_SECRET"        envDefault:"supersecretkey"`
}

func LoadConfig() (*Config, error) {
	var conf Config

	loadFlags(&conf)

	if envParseErr := env.Parse(&conf); envParseErr != nil {
		return nil, fmt.Errorf("parse env config: %s", envParseErr.Error())
	}

	if conf.DatabaseDSN == "" {
		return nil, errors.New("database DSN is not set")
	}
	return &conf, nil
}

func MustLoadConfig() *Config {
	config, err := LoadConfig()
	if err != nil {
		panic(err)
	}
	return config
}

func loadFlags(flagConfig *Config) {
	flag.StringVar(&flagConfig.RunAddress, "a", "localhost:8080", "Run address in format host:port")
	flag.StringVar(&flagConfig.DatabaseDSN, "d", "", "Database DSN")
	flag.StringVar(&flagConfig.MigrationsDir, "m", "internal/db/migrations", "Database migrations directory")
	flag.StringVar(
		&flagConfig.AccrualSystemAddress,
		"f",
		"http://localhost:8081",
		"Accrual address in format scheme://host:port",
	)

	flag.Parse()
}

package config

import (
	"errors"
	"flag"
	"fmt"
	"github.com/caarlos0/env/v11"
)

type Config struct {
	RunAddress    string `env:"RUN_ADDRESS"`
	DatabaseDSN   string `env:"DATABASE_URI"`
	MigrationsDir string `env:"MIGRATIONS_DIR"`
}

func LoadConfig() (*Config, error) {
	var flagsConfig, envConfig Config

	if envParseErr := env.Parse(&envConfig); envParseErr != nil {
		return nil, fmt.Errorf("parse env config: %s", envParseErr.Error())
	}

	loadFlags(&flagsConfig)

	conf := mergeConfig(&envConfig, &flagsConfig)
	if conf.DatabaseDSN == "" {
		return nil, errors.New("database DSN is not set")
	}
	return conf, nil
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

	flag.Parse()
}

func mergeConfig(envConfig, flagsConfig *Config) *Config {
	return &Config{
		RunAddress:    defaultIfBlank(envConfig.RunAddress, flagsConfig.RunAddress),
		DatabaseDSN:   defaultIfBlank(envConfig.DatabaseDSN, flagsConfig.DatabaseDSN),
		MigrationsDir: defaultIfBlank(envConfig.MigrationsDir, flagsConfig.MigrationsDir),
	}
}

func defaultIfBlank(value string, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

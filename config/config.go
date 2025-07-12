package config

import (
	"context"
	"strconv"

	"github.com/sethvargo/go-envconfig"
)

func Get(ctx context.Context) Config {
	c := Config{}
	if err := envconfig.Process(ctx, &c); err != nil {
		panic(err)
	}
	validate(&c)
	return c
}

type Config struct {
	Port                    string `env:"PORT"`
	Database                string `env:"DATABASE"`
	PaymentProcessorDefault string `env:"PAYMENT_PROCESSOR_DEFAULT"`
	PaymentProcessorBackup  string `env:"PAYMENT_PROCESSOR_BACKUP"`
}

func validate(cfg *Config) Config {
	if cfg.Database == "" {
		panic("required value for Database")
	}
	if cfg.PaymentProcessorDefault == "" {
		panic("required value for payment processor default")
	}
	if cfg.PaymentProcessorBackup == "" {
		panic("required value for payment processor backup")
	}

	if cfg.Port == "" {
		cfg.Port = "9999"
	} else if _, err := strconv.ParseInt(cfg.Port, 10, 16); err != nil {
		panic("invalid port number: " + cfg.Port)
	}

	return *cfg
}

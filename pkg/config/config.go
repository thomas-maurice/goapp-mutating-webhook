package config

import (
	"context"
	"os"

	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/log"
	"gopkg.in/yaml.v3"
)

type Config struct {
	GoMemLimitFactor float64 `yaml:"go_mem_limit_factor"`
}

func GetConfigFromFile(fileName string) (*Config, error) {
	b, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(b, &cfg)
	if err != nil {
		return nil, err
	}

	// Defaults the config

	logger := log.GetLogger()

	if cfg.GoMemLimitFactor <= 0 {
		logger.Warn("go_mem_limit_factor is zero or invalid, resetting it to 1")
	}

	return &cfg, nil
}

type ConfigKey struct{}

var (
	configKey ConfigKey = ConfigKey{}
)

func FromContext(ctx context.Context) (*Config, error) {
	cfg, ok := ctx.Value(configKey).(*Config)
	if ok {
		return cfg, nil
	}

	return nil, nil
}

func ToContext(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, configKey, cfg)
}

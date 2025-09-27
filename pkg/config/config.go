package config

import (
	"context"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Mutations  map[string]yaml.Node `yaml:"mutations"`
	Admissions map[string]yaml.Node `yaml:"admissions"`
}

var NoConfig = Config{
	Mutations:  make(map[string]yaml.Node),
	Admissions: make(map[string]yaml.Node),
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

	if cfg.Admissions == nil {
		cfg.Admissions = make(map[string]yaml.Node)
	}

	if cfg.Mutations == nil {
		cfg.Mutations = make(map[string]yaml.Node)
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

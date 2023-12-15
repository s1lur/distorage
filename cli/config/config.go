package config

import (
	"errors"
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
	"os"
	"path"
)

type Config struct {
	ServerURL        string `toml:"server_url"`
	ReplicationCount int    `toml:"replication_count" env-default:"5"`
}

func NewConfig(homeDir string) (*Config, error) {
	configPath := path.Join(homeDir, ".distorage", "cli.toml")
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("config file not found! please run distorage init")
	}
	cfg := &Config{}

	err := cleanenv.ReadConfig(configPath, cfg)
	if err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	return cfg, nil
}

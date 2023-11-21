package config

import (
	"errors"
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
	"os"
)

type (
	Config struct {
		Port         string `toml:"port"`
		ServerIpAddr string `toml:"server_ip_addr"`
		BasePath     string `toml:"base_path" env-default:"~/.distorage/"`
		Addr         string `toml:"addr"`
	}
)

func NewConfig() (*Config, error) {
	if len(os.Args) < 2 {
		return nil, errors.New("please provide path to config file")
	}
	if len(os.Args) > 2 {
		return nil, errors.New("too many arguments")
	}
	cfg := &Config{}

	err := cleanenv.ReadConfig(os.Args[1], cfg)
	if err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	return cfg, nil
}

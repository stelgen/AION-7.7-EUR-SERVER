// Package config — загрузка config.yaml.
package config

import (
	"os"

	"gopkg.in/yaml.v3"

	"aion-captcha/internal/server"
	"aion-captcha/internal/ship"
)

// Config — полный конфиг (layout совместим с aion-logd: server/render/ship).
type Config struct {
	Server server.Config `yaml:"server"`
	Ship   ship.Cfg      `yaml:"ship"`
}

// Load — прочитать и заполнить дефолты.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	c.Server.FillDefaults()
	return &c, nil
}

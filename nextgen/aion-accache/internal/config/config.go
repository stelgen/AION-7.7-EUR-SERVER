// Package config: yaml-конфиг aion-accache (зеркало семантики common.xml ACS).
package config

import (
	"os"

	"gopkg.in/yaml.v3"

	"aion-accache/internal/ship"
)

type DB struct {
	Enabled bool   `yaml:"enabled"`
	Conn    string `yaml:"conn"` // СЕКРЕТ: только на VM/env
}

type Server struct {
	Listen     string `yaml:"listen"`      // ":2220"
	Verbose    bool   `yaml:"verbose"`     // дамп кадров
	CaptureAll bool   `yaml:"capture_all"` // raw hex в logs (как logd)
	BaseDir    string `yaml:"base_dir"`    // logs/
}

type Cfg struct {
	Server Server `yaml:"server"`
	DB     DB     `yaml:"db"`
	Ship   ship.Cfg `yaml:"ship"`
}

// Load читает yaml (файл опционален — дефолты кодом).
func Load(path string) (*Cfg, error) {
	c := &Cfg{}
	c.Server.Listen = ":2220"
	c.Server.BaseDir = "logs"
	if path == "" {
		return c, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(b, c); err != nil {
		return nil, err
	}
	return c, nil
}

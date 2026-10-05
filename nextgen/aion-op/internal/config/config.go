// Package config — YAML-топология стека: группы, сервисы, порты, порядок, пара.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type VM struct {
	Mode    string `yaml:"mode"`
	SSHHost string `yaml:"ssh_host"`
	SSHPort int    `yaml:"ssh_port"`
	SSHUser string `yaml:"ssh_user"`
	SSHKey  string `yaml:"ssh_key"`
}

type Operator struct {
	UIPort     int    `yaml:"ui_port"`
	Bind       string `yaml:"bind"` // дефолт 127.0.0.1 (снаружи — только ssh-туннель)
	RefreshSec int    `yaml:"refresh_sec"`
	Mode       string `yaml:"mode"`    // observe | operate
	DryRun     *bool  `yaml:"dry_run"` // nil = true (безопасный дефолт); actions только планируются
}

type Group struct {
	Title string `yaml:"title"`
	Icon  string `yaml:"icon"`
}

type Service struct {
	ID          string `yaml:"id"`
	Group       string `yaml:"group"`
	Display     string `yaml:"display"`
	Exe         string `yaml:"exe"`
	Task        string `yaml:"task"`
	KillTask    string `yaml:"kill_task"` // /IT-задача-киллер (обязательна для interactive/heavy)
	Ports       []int  `yaml:"ports"`
	Order       int    `yaml:"order"`
	Interactive bool   `yaml:"interactive"`
	Heavy       bool   `yaml:"heavy"`
	ObserveOnly bool   `yaml:"observe_only"`
	Locked      bool   `yaml:"locked"`
	Pair        string `yaml:"pair"`
	PairMaster  bool   `yaml:"pair_master"`
}

type WorldPair struct {
	NPCPort       int `yaml:"npc_port"`
	ExpectedConns int `yaml:"expected_conns"`
}

type StoreCfg struct {
	Path          string `yaml:"path"` // SQLite WAL
	RetentionDays int    `yaml:"retention_days"`
}

type PprofCfg struct {
	Enabled bool   `yaml:"enabled"` // loopback-only
	Addr    string `yaml:"addr"`
}

type LogFile struct {
	Svc  string `yaml:"svc"`  // id сервиса
	Path string `yaml:"path"` // {{date}} → YYYY-MM-DD
}

type LogsCfg struct {
	PollSec   int       `yaml:"poll_sec"`
	TailLines int       `yaml:"tail_lines"`
	Files     []LogFile `yaml:"files"`
}

type MetricsCfg struct {
	PollSec int `yaml:"poll_sec"`
}

type Config struct {
	VM        VM               `yaml:"vm"`
	Operator  Operator         `yaml:"operator"`
	Services  []Service        `yaml:"services"`
	Groups    map[string]Group `yaml:"groups"`
	WorldPair WorldPair        `yaml:"world_pair"`
	Store     StoreCfg         `yaml:"store"`
	Pprof     PprofCfg         `yaml:"pprof"`
	Logs      LogsCfg          `yaml:"logs"`
	Metrics   MetricsCfg       `yaml:"metrics"`
}

func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("читаю %s: %w", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("yaml %s: %w", path, err)
	}
	c.normalize()
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) normalize() {
	if c.Operator.DryRun == nil {
		t := true
		c.Operator.DryRun = &t // безопасный дефолт: действия только планируются
	}
	if c.Operator.UIPort == 0 {
		c.Operator.UIPort = 10200
	}
	if c.Operator.Bind == "" {
		c.Operator.Bind = "127.0.0.1"
	}
	if c.Operator.RefreshSec <= 0 {
		c.Operator.RefreshSec = 10
	}
	if c.Operator.Mode == "" {
		c.Operator.Mode = "observe"
	}
	if c.VM.Mode == "" {
		c.VM.Mode = "mock"
	}
	if c.VM.SSHPort == 0 {
		c.VM.SSHPort = 22
	}
	if c.WorldPair.ExpectedConns == 0 {
		c.WorldPair.ExpectedConns = 16
	}
	if c.WorldPair.NPCPort == 0 {
		c.WorldPair.NPCPort = 2002
	}
	if c.Store.Path == "" {
		c.Store.Path = "aionop.db"
	}
	if c.Store.RetentionDays == 0 {
		c.Store.RetentionDays = 30
	}
	if c.Pprof.Addr == "" {
		c.Pprof.Addr = "127.0.0.1:10201"
	}
	if c.Logs.PollSec <= 0 {
		c.Logs.PollSec = 20
	}
	if c.Logs.TailLines <= 0 {
		c.Logs.TailLines = 200
	}
	if c.Metrics.PollSec <= 0 {
		c.Metrics.PollSec = 15
	}
}

func (c *Config) Validate() error {
	if c.Operator.Mode != "observe" && c.Operator.Mode != "operate" {
		return fmt.Errorf("operator.mode=%q: только observe|operate", c.Operator.Mode)
	}
	if c.Operator.DryRun == nil || *c.Operator.DryRun {
		// dry-run обязателен по умолчанию; operate без явного dry_run:false — отвергаем
		if c.Operator.Mode == "operate" {
			return fmt.Errorf("operate требует явного operator.dry_run: false (осознанное решение)")
		}
	}
	if c.VM.Mode != "mock" && c.VM.Mode != "ssh" && c.VM.Mode != "local" {
		return fmt.Errorf("vm.mode=%q: только mock|ssh|local", c.VM.Mode)
	}
	if len(c.Services) == 0 {
		return fmt.Errorf("services пуст")
	}
	if len(c.Groups) == 0 {
		return fmt.Errorf("groups пуст")
	}
	ids := map[string]bool{}
	for _, s := range c.Services {
		if s.ID == "" || s.Exe == "" {
			return fmt.Errorf("сервис без id/exe: %+v", s)
		}
		if ids[s.ID] {
			return fmt.Errorf("дубль id %q", s.ID)
		}
		ids[s.ID] = true
	}
	for _, s := range c.Services {
		if _, ok := c.Groups[s.Group]; !ok {
			return fmt.Errorf("сервис %q ссылается на неизвестную группу %q", s.ID, s.Group)
		}
	}
	for _, f := range c.Logs.Files {
		if f.Path == "" {
			return fmt.Errorf("logs.files: пустой path")
		}
		if _, ok := ids[f.Svc]; !ok {
			return fmt.Errorf("logs.files: неизвестный сервис %q", f.Svc)
		}
	}
	return nil
}

func (c *Config) ByID(id string) (Service, bool) {
	for _, s := range c.Services {
		if s.ID == id {
			return s, true
		}
	}
	return Service{}, false
}

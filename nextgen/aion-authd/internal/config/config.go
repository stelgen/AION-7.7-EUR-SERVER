// Package config — yaml-конфиг aion-authd (зеркало стиля aion-gate).
// Секреты (db.connStr) — ТОЛЬКО в конфиге на VM, в гит не попадают
// (config.example.yaml = плейсхолдеры).
package config

import (
	"fmt"
	"os"

	"aion-authd/internal/ship"
	"gopkg.in/yaml.v3"
)

// DBConfig — слой БД (AionAccounts). driver: mem (дефолт, тесты/стенд) | mssql.
type DBConfig struct {
	Driver     string `yaml:"driver"`     // mem | mssql
	ConnStr    string `yaml:"connStr"`    // СЕКРЕТ: только на VM; env AUTHD_CONNSTR перекрывает
	AutoCreate bool   `yaml:"autoCreate"` // автосоздание акка по ASCII-логину (live-факт 06.10)

	// Переопределяемые SQL (дефолты = C1-схема ReleaseAuthDBSchema.sql;
	// реальные имена таблиц/проц AionAccounts сверяются на R0 sp_helptext).
	QAccount  string `yaml:"qAccount"`  // SELECT атрибутов акка по имени
	QInsert   string `yaml:"qInsert"`   // INSERT + возврат uid
	QBlocks   string `yaml:"qBlocks"`   // block_msg по uid
	QLogLogin string `yaml:"qLogLogin"` // ap_SLog-аналог (last_login/last_ip)
}

// Config — корень.
type Config struct {
	ServerPort  int    `yaml:"serverPort"`  // 2110 (serverExPort — наш гейт)
	AuthVersion uint32 `yaml:"authVersion"` // [03]-greeting V (live 0x0000c621)

	ServerID  byte   `yaml:"serverID"`  // id мира в ответах play-ok
	WorldIP   string `yaml:"worldIP"`   // IP мира для type=4 (server-info)
	WorldPort uint16 `yaml:"worldPort"` // порт мира (7777)
	MaxUsers  uint32 `yaml:"maxUsers"`  // поле serverlist (live 2000)
	Type3Unk1 uint32 `yaml:"type3Unk1"` // живой dword из serverlist (a0c69f0b; сверка на R5-диффе)

	StrictAsmBlob           bool   `yaml:"strictAsmBlob"`           // true: login-blob строго 191Б (86Б → тишина — live)
	AutoCreate              bool   `yaml:"autoCreate"`              // дублирует db.autoCreate, если db пуст
	ReloginPolicy           string `yaml:"reloginPolicy"`           // silent (live-паритет: молчим) | fail7
	OnlineTTLSec            int    `yaml:"onlineTtlSec"`            // 300 (live: флаг живёт 2-6 мин)
	SweepSec                int    `yaml:"sweepSec"`                // период чистки онлайн-флагов
	ClearOnlineOnDisconnect bool   `yaml:"clearOnlineOnDisconnect"` // false: [01] флаг НЕ снимает (live)

	FailCodeBadUser uint32 `yaml:"failCodeBadUser"` // не-ASCII/пустой логин (live → 18Б fail)
	FailCodeBlocked uint32 `yaml:"failCodeBlocked"` // block_msg/block_flag
	FailCodeDB      uint32 `yaml:"failCodeDB"`      // ошибка БД (SYSTEM_ERROR)

	// Мир-канал Server64 (serverPort 2104, C1 WorldSrvSocket; docs/authd-2104-recon-20261009.md).
	// GSPort=0 = листенер ВЫКЛ (дефолт: не конкурировать с ориг до R6).
	GSPort         int    `yaml:"gsPort"`         // 2104
	GSAuthVersion  uint32 `yaml:"gsAuthVersion"`  // greeting [03][V][1] (live 2017012601)
	GSHeartbeatSec int    `yaml:"gsHeartbeatSec"` // ping 2104 (live 60)
	GSAcks         bool   `yaml:"gsAcks"`         // квитанции 13-44 на события мира (T2, дефолт false)
	GSRelayTailHex string `yaml:"gsRelayTailHex"` // tail type-0 релея (default = живой корпус)

	DB   DBConfig `yaml:"db"`
	Ship ship.Cfg `yaml:"ship"`
}

// FillDefaults — дефолты по live-фактам.
func (c *Config) FillDefaults() {
	if c.ServerPort == 0 {
		c.ServerPort = 2110
	}
	if c.AuthVersion == 0 {
		c.AuthVersion = 0x0000c621
	}
	if c.ServerID == 0 {
		c.ServerID = 1
	}
	if c.WorldIP == "" {
		c.WorldIP = "192.168.0.125"
	}
	if c.WorldPort == 0 {
		c.WorldPort = 7777
	}
	if c.MaxUsers == 0 {
		c.MaxUsers = 2000
	}
	if c.Type3Unk1 == 0 {
		c.Type3Unk1 = 0xa0c69f0b
	}
	c.StrictAsmBlob = true // live: authd требует asm-размер blob (86Б → тишина)
	if c.ReloginPolicy == "" {
		c.ReloginPolicy = "silent"
	}
	if c.OnlineTTLSec == 0 {
		c.OnlineTTLSec = 300
	}
	if c.SweepSec == 0 {
		c.SweepSec = 30
	}
	if c.FailCodeBadUser == 0 {
		c.FailCodeBadUser = 2
	}
	if c.FailCodeBlocked == 0 {
		c.FailCodeBlocked = 22
	}
	if c.FailCodeDB == 0 {
		c.FailCodeDB = 1
	}
	if c.GSAuthVersion == 0 {
		c.GSAuthVersion = 2017012601 // Server64-лог: «Protocol Version authVersion:2017012601»
	}
	if c.GSHeartbeatSec == 0 {
		c.GSHeartbeatSec = 60 // live packet-лог 09.10
	}
	if c.DB.Driver == "" {
		c.DB.Driver = "mem"
	}
	if c.DB.ConnStr == "" {
		c.DB.ConnStr = os.Getenv("AUTHD_CONNSTR")
	}
	c.AutoCreate = true // live: автосоздание по ASCII-логину (06.10)
	if c.DB.Driver == "mssql" || c.DB.Driver == "sqlserver" {
		c.AutoCreate = c.DB.AutoCreate // для mssql — явный конфиг (безопаснее на чужой БД)
	}
}

// Load — чтение yaml; файл опционален (нет → дефолты).
func Load(path string) (*Config, error) {
	cfg := &Config{}
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("config: %w", err)
		}
		if err := yaml.Unmarshal(b, cfg); err != nil {
			return nil, fmt.Errorf("config: %w", err)
		}
	}
	cfg.FillDefaults()
	return cfg, nil
}

// aion-logd — замена LogServer64. Один исходник, windows+linux.
// Л1–Л4 + ship-стандарт телеметрии (nextgen/TELEMETRY-SPEC.md), 05.10.2026.
package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"aion-logd/internal/logdb"
	"aion-logd/internal/ship"
	"aion-logd/internal/server"
	"aion-logd/internal/writer"
	_ "github.com/microsoft/go-mssqldb"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server server.Config `yaml:"server"`
	LogDB  logdb.Cfg     `yaml:"logdb"`
	Ship   ship.Cfg      `yaml:"ship"`
}

func loadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func main() {
	cfgPath := flag.String("config", "config.yaml", "путь к конфигу")
	flag.Parse()

	cfg, err := loadConfig(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ship-телеметрия (Л5-стандарт): syslog/HTTP/файл; недоступный приёмник не мешает.
	sh := ship.New(cfg.Ship)
	if sh.Enabled() {
		go sh.Run(ctx)
		log.Printf("ship: enabled (syslog=%s tcp=%s http=%v file=%v)",
			cfg.Ship.Syslog.Net, cfg.Ship.Syslog.Host, cfg.Ship.HTTP.URL != "", cfg.Ship.File.Enabled)
	}

	if v := os.Getenv("AIONLOG_MIRROR_UP"); v != "" {
		log.Printf("mirror-режим: listen %s -> upstream %s", cfg.Server.Listen, v)
		_ = server.Mirror(ctx, cfg.Server.Listen, v)
		return
	}
	wr := writer.New(cfg.Server.BaseDir, cfg.Server.Dirs)
	defer wr.CloseAll()

	srv := server.New(cfg.Server, wr)
	srv.SetShip(sh)
	if cfg.LogDB.Enabled && cfg.LogDB.Conn != "" {
		d, err := cfg.LogDB.Open()
		if err != nil {
			log.Fatalf("logdb: %v", err)
		}
		d.Notify = func(ev, msg string, kv map[string]any) {
			sh.Send(ship.Event{Ev: ev, Svc: "logdb", Msg: msg, Data: kv})
		}
		srv.SetDB(d)
		go d.RunTimers(ctx)
		log.Printf("logdb: enabled world=%d server=%d freedisk=%ds status=%ds",
			cfg.LogDB.WorldID, cfg.LogDB.ServerID, cfg.LogDB.FreediskSec, cfg.LogDB.StatusSec)
	}

	ln, err := net.Listen("tcp", cfg.Server.Listen)
	if err != nil {
		log.Fatalf("listen %s: %v", cfg.Server.Listen, err)
	}
	log.Printf("aion-logd: listen %s builder=%d base=%s init_svc=%d retention=%dd",
		cfg.Server.Listen, cfg.Server.Builder, cfg.Server.BaseDir, cfg.Server.InitSvc, cfg.Server.RetentionDays)
	sh.Send(ship.Event{Ev: ship.EvStart, Msg: "aion-logd started", Data: map[string]any{
		"listen": cfg.Server.Listen, "builder": cfg.Server.Builder,
		"logdb": cfg.LogDB.Enabled, "capture_all": cfg.Server.CaptureAll,
		"init_svc": cfg.Server.InitSvc, "retention_days": cfg.Server.RetentionDays}})

	if err := srv.Run(ctx, ln); err != nil {
		log.Fatalf("run: %v", err)
	}
	sent, dropped := sh.Stats()
	log.Printf("aion-logd: shutdown ok (ship sent=%d dropped=%d)", sent, dropped)
	sh.Send(ship.Event{Ev: ship.EvStop, Msg: "aion-logd stopped",
		Data: map[string]any{"sent": sent, "dropped": dropped}})
}
// aion-logd — замена LogServer64 (Phase: skeleton). Один исходник, windows+linux.
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
	"aion-logd/internal/server"
	"aion-logd/internal/writer"
	_ "github.com/microsoft/go-mssqldb"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server server.Config `yaml:"server"`
	LogDB  logdb.Cfg     `yaml:"logdb"`
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

	wr := writer.New(cfg.Server.BaseDir, cfg.Server.Dirs)
	defer wr.CloseAll()

	srv := server.New(cfg.Server, wr)
	if cfg.LogDB.Enabled && cfg.LogDB.Conn != "" {
		d, err := cfg.LogDB.Open()
		if err != nil {
			log.Fatalf("logdb: %v", err)
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
	log.Printf("aion-logd: listen %s builder=%d base=%s (skeleton: unknown payload → .raw)",
		cfg.Server.Listen, cfg.Server.Builder, cfg.Server.BaseDir)

	if err := srv.Run(ctx, ln); err != nil {
		log.Fatalf("run: %v", err)
	}
	log.Printf("aion-logd: shutdown ok")
}

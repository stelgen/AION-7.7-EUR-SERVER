// aion-authd — замена L2Authd.exe для AION 7.7 EU (Go, трек B #5).
//
// MVP (07.10.2026): wire 2110 (наш гейт) + логика логина на live-фактах +
// DB-слой (mem/mssql). Порт 2104 (world/GS-канал Server64) — НЕ реализован
// (роль уточняется дизasmом на R0; включать свитч без него нельзя — см. README).
//
// Дисциплина: прод трогаем ТОЛЬКО по «го» юзера (PROMPT-AUTHD.md).
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"

	"aion-authd/internal/config"
	"aion-authd/internal/server"
	"aion-authd/internal/ship"
	"aion-authd/internal/store"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "путь к config.yaml")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	sh := ship.New(cfg.Ship)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sh.Run(ctx)

	st, err := openStore(cfg)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()

	srv := server.New(cfg, sh, st)
	go srv.RunSweeper(ctx)

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(cfg.ServerPort))
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	log.Printf("aion-authd: :%d V=%#x serverID=%d world=%s:%d db=%s autoCreate=%v onlineTTL=%ds relogin=%s (ship=%v)",
		cfg.ServerPort, cfg.AuthVersion, cfg.ServerID, cfg.WorldIP, cfg.WorldPort,
		cfg.DB.Driver, cfg.AutoCreate, cfg.OnlineTTLSec, cfg.ReloginPolicy, sh.Enabled())
	log.Printf("⚠ MVP: порт 2104 (Server64/world-канал) НЕ реализован — свитч прод только после R0/R5 верификации (см. README)")
	sh.Send(ship.Event{Ev: ship.EvStart, Msg: "aion-authd started", Data: map[string]any{
		"port": cfg.ServerPort, "auth_version": cfg.AuthVersion, "db": cfg.DB.Driver,
	}})
	if err := srv.Serve(ln); err != nil {
		sh.Send(ship.Event{Ev: ship.EvStop, Msg: "aion-authd stopped", Err: err.Error()})
		log.Fatal(err)
	}
}

// openStore — выбор БД по конфигу: mem (дефолт) | mssql.
func openStore(cfg *config.Config) (store.Store, error) {
	switch cfg.DB.Driver {
	case "mem":
		return store.NewMap(), nil
	case "mssql", "sqlserver":
		s, err := store.NewSQL(cfg.DB.Driver, cfg.DB.ConnStr)
		if err != nil {
			return nil, err
		}
		s.SetQueries(cfg.DB.QAccount, cfg.DB.QInsert, cfg.DB.QBlocks, cfg.DB.QLogLogin)
		return s, nil
	default:
		return nil, fmt.Errorf("store: неизвестный driver=%q (mem|mssql)", cfg.DB.Driver)
	}
}

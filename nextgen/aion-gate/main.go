package main

import (
	"context"
	"flag"
	"log"
	"net"
	"strconv"

	"aion-gate/internal/config"
	"aion-gate/internal/server"
	"aion-gate/internal/ship"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "путь к config.yaml")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	g := &cfg.Gate

	sh := ship.New(g.Ship)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sh.Run(ctx)

	srv, err := server.New(*g, sh)
	if err != nil {
		log.Fatalf("server: %v", err)
	}
	if err := srv.DialAuthd(); err != nil {
		// Оригинал при недоступном authd молчит и не реконнектит — не фатально.
		log.Printf("authd: %v (работаю без authd, authReconnectInterval=%d)", err, g.AuthReconnectInterval)
	}

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(g.ServerPort))
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	log.Printf("aion-gate: :%d → authd %s:%d (ship=%v)", g.ServerPort, g.AuthAddr, g.AuthPort, sh.Enabled())
	sh.Send(ship.Event{Ev: ship.EvStart, Msg: "aion-gate started", Data: map[string]any{
		"port": g.ServerPort, "authd": net.JoinHostPort(g.AuthAddr, strconv.Itoa(g.AuthPort)),
	}})
	if err := srv.Serve(ln); err != nil {
		sh.Send(ship.Event{Ev: ship.EvStop, Msg: "aion-gate stopped", Err: err.Error()})
		log.Fatal(err)
	}
}

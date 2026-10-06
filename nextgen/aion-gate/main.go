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
	switch g.Mode {
	case "authgate":
		if err := srv.DialAuthd(); err != nil {
			// Оригинал при недоступном authd молчит и не реконнектит — не фатально.
			log.Printf("authd: %v (работаю без authd, authReconnectInterval=%d)", err, g.AuthReconnectInterval)
		}
	case "classic":
		log.Printf("mode=classic: standalone-флоу 4.8 (без authd; SessionKey случайный, GS-прокси TODO)")
	case "fork":
		log.Printf("mode=fork: клиент :%d ↔ оригинал %s:%d (raw-релей + shadow-сравнение; authd не подключаем)",
			g.ServerPort, g.ForkOrigAddr, g.ForkOrigPort)
	default:
		log.Fatalf("config: неизвестный mode=%q (authgate|classic|fork)", g.Mode)
	}

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(g.ServerPort))
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	log.Printf("aion-gate: mode=%s :%d rsa_exponent=%d authd=%s:%d (ship=%v)",
		g.Mode, g.ServerPort, g.RsaExponent, g.AuthAddr, g.AuthPort, sh.Enabled())
	log.Printf("authd-fail: loginTimeout=%ds playTimeout=%ds onlineTtl=%ds codes login=%d play=%d online=%d failClose=%ds",
		g.LoginTimeoutSec, g.PlayTimeoutSec, g.OnlineTtlSec, g.LoginFailCode, g.PlayFailCode, g.LoginFailOnline, g.FailCloseSec)
	sh.Send(ship.Event{Ev: ship.EvStart, Msg: "aion-gate started", Data: map[string]any{
		"port": g.ServerPort, "mode": g.Mode, "rsa_exponent": g.RsaExponent,
		"authd": net.JoinHostPort(g.AuthAddr, strconv.Itoa(g.AuthPort)),
	}})
	if err := srv.Serve(ln); err != nil {
		sh.Send(ship.Event{Ev: ship.EvStop, Msg: "aion-gate stopped", Err: err.Error()})
		log.Fatal(err)
	}
}
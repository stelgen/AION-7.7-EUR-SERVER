package main

import (
	"flag"
	"log"
	"net"
	"strconv"

	"aion-gate/internal/config"
	"aion-gate/internal/server"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "путь к config.yaml")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	g := &cfg.Gate

	srv, err := server.New(*g)
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
	log.Printf("aion-gate: :%d → authd %s:%d (ship=%v)", g.ServerPort, g.AuthAddr, g.AuthPort, g.Ship["enabled"])
	log.Fatal(srv.Serve(ln))
}

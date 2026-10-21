// aion-cached — R1 прототип CacheD64 wire-фреймворка (Go).
// Канал-карта: 2006=server (RQ/RP), 2007=admin interactive (GQ/GP), 2009=npcDb.
package main

import (
	"flag"
	"log"

	"aion-cached/internal/server"
)

func main() {
	cfgPath := flag.String("config", "", "path to config file (optional)")
	flag.Parse()

	cfg := server.DefaultConfig()
	if *cfgPath != "" {
		if err := cfg.Load(*cfgPath); err != nil {
			log.Fatalf("config load: %v", err)
		}
	}

	srv := server.New(cfg)
	log.Fatal(srv.Run())
}

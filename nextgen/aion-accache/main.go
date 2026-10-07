// aion-accache — перепись AccountCacheServer 7.7 (порт 2220).
// Каркас R2: фрейм + dispatch + RAM-кэш + db-интерфейс, БЕЗ capture.
// SQLStore (go-mssqldb) подключается в R3; connStr — секрет только на VM/env.
package main

import (
	"flag"
	"log"

	"aion-accache/internal/cache"
	"aion-accache/internal/config"
	"aion-accache/internal/server"
	"aion-accache/internal/ship"
)

var version = "dev"

func main() {
	cfgPath := flag.String("config", "config.yaml", "yaml config")
	vv := flag.Bool("v", false, "verbose")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if *vv {
		cfg.Server.Verbose = true
	}

	sh := ship.New(cfg.Ship)
	go sh.Run(nil)
	sh.Send(ship.Event{Ev: "start", Svc: "accache", Msg: "version=" + version + " listen=" + cfg.Server.Listen})

	st := cache.New()
	srv := server.New(cfg, st, nil, sh) // db.Executor = nil до R3 (SQLStore)
	if err := srv.Serve(); err != nil {
		sh.Send(ship.Event{Ev: "stop", Svc: "accache", Msg: "err=" + err.Error()})
		log.Fatalf("serve: %v", err)
	}
}

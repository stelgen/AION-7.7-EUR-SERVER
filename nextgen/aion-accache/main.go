// aion-accache — перепись AccountCacheServer 7.7 (порт 2220).
// Каркас R2: фрейм + dispatch + RAM-кэш + db-интерфейс, БЕЗ capture.
// SQLStore (go-mssqldb) подключается в R3; connStr — секрет только на VM/env.
package main

import (
	"context"
	"flag"
	"log"

	"aion-accache/internal/cache"
	"aion-accache/internal/config"
	"aion-accache/internal/db"
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
	// ГОТЧА (22.10, live-деплой): Run(nil) паниковал на ctx.Done() — nil-интерфейс;
	// всегда context.Background().
	go sh.Run(context.Background())
	sh.Send(ship.Event{Ev: "start", Svc: "accache", Msg: "version=" + version + " listen=" + cfg.Server.Listen})

	st := cache.New()
	exec := db.Executor(nil) // SQLStore (R3) или nil (RAM-only)
	if cfg.DB.Enabled {
		store, err := db.NewSQLStore(cfg.DB.Conn)
		if err != nil {
			sh.Send(ship.Event{Ev: "stop", Svc: "accache", Msg: "db: " + err.Error()})
			log.Fatalf("db: %v", err)
		}
		exec = store
		if cfg.DB.ReadOnly {
			exec = db.NewReadOnly(exec)
			log.Printf("aion-accache: SQLStore READ-ONLY (тень A/B: SELECT идут, Exec глушится)")
		} else {
			log.Printf("aion-accache: SQLStore подключен (procs AionAccountCacheD)")
		}
	}
	srv := server.New(cfg, st, exec, sh)
	if err := srv.Serve(); err != nil {
		sh.Send(ship.Event{Ev: "stop", Svc: "accache", Msg: "err=" + err.Error()})
		log.Fatalf("serve: %v", err)
	}
}

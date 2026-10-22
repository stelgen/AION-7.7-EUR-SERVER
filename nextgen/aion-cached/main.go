// aion-cached — CacheD64 wire-фреймворк (Go). R4 MVP с DB-персистом настроек.
// Каналы: 2006=server (RQ/RP), 2007=admin, 2009=npcDb. Сессии marker AE/AF (Т26).
package main

import (
	"flag"
	"log"

	"aion-cached/internal/db"
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
	// DB-персист настроек (Т30): db_conn из конфига или env
	conn := cfg.DBConn
	if conn == "" {
		log.Printf("[cached] no db_conn — settings RAM-only")
	} else {
		store, err := db.NewSQLStore(conn)
		if err != nil {
			log.Printf("[cached] DB connect FAIL: %v — settings RAM-only", err)
		} else {
			srv.SDB = &server.SettingsDB{Ex: store}
			log.Printf("[cached] SettingsDB connected (AionAccountCacheD_rc)")
		}
	}
	log.Fatal(srv.Run())
}

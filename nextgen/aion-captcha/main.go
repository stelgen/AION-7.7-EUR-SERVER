// aion-captcha — замена CAPTCHAImageServer.exe. Один исходник, windows+linux.
// Протокол capture-верифицирован (docs/captcha-protocol-20261005.md).
// Телеметрия по nextgen/TELEMETRY-SPEC.md (ship из aion-logd).
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"aion-captcha/internal/config"
	"aion-captcha/internal/server"
	"aion-captcha/internal/ship"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "путь к конфигу")
	verbose := flag.Bool("v", false, "verbose stdout")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if *verbose {
		cfg.Server.Verbose = true
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sh := ship.New(cfg.Ship)
	if sh.Enabled() {
		go sh.Run(ctx)
		log.Printf("ship: enabled (syslog=%s http=%v file=%v)", cfg.Ship.Syslog.Net, cfg.Ship.HTTP.URL != "", cfg.Ship.File.Enabled)
	}

	srv := server.New(cfg.Server, sh)
	if err := srv.Run(ctx); err != nil {
		log.Fatalf("server: %v", err)
	}
}

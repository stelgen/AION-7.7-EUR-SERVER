// aion-op — единый оператор стека AION 7.7 (Phase 0: observe-only, прод не трогаем).
package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"aion-op/internal/config"
	"aion-op/internal/probe"
	"aion-op/internal/web"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "путь к YAML-конфигу")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	var prober probe.Prober
	switch cfg.VM.Mode {
	case "mock":
		down := map[string]bool{}
		if v := os.Getenv("AIONOP_MOCK_DOWN"); v != "" {
			for _, id := range strings.Split(v, ",") {
				down[strings.TrimSpace(id)] = true
			}
		}
		prober = probe.NewMock(cfg, down)
	case "ssh":
		prober = probe.NewSSH(cfg)
	default:
		log.Fatalf("vm.mode: неизвестный режим %q (mock|ssh)", cfg.VM.Mode)
	}

	srv := web.New(cfg, prober)
	log.Printf("aion-op: probe=%s ui=:%d mode=%s (Phase 0: observe-only, кнопки заблокированы)",
		cfg.VM.Mode, cfg.Operator.UIPort, cfg.Operator.Mode)
	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}

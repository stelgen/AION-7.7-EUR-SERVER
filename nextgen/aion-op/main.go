// aion-op — единый оператор стека AION 7.7 (Phase 0.5: observe-only, прод не трогаем).
// Глаза: лог-тейлеры+парсер, метрики RAM/handles/commit, SQLite-WAL, алерты, pprof.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof" // только на loopback (cfg.Pprof.Addr)
	"os"
	"strings"
	"time"

	"aion-op/internal/alerts"
	"aion-op/internal/config"
	"aion-op/internal/logs"
	"aion-op/internal/metrics"
	"aion-op/internal/probe"
	"aion-op/internal/store"
	"aion-op/internal/web"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "путь к YAML-конфигу")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	st, err := store.Open(cfg.Store.Path, cfg.Store.RetentionDays)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()

	var prober probe.Prober
	var runner probe.Runner
	var tailer logs.Tailer
	var coll metrics.Collector
	switch cfg.VM.Mode {
	case "mock":
		down := map[string]bool{}
		if v := os.Getenv("AIONOP_MOCK_DOWN"); v != "" {
			for _, id := range strings.Split(v, ",") {
				down[strings.TrimSpace(id)] = true
			}
		}
		prober = probe.NewMock(cfg, down)
		tailer = logs.NewMock()
		coll = metrics.NewMock(cfg)
	case "ssh":
		ssh := probe.NewSSH(cfg)
		prober, runner = ssh, ssh
		tailer = logs.NewSSH(cfg, runner)
		coll = metrics.NewSSH(cfg, runner)
	default:
		log.Fatalf("vm.mode: неизвестный режим %q (mock|ssh)", cfg.VM.Mode)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng := alerts.New(st)
	holder := &metrics.Holder{}

	// События логов → SQLite + движок алертов.
	evCh := make(chan logs.Event, 512)
	go tailer.Start(ctx, evCh)
	go func() {
		for ev := range evCh {
			st.AddEvent(store.EventRow{
				Ts: ev.When.Unix(), Svc: ev.Svc, Kind: ev.Kind, Sev: ev.Sev, Text: ev.Text,
			})
			eng.FeedEvent(ev)
		}
	}()

	// Метрики: срез → holder + движок + SQLite (WAL, retention по конфигу).
	go func() {
		t := time.NewTicker(time.Duration(cfg.Metrics.PollSec) * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				ms := coll.Collect(ctx)
				holder.Set(ms)
				eng.FeedMetrics(ms)
				if ms.Err == "" {
					rows := make([]store.MetricRow, len(ms.Procs))
					for i, p := range ms.Procs {
						rows[i] = store.MetricRow{
							Name: p.Name, PID: p.PID, MemMB: p.MemMB,
							Handles: p.Handles, DPM: p.HandlesDPM,
						}
					}
					st.AddMetrics(ms.When, rows)
					st.AddSys(ms.When, ms.Sys.FreePhysMB, ms.Sys.FreeCommitMB)
				}
			}
		}
	}()

	if cfg.Pprof.Enabled {
		go func() {
			log.Printf("pprof: http://%s/debug/pprof/ (loopback)", cfg.Pprof.Addr)
			_ = http.ListenAndServe(cfg.Pprof.Addr, nil)
		}()
	}

	srv := web.New(cfg, prober, st, eng, holder)
	log.Printf("aion-op: probe=%s ui=:%d mode=%s store=%s (Phase 0.5: observe-only, кнопки заблокированы)",
		cfg.VM.Mode, cfg.Operator.UIPort, cfg.Operator.Mode, cfg.Store.Path)
	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}

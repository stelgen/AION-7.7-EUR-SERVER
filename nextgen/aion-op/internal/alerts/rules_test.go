package alerts

import (
	"path/filepath"
	"testing"
	"time"

	"aion-op/internal/core"
	"aion-op/internal/logs"
	"aion-op/internal/metrics"
	"aion-op/internal/store"
)

func newEng(t *testing.T) *Engine {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "t.db"), 30)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(s.Close)
	return New(s)
}

func statuses(main core.State) []core.SvcStatus {
	return []core.SvcStatus{
		{ID: "main", Display: "Server64", State: main},
		{ID: "npc", Display: "NPCSvr64", State: core.StateRunning},
		{ID: "gate", Display: "AuthGateD", State: core.StateRunning},
		{ID: "chat", Display: "ChannelChat", State: core.StateStopped, Locked: true},
	}
}

func byIdem(a []store.AlertRow) map[string]store.AlertRow {
	m := map[string]store.AlertRow{}
	for _, r := range a {
		m[r.Idem] = r
	}
	return m
}

func TestDownAndRecover(t *testing.T) {
	e := newEng(t)
	e.Tick(statuses(core.StateStopped), "", false)
	m := byIdem(e.Snapshot())
	if _, ok := m["down:main"]; !ok {
		t.Fatalf("нет алерта down:main: %+v", m)
	}
	if _, ok := m["down:chat"]; ok {
		t.Fatal("locked-сервис не должен алертиться")
	}
	e.Tick(statuses(core.StateRunning), "", false)
	if m := byIdem(e.Snapshot()); len(m) != 0 {
		t.Fatalf("после восстановления: %+v", m)
	}
}

func TestProbeAndPair(t *testing.T) {
	e := newEng(t)
	e.Tick(statuses(core.StateRunning), "ssh timeout", true)
	m := byIdem(e.Snapshot())
	if _, ok := m["probe"]; !ok {
		t.Fatal("нет probe-алерта")
	}
	if _, ok := m["pair"]; !ok {
		t.Fatal("нет pair-алерта")
	}
	e.Tick(statuses(core.StateRunning), "", false)
	if len(byIdem(e.Snapshot())) != 0 {
		t.Fatal("алерты не закрылись")
	}
}

func TestFreeCommitHysteresis(t *testing.T) {
	e := newEng(t)
	e.FeedMetrics(metrics.Snap{Sys: metrics.SysMem{FreeCommitMB: 4000}})
	e.Tick(statuses(core.StateRunning), "", false)
	if _, ok := byIdem(e.Snapshot())["freecommit"]; !ok {
		t.Fatal("нет freecommit-алерта при 4 ГБ")
	}
	e.FeedMetrics(metrics.Snap{Sys: metrics.SysMem{FreeCommitMB: 9000}}) // зона гистерезиса
	e.Tick(statuses(core.StateRunning), "", false)
	if _, ok := byIdem(e.Snapshot())["freecommit"]; !ok {
		t.Fatal("гистерезис: алерт должен остаться до 10 ГБ")
	}
	e.FeedMetrics(metrics.Snap{Sys: metrics.SysMem{FreeCommitMB: 11000}})
	e.Tick(statuses(core.StateRunning), "", false)
	if _, ok := byIdem(e.Snapshot())["freecommit"]; ok {
		t.Fatal("алерт должен закрыться при 11 ГБ")
	}
}

func TestHandleLeakSustained(t *testing.T) {
	e := newEng(t)
	ms := metrics.Snap{Procs: []metrics.ProcMetric{{Name: "Server64.exe", Handles: 800000, HandlesDPM: 120000}}}
	for i := 0; i < leakTicksNeeded-1; i++ {
		e.FeedMetrics(ms)
		e.Tick(statuses(core.StateRunning), "", false)
	}
	if _, ok := byIdem(e.Snapshot())["leak:Server64.exe"]; ok {
		t.Fatal("алерт раньше 3 тиков")
	}
	e.FeedMetrics(ms)
	e.Tick(statuses(core.StateRunning), "", false)
	if _, ok := byIdem(e.Snapshot())["leak:Server64.exe"]; !ok {
		t.Fatal("нет leak-алерта после 3 тиков")
	}
	clean := metrics.Snap{Procs: []metrics.ProcMetric{{Name: "Server64.exe", Handles: 800000, HandlesDPM: 0}}}
	e.FeedMetrics(clean)
	e.Tick(statuses(core.StateRunning), "", false)
	if _, ok := byIdem(e.Snapshot())["leak:Server64.exe"]; ok {
		t.Fatal("leak-алерт не закрылся")
	}
}

func TestRate2812AndCritLog(t *testing.T) {
	e := newEng(t)
	for i := 0; i < rate2812Max+10; i++ {
		e.FeedEvent(logs.Event{Kind: "proc_missing", Sev: logs.SevLow})
	}
	e.Tick(statuses(core.StateRunning), "", false)
	if _, ok := byIdem(e.Snapshot())["rate:2812"]; !ok {
		t.Fatal("нет rate:2812")
	}

	e.FeedEvent(logs.Event{Kind: "super_lag", Sev: logs.SevCrit})
	e.Tick(statuses(core.StateRunning), "", false)
	if _, ok := byIdem(e.Snapshot())["crit_log"]; !ok {
		t.Fatal("нет crit_log")
	}
	// имитируем «давно»
	e.mu.Lock()
	e.lastCrit = time.Now().Add(-critLogKeep - time.Minute)
	e.mu.Unlock()
	e.Tick(statuses(core.StateRunning), "", false)
	if _, ok := byIdem(e.Snapshot())["crit_log"]; ok {
		t.Fatal("crit_log не закрылся после 15 мин чисто")
	}
}

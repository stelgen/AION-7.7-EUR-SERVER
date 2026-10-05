package store

import (
	"path/filepath"
	"testing"
	"time"
)

func openTmp(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"), 30)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(s.Close)
	return s
}

func TestEventsRoundtrip(t *testing.T) {
	s := openTmp(t)
	now := time.Now()
	s.AddEvent(EventRow{Ts: now.Unix(), Svc: "cache", Kind: "too_slow", Sev: 2, Text: "Too slow 3312ms"})
	s.AddEvent(EventRow{Ts: now.Unix(), Svc: "main", Kind: "super_lag", Sev: 3, Text: "DeadLock or Super-Lag"})
	s.Flush()

	evs := s.RecentEvents(10)
	if len(evs) != 2 {
		t.Fatalf("events: %d", len(evs))
	}
	if evs[0].Kind != "super_lag" { // новые сначала
		t.Fatalf("order: %+v", evs[0])
	}
	if n := s.CountEvents("too_slow", time.Hour); n != 1 {
		t.Fatalf("count: %d", n)
	}
	if n := s.CountEvents("proc_missing", time.Hour); n != 0 {
		t.Fatalf("count чужого kind: %d", n)
	}
}

func TestMetricsSeriesAndPurge(t *testing.T) {
	s := openTmp(t)
	now := time.Now()
	old := now.Add(-48 * time.Hour)

	s.AddMetrics(old, []MetricRow{{Name: "Server64.exe", PID: "1", MemMB: 10000, Handles: 800000}})
	s.AddMetrics(now, []MetricRow{
		{Name: "Server64.exe", PID: "1", MemMB: 10300, Handles: 827000, DPM: 500},
		{Name: "NPCSvr64.exe", PID: "2", MemMB: 14900, Handles: 578000},
	})
	s.AddSys(now, 9900, 44000)
	s.Flush()

	pts := s.Series("Server64.exe", 240)
	if len(pts) != 1 || pts[0].Handles != 827000 {
		t.Fatalf("series: %+v", pts)
	}

	s.PurgeNow(now.Add(-24 * time.Hour))
	s.Flush()
	if pts := s.Series("Server64.exe", 24*60); len(pts) != 1 {
		t.Fatalf("purge не удалил старую метрику: %+v", pts)
	}
}

func TestAlertsLifecycle(t *testing.T) {
	s := openTmp(t)
	s.OpenAlert("down:main", 2, "Server64 не запущен")
	s.OpenAlert("down:main", 2, "Server64 не запущен") // повтор — без дублей
	s.Flush()
	if al := s.ActiveAlerts(); len(al) != 1 || al[0].Idem != "down:main" {
		t.Fatalf("active: %+v", al)
	}
	s.CloseAlert("down:main")
	s.Flush()
	if al := s.ActiveAlerts(); len(al) != 0 {
		t.Fatalf("после close: %+v", al)
	}
	// переоткрытие: текст обновился, активен снова
	s.OpenAlert("down:main", 2, "снова")
	s.Flush()
	al := s.ActiveAlerts()
	if len(al) != 1 || al[0].Text != "снова" {
		t.Fatalf("reopen: %+v", al)
	}
}

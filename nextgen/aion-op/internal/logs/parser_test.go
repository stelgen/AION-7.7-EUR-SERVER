package logs

import (
	"strings"
	"testing"
	"time"
)

var now = time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)

func TestRules(t *testing.T) {
	cases := []struct {
		line string
		kind string
		sev  int
		want bool
	}{
		{"00:13:41 : Deadlock Detected by CheckIOThreadDeadlock() ... IOThread 2 passedTick 181156", "super_lag", SevCrit, true},
		{"DeadLock or Super-Lag detected !", "super_lag", SevCrit, true},
		{"Intentional exception", "crash_precursor", SevCrit, true},
		{"Shutdown By NpcSocket Close", "world_shutdown", SevCrit, true},
		{"Server shutdown started", "shutdown", SevMed, true},
		{"AboutToPlayerTimer ::== Client(Acct: 1010) is not connected", "login_wait", SevMed, true},
		{"[2] Too slow function, execution time = 3312 mili second", "too_slow", SevMed, true},
		{"Time difference with Log-Server 153s", "date_mismatch", SevMed, true},
		{"Could not find stored procedure 'aion_LoadFameInfo'", "proc_missing", SevLow, true},
		{"Disallowed gamesession login (loginType=1)", "disallowed", SevLow, true},
		{"WARN Session id mismatched (sessionId:0)", "session_mismatch", SevLow, true},
		{"ClientLoginTry:9", "login", SevInfo, true},
		{"*new world server connection from 127.0.0.1", "world_registered", SevInfo, true},
		{"NPC Server Started", "npc_started", SevInfo, true},
		{"Strings DB unexpted id 123456", "", 0, false}, // шум — не храним
		{"обычная строка лога без ключей", "", 0, false},
	}
	for _, c := range cases {
		ev, ok := Parse("cache", c.line, now)
		if ok != c.want {
			t.Errorf("%q: ok=%v, want %v", c.line, ok, c.want)
			continue
		}
		if !ok {
			continue
		}
		if ev.Kind != c.kind || ev.Sev != c.sev {
			t.Errorf("%q: got %s/%d, want %s/%d", c.line, ev.Kind, ev.Sev, c.kind, c.sev)
		}
		if ev.Svc != "cache" {
			t.Errorf("svc: %s", ev.Svc)
		}
	}
}

func TestTimestamps(t *testing.T) {
	ev, ok := Parse("cache", "2026-10-05 02:37:57 : Could not find stored procedure 'x'", now)
	if !ok || ev.When.Format("2006-01-02 15:04:05") != "2026-10-05 02:37:57" {
		t.Fatalf("full-ts: %+v ok=%v", ev, ok)
	}
	ev, ok = Parse("cache", "00:13:41 : Too slow function", now)
	if !ok || ev.When.Format("15:04:05") != "00:13:41" || ev.When.Day() != now.Day() {
		t.Fatalf("time-ts: %+v ok=%v", ev, ok)
	}
}

func TestTrunc(t *testing.T) {
	long := strings.Repeat("x", 600)
	if got := trunc(long, 500); len(got) != 503 { // 500 байт + «…» (3 байта UTF-8)
		t.Fatalf("trunc: %d", len(got))
	}
	ev, ok := Parse("cache", "Too slow function, execution time = "+long, now)
	if !ok || len(ev.Text) != 503 {
		t.Fatalf("parse trunc: %d ok=%v", len(ev.Text), ok)
	}
}

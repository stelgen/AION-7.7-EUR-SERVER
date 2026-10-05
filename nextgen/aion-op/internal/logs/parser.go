// Package logs — парсер строк логов NC-компонентов → события (Phase 0.5).
// Правила выведены из эксплуатации 02–05.10 (CacheD крэш-цепочка, authd, Server64).
package logs

import (
	"regexp"
	"strings"
	"time"
)

// Sev — серьёзность события.
const (
	SevNoise = -1 // шум: не хранить
	SevInfo  = 0
	SevLow   = 1
	SevMed   = 2
	SevCrit  = 3
)

// Event — одно распознанное событие из лога.
type Event struct {
	When time.Time `json:"-"`
	Svc  string    `json:"svc"`  // id сервиса (cache/main/npc/logsrv/authd/gate)
	Kind string    `json:"kind"` // super_lag / proc_missing / too_slow / ...
	Sev  int       `json:"sev"`
	Text string    `json:"text"` // строка лога (обрезана)
}

type rule struct {
	kind string
	sev  int
	re   *regexp.Regexp
}

// Порядок правил = приоритет матча (критичные раньше).
var rules = []rule{
	{"super_lag", SevCrit, regexp.MustCompile(`(?i)Deadlock Detected by CheckIOThreadDeadlock|DeadLock or Super-Lag`)},
	{"crash_precursor", SevCrit, regexp.MustCompile(`Intentional exception`)},
	{"world_shutdown", SevCrit, regexp.MustCompile(`Shutdown By NpcSocket Close`)},
	{"shutdown", SevMed, regexp.MustCompile(`(?i)Server shutdown started`)},
	{"login_wait", SevMed, regexp.MustCompile(`AboutToPlayerTimer`)},
	{"too_slow", SevMed, regexp.MustCompile(`Too slow function`)},
	{"date_mismatch", SevMed, regexp.MustCompile(`Time difference with Log-Server`)},
	{"proc_missing", SevLow, regexp.MustCompile(`(?i)Could not find stored procedure`)},
	{"disallowed", SevLow, regexp.MustCompile(`Disallowed gamesession login`)},
	{"session_mismatch", SevLow, regexp.MustCompile(`Session id mismatched`)},
	{"login", SevInfo, regexp.MustCompile(`ClientLoginTry`)},
	{"world_registered", SevInfo, regexp.MustCompile(`new world server connection`)},
	{"npc_started", SevInfo, regexp.MustCompile(`NPC Server Started`)},
}

var (
	noiseRe = regexp.MustCompile(`Strings DB unexpted id`)
	tsFull  = regexp.MustCompile(`(\d{4}-\d{2}-\d{2})[ T](\d{2}:\d{2}:\d{2})`)
	tsTime  = regexp.MustCompile(`^\s*(\d{2}:\d{2}:\d{2})`)
)

// Parse — строка лога → событие. false = пропустить (нет матча или шум).
func Parse(svc, line string, now time.Time) (Event, bool) {
	if noiseRe.MatchString(line) {
		return Event{}, false // Strings DB: ~570k строк за рестарт — дедуп по правилу
	}
	for _, r := range rules {
		if !r.re.MatchString(line) {
			continue
		}
		ev := Event{When: now, Svc: svc, Kind: r.kind, Sev: r.sev, Text: trunc(line, 500)}
		if m := tsFull.FindStringSubmatch(line); m != nil {
			if t, err := time.Parse("2006-01-02 15:04:05", m[1]+" "+m[2]); err == nil {
				ev.When = t
			}
		} else if m := tsTime.FindStringSubmatch(line); m != nil {
			if t, err := time.Parse("15:04:05", m[1]); err == nil {
				ev.When = time.Date(now.Year(), now.Month(), now.Day(),
					t.Hour(), t.Minute(), t.Second(), 0, now.Location())
			}
		}
		return ev, true
	}
	return Event{}, false
}

func trunc(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

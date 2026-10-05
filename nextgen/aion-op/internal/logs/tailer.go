// Тейлеры лог-файлов VM: ssh (read-only Get-Content -Tail) | mock (сценарии). Phase 0.5.
package logs

import (
	"context"
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"strings"
	"time"

	"aion-op/internal/config"
)

// Runner — исполнитель read-only команды на VM (probe.SSH).
type Runner interface {
	Run(ctx context.Context, remote string) (string, error)
}

// Tailer отдаёт события в канал (без блокировки капнутого канала — дроп с логом).
type Tailer interface {
	Start(ctx context.Context, out chan<- Event)
}

// renderDate — подстановка {{date}} → YYYY-MM-DD (файлы err у NC = по дню).
func renderDate(path string, now time.Time) string {
	return strings.ReplaceAll(path, "{{date}}", now.Format("2006-01-02"))
}

func send(out chan<- Event, ev Event) {
	select {
	case out <- ev:
	default:
		log.Printf("logs: канал полон, событие дропнуто: %s %s", ev.Svc, ev.Kind)
	}
}

// --- SSH ---

type SSH struct {
	cfg    *config.Config
	runner Runner
	seen   map[string]map[uint32]struct{} // файл → хэши уже отправленных строк
}

func NewSSH(cfg *config.Config, r Runner) *SSH {
	return &SSH{cfg: cfg, runner: r, seen: map[string]map[uint32]struct{}{}}
}

func (s *SSH) Start(ctx context.Context, out chan<- Event) {
	t := time.NewTicker(time.Duration(s.cfg.Logs.PollSec) * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			now := time.Now()
			for _, f := range s.cfg.Logs.Files {
				path := renderDate(f.Path, now)
				cmd := fmt.Sprintf(
					"powershell -NoProfile -Command \"Get-Content -LiteralPath '%s' -Tail %d\"",
					path, s.cfg.Logs.TailLines)
				outTxt, err := s.runner.Run(ctx, cmd)
				if err != nil {
					continue // файл ещё не создан (новый день/не запускался) — штатно
				}
				s.emit(f.Svc, path, outTxt, now, out)
			}
		}
	}
}

func (s *SSH) emit(svc, path, txt string, now time.Time, out chan<- Event) {
	seen := s.seen[path]
	if seen == nil {
		seen = map[uint32]struct{}{}
		s.seen[path] = seen
	}
	for _, line := range strings.Split(txt, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		h := fnv.New32a()
		_, _ = h.Write([]byte(line))
		sum := h.Sum32()
		if _, dup := seen[sum]; dup {
			continue
		}
		if len(seen) > 20000 { // кольцо: хвост всегда 200 строк, лимит с запасом
			s.seen[path] = map[uint32]struct{}{}
			seen = s.seen[path]
		}
		seen[sum] = struct{}{}
		if ev, ok := Parse(svc, line, now); ok {
			send(out, ev)
		}
	}
}

// --- Mock: детерминированный ротационный сценарий для разработки/демо ---

type Mock struct {
	every int // событие каждые N тиков по 2с
}

func NewMock() *Mock {
	every := 2
	if v := os.Getenv("AIONOP_MOCK_EVENT_EVERY"); v != "" {
		if n, err := fmt.Sscanf(v, "%d", &every); err != nil || n != 1 || every < 1 {
			every = 2
		}
	}
	return &Mock{every: every}
}

var mockScenario = []struct{ svc, kind, text string }{
	{"main", "world_registered", "*new world server connection from 127.0.0.1"},
	{"npc", "npc_started", "NPC Server Started"},
	{"authd", "login", "ClientLoginTry:1 (Acct: 1010)"},
	{"cache", "proc_missing", "Could not find stored procedure 'aion_LoadFameInfo'"},
	{"cache", "too_slow", "[2] Too slow function, execution time = 3312 mili second, curTick(193422)"},
	{"gate", "session_mismatch", "WARN Session id mismatched (sessionId:0 vs 4, IP 127.0.0.1)"},
	{"main", "login_wait", "AboutToPlayerTimer ::== Client(Acct: 1010) is not connected"},
	{"cache", "super_lag", "DeadLock or Super-Lag detected ! CheckIOThreadDeadlock() IOThread 8"},
	{"main", "world_shutdown", "Shutdown By NpcSocket Close"},
	{"cache", "proc_missing", "Could not find stored procedure 'aion_DeleteItemByDate_20191206'"},
}

func (m *Mock) Start(ctx context.Context, out chan<- Event) {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	i := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if i%m.every != 0 {
				continue
			}
			s := mockScenario[(i/m.every)%len(mockScenario)]
			send(out, Event{
				When: time.Now(), Svc: s.svc, Kind: s.kind,
				Sev: sevOfKind(s.kind), Text: time.Now().Format("15:04:05") + " : " + s.text,
			})
			i++
		}
	}
}

func sevOfKind(kind string) int {
	for _, r := range rules {
		if r.kind == kind {
			return r.sev
		}
	}
	return SevInfo
}

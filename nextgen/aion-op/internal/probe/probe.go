// Package probe — read-only срезы состояния VM. Phase 0: mock | ssh. Phase 1: агент.
package probe

import (
	"context"
	"strconv"
	"strings"
	"time"
)

// Proc — процесс на VM.
type Proc struct {
	Name    string
	PID     string
	Session string // Services | Console
	MemKB   uint64
}

// Snapshot — один срез состояния VM (только чтение).
type Snapshot struct {
	When           time.Time
	Source         string
	Err            string            // непустой = проба не удалась, всё UNKNOWN
	Processes      map[string][]Proc // exe (lowercase) → процессы
	Ports          map[int]bool      // TCP LISTENING
	Conns2002      int               // ESTABLISHED с локальным портом 2002 (сторона Server64)
	ConsoleSession bool              // интерактивная сессия (quser console Active) — нужна /IT-сервисам
}

// Prober снимает срезы.
type Prober interface {
	Snapshot() Snapshot
}

// Runner — исполнитель read-only команд на VM (лог-тейлеры/метрики Phase 0.5:
// Get-Content -Tail, Get-Process). Никаких изменений состояния.
type Runner interface {
	Run(ctx context.Context, remote string) (string, error)
}

// --- парсеры вывода Windows (используют mock и ssh) ---

func parseTasklistCSV(out string) map[string][]Proc {
	res := map[string][]Proc{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, `"`) {
			continue
		}
		f := strings.Split(line, `","`)
		if len(f) < 5 {
			continue
		}
		name := strings.Trim(f[0], `"`)
		pid := strings.Trim(f[1], `"`)
		sess := strings.Trim(f[2], `"`)
		digits := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, f[4])
		mem, _ := strconv.ParseUint(digits, 10, 64)
		key := strings.ToLower(name)
		res[key] = append(res[key], Proc{Name: name, PID: pid, Session: sess, MemKB: mem})
	}
	return res
}

func parseNetstat(out string, npcPort int) (map[int]bool, int) {
	ports := map[int]bool{}
	conns := 0
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) < 4 || !strings.EqualFold(f[0], "TCP") {
			continue
		}
		lp := addrPort(f[1])
		switch {
		case strings.EqualFold(f[3], "LISTENING"):
			ports[lp] = true
		case strings.EqualFold(f[3], "ESTABLISHED") && lp == npcPort:
			conns++
		}
	}
	return ports, conns
}

func addrPort(addr string) int {
	i := strings.LastIndex(addr, ":")
	if i < 0 {
		return 0
	}
	p, _ := strconv.Atoi(addr[i+1:])
	return p
}

func parseQuser(out string) bool {
	for _, line := range strings.Split(out, "\n") {
		l := strings.ToLower(line)
		if strings.Contains(l, "console") &&
			(strings.Contains(l, "active") || strings.Contains(l, "актив")) {
			return true
		}
	}
	return false
}

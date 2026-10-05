package probe

import (
	"os"
	"strconv"
	"strings"
	"time"

	"aion-op/internal/config"
)

// Mock — детерминированная симуляция живого стека: разработка UI/состояний без VM.
// AIONOP_MOCK_DOWN=main,gate — погасить сервисы (демо красного), AIONOP_MOCK_CONNS=4 — окно загрузки.
type Mock struct {
	cfg   *config.Config
	down  map[string]bool
	conns int
}

func NewMock(cfg *config.Config, down map[string]bool) *Mock {
	conns := cfg.WorldPair.ExpectedConns
	if v := os.Getenv("AIONOP_MOCK_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			conns = n
		}
	}
	return &Mock{cfg: cfg, down: down, conns: conns}
}

var memMB = map[string]uint64{
	"npc": 14900, "main": 10300, "cache": 530, "ic": 1700, "sql": 1200,
	"acc": 240, "logsrv": 90, "authd": 40, "gate": 30, "captcha": 60, "pa": 80,
}

func (m *Mock) Snapshot() Snapshot {
	snap := Snapshot{
		When:           time.Now(),
		Source:         "mock",
		Processes:      map[string][]Proc{},
		Ports:          map[int]bool{},
		Conns2002:      m.conns,
		ConsoleSession: true,
	}
	pid := 1000
	for _, s := range m.cfg.Services {
		if s.Locked || m.down[s.ID] {
			continue // мёртвые в ките и «погашенные» — не запущены
		}
		pid++
		sess := "Services"
		if s.Interactive || s.Heavy {
			sess = "Console"
		}
		key := strings.ToLower(s.Exe)
		snap.Processes[key] = append(snap.Processes[key],
			Proc{Name: s.Exe, PID: strconv.Itoa(pid), Session: sess, MemKB: memMB[s.ID] * 1024})
		for _, p := range s.Ports {
			snap.Ports[p] = true
		}
	}
	// Смерть Server64 рвёт NPC-коннекты — conns в 0 (реалистичное демо).
	if m.down["main"] {
		snap.Conns2002 = 0
	}
	return snap
}

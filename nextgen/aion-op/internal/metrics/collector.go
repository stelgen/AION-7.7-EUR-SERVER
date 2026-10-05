// Package metrics — RAM/handles/FreeCommit срезы (Phase 0.5, read-only).
// Handles-динамика Server64 (827k) и NPCSvr — кандидат утечки (эксплуатация 04–05.10).
package metrics

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"aion-op/internal/config"
	"aion-op/internal/probe"
)

type ProcMetric struct {
	Name       string `json:"name"` // exe (Server64.exe)
	PID        string `json:"pid"`
	MemMB      uint64 `json:"mem_mb"`
	Handles    uint64 `json:"handles"`
	HandlesDPM int64  `json:"handles_dpm"` // дельта хендлов в минуту (утечка видна тут)
}

type SysMem struct {
	FreePhysMB   uint64 `json:"free_phys_mb"`
	FreeCommitMB uint64 `json:"free_commit_mb"`
}

type Snap struct {
	When  time.Time    `json:"when"`
	Err   string       `json:"err,omitempty"`
	Procs []ProcMetric `json:"procs"`
	Sys   SysMem       `json:"sys"`
}

// Collector снимает срез метрик.
type Collector interface {
	Collect(ctx context.Context) Snap
}

// Holder — общий слот последнего среза (main пишет, web читает).
type Holder struct {
	mu sync.RWMutex
	s  Snap
}

func (h *Holder) Set(s Snap) { h.mu.Lock(); h.s = s; h.mu.Unlock() }
func (h *Holder) Get() Snap  { h.mu.RLock(); defer h.mu.RUnlock(); return h.s }

// --- SSH ---

type SSH struct {
	cfg    *config.Config
	runner probe.Runner
	prev   map[string]uint64 // pid → handles
	prevAt time.Time
}

func NewSSH(cfg *config.Config, r probe.Runner) *SSH {
	return &SSH{cfg: cfg, runner: r, prev: map[string]uint64{}}
}

// watchNames — имена процессов для Get-Process (exe без .exe).
func watchNames(cfg *config.Config) string {
	var names []string
	for _, s := range cfg.Services {
		if s.Exe == "" {
			continue
		}
		n := strings.TrimSuffix(s.Exe, ".exe")
		names = append(names, "'"+n+"'")
	}
	return strings.Join(names, ",")
}

func (c *SSH) Collect(ctx context.Context) Snap {
	names := watchNames(c.cfg)
	cmd := fmt.Sprintf(
		"$ErrorActionPreference='SilentlyContinue';Get-Process -Name %s | "+
			"Select-Object ProcessName,Id,Handles,@{n='MB';e={[int]($_.WorkingSet64/1MB)}} | "+
			"ConvertTo-Csv -NoTypeInformation;Write-Output '__OS__';"+
			"Get-CimInstance Win32_OperatingSystem | "+
			"Select-Object FreePhysicalMemory,FreeVirtualMemory | ConvertTo-Csv -NoTypeInformation",
		names)
	out, err := c.runner.Run(ctx, cmd)
	if err != nil {
		return Snap{When: time.Now(), Err: err.Error()}
	}
	return c.build(out)
}

func (c *SSH) build(out string) Snap {
	now := time.Now()
	s := Snap{When: now}
	parts := strings.SplitN(out, "__OS__", 2)
	procs := parsePSProcs(parts[0])
	if len(parts) == 2 {
		s.Sys = parsePSOS(parts[1])
	}
	// дельты хендлов по PID
	interval := time.Duration(c.cfg.Metrics.PollSec) * time.Second
	if c.prevAt.IsZero() {
		interval = 0 // первый срез — без дельт
	}
	for i := range procs {
		p := &procs[i]
		key := p.PID
		if prev, ok := c.prev[key]; ok && interval > 0 {
			d := int64(p.Handles) - int64(prev)
			min := interval.Minutes()
			if min < 0.01 {
				min = 0.01
			}
			p.HandlesDPM = int64(float64(d) / min)
		}
		c.prev[key] = p.Handles
	}
	// убрать умершие PID из prev
	live := map[string]bool{}
	for _, p := range procs {
		live[p.PID] = true
	}
	for k := range c.prev {
		if !live[k] {
			delete(c.prev, k)
		}
	}
	c.prevAt = now
	s.Procs = procs
	return s
}

func parsePSProcs(out string) []ProcMetric {
	var res []ProcMetric
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if !strings.HasPrefix(line, `"`) {
			continue
		}
		f := strings.Split(line, `","`)
		if len(f) < 4 || strings.HasPrefix(f[0], `"ProcessName`) {
			continue
		}
		name := strings.Trim(f[0], `"`)
		pid := strings.Trim(f[1], `"`)
		h, _ := strconv.ParseUint(strings.Trim(f[2], `"`), 10, 64)
		mb, _ := strconv.ParseUint(strings.Trim(f[3], `"`), 10, 64)
		if h == 0 && mb == 0 {
			continue // пустые/умершие строки CSV
		}
		if !strings.HasSuffix(name, ".exe") {
			name += ".exe"
		}
		res = append(res, ProcMetric{Name: name, PID: pid, MemMB: mb, Handles: h})
	}
	return res
}

func parsePSOS(out string) SysMem {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if !strings.HasPrefix(line, `"`) {
			continue
		}
		f := strings.Split(line, `","`)
		if len(f) < 2 || strings.Contains(f[0], "FreePhysical") {
			continue // заголовок
		}
		phys, _ := strconv.ParseUint(strings.Trim(f[0], `"`), 10, 64)
		virt, _ := strconv.ParseUint(strings.Trim(f[1], `"`), 10, 64)
		return SysMem{FreePhysMB: phys / 1024, FreeCommitMB: virt / 1024} // KB → MB
	}
	return SysMem{}
}

// --- Mock ---

type Mock struct {
	cfg     *config.Config
	leak    map[string]bool
	handles map[string]uint64
}

func NewMock(cfg *config.Config) *Mock {
	leak := map[string]bool{}
	if v := os.Getenv("AIONOP_MOCK_LEAK"); v != "" {
		for _, n := range strings.Split(v, ",") {
			leak[strings.TrimSpace(n)] = true
		}
	}
	return &Mock{cfg: cfg, leak: leak, handles: map[string]uint64{
		"server64.exe": 827000, "npcsvr64.exe": 578000, "icserver.exe": 32000,
		"cached64.exe": 12000, "sqlservr.exe": 9000,
	}}
}

func (m *Mock) Collect(ctx context.Context) Snap {
	now := time.Now()
	s := Snap{When: now}
	s.Sys = SysMem{FreePhysMB: 9900, FreeCommitMB: 44000}
	interval := float64(m.cfg.Metrics.PollSec)
	if interval < 1 {
		interval = 1
	}
	for _, svc := range m.cfg.Services {
		if svc.Locked || svc.Exe == "" {
			continue
		}
		name := strings.ToLower(svc.Exe)
		base := m.handles[name]
		if base == 0 {
			base = 900
			m.handles[name] = base
		}
		dpm := int64(0)
		if m.leak[svc.ID] {
			base += 30000 // ≈ +120k/мин при poll 15с — демо алерта утечки
			m.handles[name] = base
			dpm = int64(30000 * 60 / interval)
		}
		s.Procs = append(s.Procs, ProcMetric{
			Name: svc.Exe, PID: fmt.Sprint(1000 + len(s.Procs)),
			MemMB: baseMemMB(svc.ID), Handles: base, HandlesDPM: dpm,
		})
	}
	return s
}

func baseMemMB(id string) uint64 {
	mb := map[string]uint64{
		"npc": 14900, "main": 10300, "cache": 530, "ic": 1700, "sql": 1200,
		"acc": 240, "logsrv": 90, "authd": 40, "gate": 30, "captcha": 60, "pa": 80,
	}
	return mb[id]
}

var _ = log.Printf // резерв для будущих диагностик

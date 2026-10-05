package probe

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"aion-op/internal/config"
)

// SSH снимает состояние внешним ssh-клиентом (BatchMode, только read-only команды:
// tasklist / netstat / quser). Никаких изменений на VM.
type SSH struct {
	cfg *config.Config
}

func NewSSH(cfg *config.Config) *SSH { return &SSH{cfg: cfg} }

func (s *SSH) run(ctx context.Context, remote string) (string, error) {
	host := fmt.Sprintf("%s@%s", s.cfg.VM.SSHUser, s.cfg.VM.SSHHost)
	args := []string{
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-o", "StrictHostKeyChecking=accept-new",
		"-p", fmt.Sprint(s.cfg.VM.SSHPort),
	}
	if s.cfg.VM.SSHKey != "" {
		args = append(args, "-i", s.cfg.VM.SSHKey)
	}
	args = append(args, host, remote)

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "ssh", args...).CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("ssh %q: %w", remote, err)
	}
	return string(out), nil
}

// Run — исполнитель read-only команд для лог-тейлеров/метрик (Phase 0.5).
// Тоже только чтение: Get-Content, Get-Process.
func (s *SSH) Run(ctx context.Context, remote string) (string, error) {
	return s.run(ctx, remote)
}

func (s *SSH) Snapshot() Snapshot {
	ctx := context.Background()
	snap := Snapshot{
		When:      time.Now(),
		Source:    fmt.Sprintf("ssh:%s", s.cfg.VM.SSHHost),
		Processes: map[string][]Proc{},
		Ports:     map[int]bool{},
	}

	tl, err := s.run(ctx, "tasklist /fo csv /nh")
	if err != nil {
		snap.Err = err.Error()
		return snap
	}
	snap.Processes = parseTasklistCSV(tl)

	ns, err := s.run(ctx, "netstat -ano -p tcp")
	if err != nil {
		snap.Err = err.Error()
		return snap
	}
	snap.Ports, snap.Conns2002 = parseNetstat(ns, s.cfg.WorldPair.NPCPort)

	if qu, err := s.run(ctx, "quser"); err == nil {
		snap.ConsoleSession = parseQuser(qu)
	}
	return snap
}

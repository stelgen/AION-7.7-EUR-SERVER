// Package probe — Local: срезы и read-only команды НА САМОЙ VM (mode=local).
// Оператор запущен на VM как SYSTEM-задача AionOp; команды исполняются локально.
package probe

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"aion-op/internal/config"
)

type Local struct {
	cfg *config.Config
}

func NewLocal(cfg *config.Config) *Local { return &Local{cfg: cfg} }

func (l *Local) Snapshot() Snapshot {
	return SnapshotVia(l, l.cfg, "local")
}

// Run — локальное исполнение read-only команды.
// PS-стейтменты ($Error…;Get-Process / Get-Content) уходят в powershell ОДНИМ -Command
// аргументом (сплиттер бы их порезал); обычные (tasklist/netstat/schtasks) — по токенам.
func (l *Local) Run(ctx context.Context, remote string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if looksPS(remote) {
		cmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", remote)
	} else {
		args := splitQuoted(remote)
		if len(args) == 0 {
			return "", errEmptyCmd
		}
		cmd = exec.CommandContext(ctx, args[0], args[1:]...)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}

func looksPS(s string) bool {
	t := strings.TrimSpace(s)
	return strings.HasPrefix(t, "$") || strings.HasPrefix(t, "Get-") ||
		strings.Contains(t, ";Get-") || strings.Contains(t, "; ")
}

// splitQuoted — простой сплиттер: пробелы вне двойных кавычек = разделитель.
// Двойные кавычки сохраняют содержимое одним токеном (без экранирования внутри).
func splitQuoted(s string) []string {
	var args []string
	var cur strings.Builder
	inQ := false
	flush := func() {
		if cur.Len() > 0 {
			args = append(args, cur.String())
			cur.Reset()
		}
	}
	for _, r := range s {
		switch {
		case r == '"':
			inQ = !inQ
		case r == ' ' && !inQ:
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return args
}

type emptyErr struct{}

func (emptyErr) Error() string { return "пустая команда" }

var errEmptyCmd = emptyErr{}

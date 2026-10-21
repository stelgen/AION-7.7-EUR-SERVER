// OP-5: async-ожидания вместо фиксированных пауз. Гонка старта гейта:
// пауза 5с была догадкой; правильный примитив = ожидание порта/лог-маркера
// с таймаутом. Использует существующие probe-каналы (read-only команды).
package act

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	defaultAwaitTimeout = 2 * time.Minute
	defaultAwaitPoll    = 500 * time.Millisecond
)

// portCheckFn — проверка порта; переопределяется в тестах (fake listener).
// Дефолт: TCP-dial из процесса (оператор живёт на VM, mode=local).
func (e *Executor) portCheck(ctx context.Context, host string, port int) (bool, error) {
	if e.portCheckFn != nil {
		return e.portCheckFn(ctx, host, port)
	}
	cmd := fmt.Sprintf(`powershell -NoProfile -Command "(New-Object Net.Sockets.TcpClient('%s',%d)).Connected"`, host, port)
	out, err := e.run(ctx, cmd)
	if err != nil {
		return false, err
	}
	return strings.Contains(strings.ToLower(out), "true"), nil
}

// waitForPort — ждать, пока host:port начнёт принимать TCP-соединения.
func (e *Executor) waitForPort(ctx context.Context, host string, port int, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = defaultAwaitTimeout
	}
	deadline := time.Now().Add(timeout)
	for {
		ok, err := e.portCheck(ctx, host, port)
		if err == nil && ok {
			return nil
		}
		if time.Now().After(deadline) {
			if err != nil {
				return fmt.Errorf("порт %s:%d не открылся за %s: %w", host, port, timeout, err)
			}
			return fmt.Errorf("порт %s:%d не открылся за %s", host, port, timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(defaultAwaitPoll):
		}
	}
}

// waitForLogMarker — ждать, пока маркер появится в хвосте лог-файла
// (read-only Get-Content -Tail через runner, как в internal/logs).
func (e *Executor) waitForLogMarker(ctx context.Context, path, marker string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = defaultAwaitTimeout
	}
	deadline := time.Now().Add(timeout)
	cmd := fmt.Sprintf(`powershell -NoProfile -Command "Get-Content -LiteralPath '%s' -Tail 200 | Select-String -SimpleMatch '%s'"`,
		path, strings.ReplaceAll(marker, "'", "''"))
	for {
		out, err := e.run(ctx, cmd)
		if err == nil && strings.TrimSpace(out) != "" {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("маркер %q не появился в %s за %s", marker, path, timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(defaultAwaitPoll):
		}
	}
}

// awaitPortStep / awaitLogStep — шаги плана. Execute распознаёт их и
// исполняет через await-хелперы (вместо «пометка плана, не команда»).
func awaitPortStep(host string, port int) string {
	return fmt.Sprintf("await-port %s %d", host, port)
}

func awaitLogStep(path, marker string) string {
	return "await-log " + path + " => " + marker
}

// parseAwaitSteps — разбор await-шага → (порт | путь+маркер).
func parseAwaitStep(t string) (host string, port int, path, marker string, isPort bool) {
	body := strings.TrimPrefix(t, "await-port ")
	if body != t {
		f := strings.Fields(body)
		if len(f) == 2 {
			var p int
			if _, err := fmt.Sscanf(f[1], "%d", &p); err == nil {
				return f[0], p, "", "", true
			}
		}
		return "", 0, "", "", false
	}
	body = strings.TrimPrefix(t, "await-log ")
	if i := strings.Index(body, " => "); i >= 0 {
		return "", 0, body[:i], body[i+4:], false
	}
	return "", 0, "", "", false
}

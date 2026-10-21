package act

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	"aion-op/internal/config"
	"aion-op/internal/store"
)

// OP-5: тесты await-хелперов на fake listener/runner.

// waitForPort: порт открывается через 100мс → ждёт и проходит.
// Эмуляция: берём свободный порт, ЗАКРЫВАЕМ, через 100мс перe-listen — dial
// начинает проходить только тогда (Listen-сокет открыт с первого тика, поэтому
// исходная версия с мгновенным Accept не работала — waitForPort проходил сразу).
func TestWaitForPortOpens(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	stop := make(chan struct{})
	defer close(stop)
	go func() {
		time.Sleep(100 * time.Millisecond)
		l2, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			return
		}
		// держим открытым до конца теста — иначе закрытие микросекундами
		// рвёт окно между тиками опроса (500мс) и dial всегда refused.
		select {
		case <-stop:
		case <-time.After(3 * time.Second):
		}
		l2.Close()
	}()

	e := New(nil, nil, nil)
	// portCheck по умолчанию запускает powershell — подменяем на dial по ln-порту
	e.portCheckFn = func(_ context.Context, host string, p int) (bool, error) {
		c, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, p), 200*time.Millisecond)
		if err != nil {
			return false, err
		}
		c.Close()
		return true, nil
	}
	start := time.Now()
	if err := e.waitForPort(context.Background(), "127.0.0.1", port, 2*time.Second); err != nil {
		t.Fatalf("waitForPort: %v", err)
	}
	if time.Since(start) < 80*time.Millisecond {
		t.Fatal("прошёл слишком рано — порт еще не был открыт")
	}
}

// waitForPort: порт не открывается → таймаут с ошибкой.
func TestWaitForPortTimeout(t *testing.T) {
	e := New(nil, nil, nil)
	e.portCheckFn = func(context.Context, string, int) (bool, error) { return false, nil }
	start := time.Now()
	err := e.waitForPort(context.Background(), "127.0.0.1", 1, 600*time.Millisecond)
	if err == nil {
		t.Fatal("ожидали таймаут")
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Fatal("вышел раньше таймаута")
	}
}

// waitForLogMarker: маркер появляется в выводе fake-runner.
func TestWaitForLogMarkerFound(t *testing.T) {
	calls := 0
	run := func(_ context.Context, _ string) (string, error) {
		calls++
		if calls >= 2 {
			return "2026-10-21 gate started OK\n", nil
		}
		return "", nil
	}
	e := New(nil, nil, run)
	if err := e.waitForLogMarker(context.Background(), "gate.log", "started OK", 2*time.Second); err != nil {
		t.Fatalf("waitForLogMarker: %v", err)
	}
	if calls < 2 {
		t.Fatalf("calls=%d", calls)
	}
}

// waitForLogMarker: маркера нет → таймаут.
func TestWaitForLogMarkerTimeout(t *testing.T) {
	run := func(context.Context, string) (string, error) { return "", nil }
	e := New(nil, nil, run)
	if err := e.waitForLogMarker(context.Background(), "gate.log", "nope", 600*time.Millisecond); err == nil {
		t.Fatal("ожидали таймаут")
	}
}

// Execute исполняет await-port-шаг через хелпер, а не как команду.
func TestExecuteRunsAwaitPortStep(t *testing.T) {
	ran := 0
	run := func(_ context.Context, cmd string) (string, error) {
		ran++
		return "OK", nil
	}
	// реальный конфиг (dry_run=false, mode=operate) + temp-store: Execute дергает
	// DryRun() и st.AddAction — nil cfg/st из New(nil,nil,run) паникуют.
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"), 30)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cfg := &config.Config{Operator: config.Operator{Mode: "operate"}}
	e := New(cfg, st, run)
	e.portCheckFn = func(context.Context, string, int) (bool, error) { return true, nil }
	p := Plan{Action: "start", Target: "gate", Steps: []string{"schtasks /run /tn AionGate", awaitPortStep("127.0.0.1", 2106)}}
	res := e.Execute(context.Background(), p, "")
	if !res.OK || ran != 1 {
		t.Fatalf("res=%+v ran=%d", res, ran)
	}
}

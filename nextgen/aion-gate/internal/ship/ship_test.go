package ship

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// UDP sink: события долетают в формате RFC5424 + JSON, недоступность не блокирует.
func TestSyslogUDPDelivers(t *testing.T) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()
	addr := pc.LocalAddr().String()

	s := New(Cfg{Enabled: true, Syslog: SyslogCfg{Net: "udp", Host: addr, Facility: 1}, SelfSec: 3600})
	ctx, cancel := context.WithCancel(context.Background())
	go s.Run(ctx)
	defer cancel()

	s.Send(Event{Ev: EvText, Svc: "svc3", Remote: "127.0.0.1:1", Msg: "hello ship"})
	_ = pc.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 4096)
	n, _, err := pc.ReadFrom(buf)
	if err != nil {
		t.Fatalf("udp read: %v", err)
	}
	line := string(buf[:n])
	if !strings.HasPrefix(line, "<14>1 ") || !strings.Contains(line, " aion-logd 2051 text - - ") {
		t.Fatalf("RFC5424 header: %q", line[:min(60, len(line))])
	}
	var ev Event
	if err := json.Unmarshal([]byte(line[strings.LastIndex(line, " - - ")+5:]), &ev); err != nil {
		t.Fatalf("json: %v (%q)", err, line)
	}
	if ev.Ev != EvText || ev.Svc != "svc3" {
		t.Fatalf("ev: %+v", ev)
	}
}

// TCP sink: octet-counted framing.
func TestSyslogTCPDelivers(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	type res struct {
		line string
	}
	ch := make(chan res, 4)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		// octet-counted frame: "NNN <PRI>1 ...json..." (одно событие = один Write)
		buf := make([]byte, 8192)
		n, err := c.Read(buf)
		if err != nil {
			return
		}
		i := 0
		for i < n && buf[i] != ' ' {
			i++
		}
		total, err := strconv.Atoi(string(buf[:i]))
		if err != nil {
			return
		}
		ch <- res{string(buf[i+1 : i+1+total])}
	}()

	s := New(Cfg{Enabled: true, Syslog: SyslogCfg{Net: "tcp", Host: ln.Addr().String()}, SelfSec: 3600})
	ctx, cancel := context.WithCancel(context.Background())
	go s.Run(ctx)
	defer cancel()

	s.Send(Event{Ev: EvConnUp, Remote: "127.0.0.1:2"})
	select {
	case r := <-ch:
		if !strings.Contains(r.line, "conn.up") {
			t.Fatalf("tcp line: %q", r.line)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("tcp timeout")
	}
}

// Недоступный приёмник: Send не блокирует, Run не падает, ошибки считаются.
func TestUnreachableSinkNoBlock(t *testing.T) {
	s := New(Cfg{Enabled: true, Syslog: SyslogCfg{Net: "tcp", Host: "127.0.0.1:1" /* закрытый порт */}, SelfSec: 3600})
	ctx, cancel := context.WithCancel(context.Background())
	go s.Run(ctx)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			s.Send(Event{Ev: EvText, Msg: "flood"}) // udp/tcp недоступен — не должно виснуть
		}
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Send заблокировался на недоступном синке")
	}
	cancel()
}

// File sink + выключенный ship = no-op.
func TestFileSinkAndDisabled(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ship")
	s := New(Cfg{Enabled: false, File: FileCfg{Enabled: true, Dir: dir}, SelfSec: 3600})
	ctx, cancel := context.WithCancel(context.Background())
	go s.Run(ctx)
	s.Send(Event{Ev: EvDB, Msg: "db ok"})
	time.Sleep(400 * time.Millisecond)
	cancel()
	ents, err := os.ReadDir(dir)
	if err != nil || len(ents) == 0 {
		t.Fatalf("файл ndjson не создан: %v", err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, ents[0].Name()))
	if !strings.Contains(string(b), `"ev":"db"`) {
		t.Fatalf("содержимое: %q", b)
	}

	off := New(Cfg{})
	if off.Enabled() {
		t.Fatal("disabled ship должен быть no-op")
	}
	off.Send(Event{Ev: EvText}) // не должно паниковать
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

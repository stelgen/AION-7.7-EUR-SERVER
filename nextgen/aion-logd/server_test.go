package main

import (
	"context"
	"encoding/binary"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"aion-logd/internal/proto"
	"aion-logd/internal/server"
	"aion-logd/internal/writer"
)

func startSrv(t *testing.T, dir string) (addr string, stop func()) {
	t.Helper()
	cfg := server.Config{Listen: "127.0.0.1:0", Builder: 10004, BaseDir: dir, MaxConn: 8}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = server.New(cfg, writer.New(dir, nil)).Run(ctx, ln); close(done) }()
	return ln.Addr().String(), func() { cancel(); <-done }
}

func TestHandshakeAndRawData(t *testing.T) {
	dir := t.TempDir()
	addr, stop := startSrv(t, dir)
	defer stop()

	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(5 * time.Second))

	// клиент: Version(builder=10005)
	if _, err := c.Write(proto.Build(proto.TypeVersion, proto.VersionBody(10005, nil))); err != nil {
		t.Fatalf("send version: %v", err)
	}

	// сервер должен ответить Version(0, builder≥10003) + VersionAndTime(2, FILETIME)
	buf := make([]byte, 4096)
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := c.Read(buf)
	if err != nil || n < 13 {
		t.Fatalf("read reply: %v n=%d", err, n)
	}
	got := buf[:n]
	p1, used, err := proto.Parse(got)
	if err != nil {
		t.Fatalf("reply1 parse: %v (% X)", err, got[:min(16, n)])
	}
	if p1.Type != proto.TypeVersion {
		t.Fatalf("reply1: type=%d want Version", p1.Type)
	}
	b, minB := proto.ParseVersion(p1.Body)
	if b < proto.MinBld || minB != proto.MinBld {
		t.Fatalf("reply1 builder=%d min=%d", b, minB)
	}
	p2, _, err := proto.Parse(got[used:])
	if err != nil {
		t.Fatalf("reply2 parse: %v", err)
	}
	if p2.Type != proto.TypeVerAndTime || proto.ParseVersionAndTime(p2.Body) == 0 {
		t.Fatalf("reply2: %+v", p2)
	}

	// type 4 (кандидат данных) → уходит в .raw, коннект жив
	if _, err := c.Write(proto.Build(proto.TypeData, []byte("FAKE\x00P\x00A\x00Y\x00"))); err != nil {
		t.Fatalf("send data: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	rawDir := filepath.Join(dir, "payload.raw")
	ents, err := os.ReadDir(rawDir)
	if err != nil || len(ents) == 0 {
		t.Fatalf("raw не записан: %v", err)
	}

	// type 3 (заглушка) — сервер не падает
	if _, err := c.Write(proto.Build(proto.TypeUnknown3, nil)); err != nil {
		t.Fatalf("send stub: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
}

func TestSplitPacketAcrossReads(t *testing.T) {
	dir := t.TempDir()
	addr, stop := startSrv(t, dir)
	defer stop()

	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(5 * time.Second))

	raw := proto.Build(proto.TypeVersion, proto.VersionBody(10006, nil))
	// режем пополам — state machine должна дождаться хвоста
	if _, err := c.Write(raw[:4]); err != nil {
		t.Fatalf("w1: %v", err)
	}
	time.Sleep(150 * time.Millisecond)
	if _, err := c.Write(raw[4:]); err != nil {
		t.Fatalf("w2: %v", err)
	}

	buf := make([]byte, 4096)
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := c.Read(buf)
	if err != nil || n == 0 {
		t.Fatalf("нет ответа на разрезанный пакет: %v n=%d", err, n)
	}
	p, _, err := proto.Parse(buf[:n])
	if err != nil || p.Type != proto.TypeVersion {
		t.Fatalf("split reply: %v %+v", err, p)
	}
	_ = binary.LittleEndian
}

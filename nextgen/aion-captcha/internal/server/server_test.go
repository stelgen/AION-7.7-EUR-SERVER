package server

import (
	"context"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"aion-captcha/internal/proto"
)

// FakeClient имитирует Server64: handshake 101 → 102; N запросов 1001 → ответы 1002.
func TestServerE2E(t *testing.T) {
	cfg := Config{}
	cfg.FillDefaults()
	cfg.Listen = "127.0.0.1:0" // авто-порт
	srv := New(cfg, nil)

	ln := startTest(t, srv)
	defer ln.Close()

	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(10 * time.Second))

	// handshake
	if _, err := c.Write(loadFixture(t, "langcheck")); err != nil {
		t.Fatal(err)
	}
	rep := readFrame(t, c)
	if rep.Type != proto.TypeLangCheckReply {
		t.Fatalf("handshake reply type=%d", rep.Type)
	}
	seq, lang, err := proto.ParseLangCheck(rep.Body)
	if err != nil {
		t.Fatal(err)
	}
	if seq != 1 || UnEnLocal(lang) != "en" {
		t.Fatalf("seq=%d lang=%q", seq, UnEnLocal(lang))
	}

	// 3 запроса с seq 1..3
	for i := 1; i <= 3; i++ {
		if _, err := c.Write(proto.BuildRequest(uint16(i), lang, proto.CodepageUTF16)); err != nil {
			t.Fatal(err)
		}
		got := readFrame(t, c)
		if got.Type != proto.TypeCaptchaReply {
			t.Fatalf("reply type=%d", got.Type)
		}
		r, err := proto.ParseReply(got.Body)
		if err != nil {
			t.Fatal(err)
		}
		if r.Seq != uint16(i) {
			t.Fatalf("reply seq=%d want %d", r.Seq, i)
		}
		if len(r.DDS) != 2176 {
			t.Fatalf("dds=%d", len(r.DDS))
		}
		if string(r.DDS[0:4]) != "DDS " {
			t.Fatalf("dds magic")
		}
		text := utf16ToASCII(r.TextUTF16)
		if len(text) != 6 || strings.Trim(text, "1234567890") != "" {
			t.Fatalf("text=%q", text)
		}
	}
}

// Налив темпа: 100 запросов должны пройти быстро (<5с) и все с уникальными seq.
func TestServerPoolFill(t *testing.T) {
	cfg := Config{}
	cfg.FillDefaults()
	cfg.Listen = "127.0.0.1:0"
	srv := New(cfg, nil)
	ln := startTest(t, srv)
	defer ln.Close()

	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(20 * time.Second))

	if _, err := c.Write(loadFixture(t, "langcheck")); err != nil {
		t.Fatal(err)
	}
	readFrame(t, c) // 102

	const n = 100
	start := time.Now()
	for i := 1; i <= n; i++ {
		if _, err := c.Write(proto.BuildRequest(uint16(i), EnLocal("en"), proto.CodepageUTF16)); err != nil {
			t.Fatal(err)
		}
	}
	for i := 1; i <= n; i++ {
		got := readFrame(t, c)
		if got.Type != proto.TypeCaptchaReply {
			t.Fatalf("type=%d", got.Type)
		}
		r, err := proto.ParseReply(got.Body)
		if err != nil {
			t.Fatal(err)
		}
		if r.Seq != uint16(i) {
			t.Fatalf("seq=%d want %d", r.Seq, i)
		}
	}
	dur := time.Since(start)
	t.Logf("100 запросов/ответов за %v", dur)
	if dur > 5*time.Second {
		t.Fatalf("налив слишком медленный: %v", dur)
	}
}

func startTest(t *testing.T, s *Server) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(cc net.Conn) { serveConn(s, cc) }(c)
		}
	}()
	return ln
}

func serveConn(s *Server, c net.Conn) {
	s.handleConn(context.Background(), c)
}

func readFrame(t *testing.T, c net.Conn) proto.Frame {
	t.Helper()
	buf := make([]byte, 0x2000)
	fr, _, err := proto.ReadFrame(c, buf)
	if err != nil {
		t.Fatalf("read frame: %v", err)
	}
	return fr
}

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name + ".hex")
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	toks := strings.Fields(string(raw))
	b := make([]byte, len(toks))
	for i, tk := range toks {
		b[i] = byte(hexByte(tk))
	}
	return b
}

func hexByte(s string) byte {
	v := 0
	for _, ch := range s {
		v <<= 4
		switch {
		case ch >= '0' && ch <= '9':
			v |= int(ch - '0')
		case ch >= 'a' && ch <= 'f':
			v |= int(ch-'a') + 10
		}
	}
	return byte(v)
}

func utf16ToASCII(b []byte) string {
	out := make([]byte, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		out = append(out, b[i])
	}
	return string(out)
}

func UnEnLocal(v uint16) string { return proto.UnEn(v) }
func EnLocal(s string) uint16   { return proto.En(s) }

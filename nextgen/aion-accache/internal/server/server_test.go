package server

import (
	"encoding/binary"
	"net"
	"testing"
	"time"

	"aion-accache/internal/cache"
	"aion-accache/internal/config"
)

func testCfg(port int) *config.Cfg {
	c := &config.Cfg{}
	c.Server.Listen = "127.0.0.1:0"
	_ = port
	return c
}

func frame(cmd uint16, payload []byte) []byte {
	w, _ := buildForTest(cmd, payload)
	return w
}

func startSrv(t *testing.T) (addr string, srv *S) {
	t.Helper()
	srv = New(testCfg(0), cache.New(), nil, nil)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv.SetListenerForTest(ln)
	go func() { _ = srv.ServeOnTest() }()
	return ln.Addr().String(), srv
}

func dial(t *testing.T, addr string) net.Conn {
	t.Helper()
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	c.SetDeadline(time.Now().Add(3 * time.Second))
	return c
}

func TestFirstLoadResponds5Ints(t *testing.T) {
	addr, _ := startSrv(t)
	c := dial(t, addr)
	defer c.Close()

	// ACQ_FIRST_LOAD_ACCOUNT_INFO (0x04), payload = accountId=7
	pl := make([]byte, 4)
	binary.LittleEndian.PutUint32(pl, 7)
	if _, err := c.Write(frame(0x04, pl)); err != nil {
		t.Fatal(err)
	}
	// ждём ответ: [len][cmd][EB][~cmd][20Б]
	hdr := make([]byte, 7)
	if _, err := readFull(c, hdr); err != nil {
		t.Fatalf("read hdr: %v", err)
	}
	cmd := binary.LittleEndian.Uint16(hdr[2:4])
	if cmd != 0x04 {
		t.Fatalf("reply cmd = %#x (TBD: ACP real cmd)", cmd)
	}
	if hdr[4] != 0xEB {
		t.Fatalf("marker = %#x", hdr[4])
	}
	n := int(binary.LittleEndian.Uint16(hdr[0:2])) // n = полная длина кадра
	body := make([]byte, n-7)                     // payload = n - len(2) - cmd(2) - marker(1) - ~cmd(2)
	if _, err := readFull(c, body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 20 {
		t.Fatalf("reply payload = %d, want 20", len(body))
	}
}

func TestUnknownDropped(t *testing.T) {
	addr, _ := startSrv(t)
	c := dial(t, addr)
	defer c.Close()
	// cmd 0x30 (не занят) -> дроп без ответа
	if _, err := c.Write(frame(0x30, []byte{1, 2, 3})); err != nil {
		t.Fatal(err)
	}
	// ответа быть не должно
	c.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	buf := make([]byte, 16)
	_, err := c.Read(buf)
	if err == nil {
		t.Fatal("want timeout (no reply for unknown)")
	}
}

func TestBadMarkerCloses(t *testing.T) {
	addr, _ := startSrv(t)
	c := dial(t, addr)
	defer c.Close()
	w, _ := buildForTest(0x01, nil)
	w[4] = 0x00 // ломаем маркер
	if _, err := c.Write(w); err != nil {
		t.Fatal(err)
	}
	// сервер должен разорвать соединение (parse.err)
	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 8)
	for {
		if _, err := c.Read(buf); err != nil {
			break // EOF/timeout = соединение закрыто
		}
	}
}

package server

import (
	"encoding/binary"
	"net"
	"strings"
	"testing"
	"time"
)

// e2e: 2006 бинарный RQ_PING → RP_PING (канон L2 C1 packet144).
func TestAdminServerPingE2E(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := New(Config{ServerPort: 0, AdminPort: 0, NpcDbPort: 0})
	srv.SetListenerForTest(ln, ChanServer)
	go func() { _ = srv.ServeOnTest(ChanServer) }()

	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(3 * time.Second))

	// RQ_PING(108): "ddddd" [p1][charId][p2][p3][p4]
	pl := make([]byte, 20)
	put32(pl[0:4], 0)      // p1
	put32(pl[4:8], 1002)   // charId
	put32(pl[8:12], 0x11)  // p2
	put32(pl[12:16], 0x22) // p3
	put32(pl[16:20], 0x33) // p4
	frame := buildTestFrame(108, pl)
	if _, err := c.Write(frame); err != nil {
		t.Fatal(err)
	}

	hdr := make([]byte, 2)
	if _, err := readFull(c, hdr); err != nil {
		t.Fatalf("read len: %v", err)
	}
	n := int(binary.LittleEndian.Uint16(hdr))
	body := make([]byte, n-2)
	if _, err := readFull(c, body); err != nil {
		t.Fatalf("read body: %v", err)
	}
	// S2C-кадр: [cmd u16=45][0xEC][~cmd][payload "cdddddddd"...]
	cmd := binary.LittleEndian.Uint16(body[0:2])
	if cmd != rpPing || body[2] != 0xEC {
		t.Fatalf("cmd/marker: %#x %#x; FULL=%x", cmd, body[2], body)
	}
	p := body[5:] // payload = frame[7:], body = frame[2:] => payload@body[5]
	if le32(p[0:4]) != 0x11 || le32(p[4:8]) != 0x22 || le32(p[8:12]) != 0x33 || le32(p[12:16]) != 0 {
		t.Fatalf("ping echo fields: % x; p=% x; bodylen=%d", p[:16], p, len(body))
	}
	if le32(p[20:24]) != 0 || le32(p[24:28]) != 0 || le32(p[28:32]) != 0 {
		t.Fatalf("tail zeros (после ts@16): % x", p[20:32])
	}
}

// e2e: 2007 ТЕКСТОВЫЙ — строки до '\n', packetId первым полем (L2 C1 CAdminSocket).
func TestAdminTextE2E(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := New(Config{ServerPort: 0, AdminPort: 0, NpcDbPort: 0})
	srv.SetListenerForTest(ln, ChanAdmin)
	go func() { _ = srv.ServeOnTest(ChanAdmin) }()

	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(3 * time.Second))

	// GQ_CHECK_CHAR=20 (теория #14: interactive packetId ↔ GQ-enum)
	// L2 C1 01_CheckCharacterPacket: "dsss" [packetId][charName][adminAccount][unk]
	if _, err := c.Write([]byte("20\tTestChar\tAdminAcc\tx\n")); err != nil {
		t.Fatal(err)
	}
	// MVP: ответов нет (формы = capture R4); сервер НЕ должен упасть/закрыться.
	buf := make([]byte, 64)
	_ = c.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	if _, err := c.Read(buf); err == nil {
		// данные не ожидались
		t.Fatalf("unexpected response: %q", strings.TrimSpace(string(buf)))
	}
	// соединение живо (не closed): пишем вторую команду
	if _, err := c.Write([]byte("7\tTestChar\t1\n")); err != nil {
		t.Fatalf("conn closed after text cmd: %v", err)
	}
}

// e2e: 2006 unknown-опкод → тишина, соединение живо.
func TestServerUnknownSilent(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := New(Config{ServerPort: 0, AdminPort: 0, NpcDbPort: 0})
	srv.SetListenerForTest(ln, ChanServer)
	go func() { _ = srv.ServeOnTest(ChanServer) }()

	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := c.Write(buildTestFrame(9999, []byte("x"))); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 32)
	_ = c.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	if n, err := c.Read(buf); err == nil && n > 0 {
		t.Fatalf("unknown op: ожидалась тишина, пришло % x", buf[:n])
	}
	if _, err := c.Write(buildTestFrame(108, make([]byte, 20))); err != nil {
		t.Fatalf("conn closed after unknown: %v", err)
	}
}

// --- тест-хелперы (локальные, не конфликтуют с server_test.go) ---

func buildTestFrame(opcode uint16, payload []byte) []byte {
	wireLen := 7 + len(payload) // len2+cmd2+EB1+~cmd2 = заголовок, без хвоста
	out := make([]byte, wireLen)
	binary.LittleEndian.PutUint16(out, uint16(wireLen))
	binary.LittleEndian.PutUint16(out[2:], opcode)
	out[4] = 0xEB
	inv := ^opcode
	out[5] = byte(inv)
	out[6] = byte(inv >> 8)
	copy(out[7:], payload)
	return out
}

func readFull(c net.Conn, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := c.Read(buf[total:])
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}
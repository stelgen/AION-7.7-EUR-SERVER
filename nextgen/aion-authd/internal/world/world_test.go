package world

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"
	"time"
)

// Голден из живого корпуса 09.10 (authd-ref/logs-2104/2026-10-07.09.packet.txt,
// логин юзера 09:32:56, Auth->World(1260),0).

const gsAuthVersion = 2017012601 // Server64: "Protocol Version authVersion:2017012601, protocolVersion:1"

func TestEncodeFraming(t *testing.T) {
	fr := Encode(5, []byte{0, 0, 0xf4, 1})
	x := binary.LittleEndian.Uint16(fr[0:2])
	if int(x)+1 != len(fr) {
		t.Fatalf("X=%d, total=%d — контракт X=total-1 нарушен", x, len(fr))
	}
	if fr[2] != 5 {
		t.Fatalf("type=%d", fr[2])
	}
}

func TestGreetingGolden(t *testing.T) {
	// C1 OnCreate: Send("cdd", 3, build, 1) → [X][03][authVersion u32][1 u32]
	g := Greeting(gsAuthVersion)
	// 2017012601 = 0x78392b79 → LE = 79 2b 39 78
	if !bytes.Equal(g, mustHex(t, "0a0003792b397801000000")) {
		t.Fatalf("greeting = %x, want 0a0003792b397801000000", g)
	}
}

func TestPingGolden(t *testing.T) {
	if got := Ping(); !bytes.Equal(got, mustHex(t, "020002")) {
		t.Fatalf("ping = %x, want 020002", got)
	}
}

func TestRelayLoginGolden(t *testing.T) {
	// payload 107Б из корпуса (uid=1010, "Stelgen")
	var payload []byte
	payload = append(payload, 0xf2, 0x03, 0x00, 0x00)                     // uid=1010
	payload = append(payload, []byte("Stelgen")...)                       // аккаунт
	payload = append(payload, make([]byte, 9)...)                         // паддинг до 16
	payload = append(payload, 0xd0, 0x07, 0x00, 0x00)                     // maxUsers=2000
	payload = append(payload, make([]byte, 17)...)                        // нулевая зона
	payload = append(payload, '0', '0', '0', '0', '0', '0', '0', 0)       // "0000000\0"
	payload = append(payload, mustHex(t, DefaultRelayTail)...)            // tail 58Б

	if len(payload) != 107 {
		t.Fatalf("payload=%dБ, в корпусе 107", len(payload))
	}
	want := append([]byte{0x6d, 0x00, 0x00}, payload...) // X=109=total-1 (110Б total), type=0

	cfg := Cfg{MaxUsers: 2000}
	got, err := RelayLogin(cfg, 1010, "Stelgen")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("relay mismatch:\n got %x\nwant %x", got, want)
	}
}

func TestReadFrameBadSize(t *testing.T) {
	for _, bad := range []string{"0000", "0100", "0120"} { // X<2 / X=0x2001
		c1, c2 := net.Pipe()
		go func(h string) {
			c1.Write(mustHex(t, h))
		}(bad)
		if _, _, err := ReadFrame(c2); err == nil {
			t.Fatalf("size %s — ожидалась ошибка", bad)
		}
		_ = c1.Close()
		_ = c2.Close()
	}
}

func TestServeE2E(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	s := New(Cfg{Port: ln.Addr().(*net.TCPAddr).Port, AuthVersion: gsAuthVersion, MaxUsers: 2000, HeartbeatSec: 1}, nil)
	go s.Serve(ln)

	wc, err := net.DialTimeout("tcp", ln.Addr().String(), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer wc.Close()

	// greeting
	typ, payload, err := ReadFrame(wc)
	if err != nil || typ != 3 || len(payload) != 8 {
		t.Fatalf("greeting: typ=%d len=%v err=%v", typ, payload, err)
	}
	if av := binary.LittleEndian.Uint32(payload[0:4]); av != gsAuthVersion {
		t.Fatalf("authVersion=%d, want %d", av, gsAuthVersion)
	}
	if pv := binary.LittleEndian.Uint32(payload[4:8]); pv != 1 {
		t.Fatalf("protocolVersion=%d, want 1", pv)
	}

	// world status → users/limit (live-корпус: 0100f401 после логина юзера)
	wc.Write(Encode(5, []byte{1, 0, 0xf4, 1})) // users=1, limit=500
	<-s.chPong
	if _, users, limit := s.Status(); users != 1 || limit != 500 {
		t.Fatalf("status users=%d limit=%d, want 1/500", users, limit)
	}

	// релей логина = голден корпуса: сырой фрейм [X u16][type][payload]
	s.NotifyLogin(1010, "Stelgen")
	var h [2]byte
	if _, err := readFull(wc, h[:]); err != nil {
		t.Fatal(err)
	}
	x := binary.LittleEndian.Uint16(h[:])
	if x != 109 {
		t.Fatalf("X=%d, want 109", x)
	}
	wantFrame, _ := RelayLogin(s.Cfg, 1010, "Stelgen")
	body := make([]byte, int(x)-1)
	if _, err := readFull(wc, body); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(body, wantFrame[2:]) {
		t.Fatalf("relay body mismatch: got %dБ want %dБ", len(body), len(wantFrame)-2)
	}

	// heartbeat в течение ~2 тиков
	wc.SetReadDeadline(time.Now().Add(3 * time.Second))
	typ, _, err = ReadFrame(wc)
	if err != nil || typ != 2 {
		t.Fatalf("heartbeat: typ=%d err=%v", typ, err)
	}
}

func TestAckEcho(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	s := New(Cfg{Port: ln.Addr().(*net.TCPAddr).Port, AuthVersion: gsAuthVersion, HeartbeatSec: 3600, Acks: true}, nil)
	go s.Serve(ln)

	wc, _ := net.DialTimeout("tcp", ln.Addr().String(), 2*time.Second)
	defer wc.Close()
	_, _, _ = ReadFrame(wc) // greeting

	// мир шлёт type 27 (uid=1010) → квитанция type 19 (uid)
	wc.Write(Encode(27, []byte{0xf2, 0x03, 0x00, 0x00}))
	wc.SetReadDeadline(time.Now().Add(2 * time.Second))
	typ, payload, err := ReadFrame(wc)
	if err != nil || typ != 19 {
		t.Fatalf("ack: typ=%d err=%v", typ, err)
	}
	if !bytes.Equal(payload, []byte{0xf2, 0x03, 0x00, 0x00}) {
		t.Fatalf("ack payload=%x", payload)
	}
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	if len(s)%2 != 0 {
		t.Fatalf("нечётный hex %q", s)
	}
	out := make([]byte, len(s)/2)
	for i := range out {
		hi := hexVal(s[2*i])
		lo := hexVal(s[2*i+1])
		if hi < 0 || lo < 0 {
			t.Fatalf("bad hex %q", s)
		}
		out[i] = byte(hi<<4 | lo)
	}
	return out
}

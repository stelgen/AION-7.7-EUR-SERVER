package world

import (
	"bytes"
	"encoding/binary"
	"net"
	"strings"
	"testing"
	"time"
)

// Голдены из живых логов L2Authd (authd-ref/logs-2104/ + 2026-10-04.04/05.packet):
// greeting ориг 04.10 05:11:21; relay 04.10 04:04:51 (stelgen, клиент 127.0.0.1).

const gsAuthVersion = 2017012601 // Server64: "Protocol Version authVersion:2017012601, protocolVersion:1"

// orig0410Relay = точный 107Б payload ориг-лога 04.10 04:04:51 (uid=1010, "Stelgen", ip 127.0.0.1).
// корпус 09.10 (клиент 184.160.77.85 -> 554da0b8 реверс); golden через strings.Replace.
const orig0410Relay = "f20300005374656c67656e00000000000000000000000000d00700000000000000000000000000000030303030303030007f000001000000000000ffffffffffffffffffffffffffffffffffffffffffffffff0000000050c2366b50c2366b000000000000000000000000"

func TestEncodeFraming(t *testing.T) {
	fr := Encode(5, []byte{0, 0, 0xf4, 1})
	x := binary.LittleEndian.Uint16(fr[0:2])
	if int(x)+2 != len(fr) {
		t.Fatalf("X=%d, total=%d — контракт X=total-2 (body+2) нарушен", x, len(fr))
	}
	if fr[2] != 5 {
		t.Fatalf("type=%d", fr[2])
	}
}

func TestGreetingGolden(t *testing.T) {
	// КАНОН ориг-лога 04.10 (Auth->World,3: 792b39780100000000 — 9Б payload!):
	// [03][authVersion][1][0x00] — без 9-го байта Server64 молчит (корень R6-отказа 11:07).
	g := Greeting(gsAuthVersion)
	if !bytes.Equal(g, mustHex(t, "0c0003792b39780100000000")) {
		t.Fatalf("greeting = %x, want 0c0003792b39780100000000", g)
	}
}

func TestPingGolden(t *testing.T) {
	if got := Ping(); !bytes.Equal(got, mustHex(t, "030002")) {
		t.Fatalf("ping = %x, want 030002", got)
	}
}

func TestRelayLoginGolden(t *testing.T) {
	cfg := Cfg{MaxUsers: 2000}
	// Корпус 09.10 (клиент 184.160.77.85 → IP-дворд 554da0b8 реверс-октеты)
	got, err := RelayLogin(cfg, 1010, "Stelgen", "184.160.77.85")
	if err != nil {
		t.Fatal(err)
	}
	want := append([]byte{0x6e, 0x00, 0x00}, mustHex(t, strings.Replace(orig0410Relay, "7f000001", "554da0b8", 1))...) // X=110, type=0
	if !bytes.Equal(got, want) {
		t.Fatalf("relay mismatch:\n got %x\nwant %x", got, want)
	}
	// 04.10-противоречие: для 127.0.0.1 ориг писал 7f000001 (direct) — конвенция неоднозначна (T2);
	// у нас принят реверс (09.10 корпус): 127.0.0.1 -> 0100007f
	got2, _ := RelayLogin(cfg, 1010, "Stelgen", "127.0.0.1")
	want2 := append([]byte{0x6e, 0x00, 0x00}, mustHex(t, strings.Replace(orig0410Relay, "7f000001", "0100007f", 1))...)
	if !bytes.Equal(got2, want2) {
		t.Fatalf("relay 127.0.0.1 mismatch")
	}
}

func TestReadFrameBadSize(t *testing.T) {
	for _, bad := range []string{"0000", "0200", "0120"} { // X<3 / X=0x2001
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

	// greeting: [03][authVersion u32][1 u32][00] — 9Б payload
	typ, payload, err := ReadFrame(wc)
	if err != nil || typ != 3 || len(payload) != 9 {
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

	// релей play = голден 04.10 (сырой фрейм [X u16][type][payload])
	var acked bool
	s.OnPlayAck = func(uid, pk1 uint32) { acked = true }
	s.RelayPlay(1010, "Stelgen", "127.0.0.1")
	var h [2]byte
	if _, err := readFull(wc, h[:]); err != nil {
		t.Fatal(err)
	}
	x := binary.LittleEndian.Uint16(h[:])
	if x != 110 {
		t.Fatalf("X=%d, want 110", x)
	}
	wantFrame, _ := RelayLogin(s.Cfg, 1010, "Stelgen", "127.0.0.1")
	body := make([]byte, int(x)-1)
	if _, err := readFull(wc, body); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(body, wantFrame[2:]) {
		t.Fatalf("relay body mismatch: got %dБ want %dБ", len(body), len(wantFrame)-2)
	}

	// world ack → OnPlayAck (uid, pk1=N)
	wc.Write(Encode(0, []byte{0xf2, 0x03, 0x00, 0x00, 0x09, 0x00, 0x00, 0x00}))
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !acked {
		time.Sleep(20 * time.Millisecond)
	}
	if !acked {
		t.Fatal("OnPlayAck не сработал на ack мира")
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

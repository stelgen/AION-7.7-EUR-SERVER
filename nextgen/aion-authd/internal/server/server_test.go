package server

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"

	"aion-authd/internal/config"
	"aion-authd/internal/store"
	"aion-authd/internal/wire"
)

// e2e: фейк-гейт (по схеме cmd/probe aion-gate) против нашего authd.

func startServer(t *testing.T) (*Server, string, *store.MapStore) {
	t.Helper()
	cfg := &config.Config{}
	cfg.FillDefaults()
	cfg.Ship.File.Enabled = false
	st := store.NewMap()
	srv := New(cfg, nil, st)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { srv.Close(); st.Close() })
	return srv, ln.Addr().String(), st
}

// asmBlob — как в logic_test ([00][decbuf34][dword][tail152] = 191Б).
func asmBlob(user string) []byte {
	b := []byte{0x00}
	dec := make([]byte, 34)
	copy(dec[0:14], user)
	b = append(b, dec...)
	b = append(b, make([]byte, 4)...) // dword
	return append(b, make([]byte, 152)...)
}

func dial(t *testing.T, addr string) net.Conn {
	t.Helper()
	c, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	return c
}

func readReply(t *testing.T, c net.Conn) (id uint32, typ byte, payload []byte) {
	t.Helper()
	var ft [1]byte
	if _, err := io.ReadFull(c, ft[:]); err != nil {
		t.Fatalf("read type: %v", err)
	}
	switch ft[0] {
	case 0x03, 0x01:
		var b [4]byte
		if _, err := io.ReadFull(c, b[:]); err != nil {
			t.Fatal(err)
		}
		return binary.LittleEndian.Uint32(b[:]), ft[0], nil
	case 0x02:
		var hdr [6]byte
		if _, err := io.ReadFull(c, hdr[:]); err != nil {
			t.Fatal(err)
		}
		body := int(binary.LittleEndian.Uint16(hdr[4:6])) - 2
		buf := make([]byte, body)
		if _, err := io.ReadFull(c, buf); err != nil {
			t.Fatal(err)
		}
		return binary.LittleEndian.Uint32(hdr[0:4]), buf[0], buf[1:]
	default:
		t.Fatalf("unexpected frame type %02x", ft[0])
	}
	return
}

func TestE2EFullFlow(t *testing.T) {
	_, addr, _ := startServer(t)
	c := dial(t, addr)
	defer c.Close()

	// 1. greeting [03][0xc621] при accept (probe-live: «приветствие есть»)
	var g [5]byte
	if _, err := io.ReadFull(c, g[:]); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(g[:], []byte{0x03, 0x21, 0xc6, 0x00, 0x00}) {
		t.Fatalf("greeting: %x", g)
	}

	// 2. [00] CltConnect + [02] login-blob 191Б → type=3 (uid=1, token)
	if _, err := c.Write(wire.ConnectFrame(7, [4]byte{192, 168, 0, 253})); err != nil {
		t.Fatal(err)
	}
	pf, _ := wire.PacketFrame(7, asmBlob("stelgen"))
	if _, err := c.Write(pf); err != nil {
		t.Fatal(err)
	}
	id, typ, payload := readReply(t, c)
	if id != 7 || typ != 3 || len(payload) != 52 {
		t.Fatalf("login: id=%d typ=%d len=%d", id, typ, len(payload))
	}
	if uid := binary.LittleEndian.Uint32(payload[0:4]); uid != 1 {
		t.Fatalf("uid: %d", uid)
	}

	// 3. [05] CM_SERVER_LIST → type=4 (payload 31: IP/порт мира)
	op5 := []byte{0x05}
	pf, _ = wire.PacketFrame(7, op5)
	if _, err := c.Write(pf); err != nil {
		t.Fatal(err)
	}
	_, typ, payload = readReply(t, c)
	if typ != 4 || len(payload) != 26 {
		t.Fatalf("serverlist: typ=%d len=%d (канон ориг 26Б)", typ, len(payload))
	}
	if payload[3] != 192 || payload[6] != 125 { // worldIP 192.168.0.125
		t.Fatalf("worldIP: %x", payload[3:7])
	}
	if port := binary.LittleEndian.Uint16(payload[7:9]); port != 7777 {
		t.Fatalf("port: %d", port)
	}

	// 4. [02] CM_PLAY → type=7 (payload 15)
	op2 := []byte{0x02}
	pf, _ = wire.PacketFrame(7, op2)
	if _, err := c.Write(pf); err != nil {
		t.Fatal(err)
	}
	_, typ, payload = readReply(t, c)
	if typ != 7 || len(payload) != 9 {
		t.Fatalf("play: typ=%d len=%d (канон ориг 9Б)", typ, len(payload))
	}
	if binary.LittleEndian.Uint32(payload[4:8]) != 1 {
		t.Fatalf("pk2: %d want accID 1", binary.LittleEndian.Uint32(payload[4:8]))
	}

	// 5. [01] CltDisconnect — онлайн-флаг НЕ снимается (live)
	if _, err := c.Write(wire.DisconnectFrame(7)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond) // [01] обработан; онлайн-флаг намеренно жив (live)
}

func TestE2EReloginSilenceAndUnknownSession(t *testing.T) {
	_, addr, _ := startServer(t)
	c := dial(t, addr)
	defer c.Close()

	var g [5]byte
	if _, err := io.ReadFull(c, g[:]); err != nil {
		t.Fatal(err)
	}
	// [02] БЕЗ [00] → [01][sid] (negative-ack, probe-live)
	pf, _ := wire.PacketFrame(99, asmBlob("x"))
	if _, err := c.Write(pf); err != nil {
		t.Fatal(err)
	}
	id, typ, _ := readReply(t, c)
	if typ != 0x01 || id != 99 {
		t.Fatalf("unknown-session: typ=%02x id=%d", typ, id)
	}

	// логин ок (sid=5), затем relogin онлайн-акка (sid=6) = ТИШИНА
	if _, err := c.Write(wire.ConnectFrame(5, [4]byte{127, 0, 0, 1})); err != nil {
		t.Fatal(err)
	}
	pf, _ = wire.PacketFrame(5, asmBlob("1"))
	if _, err := c.Write(pf); err != nil {
		t.Fatal(err)
	}
	if _, typ, _ = readReply(t, c); typ != 3 {
		t.Fatalf("first login: typ=%d", typ)
	}
	if _, err := c.Write(wire.ConnectFrame(6, [4]byte{127, 0, 0, 2})); err != nil {
		t.Fatal(err)
	}
	pf, _ = wire.PacketFrame(6, asmBlob("1"))
	if _, err := c.Write(pf); err != nil {
		t.Fatal(err)
	}
	_ = c.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	var ft [1]byte
	if _, err := io.ReadFull(c, ft[:]); err == nil {
		t.Fatalf("relogin должен молчать, пришло %02x", ft[0])
	}
}

func TestE2EUnknownOpSilence(t *testing.T) {
	_, addr, _ := startServer(t)
	c := dial(t, addr)
	defer c.Close()
	var g [5]byte
	if _, err := io.ReadFull(c, g[:]); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Write(wire.ConnectFrame(3, [4]byte{127, 0, 0, 1})); err != nil {
		t.Fatal(err)
	}
	pf, _ := wire.PacketFrame(3, []byte{0x77, 1, 2, 3})
	if _, err := c.Write(pf); err != nil {
		t.Fatal(err)
	}
	_ = c.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	var ft [1]byte
	if _, err := io.ReadFull(c, ft[:]); err == nil {
		t.Fatalf("unknown-op должен молчать, пришло %02x", ft[0])
	}
}

func TestE2EBadBlobStrict(t *testing.T) {
	_, addr, _ := startServer(t)
	c := dial(t, addr)
	defer c.Close()
	var g [5]byte
	if _, err := io.ReadFull(c, g[:]); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Write(wire.ConnectFrame(3, [4]byte{127, 0, 0, 1})); err != nil {
		t.Fatal(err)
	}
	// 86Б blob → тишина (probe-live)
	short := append([]byte{0x00}, make([]byte, 85)...)
	pf, _ := wire.PacketFrame(3, short)
	if _, err := c.Write(pf); err != nil {
		t.Fatal(err)
	}
	_ = c.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	var ft [1]byte
	if _, err := io.ReadFull(c, ft[:]); err == nil {
		t.Fatalf("strict 86Б должен молчать, пришло %02x", ft[0])
	}
	// 191Б → ответ (сбросить дедлайн silence-окна!)
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	pf, _ = wire.PacketFrame(3, asmBlob("probeacc"))
	if _, err := c.Write(pf); err != nil {
		t.Fatal(err)
	}
	if _, typ, _ := readReply(t, c); typ != 3 {
		t.Fatalf("191Б: typ=%d", typ)
	}
}

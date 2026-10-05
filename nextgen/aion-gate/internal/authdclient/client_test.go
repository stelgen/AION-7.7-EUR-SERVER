package authdclient

import (
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"aion-gate/internal/proto"
)

// fakeConn — бест-дабл net.Conn: Read выдаёт push-нутые байты, Write копит.
type fakeConn struct {
	mu     sync.Mutex
	in     []byte
	w      []byte
	closed bool
	notify chan struct{}
}

func newFakeConn() *fakeConn { return &fakeConn{notify: make(chan struct{}, 16)} }

func (c *fakeConn) push(b []byte) {
	c.mu.Lock()
	c.in = append(c.in, b...)
	c.mu.Unlock()
	select {
	case c.notify <- struct{}{}:
	default:
	}
}

func (c *fakeConn) Read(p []byte) (int, error) {
	for {
		c.mu.Lock()
		if len(c.in) > 0 {
			n := copy(p, c.in)
			c.in = c.in[n:]
			c.mu.Unlock()
			return n, nil
		}
		if c.closed {
			c.mu.Unlock()
			return 0, errors.New("closed")
		}
		c.mu.Unlock()
		<-c.notify
	}
}

func (c *fakeConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return 0, errors.New("closed")
	}
	c.w = append(c.w, p...)
	return len(p), nil
}

func (c *fakeConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	select {
	case c.notify <- struct{}{}:
	default:
	}
	return nil
}

func (c *fakeConn) out() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]byte(nil), c.w...)
}

func (c *fakeConn) outLen() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.w)
}

// лишние методы net.Conn не нужны (клиент не их использует)
func (c *fakeConn) LocalAddr() net.Addr                { return nil }
func (c *fakeConn) RemoteAddr() net.Addr               { return nil }
func (c *fakeConn) SetDeadline(t time.Time) error      { return nil }
func (c *fakeConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *fakeConn) SetWriteDeadline(t time.Time) error { return nil }

func waitFor(t *testing.T, fn func() bool) {
	t.Helper()
	for i := 0; i < 500; i++ {
		if fn() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("таймаут ожидания")
}

func TestOutboundFrames(t *testing.T) {
	conn := newFakeConn()
	c := DialConn(conn, Handler{})
	sid := uint32(0x7d5214)

	if err := c.SendConnect(sid, [4]byte{192, 168, 0, 125}); err != nil {
		t.Fatal(err)
	}
	want := proto.Assemble("cdd", byte(0), sid, binary.BigEndian.Uint32([]byte{192, 168, 0, 125}))
	waitFor(t, func() bool { return conn.outLen() >= len(want) })
	if got := conn.out()[:len(want)]; string(got) != string(want) {
		t.Fatalf("cdd: got %x want %x", got, want)
	}

	blob := []byte{1, 2, 3, 4}
	if err := c.SendPacket(sid, blob); err != nil {
		t.Fatal(err)
	}
	want2 := append(proto.Assemble("cdh", byte(2), sid, uint16(len(blob)+2)), blob...)
	waitFor(t, func() bool { return conn.outLen() >= len(want)+len(want2) })
	if got := conn.out()[len(want) : len(want)+len(want2)]; string(got) != string(want2) {
		t.Fatalf("cdh: got %x want %x", got, want2)
	}

	if err := c.SendDisconnect(sid); err != nil {
		t.Fatal(err)
	}
	want3 := proto.Assemble("cd", byte(1), sid)
	waitFor(t, func() bool { return conn.outLen() >= len(want)+len(want2)+len(want3) })
	if got := conn.out()[len(want)+len(want2):]; string(got) != string(want3) {
		t.Fatalf("cd: got %x want %x", got, want3)
	}
}

func TestInboundFrames(t *testing.T) {
	conn := newFakeConn()
	var mu sync.Mutex
	var regID, assigned, pktID uint32
	var pktTyp byte
	var payload []byte
	closed := make(chan error, 1)
	c := DialConn(conn, Handler{
		OnRegistered: func(id uint32) { mu.Lock(); regID = id; mu.Unlock() },
		OnAssigned:   func(sid uint32) { mu.Lock(); assigned = sid; mu.Unlock() },
		OnPacket:     func(id uint32, typ byte, p []byte) { mu.Lock(); pktID, pktTyp, payload = id, typ, append([]byte(nil), p...); mu.Unlock() },
		OnClosed:     func(err error) { closed <- err },
	})

	conn.push([]byte{0x01, 0x07, 0x00, 0x00, 0x00})                    // [01][id=7]
	conn.push([]byte{0x03, 0x14, 0x52, 0x7d, 0x00})                    // [03][sid=0x7d5214]
	conn.push([]byte{0x02, 0x07, 0x00, 0x00, 0x00, 0x05, 0x00, 0x04, 0xde, 0xad}) // [02][id=7][len=5][type=4][payload=de ad]

	waitFor(t, func() bool { mu.Lock(); defer mu.Unlock(); return regID == 7 && assigned == 0x7d5214 && pktTyp == 4 })
	mu.Lock()
	if pktID != 7 || string(payload) != "\xde\xad" {
		t.Fatalf("packet: id=%d typ=%d payload=%x", pktID, pktTyp, payload)
	}
	mu.Unlock()

	// нарушение протокола: неизвестный фрейм → close + ErrBadFrame
	conn.push([]byte{0x77})
	if err := <-closed; err != ErrBadFrame {
		t.Fatalf("closed err: %v", err)
	}
	if err := c.Close(); err == nil {
		// соединение уже закрыто readLoop'ом — ок
		_ = err
	}
}

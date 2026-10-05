package authdclient

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"

	"aion-gate/internal/proto"
)

// inbound-типы фреймов authd (OnRead @0x405d40): [01] id, [02] пакет, [03] sid.
const (
	InID  = 0x01
	InPkt = 0x02
	InSID = 0x03
)

// Wire 2110 gate↔authd — байт-в-байт по docs/authgate-protocol-20261005.md §3
// (дизasm 05.10, коммит 4e8181d):
//   gate→authd: [00][4B sid][4B IP] | [01][4B sid] | [02][4B sid][2B len][blob],
//               len = len(blob)+2 (самоинклюзивный, Assemble "cdh");
//   authd→gate: [01][4B id] | [03][4B sid] | [02][4B id][2B len][type][payload],
//               len = body+2, body = type+payload; body ≤ 0x1ffb.
//
// Оригинал НЕ реконнектит после потери authd (authReconnectInterval=0);
// наш Client держит одну коннекцию и сообщает о потере через Handler.OnClosed —
// реконнект решает внешний код (аккуратно, после рестарта authd).

const (
	FrameConnect    = 0x00 // "cdd"
	FrameDisconnect = 0x01 // "cd"
	FramePacket     = 0x02 // "cdh"
)

var (
	ErrNotConnected = errors.New("authdclient: not connected")
	ErrBadFrame     = errors.New("authdclient: protocol violation")
	ErrTooBig       = errors.New("authdclient: body too large")
)

// Handler — inbound колбеки (вызываются из read-горутины).
type Handler struct {
	OnRegistered func(id uint32)                            // [01] authd id
	OnAssigned   func(sid uint32)                           // [03] authd назначил сессию
	OnPacket     func(id uint32, typ byte, payload []byte)  // [02] тип<0x15
	OnClosed     func(err error)                            // коннект потерян
}

type Client struct {
	conn net.Conn
	mu   sync.Mutex
	h    Handler
	max  int
}

// Dial подключается к authd и запускает read-горутину.
func Dial(addr string, h Handler) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return DialConn(conn, h), nil
}

// DialConn — обёртка готового conn (для тестов через net.Pipe).
func DialConn(conn net.Conn, h Handler) *Client {
	c := &Client{conn: conn, h: h, max: 0x1ffb}
	go c.readLoop()
	return c
}

// SendConnect — CltConnect @0x406000: Assemble("cdd", 0, sid, IP).
// ipBytes — байты IP в сетевом порядке (a.b.c.d): Assemble 'd' пишет u32 LE от
// big-endian значения → на проводе байты IP сохраняются 1-в-1.
func (c *Client) SendConnect(sid uint32, ipBytes [4]byte) error {
	return c.write(proto.Assemble("cdd", byte(0), sid,
		binary.BigEndian.Uint32(ipBytes[:])))
}

// SendDisconnect — CltDisconnect @0x406050: Assemble("cd", 1, sid).
func (c *Client) SendDisconnect(sid uint32) error {
	return c.write(proto.Assemble("cd", byte(1), sid))
}

// SendPacket — CltPacket @0x4060a0: Assemble("cdh", 2, sid, len(blob)+2)+blob.
// blob — payload релея (например "cbdb"-сборка RecvLogin).
func (c *Client) SendPacket(sid uint32, blob []byte) error {
	frame := proto.Assemble("cdh", byte(2), sid, uint16(len(blob)+2))
	return c.write(append(frame, blob...))
}

func (c *Client) write(b []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return ErrNotConnected
	}
	_, err := c.conn.Write(b)
	return err
}

// Close закрывает соединение (readLoop вызовет OnClosed).
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

func (c *Client) readLoop() {
	var err error
	for {
		var ft [1]byte
		if _, err = io.ReadFull(c.conn, ft[:]); err != nil {
			break
		}
		switch ft[0] {
		case InID, InSID:
			var rest [4]byte
			if _, err = io.ReadFull(c.conn, rest[:]); err != nil {
				break
			}
			id := binary.LittleEndian.Uint32(rest[:])
			if ft[0] == InID && c.h.OnRegistered != nil {
				c.h.OnRegistered(id)
			}
			if ft[0] == InSID && c.h.OnAssigned != nil {
				c.h.OnAssigned(id)
			}
		case InPkt:
			var hdr [6]byte
			if _, err = io.ReadFull(c.conn, hdr[:]); err != nil {
				break
			}
			id := binary.LittleEndian.Uint32(hdr[0:4])
			body := int(binary.LittleEndian.Uint16(hdr[4:6])) - 2
			if body < 1 || body > c.max {
				err = ErrTooBig
				break
			}
			buf := make([]byte, body)
			if _, err = io.ReadFull(c.conn, buf); err != nil {
				break
			}
			if c.h.OnPacket != nil {
				c.h.OnPacket(id, buf[0], buf[1:])
			}
		default:
			err = ErrBadFrame
		}
		if err != nil {
			break
		}
	}
	c.mu.Lock()
	conn := c.conn
	c.conn = nil
	c.mu.Unlock()
	if conn != nil {
		conn.Close()
	}
	if c.h.OnClosed != nil {
		c.h.OnClosed(err)
	}
}

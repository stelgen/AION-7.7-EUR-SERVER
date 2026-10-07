// Package proto: wire-фрейм AccountCacheServer 7.7 (снят дизasmом, dispatch-77.md).
//
//	[u16 lenMinus2 LE][u16 cmd LE][u8 0xEB][u16 ~cmd LE][payload]
//	lenMinus2 = полная длина на проводе минус 2 (т.е. cmd+EB+~cmd+payload).
//	Лимит payload 0x2000, cmd >= 0x6C = reject (AC_Socket::OnRead @0x1400799F0).
package proto

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	Marker      = 0xEB
	MaxCmd      = 0x6B // cmd >= 0x6C -> reject
	MaxPayload  = 0x2000
	HeaderWire  = 5 // len(2) + cmd(2) + marker(1)
	HeaderBody  = 3 // cmd(2) + marker(1) после len-поля... см. Decode
)

var (
	ErrMarker      = errors.New("accache: bad marker")
	ErrCmdRange    = errors.New("accache: cmd out of range")
	ErrTooLarge    = errors.New("accache: packet too large")
	ErrShort       = errors.New("accache: short packet")
)

// Frame — разобранный пакет.
type Frame struct {
	Cmd     uint16
	Payload []byte // копия payload (без заголовков)
}

// BuildLen — значение len-поля для полной длины на проводе wireLen.
func BuildLen(wireLen int) uint16 {
	return uint16(wireLen - 2)
}

// Build собирает полный кадр на проводе.
func Build(cmd uint16, payload []byte) ([]byte, error) {
	if int(cmd) > MaxCmd {
		return nil, ErrCmdRange
	}
	body := 2 + 1 + 2 + len(payload) // cmd + EB + ~cmd + payload
	wire := make([]byte, 2+body)
	binary.LittleEndian.PutUint16(wire[0:], BuildLen(2+body))
	binary.LittleEndian.PutUint16(wire[2:], cmd)
	wire[4] = Marker
	binary.LittleEndian.PutUint16(wire[5:], ^cmd)
	copy(wire[7:], payload)
	return wire, nil
}

// Validate проверяет заголовок пакета (cmd, 0xEB, ~cmd) без payload.
// buf = пакет БЕЗ len-поля (2 байта): [cmd u16][0xEB][~cmd u16]...payload.
func Validate(buf []byte) (uint16, error) {
	if len(buf) < 5 {
		return 0, ErrShort
	}
	cmd := binary.LittleEndian.Uint16(buf[0:2])
	if buf[2] != Marker {
		return 0, ErrMarker
	}
	if inv := binary.LittleEndian.Uint16(buf[3:5]); inv != ^cmd {
		return 0, fmt.Errorf("%w: cmd=%#x inv=%#x", ErrMarker, cmd, inv)
	}
	if int(cmd) > MaxCmd {
		return 0, ErrCmdRange
	}
	return cmd, nil
}

// Reader — потоковый читатель кадров (аналог state-машины AC_Socket::OnRead).
type Reader struct {
	r      io.Reader
	pend   []byte // недокопленный payload
	plen   int
	state  int // 0 = ждём заголовок, 1 = копим payload
	MaxWire int
}

func NewReader(r io.Reader) *Reader {
	return &Reader{r: r, MaxWire: MaxPayload + 5}
}

// ReadFrame возвращает следующий кадр (блокирующе).
func (rd *Reader) ReadFrame() (*Frame, error) {
	// 1. len-поле
	var lenb [2]byte
	if _, err := io.ReadFull(rd.r, lenb[:]); err != nil {
		return nil, err
	}
	n := int(binary.LittleEndian.Uint16(lenb[:])) + 2 // полная длина на проводе
	if n < HeaderWire {
		return nil, ErrShort
	}
	if n > rd.MaxWire {
		return nil, ErrTooLarge
	}
	body := make([]byte, n-2) // cmd+EB+~cmd+payload
	if _, err := io.ReadFull(rd.r, body); err != nil {
		return nil, err
	}
	cmd, err := Validate(body)
	if err != nil {
		return nil, err
	}
	return &Frame{Cmd: cmd, Payload: body[5:]}, nil
}

// WriteFrame шлёт кадр.
func WriteFrame(w io.Writer, cmd uint16, payload []byte) error {
	b, err := Build(cmd, payload)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

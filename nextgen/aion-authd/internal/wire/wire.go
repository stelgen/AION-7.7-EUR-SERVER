// Package wire — фрейминг authd-wire 2110 (PLAINTEXT, без крипты).
//
// Контракт live-доказан (дизasm AuthGateD 0x406000/0x406050/0x4060a0/0x405d40,
// наш aion-gate internal/authdclient, probe 07.10):
//
//	gate→authd: [00][sid u32 LE][ip u32 LE]                 CltConnect (9Б)
//	              ip = BE-значение октетов: для 192.168.0.253 гейт шлёт
//	              LE(0xC0A800FD) = FD 00 A8 C0 — парсить LE и читать октеты BE.
//	            [01][sid u32 LE]                           CltDisconnect (5Б)
//	            [02][sid u32 LE][len u16 LE][blob]         CltPacket — len САМОинклюзивный = len(blob)+2
//	authd→gate: [03][V u32 LE]                               приветствие при accept (live V=0x0000c621)
//	            [01][sid u32 LE]                           negative-ack «unknown session» (probe-live)
//	            [02][id u32 LE][len u16 LE][type][payload]  len = body+2, body = [type]+payload
//
// body ≤ 0x1ffb; [01]/[03] фиксированно 5Б; неизвестный type-байт = protocol violation
// (гейт в этом случае рвёт коннекцию — не подсовывать ему мусор).
package wire

import (
	"encoding/binary"
	"errors"
	"io"
)

// inbound-типы (от гейта).
const (
	FConnect    = 0x00
	FDisconnect = 0x01
	FPacket     = 0x02
)

// MaxBody — лимит тела [02] (гейт: body ≤ 0x1ffb, иначе ErrTooBig + close).
const MaxBody = 0x1ffb

var (
	ErrBadFrame = errors.New("wire: неизвестный тип фрейма")
	ErrTooBig   = errors.New("wire: тело слишком большое")
	ErrShort    = errors.New("wire: короткий фрейм")
)

// Frame — разобранный фрейм от гейта.
type Frame struct {
	Type byte   // 0x00/0x01/0x02
	Sid  uint32 // id сессии на стороне гейта
	IP   [4]byte
	Blob []byte // только для 0x02
}

// ReadFrame — читает один фрейм из потока (блокирующе).
func ReadFrame(r io.Reader) (Frame, error) {
	var ft [1]byte
	if _, err := io.ReadFull(r, ft[:]); err != nil {
		return Frame{}, err
	}
	f := Frame{Type: ft[0]}
	switch ft[0] {
	case FConnect:
		var b [8]byte
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return Frame{}, err
		}
		f.Sid = binary.LittleEndian.Uint32(b[0:4])
		f.IP = ipFromLE(binary.LittleEndian.Uint32(b[4:8]))
		return f, nil
	case FDisconnect:
		var b [4]byte
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return Frame{}, err
		}
		f.Sid = binary.LittleEndian.Uint32(b[:])
		return f, nil
	case FPacket:
		var hdr [6]byte
		if _, err := io.ReadFull(r, hdr[:]); err != nil {
			return Frame{}, err
		}
		f.Sid = binary.LittleEndian.Uint32(hdr[0:4])
		body := int(binary.LittleEndian.Uint16(hdr[4:6])) - 2 // самоинклюзивный
		if body < 1 {
			return Frame{}, ErrShort
		}
		if body > MaxBody {
			return Frame{}, ErrTooBig
		}
		f.Blob = make([]byte, body)
		if _, err := io.ReadFull(r, f.Blob); err != nil {
			return Frame{}, err
		}
		return f, nil
	default:
		return Frame{}, ErrBadFrame
	}
}

// ipFromLE — IP из u32, прочитанного LE: октеты в BE-порядке значения.
func ipFromLE(v uint32) [4]byte {
	return [4]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}
}

// ---------- сборка (authd→gate; и gate→authd для тестов/инструментов) ----------

// Greeting — приветствие при accept: [03][V u32 LE] (live V=0x0000c621).
func Greeting(v uint32) []byte {
	b := []byte{0x03, 0, 0, 0, 0}
	binary.LittleEndian.PutUint32(b[1:5], v)
	return b
}

// UnknownSession — negative-ack: [01][sid u32 LE] (probe-live: ответ на пакет
// для sid без CltConnect).
func UnknownSession(sid uint32) []byte {
	b := []byte{0x01, 0, 0, 0, 0}
	binary.LittleEndian.PutUint32(b[1:5], sid)
	return b
}

// ReplyPkt — ответ по сессии: [02][id u32 LE][len u16 LE = body+2][type][payload],
// body = 1+len(payload) ≤ MaxBody.
func ReplyPkt(id uint32, typ byte, payload []byte) ([]byte, error) {
	if 1+len(payload) > MaxBody {
		return nil, ErrTooBig
	}
	out := make([]byte, 7, 7+len(payload))
	out[0] = 0x02
	binary.LittleEndian.PutUint32(out[1:5], id)
	binary.LittleEndian.PutUint16(out[5:7], uint16(len(payload)+3)) // body+2 = 1+payload+2
	out = append(out, typ)
	out = append(out, payload...)
	return out, nil
}

// ---- служебные (симуляция стороны гейта: тесты, будущий fork-proxy в authd) ----

// ConnectFrame — [00][sid u32 LE][ip u32 LE от BE-значения октетов].
func ConnectFrame(sid uint32, ip [4]byte) []byte {
	b := make([]byte, 9)
	b[0] = 0x00
	binary.LittleEndian.PutUint32(b[1:5], sid)
	binary.LittleEndian.PutUint32(b[5:9], uint32(ip[0])<<24|uint32(ip[1])<<16|uint32(ip[2])<<8|uint32(ip[3]))
	return b
}

// DisconnectFrame — [01][sid u32 LE].
func DisconnectFrame(sid uint32) []byte {
	b := make([]byte, 5)
	b[0] = 0x01
	binary.LittleEndian.PutUint32(b[1:5], sid)
	return b
}

// PacketFrame — [02][sid u32 LE][len u16 LE = len(blob)+2][blob].
func PacketFrame(sid uint32, blob []byte) ([]byte, error) {
	if len(blob)+2 > MaxBody+2 {
		return nil, ErrTooBig
	}
	out := make([]byte, 7, 7+len(blob))
	out[0] = 0x02
	binary.LittleEndian.PutUint32(out[1:5], sid)
	binary.LittleEndian.PutUint16(out[5:7], uint16(len(blob)+2))
	out = append(out, blob...)
	return out, nil
}

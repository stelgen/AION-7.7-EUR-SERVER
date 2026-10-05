package proto

import (
	"encoding/binary"
	"io"
)

// Фрейминг 2106: [u16 LE total][payload]; total ВКЛЮЧАЕТ сам len-филд
// (welcome 194 = len 0xC2 + 192 ECB). Клиентские 34/74/26/186/314 — та же схема.

const FrameOverhead = 2

// WriteFrame оборачивает payload в [u16 LE total] (total = payload+2).
func WriteFrame(payload []byte) []byte {
	out := make([]byte, len(payload)+FrameOverhead)
	binary.LittleEndian.PutUint16(out, uint16(len(payload)+FrameOverhead))
	copy(out[FrameOverhead:], payload)
	return out
}

// ReadFrame читает один фрейм из потока 2106 и возвращает payload (без len-филда).
func ReadFrame(r io.Reader) ([]byte, error) {
	var l [FrameOverhead]byte
	if _, err := io.ReadFull(r, l[:]); err != nil {
		return nil, err
	}
	total := binary.LittleEndian.Uint16(l[:])
	if total < FrameOverhead {
		return nil, ErrBadPacketLen
	}
	buf := make([]byte, total-FrameOverhead)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

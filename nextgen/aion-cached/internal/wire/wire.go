// Package wire — CacheD64 wire-канон = ACCACHE-ФОРМАТ (доказан golden-кадрами
// server_test.go 0f000100ebfeff... и accache R2.5, "len=total" канон 10.10):
//
//	[u16 LE totalLen][u16 LE cmd][u8 marker][u16 LE ~cmd][payload][u16 LE totalLen]
//
//	marker: C2S=0xEB (Server64→CacheD), S2C=0xEC (CacheD→Server64; PutCmd_ACP
//	инкрементит маркер — та же схема что AccountCacheServer).
//	~cmd = побитовая инверсия cmd (u16). totalLen = ПОЛНАЯ длина кадра включая оба len-поля.
//	Хвост-сумма = u16 LE totalLen (accache: checkSumm == packetSize+2 == полная длина).
//
// ⚠ ИСТОРИЯ (не повторять): R1-каркас был построен по L2 C1-канону (op 1 байт,
// без маркера) — НЕСОВМЕСТИМ с golden-кадрами Aion. R3 (22.10) унифицировано
// на accache-канон: u16-opcode закрывает и 381 RQ > 255 (теория #8 ЗАКРЫТА —
// 2-байтовый опкод доказан golden-кадрами, не нужен «флаг ширины»).
package wire

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	// MarkerC2S — Server64→CacheD.
	MarkerC2S = 0xEB
	// MarkerS2C — CacheD→Server64 (PutCmd_ACP инкремент — accache-канон).
	MarkerS2C = 0xEC
	// HeaderWire — len(2) + cmd(2) + marker(1) + ~cmd(2).
	HeaderWire = 7
	// MaxPayload — лимит тела (accache OnRead: cmp 0x2000).
	MaxPayload = 0x2000
)

// Ошибки парсинга кадра.
var (
	ErrShort       = errors.New("wire: short frame")
	ErrBadSize     = errors.New("wire: bad packet size")
	ErrBadChecksum = errors.New("wire: packet checksum different")
	ErrBadMarker   = errors.New("wire: bad marker")
)

// Frame — разобранный кадр.
type Frame struct {
	Cmd     uint16
	Payload []byte // без cmd/marker/~cmd/csum
}

// BuildC2S — сборка кадра Server64→CacheD (маркер 0xEB).
func BuildC2S(cmd uint16, payload []byte) []byte { return build(MarkerC2S, cmd, payload) }

// BuildS2C — сборка ответа CacheD→Server64 (маркер 0xEC, accache PutCmd_ACP).
func BuildS2C(cmd uint16, payload []byte) []byte { return build(MarkerS2C, cmd, payload) }

func build(marker byte, cmd uint16, payload []byte) []byte {
	total := HeaderWire + len(payload)
	out := make([]byte, total)
	binary.LittleEndian.PutUint16(out[0:], uint16(total)) // len = ПОЛНАЯ длина (и есть контроль — отдельного хвоста НЕТ, golden 0f000100ebfeff... = 15 без хвоста)
	binary.LittleEndian.PutUint16(out[2:], cmd)           // cmd u16 LE
	out[4] = marker                                       // 0xEB / 0xEC
	inv := ^cmd                                           // ~cmd u16
	binary.LittleEndian.PutUint16(out[5:], inv)           // ~cmd
	copy(out[HeaderWire:], payload)                       // payload
	return out
}

// Validate — проверка тела (buf = кадр БЕЗ первого len-поля, как accache Validate):
// buf[0:2]=cmd, buf[2]=marker, buf[3:5]=~cmd; возвращает cmd.
func Validate(buf []byte, wantMarker byte) (uint16, error) {
	if len(buf) < 5 {
		return 0, ErrShort
	}
	cmd := binary.LittleEndian.Uint16(buf[0:2])
	if buf[2] != wantMarker {
		return 0, ErrBadMarker
	}
	if inv := binary.LittleEndian.Uint16(buf[3:5]); inv != ^cmd {
		return 0, fmt.Errorf("%w: cmd=%#x inv=%#x", ErrBadMarker, cmd, inv)
	}
	return cmd, nil
}

// Decoder — state-машина разбора потока (порт accache OnRead).
type Decoder struct {
	Marker byte // ожидаемый маркер входящих кадров (C2S-сторона: 0xEB)
}

// NewDecoder — C2S-декодер (наш сервер читает 0xEB от Server64).
func NewDecoder() *Decoder { return &Decoder{Marker: MarkerC2S} }

// Feed — разбор ОДНОГО кадра из buf.
// Возвращает число съеденных байт, кадр или ошибку.
func (d *Decoder) Feed(buf []byte) (int, *Frame, error) {
	if len(buf) < 2 {
		return 0, nil, ErrShort
	}
	total := int(binary.LittleEndian.Uint16(buf[0:2])) // ПОЛНАЯ длина кадра
	if total < HeaderWire {                            // минимум: заголовок
		return 0, nil, fmt.Errorf("%w: total=%d", ErrBadSize, total)
	}
	if total > MaxPayload+HeaderWire {
		return 0, nil, fmt.Errorf("%w: total=%d > max", ErrBadSize, total)
	}
	if len(buf) < total {
		return 0, nil, ErrShort
	}
	cmd, err := Validate(buf[2:total-2], d.Marker)
	if err != nil {
		return total, nil, err
	}
	return total, &Frame{Cmd: cmd, Payload: buf[HeaderWire:total]}, nil
}
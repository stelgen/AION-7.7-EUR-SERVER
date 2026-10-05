// Package proto — wire-протокол NC-logd (вскрыт дизasm'ом LogServer64.exe+pdb, 05.10.2026).
//
// Фрейминг:  [u16 LE total_len][u8 type(0..4)][0xBB][u8 ~type][payload...]
// Ограничение: total_len ≤ 0x2000, type < 5 (GetCmd_LP: packetType/marker/инверсия).
//
// Версионный handshake:
//
//	клиент → сервер: Version(0):  [00][BB][FF][builder u32][min u32=10003][SYSTEMTIME 16] (total 29)
//	сервер → клиент: Version(0):  тот же layout (LogServerSocket::OnCreate: builder из конфига,
//	                               min=0x2713), затем/либо VersionAndTime(2):
//	                               [02][BB][FD][qword FILETIME] (total 13)
//
// Мэджик 0x68DB8BAD в коде = деление FILETIME тиков на 10^7 → секунды.
//
// Alive: AlivePacket/C_PING/C_RECONNECT в строках бинаря; клиент ретраит коннект,
// недоступность logd безопасна («Can't connect to log server» — rоняет ли? нет).
//
// Payload лог-батча: ProcessLog/ProcessLogBatch(LogSvcType, SYSTEMTIME*, wchar_t*) +
// ParseLogData(wchar*, char*...) + DecodeBotLog — точный layout байт подтверждается
// живым capture на параллельном прогоне (тип 4 — кандидат лог-данных; в skeleton
// пишем raw-hex + эвристичный UTF-16 текст в .raw файл, НЕ в боевые файлы).
package proto

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

const (
	Marker   = 0xBB
	MaxTotal = 0x2000
	MinBld   = 10003 // 0x2713

	TypeVersion       = 0
	TypeVerAndTimeReq = 1 // обработчик строит ответ-2
	TypeVerAndTime    = 2 // [02][BB][FD][qword FILETIME], total 13
	TypeUnknown3      = 3 // заглушка в оригинале (xor eax,eax)
	TypeData          = 4 // кандидат лог-данных (payload layout TBD, см. живой capture)
)

// ErrBadPacket — невалидный пакет (маркер/длина).
var ErrBadPacket = errors.New("logd: bad packet")

// Packet — разобранный пакет.
type Packet struct {
	Type byte
	Body []byte // после [type][BB][~type]
	Raw  []byte // целиком, включая len
}

// Build — собрать пакет: [len u16][type][BB][~type][body...]
func Build(typ byte, body []byte) []byte {
	total := 5 + len(body)
	b := make([]byte, total)
	binary.LittleEndian.PutUint16(b[0:2], uint16(total))
	b[2] = typ
	b[3] = Marker
	b[4] = ^typ
	copy(b[5:], body)
	return b
}

// Parse — разобрать сырой буфер (может содержать ≥1 пакета подряд).
// Возвращает пакет, число съеденных байт, ошибку (ErrShort для неполного).
func Parse(buf []byte) (*Packet, int, error) {
	if len(buf) < 5 {
		return nil, 0, ErrShort
	}
	total := int(binary.LittleEndian.Uint16(buf[0:2]))
	if total < 5 || total > MaxTotal {
		return nil, 0, fmt.Errorf("%w: len %d", ErrBadPacket, total)
	}
	if len(buf) < total {
		return nil, 0, ErrShort
	}
	typ := buf[2]
	if typ > 4 || buf[3] != Marker || buf[4] != ^typ {
		return nil, 0, fmt.Errorf("%w: type=%d marker=%02x inv=%02x", ErrBadPacket, typ, buf[3], buf[4])
	}
	p := &Packet{Type: typ, Body: buf[5:total], Raw: buf[:total:total]}
	return p, total, nil
}

// ErrShort — ждём ещё байт.
var ErrShort = errors.New("logd: short read")

// VersionBody — тело Version: [builder u32][min u32=10003][SYSTEMTIME 16] (29 total).
func VersionBody(builder uint32, systime []byte) []byte {
	b := make([]byte, 24)
	binary.LittleEndian.PutUint32(b[0:4], builder)
	binary.LittleEndian.PutUint32(b[4:8], MinBld)
	if len(systime) == 16 {
		copy(b[8:24], systime)
	} else {
		putFileTime(b[8:16]) // дефолт: текущее FILETIME
	}
	return b
}

// ParseVersion — вытащить builderNumber из тела Version.
func ParseVersion(body []byte) (builder, minBld uint32) {
	if len(body) >= 8 {
		return binary.LittleEndian.Uint32(body[0:4]), binary.LittleEndian.Uint32(body[4:8])
	}
	if len(body) >= 4 {
		return binary.LittleEndian.Uint32(body[0:4]), 0
	}
	return 0, 0
}

// VersionAndTime — [02][BB][FD][qword FILETIME] (total 13).
func VersionAndTime() []byte {
	b := make([]byte, 8)
	putFileTime(b)
	return Build(TypeVerAndTime, b)
}

// ParseVersionAndTime — qword из тела (FILETIME).
func ParseVersionAndTime(body []byte) uint64 {
	if len(body) >= 8 {
		return binary.LittleEndian.Uint64(body[0:8])
	}
	return 0
}

func putFileTime(dst []byte) {
	// FILETIME: 100нс с 1601-01-01 — совместимо с ожиданиями клиента (мэджик /10^7).
	ft := uint64(time.Now().UnixNano())/10 + 116444736000000000
	binary.LittleEndian.PutUint64(dst, ft)
}

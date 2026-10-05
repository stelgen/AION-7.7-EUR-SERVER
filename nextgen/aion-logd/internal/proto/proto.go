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
	Marker   = 0xBB // сервер→клиент
	CMarker  = 0xBA // клиент→сервер (capture 05.10: ВСЕ клиентские пакеты)
	MaxTotal = 0x2000
	MinBld   = 10003 // 0x2713

	TypeVersion       = 0 // клиент: 13 total [builder u32][min u32] (БЕЗ SYSTEMTIME)
	TypeVerAndTimeReq = 1 // обработчик строит ответ-2
	TypeVerAndTime    = 2 // [02][BB][FD][qword FILETIME], total 13
	TypeServerStarted = 3
	TypeControl       = 4 // S→C периодика: body 8 нулей (capture: 6 шт/2 на клиента/4 мин) — анти-флап // клиент: body 12 = 3×u32 (SendServerStarted)
	TypeData          = 4 // TBD
	TypeStatus        = 5
	TypeTextLog       = 9  // текст-логи/онлайн-таблица — парсер internal/textlog (Л1, 05.10)
	TypeAlive         = 11 // EncodeAlive (клиент ping; capture: [0B][BA][F4][qword]) // клиент: body 194 const — статус-поток ~1/2с (floats/счётчики + SYSTEMTIME-хвост)
)

// ErrBadPacket — невалидный пакет (маркер/длина).
var ErrBadPacket = errors.New("logd: bad packet")

// Packet — разобранный пакет.
type Packet struct {
	Type byte
	Body []byte // после [type][BB][~type]
	Raw  []byte // целиком, включая len
}

// Build — собрать пакет (серверный маркер 0xBB): [len u16][type][BB][~type][body...]
func Build(typ byte, body []byte) []byte {
	return buildWith(Marker, typ, body)
}

// BuildC — клиентский пакет (маркер 0xBA) — для фейк-клиентов/тестов.
func BuildC(typ byte, body []byte) []byte {
	return buildWith(CMarker, typ, body)
}

func buildWith(m byte, typ byte, body []byte) []byte {
	total := 5 + len(body)
	b := make([]byte, total)
	binary.LittleEndian.PutUint16(b[0:2], uint16(total))
	b[2] = typ
	b[3] = m
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
	if typ > 12 || (buf[3] != Marker && buf[3] != CMarker) || buf[4] != ^typ {
		return nil, 0, fmt.Errorf("%w: type=%d marker=%02x inv=%02x", ErrBadPacket, typ, buf[3], buf[4])
	}
	p := &Packet{Type: typ, Body: buf[5:total], Raw: buf[:total:total]}
	return p, total, nil
}

// ErrShort — ждём ещё байт.
var ErrShort = errors.New("logd: short read")

// VersionBody — тело Version: [builder u32][min u32=10003][SYSTEMTIME 16] (29 total).
// SYSTEMTIME = 8 WORD (year,month,dow,day,hour,min,sec,ms) ЛОКАЛЬНОЕ время VM —
// клиент конвертит его в FILETIME и сравнивает со своим (мэджик 0x68DB8BAD=/10^7):
// мусор в поле даёт «Time difference»-отказ (клиент молчит — итерация 2 это доказала).
func VersionBody(builder uint32, systime []byte) []byte {
	b := make([]byte, 24)
	binary.LittleEndian.PutUint32(b[0:4], builder)
	binary.LittleEndian.PutUint32(b[4:8], MinBld)
	if len(systime) == 16 {
		copy(b[8:24], systime)
	} else {
		copy(b[8:24], SysTime(time.Now()))
	}
	return b
}

// SysTime — SYSTEMTIME (8 WORD, локальное время как GetLocalTime).
func SysTime(t time.Time) []byte {
	y, m, d := t.Date()
	dow := (int(t.Weekday()) + 6) % 7 // WORD: 0=понедельник... по SDK 0=воскресенье
	dow = int(t.Weekday())            // на деле 0=Sunday..6=Saturday
	b := make([]byte, 16)
	binary.LittleEndian.PutUint16(b[0:2], uint16(y))
	binary.LittleEndian.PutUint16(b[2:4], uint16(m))
	binary.LittleEndian.PutUint16(b[4:6], uint16(dow))
	binary.LittleEndian.PutUint16(b[6:8], uint16(d))
	binary.LittleEndian.PutUint16(b[8:10], uint16(t.Hour()))
	binary.LittleEndian.PutUint16(b[10:12], uint16(t.Minute()))
	binary.LittleEndian.PutUint16(b[12:14], uint16(t.Second()))
	binary.LittleEndian.PutUint16(b[14:16], uint16(t.Nanosecond()/1e6))
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

// AliveReply — ответ сервера на клиентский Alive(type 11): тот же тип, маркер 0xBB,
// qword = локальное-как-UTC FILETIME (клиент сверяет с локальным: иначе «Time difference»
// и Close через 153с — источник RunAsDate-инцидента и нашего флапа).
func AliveReply() []byte {
	b := make([]byte, 8)
	putFileTime(b)
	return Build(TypeAlive, b)
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

// ClientVersion — тело клиентского Version: [builder u32][min u32] (13 total, БЕЗ
// SYSTEMTIME; capture 05.10: builder=200604, min=10003).
func ClientVersion(builder, minBld uint32) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint32(b[0:4], builder)
	binary.LittleEndian.PutUint32(b[4:8], minBld)
	return BuildC(TypeVersion, b)
}

// ServerStarted — body 12 = 3×u32 (SendServerStarted(h,h,h)); capture: (1,2,0)/(1,3,0)/(1,4,-22).
func ServerStarted(a, b, c int32) []byte {
	x := make([]byte, 12)
	binary.LittleEndian.PutUint32(x[0:4], uint32(a))
	binary.LittleEndian.PutUint32(x[4:8], uint32(b))
	binary.LittleEndian.PutUint32(x[8:12], uint32(c))
	return Build(TypeServerStarted, x)
}

func putFileTime(dst []byte) {
	// КРИТИЧНО (флап-разгадка 05.10, дизasm ProcessAliveResponse NPCSvr): клиент вычитает
	// СЕРВЕРНЫЙ qword из СВОЕГО GetSystemTimeAsFileTime (UTC-тики) — diff в мс > 60000 (0xEA60)
	// = «Time difference too big» → Close через ~153с. ПОЭТОМУ qword = ЧЕСТНЫЙ UTC FILETIME
	// (как у любого нормального сервера). Локальное-как-UTC (предыдущая попытка) давало
	// diff = −TZ (3ч = 10.8M мс ≫ 60000) → флап. SYSTEMTIME-поля в Version (другая ветка
	// проверки) остаются ЛОКАЛЬНЫМИ (SysTime()).
	ft := uint64(time.Now().UnixNano())/10 + 116444736000000000
	binary.LittleEndian.PutUint64(dst, ft)
}

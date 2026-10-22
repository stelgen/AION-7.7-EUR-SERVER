// Package wire — CacheD wire-канон из CServerSocket.cpp (read-only ref: nextgen/cached-ref/l2-c1-cached).
//
// Фрейм: [u16 LE self-inclusive len][opcode][payload...][2Б csum].
//
//	Canon (L2CacheD 0x0043C2E0):
//	  m_packetSize = (buf[off+1]<<8) + buf[off] - 2   // размер тела ПОСЛЕ len-филда
//	  DummyCrypt::Decrypt(payload, m_key, m_packetSize) // NO-OP (см. DummyCrypt ниже)
//	  m_key += m_packetSize                              // rolling (vestigial)
//	  checkSumm = (preLastByte + (lastByte<<8)) - 2 == m_packetSize
//	  → u16 LE двух последних байт тела = m_packetSize + 2 = полная wire-длина кадра.
//
// R2 ВЕРДИКТ (21.10, доказано двумя независимыми источниками):
//  1. L2 C1 (сурсы MasterToma): L2Core/src/crypt/DummyCrypt.cpp — тело Decrypt
//     ПУСТОЕ (UNUSED(data/key/length)) — «Dummy» в буквальном смысле, RPC без крипты.
//  2. Aion 7.7 CacheD64.exe (дизasm): крипто-фэмили Utils.obj —
//     Decrypt @0x1401b0360 / Encrypt @0x1401b03e0 / Encrypt @0x1401b0440 /
//     Decrypt2 @0x1401b04c0 / Encrypt2 @0x1401b0570 — НОЛЬ ссылок в бинаре
//     (E8/E9/FF15/lea/ptr-скан пуст, incl. полный .data 19МБ) = мёртвый код.
//     ВЫВОД: wire CacheD64 2006 = plaintext, единственная защита = csum-хвост.
package wire

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	// HeaderSize — размер u16 LE len-филда.
	HeaderSize = 2
	// ChecksumSize — хвост [preLast][last] (u16 LE wireLen).
	ChecksumSize = 2
	// MaxPacketSize — BUFFER_SIZE L2 (8192); L2 rejects packetSize > 8192 (canon: m_packetSize > BUFFER_SIZE).
	MaxPacketSize = 8192
)

// Ошибки парсинга кадра.
var (
	// ErrShort — вход недостаточен даже для чтения len-филда (или len указывает больше, чем есть).
	ErrShort = errors.New("wire: short frame")
	// ErrBadSize — packetSize <= 0 или > MaxPacketSize (L2: «bad packet size» → Close).
	ErrBadSize = errors.New("wire: bad packet size")
	// ErrBadChecksum — контрольная сумма не совпала (L2: «Packet checksum is different» → Close).
	ErrBadChecksum = errors.New("wire: packet checksum is different")
	// ErrUnknownProto — opcode за пределами таблицы хандлеров (L2: «unknown protocol» → Close).
	ErrUnknownProto = errors.New("wire: unknown protocol opcode")
)

// DummyCrypt — NO-OP (R2 вердикт 21.10).
//
// Доказательства:
//  - L2 C1 MasterToma сурсы: L2Core/src/crypt/DummyCrypt.cpp (0x00435050) — тело
//    функции ПУСТОЕ: UNUSED(data); UNUSED(key); UNUSED(length); — декрипта нет.
//  - Aion 7.7 CacheD64.exe: семейство Utils.obj Decrypt/Encrypt/Decrypt2/Encrypt2
//    (map: 0x1401b0360/0x1401b03e0/0x1401b0440/0x1401b04c0/0x1401b0570) не имеет
//    НИ ОДНОЙ ссылки в бинаре — мёртвый код; алгоритм семейства снят (XOR-цепочка
//    со staticTable 64Б @0x1402dccb0 и key[i&7]) — но он НЕ В WIRE-ПУТИ.
//
// Функция сохранена (API-стабильность + vestigial rolling-key канона).
func DummyCrypt(data []byte, key uint32) {
	_, _ = data, key // NO-OP: payload идёт в хандлер как есть (plaintext)
}

// Frame — разобранный кадр канала.
// Opcode — uint16 (ТЗ map[uint16]Handler); на проводе — L2 C1 canon, 1 байт.
type Frame struct {
	Opcode  uint16
	Payload []byte // без opcode и без csum-хвоста
}

// Build — каноничный сборщик кадра (VAssemble-аналог): [u16 LE len][opcode][payload][csum].
// csum = u16 LE от (packetSize + 2) = полной wire-длины.
// Возвращает НЕзашифрованный wire-блоб (plaintext — R2 вердикт).
// OpcodeWidth = 1 (L2 C1 канон); 2-байтовая форма = BuildW (журнал теорий #8:
// RQ 381 > 255 у Aion — ширина НЕПОДТВЕРЖДЕНА, арбитр = live capture).
func Build(opcode uint16, payload []byte) []byte {
	return BuildW(Opcode1Byte, opcode, payload)
}

// BuildW — сборщик с явной шириной опкода на проводе.
func BuildW(width OpcodeWidth, opcode uint16, payload []byte) []byte {
	ps := int(width) + len(payload) // opcode + payload = размер тела без csum
	wireLen := HeaderSize + ps + ChecksumSize

	body := make([]byte, 0, ps+ChecksumSize)
	if width == Opcode2ByteLE {
		body = append(body, byte(opcode&0xFF), byte(opcode>>8))
	} else {
		body = append(body, byte(opcode)) // канон L2 C1: opcode 1 байт
	}
	body = append(body, payload...)
	// csum: (preLast + last<<8) - 2 == packetSize → u16 LE = packetSize + 2 = wireLen
	csum := uint32(ps + ChecksumSize + 2)
	body = append(body, byte(csum&0xFF), byte((csum>>8)&0xFF))

	out := make([]byte, wireLen)
	binary.LittleEndian.PutUint16(out, uint16(wireLen))
	copy(out[HeaderSize:], body)
	return out
}

// Encoder — stateless-сборщик исходящих кадров (инварианты канона: len, csum-хвост).
// L2 C1 Send НЕ шифрует — крипта outbound отсутствует (см. CServerSocket::Send).
type Encoder struct{}

// Encode собирает кадр канона.
func (Encoder) Encode(opcode uint16, payload []byte) []byte { return Build(opcode, payload) }

// Decoder — состояние соединения: rolling key (m_key).
type Decoder struct {
	key         uint32
	opcodeWidth OpcodeWidth
}

// OpcodeWidth — ширина опкода на проводе. Дефолт 1 = L2 C1 канон.
type OpcodeWidth int

const (
	Opcode1Byte   OpcodeWidth = 1
	Opcode2ByteLE OpcodeWidth = 2
)

// NewDecoder создаёт декодер с seed-ключом из конфига.
func NewDecoder(seed uint32) *Decoder { return &Decoder{key: seed, opcodeWidth: Opcode1Byte} }

// SetOpcodeWidth — переключение ширины опкода (R3: RQ 381 > 255 ⇒ Aion может
// использовать 2-байтовые опкоды; арбитр = live capture, журнал теорий #8).
func (d *Decoder) SetOpcodeWidth(w OpcodeWidth) { d.opcodeWidth = w }

// Decrypt расшифровывает payload на месте (DummyCrypt).
func (d *Decoder) Decrypt(payload []byte) { DummyCrypt(payload, d.key) }

// Roll продвигает rolling key: m_key += m_packetSize.
func (d *Decoder) Roll(packetSize int) { d.key += uint32(packetSize) }

// Feed — state-машина разбора ОДНОГО полного кадра из buf:
// [u16 LE self-len][зашифрованное тело].
// Возвращает число съеденных байт (может быть 0 при ErrShort/ErrBadSize — буфер не трогаем)
// или ошибку. Rolling key катится при УСПЕХЕ и при браке csum (как в ориге — m_key += m_packetSize
// выполняется ДО проверки csum).
func (d *Decoder) Feed(buf []byte) (int, *Frame, error) {
	if len(buf) < HeaderSize {
		return 0, nil, ErrShort
	}
	packetSize := int(binary.LittleEndian.Uint16(buf)) - HeaderSize
	if packetSize <= 0 || packetSize > MaxPacketSize {
		// L2 canon: bad size → socket Close; R1: дропаем соединение на стороне вызывателя.
		return 0, nil, fmt.Errorf("%w: %d", ErrBadSize, packetSize)
	}
	if len(buf) < HeaderSize+packetSize {
		return 0, nil, ErrShort
	}
	frame := buf[HeaderSize : HeaderSize+packetSize]

	d.Decrypt(frame)
	d.Roll(packetSize) // канон: m_key += m_packetSize — до csum-проверки

	last := int(frame[packetSize-1])
	preLast := int(frame[packetSize-2])
	checkSum := (preLast + (last << 8)) - 2
	if checkSum != packetSize {
		return packetSize + HeaderSize, nil, fmt.Errorf("%w: proto=%d len=%d check=%d",
			ErrBadChecksum, frame[0], packetSize, checkSum)
	}

	var op uint16
	var pl []byte
	if d.opcodeWidth == Opcode2ByteLE {
		if packetSize < 2+ChecksumSize {
			return packetSize + HeaderSize, nil, fmt.Errorf("%w: 2-byte opcode in tiny frame", ErrBadSize)
		}
		op = binary.LittleEndian.Uint16(frame[0:2])
		pl = frame[2 : packetSize-ChecksumSize]
	} else {
		op = uint16(frame[0])
		pl = frame[1 : packetSize-ChecksumSize]
	}
	return packetSize + HeaderSize, &Frame{Opcode: op, Payload: pl}, nil
}

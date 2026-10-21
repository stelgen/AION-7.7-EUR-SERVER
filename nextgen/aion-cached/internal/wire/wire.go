// Package wire — L2 C1 CacheD wire-канон из CServerSocket.cpp (read-only ref: nextgen/cached-ref/l2-c1-cached).
//
// Фрейм: [u16 LE self-inclusive len][opcode][payload...][2Б csum].
//
//	Canon (L2CacheD 0x0043C2E0):
//	  m_packetSize = (buf[off+1]<<8) + buf[off] - 2   // размер тела ПОСЛЕ len-филда
//	  DummyCrypt::Decrypt(payload, m_key, m_packetSize) // XOR-стрим, rolling key
//	  m_key += m_packetSize                              // rolling
//	  checkSumm = (preLastByte + (lastByte<<8)) - 2 == m_packetSize
//	  → u16 LE двух последних байт тела = m_packetSize + 2 = полная wire-длина кадра.
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

// DummyCrypt — XOR-стрим L2 C1 (симметричная операция): payload[i] ^= byte(key + i).
// Rolling: seed += m_packetSize ПОСЛЕ каждого пакета.
// R2 TODO: дизasm DummyCrypt для подтверждения точной формы (усечение ключа/порядок).
func DummyCrypt(data []byte, key uint32) {
	for i := range data {
		data[i] ^= byte(key + uint32(i))
	}
}

// Frame — разобранный кадр канала.
// Opcode — uint16 (ТЗ map[uint16]Handler); на проводе — L2 C1 canon, 1 байт.
type Frame struct {
	Opcode  uint16
	Payload []byte // без opcode и без csum-хвоста
}

// Build — каноничный сборщик кадра (VAssemble-аналог): [u16 LE len][opcode][payload][csum].
// csum = u16 LE от (packetSize + 2) = полной wire-длины.
// Возвращает НЕзашифрованный wire-блоб (L2 C1 Send не шифрует ответы — канон CServerSocket::Send).
func Build(opcode uint16, payload []byte) []byte {
	ps := 1 + len(payload) // opcode(1Б на проводе) + payload = размер тела без csum
	wireLen := HeaderSize + ps + ChecksumSize

	body := make([]byte, 0, ps+ChecksumSize)
	body = append(body, byte(opcode)) // канон L2 C1: opcode 1 байт
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
	key uint32
}

// NewDecoder создаёт декодер с seed-ключом из конфига.
func NewDecoder(seed uint32) *Decoder { return &Decoder{key: seed} }

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

	return packetSize + HeaderSize, &Frame{Opcode: uint16(frame[0]), Payload: frame[1 : packetSize-ChecksumSize]}, nil
}

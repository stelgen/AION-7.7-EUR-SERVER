// Package crypt — крипта Game-протокола Aion 7.x (7.5-7.8, live-подтверждена capture 08.10).
// Схема: XOR-стрим staticKey[i&63]^key8[i&7]^prev + rolling key += len(body) за пакет.
// Канон: Mobius 7.7 / aion-germany 7.8 Crypt.java + EncryptionKeyPair.java; арбитр = capture (0 invalid/7500).
package crypt

import "encoding/binary"

// StaticKey — вшитый в клиент 64-байтовый ключ (латиница-строка из эталонов).
var StaticKey = []byte("nKO/WctQ0AVLbpzfBkS6NevDYT8ourG5CRlmdjyJ72aswx4EPq1UgZhFMXH?3iI9")

// Серв-код второго байта S2C-тела / клиентский код C2S-тела.
const (
	ServerPacketCode = 0x56 // 7.5 (крипта 7.8 = 1-в-1)
	ClientPacketCode = 0x75 // 4.3..7.x
)

// key suffix из эталона EncryptionKeyPair.
var keySuffix = [4]byte{0xA1, 0x6C, 0x54, 0x87}

// Dir — направление пакета.
type Dir int

const (
	S2C Dir = iota // сервер -> клиент
	C2S            // клиент -> сервер
)

// KeyPair — rolling-ключ соединения (8 байт: baseKey LE + суффикс).
// Server и client ключи стартуют одинаково, катаются независимо (+len тела за пакет).
type KeyPair struct {
	b [8]byte
}

// NewKeyPair строит ключ из baseKey (u32, шлётся в SM_KEY в обфусцированном виде).
func NewKeyPair(base uint32) *KeyPair {
	k := &KeyPair{}
	binary.LittleEndian.PutUint32(k.b[:4], base)
	copy(k.b[4:], keySuffix[:])
	return k
}

// BaseFromFalseKey восстанавливает baseKey из значения, полученного клиентом в SM_KEY.
// Формула 7.x: falseKey = (base ^ 0xCD92E4D9) + 0x3FF2CCDF.
func BaseFromFalseKey(falseKey uint32) uint32 {
	return ((falseKey - 0x3FF2CCDF) & 0xFFFFFFFF) ^ 0xCD92E4D9
}

// FalseKey — обфускация baseKey для отправки в SM_KEY.
func FalseKey(base uint32) uint32 {
	return (base ^ 0xCD92E4D9) + 0x3FF2CCDF
}

// EncodeOpcode — обфускация опкода S2C (Crypt.encodeOpcodec, 7.5).
func EncodeOpcode(op uint16) uint16 {
	return (op + 0xD8) ^ 0xD9
}

// DecodeOpcode — обратная расшифровка опкода S2C.
func DecodeOpcode(e uint16) uint16 {
	return ((e ^ 0xD9) - 0xD8) & 0xFFFF
}

// cryptBody шифрует/дешифрует тело XOR-стримом in-place по схеме эталона.
func cryptBody(b []byte, k *KeyPair, decrypt bool) {
	if len(b) == 0 {
		return
	}
	// prev = ШИФРОВАННЫЙ байт: decrypt = до xor, encrypt = после xor (канон Java).
	var prev byte
	if decrypt {
		prev = b[0]
		b[0] ^= k.b[0]
	} else {
		b[0] ^= k.b[0]
		prev = b[0]
	}
	for i := 1; i < len(b); i++ {
		cur := b[i]
		if decrypt {
			b[i] ^= StaticKey[i&63] ^ k.b[i&7] ^ prev
		} else {
			b[i] ^= StaticKey[i&63] ^ k.b[i&7] ^ prev
		}
		if decrypt {
			prev = cur
		} else {
			prev = b[i]
		}
	}
}

// Encrypt — шифрует тело S2C-пакета (body = [E][0x56][~E][payload]) и катит ключ на len(body).
func (k *KeyPair) Encrypt(body []byte) {
	cryptBody(body, k, false)
	k.roll(len(body))
}

// Decrypt — дешифрует тело пакета (dir задаёт ожидаемый код-байт), катит ключ при успехе.
// Возвращает decoded body; ok=false если валидация заголовка провалилась (ключ НЕ катится).
func (k *KeyPair) Decrypt(body []byte, dir Dir) ([]byte, bool) {
	if len(body) < 5 {
		return body, false
	}
	dec := make([]byte, len(body))
	copy(dec, body)
	cryptBody(dec, k, true)
	code := dec[2]
	x := binary.LittleEndian.Uint16(dec[0:2])
	inv := binary.LittleEndian.Uint16(dec[3:5])
	ok := (x^0xFFFF) == inv
	if dir == S2C {
		ok = ok && code == ServerPacketCode
	} else {
		ok = ok && code == ClientPacketCode
	}
	if !ok {
		return dec, false
	}
	k.roll(len(body))
	return dec, true
}

func (k *KeyPair) roll(n int) {
	v := binary.LittleEndian.Uint64(k.b[:])
	v += uint64(n)
	binary.LittleEndian.PutUint64(k.b[:], v)
}

// Bytes — текущее значение ключа (для логов).
func (k *KeyPair) Bytes() [8]byte { return k.b }
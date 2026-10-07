// Package wire — фрейминг Game-протокола: [u16 size LE self-inclusive][body].
// Тело S2C: [E=(op+0xD8)^0xD9 u16][0x56][~E u16][payload]; тело C2S: [op u16][0x75][~op u16][payload].
// Канон: Mobius AionServerPacket.write / AionPacketHandler.handle; арбитр = capture (OPCODES.md).
package wire

import "encoding/binary"

// Split режет TCP-поток на тела фреймов (без size-поля).
// Возвращает тела и число «хвостов» байт (некратность — только на обрыве потока).
func Split(stream []byte) (bodies [][]byte, tail int) {
	i := 0
	for i+2 <= len(stream) {
		size := int(binary.LittleEndian.Uint16(stream[i : i+2]))
		if size < 5 || i+size > len(stream) { // мин тело 3 (упрощённый фрейминг в тестах)
			break
		}
		bodies = append(bodies, stream[i+2:i+size])
		i += size
	}
	return bodies, len(stream) - i
}

// BuildServerFrame собирает полный фрейм S2C: [size][E][0x56][~E][payload] (без шифрования).
func BuildServerFrame(op uint16, payload []byte) []byte {
	e := uint16(int(op+0xD8) ^ 0xD9)
	body := make([]byte, 5+len(payload))
	binary.LittleEndian.PutUint16(body[0:2], e)
	body[2] = 0x56
	binary.LittleEndian.PutUint16(body[3:5], ^e&0xFFFF)
	copy(body[5:], payload)
	return Frame(body)
}

// Frame оборачивает тело в [u16 size LE self-inclusive].
func Frame(body []byte) []byte {
	out := make([]byte, 2+len(body))
	binary.LittleEndian.PutUint16(out[0:2], uint16(len(out)))
	copy(out[2:], body)
	return out
}

// BodySize — размер фрейма для данного тела (size поле = len+2).
func BodySize(body []byte) uint16 { return uint16(len(body) + 2) }

// ParseServerBody разбирает расшифрованное S2C-тело → (op, payload, ok).
func ParseServerBody(dec []byte) (op uint16, payload []byte, ok bool) {
	if len(dec) < 5 {
		return 0, nil, false
	}
	e := binary.LittleEndian.Uint16(dec[0:2])
	if dec[2] != 0x56 || binary.LittleEndian.Uint16(dec[3:5]) != ^e&0xFFFF {
		return 0, nil, false
	}
	op = uint16(int(e^0xD9) - 0xD8)
	return op, dec[5:], true
}

// ParseClientBody разбирает расшифрованное C2S-тело → (op, payload, ok).
func ParseClientBody(dec []byte) (op uint16, payload []byte, ok bool) {
	if len(dec) < 5 {
		return 0, nil, false
	}
	a := binary.LittleEndian.Uint16(dec[0:2])
	if dec[2] != 0x75 || binary.LittleEndian.Uint16(dec[3:5]) != ^a&0xFFFF {
		return 0, nil, false
	}
	return a, dec[5:], true
}

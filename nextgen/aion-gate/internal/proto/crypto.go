package proto

import (
	"encoding/binary"
	"errors"
)

// Крипто-функции AuthGateD (док §1, верифицировано capture 03.10).
// EncryptPrimary @0x417a20 / DecryptSecondary @0x417ad0.

var (
	ErrBadPacketLen = errors.New("proto: bad packet length")
	ErrChecksum     = errors.New("proto: checksum mismatch")
)

// EncryptPrimary: len_out=(len+7)&~7 → скрамбл DWORD LE (data[0] не меняется,
// data[k]^=cumsum(data_old[0..k]) для k≥1) → dword-чексумма (cumsum всех dword
// данных) за данными → ECB на (n+8) байт. Используется для welcome (key1).
func EncryptPrimary(bf *Blowfish, data []byte) []byte {
	n := (len(data) + 7) &^ 7
	buf := make([]byte, n+8)
	copy(buf, data)
	var sum uint32
	for k := 0; k < n/4; k++ {
		old := binary.LittleEndian.Uint32(buf[k*4:])
		sum += old
		if k > 0 {
			binary.LittleEndian.PutUint32(buf[k*4:], old^sum)
		}
	}
	binary.LittleEndian.PutUint32(buf[n:], sum)
	out := make([]byte, n+8)
	for off := 0; off < n+8; off += 8 {
		bf.Encrypt(out[off:off+8], buf[off:off+8])
	}
	return out
}

// DecryptSecondary: ECB-dec + проверка XOR-чексуммы
// (data[(len-8)/4] == XOR(data[0..(len-8)/4))), len кратно 8.
// Используется для клиентских пакетов (key2), скрамбл НЕ применяется.
func DecryptSecondary(bf *Blowfish, data []byte) ([]byte, error) {
	if len(data) == 0 || len(data)%8 != 0 {
		return nil, ErrBadPacketLen
	}
	dec := make([]byte, len(data))
	for off := 0; off < len(data); off += 8 {
		bf.Decrypt(dec[off:off+8], data[off:off+8])
	}
	n := len(dec)/4 - 1
	var x uint32
	for k := 0; k < n; k++ {
		x ^= binary.LittleEndian.Uint32(dec[k*4:])
	}
	if x != binary.LittleEndian.Uint32(dec[n*4:]) {
		return nil, ErrChecksum
	}
	return dec[:n*4], nil
}

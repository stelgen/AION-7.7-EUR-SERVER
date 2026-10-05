package proto

import (
	"encoding/binary"
	"errors"
)

// Крипто-функции AuthGateD — семантика снята дизasm'ом ночь-4
// (docs/session-20261005-authgate-disasm407d50.md):
// EncryptPrimary @0x417a20, EncryptSecondary @0x417a80, DecryptSecondary @0x417ad0.
// ECB-примитивы: 0x4178a0 = encrypt, 0x417870 = decrypt; key-schedule 0x4178d0 стандартный.

var (
	ErrBadPacketLen = errors.New("proto: bad packet length")
	ErrChecksum     = errors.New("proto: checksum mismatch")
)

// EncryptPrimary @0x417a20 (welcome, key1) — сверено с asm 1-в-1:
// n'=roundup8(len); S=old[0] (dword0 НЕ трогается);
// для k=1..n'/4-1: S+=old[k]; new[k]=old[k]^S (S ВКЛЮЧАЕТ old[k]);
// dword[n'/4] = S (чексумма = финальный cumsum, offset n');
// len_out = n'+8; ECB(buf, n'+8).
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

// EncryptSecondary @0x417a80 (server→client, key2): n'=roundup8(len);
// csum = XOR dword[0..n'/4) кладётся в dword[n'/4] (СРАЗУ за данными,
// паддинг-нули ПОСЛЕ чексуммы); len_out = n'+8; ECB.
// Capture-структура 42b-echo ([A][28×0] → cipher [P][Q][Q][Q][P]) подтверждает.
func EncryptSecondary(bf *Blowfish, payload []byte) []byte {
	n := (len(payload) + 7) &^ 7
	buf := make([]byte, n+8)
	copy(buf, payload)
	var x uint32
	for k := 0; k < n/4; k++ {
		x ^= binary.LittleEndian.Uint32(buf[k*4:])
	}
	binary.LittleEndian.PutUint32(buf[n:], x)
	out := make([]byte, n+8)
	for off := 0; off < n+8; off += 8 {
		bf.Encrypt(out[off:off+8], buf[off:off+8])
	}
	return out
}

// DecryptSecondary @0x417ad0 (client→server, key2): len кратно 8; ECB-dec;
// XOR dword[0..(len-8)/4) обязан равняться dword[(len-8)/4] (чексумма сразу
// за roundup8-данными); хвостовой dword (pad) не участвует.
// Возвращает данные (len-8 байт).
func DecryptSecondary(bf *Blowfish, data []byte) ([]byte, error) {
	if len(data) == 0 || len(data)%8 != 0 {
		return nil, ErrBadPacketLen
	}
	dec := make([]byte, len(data))
	for off := 0; off < len(data); off += 8 {
		bf.Decrypt(dec[off:off+8], data[off:off+8])
	}
	k := (len(dec) - 8) / 4
	var x uint32
	for i := 0; i < k; i++ {
		x ^= binary.LittleEndian.Uint32(dec[i*4:])
	}
	if x != binary.LittleEndian.Uint32(dec[k*4:]) {
		return nil, ErrChecksum
	}
	return dec[:len(dec)-8], nil
}
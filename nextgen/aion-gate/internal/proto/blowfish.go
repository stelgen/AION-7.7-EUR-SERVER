package proto

import (
	"encoding/binary"
	"errors"
)

// Стандартный Blowfish ECB. P/S — пи-константы из .rdata AuthGateD (@0x437570/@0x4375b8;
// Blowfish_Init @0x4178d0 использует те же стандартные константы, key-schedule стандартный).
// Реализация без внешних зависимостей (формы x/crypto/blowfish, длина ключа 1..56).
// Кросс-проверена векторами testdata/blowfish-vectors.json (python-blowfish).

const maxKeyLen = 56

var ErrInvalidKeyLen = errors.New("blowfish: invalid key length")

type Blowfish struct {
	p [18]uint32
	s [4][256]uint32
}

func NewBlowfish(key []byte) (*Blowfish, error) {
	if len(key) == 0 || len(key) > maxKeyLen {
		return nil, ErrInvalidKeyLen
	}
	b := &Blowfish{}
	b.p, b.s = mustParsePIOS()
	expandKey(key, b)
	return b, nil
}

// Encrypt шифрует один 8-байтный блок ECB (dst может совпадать со src).
func (b *Blowfish) Encrypt(dst, src []byte) {
	if len(src) != 8 || len(dst) < 8 {
		panic("blowfish: 8-byte block required")
	}
	l := binary.LittleEndian.Uint32(src[0:4])
	r := binary.LittleEndian.Uint32(src[4:8])
	l, r = encryptBlock(l, r, b)
	binary.LittleEndian.PutUint32(dst[0:4], l)
	binary.LittleEndian.PutUint32(dst[4:8], r)
}

// Decrypt дешифрует один 8-байтный блок ECB.
func (b *Blowfish) Decrypt(dst, src []byte) {
	if len(src) != 8 || len(dst) < 8 {
		panic("blowfish: 8-byte block required")
	}
	l := binary.LittleEndian.Uint32(src[0:4])
	r := binary.LittleEndian.Uint32(src[4:8])
	l, r = decryptBlock(l, r, b)
	binary.LittleEndian.PutUint32(dst[0:4], l)
	binary.LittleEndian.PutUint32(dst[4:8], r)
}

func bfF(s *[4][256]uint32, x uint32) uint32 {
	return ((s[0][byte(x>>24)] + s[1][byte(x>>16)]) ^ s[2][byte(x>>8)]) + s[3][byte(x)]
}

func encryptBlock(l, r uint32, b *Blowfish) (uint32, uint32) {
	xl, xr := l, r
	for i := 0; i < 16; i++ {
		xl ^= b.p[i]
		xr ^= bfF(&b.s, xl)
		xl, xr = xr, xl
	}
	xl, xr = xr, xl // undo последнего swap
	xr ^= b.p[16]
	xl ^= b.p[17]
	return xl, xr
}

func decryptBlock(l, r uint32, b *Blowfish) (uint32, uint32) {
	xl, xr := l, r
	for i := 17; i > 1; i-- {
		xl ^= b.p[i]
		xr ^= bfF(&b.s, xl)
		xl, xr = xr, xl
	}
	xl, xr = xr, xl // undo последнего swap
	xr ^= b.p[1]
	xl ^= b.p[0]
	return xl, xr
}

func expandKey(key []byte, b *Blowfish) {
	j := 0
	for i := 0; i < 18; i++ {
		var d uint32
		for k := 0; k < 4; k++ {
			d = d<<8 | uint32(key[j])
			j++
			if j >= len(key) {
				j = 0
			}
		}
		b.p[i] ^= d
	}
	var l, r uint32
	for i := 0; i < 18; i += 2 {
		l, r = encryptBlock(l, r, b)
		b.p[i], b.p[i+1] = l, r
	}
	for i := 0; i < 4; i++ {
		for k := 0; k < 256; k += 2 {
			l, r = encryptBlock(l, r, b)
			b.s[i][k], b.s[i][k+1] = l, r
		}
	}
}

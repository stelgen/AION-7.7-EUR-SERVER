package proto

import (
	_ "embed"
	"encoding/binary"
)

// LUT @0x437160 (256 u32 LE) — дамп из AuthGateD_original.exe (05.10).

//go:embed testdata/gate-lut.bin
var lutBin []byte

var lut [256]uint32

func init() {
	if len(lutBin) != 1024 {
		panic("gate-lut.bin: ожидалось 1024 байта")
	}
	for i := range lut {
		lut[i] = binary.LittleEndian.Uint32(lutBin[i*4:])
	}
}

// StaticKeyHex — статический key1 (известный ключ AION-клиента).
const StaticKeyHex = "6b60cb5b82ce90b1cc2b6c556c6c6c6c"

// StaticKey — key1 из testdata LUT (генерируется, но в гейте константен).
func StaticKey() [16]byte { return GenerateInitialKey(0x04bd) }

// GenerateInitialKey (seed 0x04bd): dh=seed>>8, dl=seed&0xff, base=dl^dh;
// k[i] = LUT[dl] ^ LUT[(base^i)&0xff] ^ LUT[dh] (i=0..2, u32 LE) + 4×0x6c.
func GenerateInitialKey(seed uint16) [16]byte {
	dl, dh := byte(seed&0xff), byte(seed>>8)
	base := dl ^ dh
	var key [16]byte
	for i := 0; i < 3; i++ {
		v := lut[dl] ^ lut[base^byte(i)] ^ lut[dh]
		binary.LittleEndian.PutUint32(key[i*4:], v)
	}
	for j := 12; j < 16; j++ {
		key[j] = 0x6c
	}
	return key
}

// Key2FromLUT @0x4075d0: r = rand()&0xff; key2 = 4 DWORD LE:
// LUT[r], LUT[r&0xc1], LUT[r&0xf2], LUT[r&0x23]. Отдаётся клиенту внутри welcome.
func Key2FromLUT(r byte) [16]byte {
	idx := [4]uint32{uint32(r), uint32(r) & 0xc1, uint32(r) & 0xf2, uint32(r) & 0x23}
	var key [16]byte
	for i, j := range idx {
		binary.LittleEndian.PutUint32(key[i*4:], lut[j])
	}
	return key
}

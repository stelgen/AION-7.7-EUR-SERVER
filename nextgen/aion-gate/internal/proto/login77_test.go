package proto

// Тесты Фазы-1 (mobius-77-flow-review-20261006 §Тесты 3-4):
//  3. TestSplitLoginOp       — К-3/P1-3: op возвращается, ct/tail не смещаются при op=0x0B
//  4. TestDecodeK1TwoLayouts — К-4/P1-4: k=1 → 4.8 (@94/@108) и эталон-7.7 (@64/@96) гипотезы

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"testing"
)

func enc77(k *RSAKey, m []byte) []byte {
	return new(big.Int).Exp(new(big.Int).SetBytes(m), big.NewInt(65537), k.Priv.PublicKey.N).FillBytes(make([]byte, 128))
}

// Тест 3: SplitLogin возвращает op=pt[0]; ct/tail не зависят от значения op.
func TestSplitLoginOp(t *testing.T) {
	k, err := GenerateRSAKey(65537)
	if err != nil {
		t.Fatal(err)
	}
	m := make([]byte, 128)
	copy(m[94:102], "opuser12")
	ct := enc77(k, m)
	tail := make([]byte, 55)
	tail[20] = 0x20

	pt0B := append([]byte{0x0B}, ct...)
	pt0B = append(pt0B, tail...)
	op, chunks, tl, ok := SplitLogin(pt0B)
	if !ok || op != 0x0B || len(chunks) != 1 || len(tl) != 55 {
		t.Fatalf("op=0x0B: ok=%v op=%02x chunks=%d tail=%d", ok, op, len(chunks), len(tl))
	}
	if !bytes.Equal(chunks[0], ct) {
		t.Fatal("ct сместился при op=0x0B")
	}
	if !bytes.Equal(tl, tail) {
		t.Fatal("tail сместился при op=0x0B")
	}

	pt00 := append([]byte{0x00}, ct...)
	pt00 = append(pt00, tail...)
	op2, chunks2, tl2, ok2 := SplitLogin(pt00)
	if !ok2 || op2 != 0x00 || !bytes.Equal(chunks2[0], chunks[0]) || !bytes.Equal(tl2, tl) {
		t.Fatalf("op=0x00: ok=%v op=%02x (ct/tail должны быть идентичны)", ok2, op2)
	}

	// loginex: op тоже возвращается
	ptEx := append([]byte{0x0B}, ct...)
	ptEx = append(ptEx, ct...)
	ptEx = append(ptEx, make([]byte, 47)...)
	opE, chunksE, _, okE := SplitLogin(ptEx)
	if !okE || opE != 0x0B || len(chunksE) != 2 {
		t.Fatalf("loginex: ok=%v op=%02x chunks=%d", okE, opE, len(chunksE))
	}
}

// Тест 4: k=1 — две гипотезы раскладки: 4.8 (@94/@108) приоритет; при не-printable —
// эталон 7.7 (user=m[64:96](32), pwd=m[96:128](32), otp LE m[124:128]). ОБЕ в лог-полях.
func TestDecodeK1TwoLayouts(t *testing.T) {
	// m с user@94 → детект 4.8
	m48 := make([]byte, 128)
	copy(m48[94:101], "user48x")
	copy(m48[108:112], "pw48")
	binary.LittleEndian.PutUint32(m48[124:128], 0xFFFFFFFF)
	d, ok := DecodeLoginPlain([][]byte{m48})
	if !ok || d.Layout != "48" || d.User != "user48x" || d.Pwd != "pw48" || d.Otp != 0xFFFFFFFF {
		t.Fatalf("4.8: ok=%v layout=%q %+v", ok, d.Layout, d)
	}
	if d.Alt77User != "" {
		t.Fatalf("4.8: alt77=%q (не должен читаться при валидной 4.8)", d.Alt77User)
	}

	// m с user@64 → детект 7.7 (4.8-гипотеза пустая: m[94]=0 → не-printable)
	m77 := make([]byte, 128)
	copy(m77[64:72], "user7777")
	copy(m77[96:112], "pwd-77-16bytes.")
	binary.LittleEndian.PutUint32(m77[124:128], 0x01020304)
	d77, ok77 := DecodeLoginPlain([][]byte{m77})
	if !ok77 || d77.Layout != "77" || d77.User != "user7777" || d77.Pwd != "pwd-77-16bytes." || d77.Otp != 0x01020304 {
		t.Fatalf("7.7: ok=%v layout=%q %+v", ok77, d77.Layout, d77)
	}
	if d77.Alt77User != "user7777" {
		t.Fatalf("7.7: alt77=%q (обе гипотезы должны быть в полях для лога)", d77.Alt77User)
	}

	// ни одна раскладка не валидна → ok=false, ОБЕ гипотезы заполнены для лога
	mBad := make([]byte, 128)
	mBad[64] = 0xEE // 7.7-user: мусор с первого байта → не-printable
	mBad[94] = 0xEE // 4.8-user: мусор с первого байта → не-printable
	dBad, okBad := DecodeLoginPlain([][]byte{mBad})
	if okBad {
		t.Fatalf("bad: ok=%v (обе гипотезы невалидны)", okBad)
	}
	if dBad.Alt77User == "" && dBad.User == "" {
		t.Fatal("bad: поля гипотез пусты — логу нечего показать")
	}
}

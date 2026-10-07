package crypt

import (
	"encoding/binary"
	"os"
	"testing"
)

// TestFalseKeyRoundtrip — канон: falseKey = (base ^ 0xCD92E4D9) + 0x3FF2CCDF.
func TestFalseKeyRoundtrip(t *testing.T) {
	for _, base := range []uint32{0xCE0879A6, 0x00000001, 0xFFFFFFFF, 0x12345678} {
		fk := FalseKey(base)
		if got := BaseFromFalseKey(fk); got != base {
			t.Fatalf("base %08X: got %08X", base, got)
		}
	}
}

// TestEncodeDecodeOpcode — канон: E = (op + 0xD8) ^ 0xD9; SM_KEY 0x48 -> 0x01F9 (live-подтверждено).
func TestEncodeDecodeOpcode(t *testing.T) {
	if e := EncodeOpcode(0x48); e != 0x01F9 {
		t.Fatalf("SM_KEY encode: got %04X want 01F9", e)
	}
	if op := DecodeOpcode(0x01F9); op != 0x48 {
		t.Fatalf("SM_KEY decode: got %04X want 0048", op)
	}
	for op := uint16(0); ; op += 0x1237 {
		if DecodeOpcode(EncodeOpcode(op)) != op {
			t.Fatalf("roundtrip fail op=%04X", op)
		}
		if op > 0xF000 {
			break
		}
	}
}

// TestGoldenSMKeyAndC2S — РЕАЛЬНЫЕ фреймы из capture 0810d (юзер-сессия 10:0x):
// SM_KEY открытый, затем первый шифрованный C2S (CM_VERSION_CHECK 0x00D6, len=22).
func TestGoldenSMKeyAndC2S(t *testing.T) {
	smkey, err := os.ReadFile("testdata/golden_smkey.bin")
	if err != nil {
		t.Skipf("testdata отсутствует: %v", err)
	}
	if len(smkey) != 11 || binary.LittleEndian.Uint16(smkey[0:2]) != 11 {
		t.Fatalf("golden smkey size: %d bytes, hex %X", len(smkey), smkey)
	}
	// первый открытый фрейм: тело [E=01F9][56][06FE][falseKey u32]
	body := smkey[2:]
	e := binary.LittleEndian.Uint16(body[0:2])
	if e != 0x01F9 || body[2] != ServerPacketCode {
		t.Fatalf("smkey header: %X", body)
	}
	falseKey := binary.LittleEndian.Uint32(body[5:9])
	base := BaseFromFalseKey(falseKey)
	if base == 0 {
		t.Fatalf("base==0")
	}
	k := NewKeyPair(base)

	// первый C2S-фрейм после SM_KEY: дешифровка + валидация + op
	c2s, err := os.ReadFile("testdata/golden_c2s.bin")
	if err != nil {
		t.Fatalf("golden c2s: %v", err)
	}
	bodyC := c2s[2:]
	dec, ok := k.Decrypt(bodyC, C2S)
	if !ok {
		t.Fatalf("decrypt golden C2S failed: %X", bodyC)
	}
	a := binary.LittleEndian.Uint16(dec[0:2])
	if a != 0x00D6 {
		t.Fatalf("golden C2S op: got %04X want 00D6 (CM_VERSION_CHECK)", a)
	}
}

// TestEncryptDecryptRoundtrip — наш S2C-пакет должен дешифроваться нашим же ключом.
func TestEncryptDecryptRoundtrip(t *testing.T) {
	// канон: сервер и клиент держат НЕЗАВИСИМЫЕ копии ключа (каждый катит свой)
	payload := []byte("payload-test-data")
	kEnc := NewKeyPair(0xDEADBEEF)
	k := NewKeyPair(0xDEADBEEF)
	body := make([]byte, 5+len(payload))
	binary.LittleEndian.PutUint16(body[0:2], EncodeOpcode(0x0E))
	body[2] = ServerPacketCode
	binary.LittleEndian.PutUint16(body[3:5], ^EncodeOpcode(0x0E)&0xFFFF)
	copy(body[5:], payload)
	kEnc.Encrypt(body)
	dec, ok := k.Decrypt(body, S2C)
	if !ok {
		t.Fatalf("roundtrip decrypt fail")
	}
	if string(dec[5:]) != "payload-test-data" {
		t.Fatalf("payload mismatch: %q", dec[5:])
	}
}
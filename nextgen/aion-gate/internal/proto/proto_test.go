package proto

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"testing"
)

// Тест-векторы Blowfish: python-blowfish (pip) → testdata/blowfish-vectors.json.
func TestBlowfishVectors(t *testing.T) {
	raw, err := os.ReadFile("testdata/blowfish-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vecs []struct {
		Key, Pt, Ct string
	}
	if err := json.Unmarshal(raw, &vecs); err != nil {
		t.Fatal(err)
	}
	if len(vecs) < 4 {
		t.Fatalf("мало векторов: %d", len(vecs))
	}
	for i, v := range vecs {
		key, _ := hex.DecodeString(v.Key)
		pt, _ := hex.DecodeString(v.Pt)
		ct, _ := hex.DecodeString(v.Ct)
		bf, err := NewBlowfish(key)
		if err != nil {
			t.Fatalf("vec%d: %v", i, err)
		}
		out := make([]byte, 8)
		bf.Encrypt(out, pt)
		if hex.EncodeToString(out) != v.Ct {
			t.Errorf("vec%d encrypt: got %x want %s", i, out, v.Ct)
		}
		bf.Decrypt(out, ct)
		if hex.EncodeToString(out) != v.Pt {
			t.Errorf("vec%d decrypt: got %x want %s", i, out, v.Pt)
		}
	}
}

// key1 (static) из LUT-формулы обязан совпасть с известным ключом AION-клиента.
func TestStaticKey(t *testing.T) {
	k := GenerateInitialKey(0x04bd)
	got := hex.EncodeToString(k[:])
	if got != StaticKeyHex {
		t.Fatalf("key1: got %s want %s", got, StaticKeyHex)
	}
	s := StaticKey()
	if hex.EncodeToString(s[:]) != StaticKeyHex {
		t.Fatal("StaticKey != GenerateInitialKey(0x04bd)")
	}
}

func TestKey2FromLUT(t *testing.T) {
	r := byte(0x37)
	k := Key2FromLUT(r)
	// самопроверка: dwords собираются из тех же слотов LUT
	var lut [256]uint32
	for i := range lut {
		lut[i] = binary.LittleEndian.Uint32(lutBin[i*4:])
	}
	var want [16]byte
	idx := [4]uint32{uint32(r), uint32(r) & 0xc1, uint32(r) & 0xf2, uint32(r) & 0x23}
	for i, j := range idx {
		binary.LittleEndian.PutUint32(want[i*4:], lut[j])
	}
	if k != want {
		t.Fatalf("key2: got %x want %x", k, want)
	}
}

// Wire-сборка 2110: fmt-строки из .data (0x42c780/84/88/dc).
func TestAssembleWire(t *testing.T) {
	got := Assemble("cdd", byte(0), uint32(0x11223344), uint32(0x7f000001))
	want := []byte{0x00, 0x44, 0x33, 0x22, 0x11, 0x01, 0x00, 0x00, 0x7f}
	if string(got) != string(want) {
		t.Fatalf("cdd: got %x want %x", got, want)
	}
	got = Assemble("cdh", byte(2), uint32(0x7d5214), uint16(0x0301))
	want = []byte{0x02, 0x14, 0x52, 0x7d, 0x00, 0x01, 0x03}
	if string(got) != string(want) {
		t.Fatalf("cdh: got %x want %x", got, want)
	}
	got = Assemble("cbdb", byte(0), []byte{1, 2, 3}, uint32(0x80000000), []byte{0xde})
	want = []byte{0x00, 1, 2, 3, 0x00, 0x00, 0x00, 0x80, 0xde}
	if string(got) != string(want) {
		t.Fatalf("cbdb: got %x want %x", got, want)
	}
	if Assemble("cd", byte(1), uint32(9))[0] != 0x01 {
		t.Fatal("cd: тип 1 не на месте")
	}
}

// Welcome: 194B, len-филд 0xC2 00, plaintext[0]=0x23, sid-байты на dword0 (capture-паритет).
func TestWelcome194(t *testing.T) {
	key1 := GenerateInitialKey(0x04bd)
	bf, err := NewBlowfish(key1[:])
	if err != nil {
		t.Fatal(err)
	}
	a := &WelcomeArgs{
		PlainByte: 0x23,
		SessionID: 0x7d5214,
		LoginType: 2, B0: 'e', B1: 'r', B2: 0,
	}
	w := BuildWelcome(a, bf)
	if len(w) != WelcomeLen {
		t.Fatalf("welcome: len %d want %d", len(w), WelcomeLen)
	}
	if w[0] != 0xC2 || w[1] != 0x00 {
		t.Fatalf("welcome: len-филд %02x %02x", w[0], w[1])
	}
	// ECB-dec(key1) вручную (обратный к EncryptPrimary без unscramble: dword0 = plaintext dword0)
	dec := make([]byte, 0, 192)
	blk := make([]byte, 8)
	for off := 2; off < 194; off += 8 {
		bf.Decrypt(blk, w[off:off+8])
		dec = append(dec, blk...)
	}
	if dec[0] != 0x23 {
		t.Fatalf("plaintext[0]=%02x want 0x23", dec[0])
	}
	if string(dec[1:4]) != "\x14\x52\x7d" {
		t.Fatalf("sid-байты dword0: %x", dec[1:4])
	}
	if len(dec) != 192 {
		t.Fatalf("ECB-часть: %d want 192", len(dec))
	}
}

func TestDecryptSecondaryRoundtrip(t *testing.T) {
	bf, _ := NewBlowfish([]byte("key2-test-key2-t"))
	msg := []byte("plaintext-16-bytes!!") // 20B
	enc := make([]byte, 24)
	copy(enc, msg)
	var x uint32
	for k := 0; k < 5; k++ {
		x ^= binary.LittleEndian.Uint32(enc[k*4:])
	}
	binary.LittleEndian.PutUint32(enc[20:], x)
	for off := 0; off < 24; off += 8 {
		bf.Encrypt(enc[off:off+8], enc[off:off+8])
	}
	got, err := DecryptSecondary(bf, enc)
	if err != nil {
		t.Fatalf("DecryptSecondary: %v", err)
	}
	if string(got) != string(msg) {
		t.Fatalf("roundtrip: %q", got)
	}
	enc[3] ^= 0xff
	if _, err := DecryptSecondary(bf, enc); err != ErrChecksum {
		t.Fatalf("ожидался ErrChecksum, got %v", err)
	}
}

// Скрамбл модуля — детерминирован и меняет буфер (byte-exact верифицируем capture'ом).
func TestScrambleModulus(t *testing.T) {
	var m [128]byte
	for i := range m {
		m[i] = byte(i)
	}
	before := m
	ScrambleModulus(&m)
	same := true
	for i := range m {
		if m[i] != before[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("scrambleModulus не изменил буфер")
	}
	ScrambleModulus(&m) // повтор — детерминированность (не идемпотентность)
	var m2 [128]byte
	for i := range m2 {
		m2[i] = byte(i)
	}
	ScrambleModulus(&m2)
	ScrambleModulus(&m2)
	for i := range m {
		if m[i] != m2[i] {
			t.Fatal("scrambleModulus недетерминирован")
		}
	}
}

// rsaEncrypt — «клиентская» публичная операция: c = x^e mod n (BE 32Б).
func rsaEncrypt(k *RSAKey, msg []byte) []byte {
	m := new(big.Int).SetBytes(msg)
	c := new(big.Int).Exp(m, big.NewInt(int64(k.Priv.PublicKey.E)), k.Priv.PublicKey.N)
	return c.FillBytes(make([]byte, 32))
}

func TestRSAPoolRoundtrip(t *testing.T) {
	pool, err := NewKeyPool()
	if err != nil {
		t.Fatal(err)
	}
	k := pool.Get()
	x := []byte("0123456789abcdef0123456789abcdef")
	m, err := k.DecryptBlock(rsaEncrypt(k, x))
	if err != nil {
		t.Fatal(err)
	}
	if string(m) != string(x) {
		t.Fatalf("rsa roundtrip: %q", m)
	}
	if pool.counter != 1 {
		t.Fatalf("pool counter: %d", pool.counter)
	}
}
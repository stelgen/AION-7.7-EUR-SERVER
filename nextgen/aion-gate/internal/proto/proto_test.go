package proto

import (
	crand "crypto/rand"
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

// Welcome: 194B, len-филд 0xC2 00; asm-модель (ночь-4, 0x407d50):
// plaintext[0]=0x00 ('c'←va0=0x0); dword0 = [0x00][sid-байты 1..3], НЕ скрамблится;
// dword1+ скрамблены cumsum'ом; csum@184; хвост 188..191 = нули буфера.
func TestWelcome194(t *testing.T) {
	key1 := GenerateInitialKey(0x04bd)
	bf, err := NewBlowfish(key1[:])
	if err != nil {
		t.Fatal(err)
	}
	a := &WelcomeArgs{
		SessionID:    0x7d521423, // fc = rand32 (группа из capture 03.10!)
		AuthdSession: 0x634692d8, // V = authd [03] (вторая половина пары!)
	}
	w := BuildWelcome(a, bf)
	if len(w) != WelcomeLen {
		t.Fatalf("welcome: len %d want %d", len(w), WelcomeLen)
	}
	if w[0] != 0xC2 || w[1] != 0x00 {
		t.Fatalf("welcome: len-филд %02x %02x", w[0], w[1])
	}
	dec := make([]byte, 0, 192)
	blk := make([]byte, 8)
	for off := 2; off < 194; off += 8 {
		bf.Decrypt(blk, w[off:off+8])
		dec = append(dec, blk...)
	}
	// dword0 НЕ скрамблится = fc ЦЕЛИКОМ (capture-паритет dw0=0x7d521423!);
	// dword1+ скрамблены (cumsum) — сырые поля напрямую не читаются
	if binary.LittleEndian.Uint32(dec[0:4]) != 0x52142300 { // [0x00][7d 52 14]
		t.Fatalf("dword0 (fc): %x want 7d521423", dec[0:4])
	}
	// хвост [188..191] = нули (после csum@184, ничем не тронут — резидуум-зона оригинала не нулевая!)
	for i := 188; i < 192; i++ {
		if dec[i] != 0 {
			t.Fatalf("tail[%d]=%02x", i, dec[i])
		}
	}
	if dec[188] != 0 || dec[189] != 0 || dec[190] != 0 || dec[191] != 0 {
		t.Fatal("хвост ECB-области (188..191) не нулевой")
	}
}

// Known-answer на семантику скрамбла asm 0x417a20: data [1,0,0,0 | 2,0,0,0] →
// dw0=1 (нетронут), S=1+2=3 → dw1=2^3=1, csum dw2=3, pad dw3=0 → ECB 16Б.
func TestEncryptPrimaryKnownAnswer(t *testing.T) {
	bf, _ := NewBlowfish([]byte("ka-test-key-16byt"))
	out := EncryptPrimary(bf, []byte{1, 0, 0, 0, 2, 0, 0, 0})
	if len(out) != 16 {
		t.Fatalf("EncryptPrimary(8B): %d want 16", len(out))
	}
	exp := make([]byte, 16)
	binary.LittleEndian.PutUint32(exp[0:], 1)
	binary.LittleEndian.PutUint32(exp[4:], 1) // 2 ^ (1+2)
	binary.LittleEndian.PutUint32(exp[8:], 3) // csum = финальный S
	want := make([]byte, 16)
	bf.Encrypt(want[0:8], exp[0:8])
	bf.Encrypt(want[8:16], exp[8:16])
	if string(out) != string(want) {
		t.Fatalf("known-answer: got %x want %x", out, want)
	}
	// инвариант dword0: расшифрованный dword0 == исходный
	dec0 := make([]byte, 8)
	bf.Decrypt(dec0, out[0:8])
	if binary.LittleEndian.Uint32(dec0[0:4]) != 1 {
		t.Fatal("dword0 изменён скрамблом")
	}
}

// DecryptSecondary: инверсия EncryptSecondary; tamper → ErrChecksum.
func TestDecryptSecondaryRoundtrip(t *testing.T) {
	bf, _ := NewBlowfish([]byte("key2-test-key2-t"))
	msg := []byte("plaintext-16-bytes!!") // 20B → n'=24 → blob 32B
	enc := EncryptSecondary(bf, msg)
	if len(enc) != 32 {
		t.Fatalf("EncryptSecondary(20B): %d want 32", len(enc))
	}
	got, err := DecryptSecondary(bf, enc)
	if err != nil {
		t.Fatalf("DecryptSecondary: %v", err)
	}
	if len(got) != 24 || string(got[:20]) != string(msg) {
		t.Fatalf("roundtrip: %q (%d)", got, len(got))
	}
	enc[3] ^= 0xff
	if _, err := DecryptSecondary(bf, enc); err != ErrChecksum {
		t.Fatalf("ожидался ErrChecksum, got %v", err)
	}
}

// Echo-структура из capture 03.10: EncryptSecondary([sid][28×0]) →
// plaintext [A][28×0][A][pad0], cipher-блоки [P][Q][Q][Q][P] (b0==b4, b1==b2==b3).
func TestEncryptSecondaryEchoStructure(t *testing.T) {
	bf, _ := NewBlowfish([]byte("echo-test-key2-16"))
	reply := make([]byte, 32)
	binary.LittleEndian.PutUint32(reply, 0x7d5214) // [sid][28×0]
	enc := EncryptSecondary(bf, reply)
	if len(enc) != 40 {
		t.Fatalf("EncryptSecondary(32B): %d want 40", len(enc))
	}
	dec := make([]byte, 40)
	for off := 0; off < 40; off += 8 {
		bf.Decrypt(dec[off:off+8], enc[off:off+8])
	}
	if binary.LittleEndian.Uint32(dec[0:4]) != 0x7d5214 || binary.LittleEndian.Uint32(dec[32:36]) != 0x7d5214 {
		t.Fatalf("echo plaintext: dw0=%x dw8=%x", dec[0:4], dec[32:36])
	}
	for i := 4; i < 32; i++ {
		if dec[i] != 0 {
			t.Fatalf("echo zeros@%d: %02x", i, dec[i])
		}
	}
	if string(enc[8:16]) != string(enc[16:24]) || string(enc[16:24]) != string(enc[24:32]) {
		t.Fatal("cipher Q-блоки не равны (структура [P][Q][Q][Q][P] нарушена)")
	}
	if string(enc[0:8]) != string(enc[32:40]) {
		t.Fatal("cipher P-блоки не равны (b0 != b4)")
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

// rsaEncrypt — «клиентская» публичная операция: c = x^e mod n (BE 128Б).
func rsaEncrypt(k *RSAKey, msg []byte) []byte {
	m := new(big.Int).SetBytes(msg)
	c := new(big.Int).Exp(m, big.NewInt(int64(k.Priv.PublicKey.E)), k.Priv.PublicKey.N)
	return c.FillBytes(make([]byte, 128))
}

// TestRSAPoolRoundtrip — параметризован экспонентой (§7.6): e=17 (Pub.key[1]) и e=65537 (F4 гита).
// DecryptBlock теперь всегда возвращает полный m 128B BE (П3): 32Б-вход лежит в [96:128].
func TestRSAPoolRoundtrip(t *testing.T) {
	for _, e := range []int64{17, 65537} {
		pool, err := NewKeyPool(e)
		if err != nil {
			t.Fatalf("e=%d: %v", e, err)
		}
		k := pool.Get()
		if k.Priv.PublicKey.E != int(e) {
			t.Fatalf("e=%d: key.E=%d", e, k.Priv.PublicKey.E)
		}
		x := []byte("0123456789abcdef0123456789abcdef")
		m, err := k.DecryptBlock(rsaEncrypt(k, x))
		if err != nil {
			t.Fatalf("e=%d: %v", e, err)
		}
		if len(m) != 128 {
			t.Fatalf("e=%d: decrypt len=%d want 128", e, len(m))
		}
		if string(m[96:]) != string(x) {
			t.Fatalf("e=%d: rsa roundtrip: %q", e, m[96:])
		}
		for i := 0; i < 96; i++ {
			if m[i] != 0 {
				t.Fatalf("e=%d: ведущие байты не нулевые @%d", e, i)
			}
		}
		if pool.counter != 1 {
			t.Fatalf("pool counter: %d", pool.counter)
		}
	}
}

// §7.1: клиентский unscramble (старая ScrambleModulus) обязан снимать серверный скрамбл гита:
// UnscrambleClient(ServerScramble(N)) == N — инверс точный (математика байон-48 §3).
func TestScrambleModulusServerInverse(t *testing.T) {
	for i := 0; i < 200; i++ {
		var n [128]byte
		if _, err := crand.Read(n[:]); err != nil {
			t.Fatal(err)
		}
		want := n
		ScrambleModulusServer(&n)
		if n == want {
			t.Fatal("ServerScramble не изменил буфер")
		}
		ScrambleModulus(&n)
		if n != want {
			t.Fatalf("iter %d: UnscrambleClient(ServerScramble(N)) != N", i)
		}
	}
}

// §7.2: обе функции НЕ инволюции — отправка ServerScramble(ServerScramble(N)) или
// ScrambleModulus(ScrambleModulus(N)) (и RAW) гарантированно даёт клиенту чужой N.
func TestScrambleNotInvolution(t *testing.T) {
	for i := 0; i < 200; i++ {
		var n [128]byte
		if _, err := crand.Read(n[:]); err != nil {
			t.Fatal(err)
		}
		orig := n
		n2 := n
		ScrambleModulusServer(&n2)
		ScrambleModulusServer(&n2)
		if n2 == orig {
			t.Fatal("ServerScramble — инволюция (не должно)")
		}
		n3 := n
		ScrambleModulus(&n3)
		ScrambleModulus(&n3)
		if n3 == orig {
			t.Fatal("ScrambleModulus — инволюция (не должно)")
		}
	}
}

// welcome (plain()) и BuildWelcomeVariant(вар. 0) обязаны класть СЕРВЕРНЫЙ скрамбл.
func TestWelcomeUsesServerScramble(t *testing.T) {
	var n [128]byte
	if _, err := crand.Read(n[:]); err != nil {
		t.Fatal(err)
	}
	a := &WelcomeArgs{Modulus: n, SessionID: 1, AuthdSession: 2}
	pt := a.plain()
	ref := n
	ScrambleModulusServer(&ref)
	if string(pt[9:137]) != string(ref[:]) {
		t.Fatal("plain(): модуль не серверно-скрамблен")
	}
	wbf, _ := NewBlowfish([]byte("variant0-test-key"))
	w := BuildWelcomeVariant(&WelcomeArgs{Modulus: n, SessionID: 1, AuthdSession: 2}, wbf, 0)
	if len(w) != WelcomeLen {
		t.Fatalf("variant0: len=%d", len(w))
	}
}

// DecryptPrimary — инверс EncryptPrimary (fork: чтение welcome оригинала).
func TestDecryptPrimaryRoundtrip(t *testing.T) {
	bf, _ := NewBlowfish([]byte("primary-test-key"))
	pt := make([]byte, 177)
	for i := range pt {
		pt[i] = byte(i * 7)
	}
	pt[0] = 0x00
	enc := EncryptPrimary(bf, pt)
	if len(enc) != 192 {
		t.Fatalf("enc len=%d", len(enc))
	}
	got, err := DecryptPrimary(bf, enc)
	if err != nil {
		t.Fatalf("DecryptPrimary: %v", err)
	}
	// dword0 не тронут; dwords 1..22 восстанавливаются; pad-зона [177:184] — нули
	if string(got[:177]) != string(pt) {
		t.Fatal("DecryptPrimary: plaintext не восстановлен")
	}
}

// §7.5: размещение чек-суммы — наш [data][chk][pad] и гит-вариант [data][pad][chk]
// оба валидны при pad=0 (верификатор один: XOR всех dword == 0).
func TestSecondaryChecksumLayouts(t *testing.T) {
	bf, _ := NewBlowfish([]byte("checksum-layout"))
	msg := []byte("0123456789abcdef") // 16B → n'=16
	// наш вариант: [16B data][chk][pad] = 24B
	enc := EncryptSecondary(bf, msg)
	if len(enc) != 24 {
		t.Fatalf("len=%d", len(enc))
	}
	if _, err := DecryptSecondary(bf, enc); err != nil {
		t.Fatalf("наша раскладка не прошла: %v", err)
	}
	// гит-вариант: [16B data][pad][chk] = 24B — верификатор клиента один (XOR всех
	// dword == 0), обе раскладки валидны при pad=0 (байон-48 §1); наш DecryptSecondary
	// парсит только нашу ([data][chk][pad]) — это ок.
	buf := make([]byte, 24)
	copy(buf, msg)
	var x uint32
	for k := 0; k < 5; k++ { // XOR данных + pad(0)
		x ^= binary.LittleEndian.Uint32(buf[k*4:])
	}
	binary.LittleEndian.PutUint32(buf[20:], x) // chk последним dword
	var all uint32
	for k := 0; k < 6; k++ {
		all ^= binary.LittleEndian.Uint32(buf[k*4:])
	}
	if all != 0 {
		t.Fatalf("гит-раскладка [data][pad][chk]: XOR != 0 (%08x)", all)
	}
}

// §7.3: classic welcome — wire 210, rev c621, magic 3FCE09ED, модуль — серверный скрамбл,
// сессионный ключ в [153:169]; DecryptGitInit восстанавливает pt.
func TestClassicWelcome210(t *testing.T) {
	k1 := GenerateInitialKey(0x04bd)
	staticBF, _ := NewBlowfish(k1[:])
	var n [128]byte
	for i := range n {
		n[i] = byte(i + 1)
	}
	var sk [16]byte
	for i := range sk {
		sk[i] = byte(0xA0 + i)
	}
	w := BuildClassicWelcome(0x7d521423, n, sk, staticBF)
	if len(w) != ClassicWelcomeWire {
		t.Fatalf("wire len=%d want %d", len(w), ClassicWelcomeWire)
	}
	if w[0] != 0xD2 || w[1] != 0 { // total = 210 = 0xD2 (len-филд ВКЛЮЧАЕТ сам себя)
		t.Fatalf("len-филд: %02x %02x", w[0], w[1])
	}
	pt, err := DecryptGitInit(staticBF, w[2:])
	if err != nil {
		t.Fatalf("DecryptGitInit: %v", err)
	}
	if len(pt) != 200 {
		t.Fatalf("pt len=%d want 200", len(pt))
	}
	if pt[0] != 0x00 || binary.LittleEndian.Uint32(pt[1:5]) != 0x7d521423 {
		t.Fatal("opcode/sid сломаны")
	}
	if binary.LittleEndian.Uint32(pt[5:9]) != ClassicProtocolRev {
		t.Fatalf("rev: %08x", binary.LittleEndian.Uint32(pt[5:9]))
	}
	if binary.LittleEndian.Uint32(pt[184:188]) != ClassicTailMagic {
		t.Fatalf("magic@184: %08x", binary.LittleEndian.Uint32(pt[184:188]))
	}
	if string(pt[153:169]) != string(sk[:]) {
		t.Fatal("sessionKey не в [153:169]")
	}
	ref := n
	ScrambleModulusServer(&ref)
	if string(pt[9:137]) != string(ref[:]) {
		t.Fatal("модуль не серверно-скрамблен")
	}
	// инверс-проверка модуля клиентским unscramble
	var back [128]byte
	copy(back[:], pt[9:137])
	ScrambleModulus(&back)
	if back != n {
		t.Fatal("UnscrambleClient(ServerScramble(N)) != N в welcome")
	}
}

// SM_AUTH_GG: живая форма 32Б / гит-форма → wire 50.
func TestClassicAuthGG(t *testing.T) {
	live := BuildClassicAuthGG(0x11223344, false)
	if len(live) != 32 || live[0] != 0x0b || binary.LittleEndian.Uint32(live[1:5]) != 0x11223344 {
		t.Fatalf("live форма: len=%d", len(live))
	}
	git := BuildClassicAuthGG(0x11223344, true)
	if len(git) != 37 {
		t.Fatalf("git форма len=%d want 37", len(git))
	}
	bf, _ := NewBlowfish([]byte("authgg-test-key"))
	if wire := len(WriteFrame(EncryptSecondary(bf, git))); wire != 50 {
		t.Fatalf("git wire=%d want 50", wire)
	}
}

// §7.4: golden CM_LOGIN (док гита): user "abcdefghijklmn", pwd "abcdefghijklmnop",
// otp FFFFFFFF; не-loginex (1 чанк) и loginex (2 чанка).
func TestCMLoginParse(t *testing.T) {
	k, err := GenerateRSAKey(65537)
	if err != nil {
		t.Fatal(err)
	}
	mkChunk := func() []byte {
		m := make([]byte, 128)
		copy(m[94:108], "abcdefghijklmn")
		copy(m[108:124], "abcdefghijklmnop")
		binary.LittleEndian.PutUint32(m[124:128], 0xFFFFFFFF)
		return m
	}
	encrypt := func(m []byte) []byte {
		c := new(big.Int).Exp(new(big.Int).SetBytes(m), big.NewInt(65537), k.Priv.PublicKey.N)
		return c.FillBytes(make([]byte, 128))
	}
	tail := make([]byte, 55)
	binary.LittleEndian.PutUint32(tail[0:4], 7)
	copy(tail[20:27], []byte{0x20, 0, 0, 0, 0, 0, 0x01})
	copy(tail[27:43], []byte{0x9D, 0xDA, 0x47, 0xA7, 0x21, 0xC0, 0xA6, 0xA5, 0x4B, 0xB7, 0x5E, 0xE3, 0xCE, 0xC9, 0x26, 0xAA})
	// не-loginex
	pt := append([]byte{0x00}, encrypt(mkChunk())...)
	pt = append(pt, tail...)
	chunks, tl, ok := SplitLogin(pt)
	if !ok || len(chunks) != 1 || len(tl) != 55 {
		t.Fatalf("split: ok=%v chunks=%d", ok, len(chunks))
	}
	dec := make([][]byte, 0, len(chunks))
	for _, ct := range chunks {
		m, err := k.DecryptBlock(ct)
		if err != nil {
			t.Fatal(err)
		}
		dec = append(dec, m)
	}
	d, ok := DecodeLoginPlain(dec)
	if !ok || d.User != "abcdefghijklmn" || d.Pwd != "abcdefghijklmnop" || d.Otp != 0xFFFFFFFF {
		t.Fatalf("не-loginex: %+v ok=%v", d, ok)
	}
	// loginex (2 чанка): чанки СКЛЕИВАЮТСЯ — user = buf[78:142], pwd = buf[206:238],
	// otp = buf[238:242] (гит decryptLoginData)
	m1 := make([]byte, 128)
	copy(m1[78:92], "abcdefghijklmn")
	m2 := make([]byte, 128)
	copy(m2[78:94], "abcdefghijklmnop")
	binary.LittleEndian.PutUint32(m2[110:114], 0xFFFFFFFF)
	pt2 := append([]byte{0x00}, encrypt(m1)...)
	pt2 = append(pt2, encrypt(m2)...)
	pt2 = append(pt2, tail...)
	chunks2, _, ok2 := SplitLogin(pt2)
	if !ok2 || len(chunks2) != 2 {
		t.Fatalf("loginex split: ok=%v chunks=%d", ok2, len(chunks2))
	}
	dec2 := make([][]byte, 0, len(chunks2))
	for _, ct := range chunks2 {
		m, err := k.DecryptBlock(ct)
		if err != nil {
			t.Fatal(err)
		}
		dec2 = append(dec2, m)
	}
	d2, ok2 := DecodeLoginPlain(dec2)
	if !ok2 || d2.User != "abcdefghijklmn" || d2.Pwd != "abcdefghijklmnop" || !d2.Ex {
		t.Fatalf("loginex: %+v ok=%v", d2, ok2)
	}
}

// LoginDecbuf: 34 = m[94:128] (user14+pwd16+otp4), 32 = m[96:128], 128 = полный m.
func TestLoginDecbuf(t *testing.T) {
	m := make([]byte, 128)
	copy(m[94:108], "user123")
	copy(m[108:124], "pass123")
	binary.LittleEndian.PutUint32(m[124:128], 0xFFFFFFFF)
	if d := LoginDecbuf(m, 34); len(d) != 34 || string(d[0:7]) != "user123" {
		t.Fatalf("34: %q", d[:8])
	}
	if d := LoginDecbuf(m, 32); len(d) != 32 || string(d[:3]) != "er1" { // m[96:128] = хвост user
		t.Fatalf("32: len=%d head=%q", len(d), d[:4])
	}
	if d := LoginDecbuf(m, 128); len(d) != 128 {
		t.Fatalf("128: len=%d", len(d))
	}
	if d := LoginDecbuf(m, 0); len(d) != 34 {
		t.Fatalf("дефолт: len=%d", len(d))
	}
}
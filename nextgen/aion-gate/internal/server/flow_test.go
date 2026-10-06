package server

// Тесты Фазы-1 закрытия живого флоу (docs/mobius-77-flow-review-20261006 §Тесты):
//  1. TestLoginRelayTailLoginex   — К-1/P0-1: loginex blob несёт tail ИЗ ХВОСТА, dword НЕ из ct-зоны
//  2. TestStateMachineTransitions — К-2/P0-2: CONNECTED→AUTHED_GG→AUTHED_LOGIN, unknown = лог (НЕ cc45)
//  5. TestUsernameNormalize       — К-5/P1-5: trim+toLowerCase перед BuildLoginDecbuf
//  6. TestSessionKeyRnd           — К-6/P2-6: эмуляция play-ok даёт РАЗНЫЕ playOk1/2 (crypto/rand)
// Golden-тесты welcome/echo/фреймов (server_test.go) не тронуты.

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"aion-gate/internal/authdclient"
	"aion-gate/internal/config"
	"aion-gate/internal/proto"
)

// newFlowSrvRaw — сервер (БЕЗ authd — путь эмуляции 26b) + клиент с welcome.
func newFlowSrvRaw(t *testing.T) (*Server, net.Conn, *Session) {
	t.Helper()
	srv, err := New(config.Gate{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go srv.Serve(ln)
	cl, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cl.Close() })
	wl := make([]byte, 2)
	if _, err := asRead(cl, wl); err != nil {
		t.Fatal(err)
	}
	ecb := make([]byte, 192)
	if _, err := asRead(cl, ecb); err != nil {
		t.Fatal(err)
	}
	sess := srv.SessionByIndex(0)
	if sess == nil {
		t.Fatal("нет сессии")
	}
	return srv, cl, sess
}

// newFlowSrv — сервер + fake-authd + клиент с welcome; возвращает сессию.
func newFlowSrv(t *testing.T) (*Server, *fakeAuthd, net.Conn, *Session) {
	t.Helper()
	srv, cl, sess := newFlowSrvRaw(t)
	ac, as := net.Pipe()
	f := runFakeAuthd(t, as)
	srv.SetAuthd(authdclient.DialConn(ac, srv.AuthdHandler()))
	return srv, f, cl, sess
}

// rsaEnc — RSA-чанк из m (pub экспонента сессии).
func rsaEnc(sess *Session, m []byte) []byte {
	pub := &sess.RSA.Priv.PublicKey
	return new(big.Int).Exp(new(big.Int).SetBytes(m), big.NewInt(int64(pub.E)), pub.N).FillBytes(make([]byte, 128))
}

// sendPt — клиент шлёт plaintext зашифрованным key2.
func sendPt(cl net.Conn, sess *Session, pt []byte) {
	_, _ = cl.Write(proto.WriteFrame(proto.EncryptSecondary(sess.BF2, pt)))
}

// authgg07 — CM_AUTH_GG эталонной формы [07][sid][19×0] → 42b reply + State=AUTHED_GG.
func authgg07(t *testing.T, cl net.Conn, sess *Session) {
	t.Helper()
	pt := make([]byte, 24)
	pt[0] = 0x07
	binary.LittleEndian.PutUint32(pt[1:5], sess.ID)
	sendPt(cl, sess, pt)
	rl := make([]byte, 2)
	cl.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := asRead(cl, rl); err != nil {
		t.Fatalf("authgg reply: %v", err)
	}
	cl.SetReadDeadline(time.Time{})
	if binary.LittleEndian.Uint16(rl) != 42 {
		t.Fatalf("authgg reply len: %d want 42", binary.LittleEndian.Uint16(rl))
	}
	if sess.State != stAuthedGG {
		t.Fatalf("state после AUTH_GG = %s want AUTHED_GG", stateName(sess.State))
	}
}

// Тест 1 (К-1/P0-1): loginex pt-304 (op+2 ct-чанка+tail 47) → blob содержит tail ИЗ
// ХВОСТА, dword НЕ из ct-зоны (pt[148:152] — внутри ct при k≥2).
func TestLoginRelayTailLoginex(t *testing.T) {
	srv, f, cl, sess := newFlowSrv(t)
	_ = srv
	authgg07(t, cl, sess)

	// m1/m2 (loginex-раскладка 06.10): user = buf[78:142], pwd = buf[206:238], otp = buf[238:242].
	user := "loginexuser" + strings.Repeat("a", 39) // 50Б @buf[78:128], далее нули
	pwd := "pwdloginex"
	m1 := make([]byte, 128)
	m2 := make([]byte, 128)
	for i := 1; i < 78; i++ { // известный ct-филлер (top-байт 0 — m < N)
		m1[i] = 0xEE
		m2[i] = 0xEE
	}
	copy(m1[78:128], user)
	copy(m2[78:110], pwd)
	binary.LittleEndian.PutUint32(m2[110:114], 0xFFFFFFFF) // otp

	tail := make([]byte, 47) // live-структура: [sid][нули][0x20][7×0][magic1][magic2][dword=0]
	binary.LittleEndian.PutUint32(tail[0:4], sess.ID)
	tail[19] = 0x20
	copy(tail[27:35], []byte{0x68, 0xff, 0xda, 0xb3, 0xe2, 0xfd, 0xa8, 0x92})
	copy(tail[35:43], []byte{0x2d, 0x9c, 0xc7, 0xba, 0xa8, 0x7e, 0x0d, 0x49})

	pt := make([]byte, 0, 304)
	pt = append(pt, 0x00) // live 7.7 EU клиент шлёт CM_LOGIN op=0x00 (форк-дамп 00:43)
	pt = append(pt, rsaEnc(sess, m1)...)
	pt = append(pt, rsaEnc(sess, m2)...)
	pt = append(pt, tail...)
	if len(pt) != 304 {
		t.Fatalf("pt=%d want 304", len(pt))
	}
	sendPt(cl, sess, pt)
	waitFor(t, func() bool { f.mu.Lock(); defer f.mu.Unlock(); return len(f.packets) == 1 })
	f.mu.Lock()
	blob := f.packets[0]
	f.mu.Unlock()

	decbuf := proto.BuildLoginDecbuf(user, pwd, 0xFFFFFFFF, 34)
	wantDword := uint32(0) // хвост ровно [..][0x20-блок][magic][00000000] → dword после блока = 0
	wantBlob := proto.Assemble("cbdb", byte(0), decbuf, wantDword, tail)
	if !bytes.Equal(blob, wantBlob) {
		t.Fatalf("loginex blob: got len=%d want len=%d\ngot  %x\nwant %x", len(blob), len(wantBlob), blob, wantBlob)
	}
	// dword НЕ из ct-зоны: старый код брал pt[148:152] = ct1[147:151] (филлер 0xEE → не 0)
	ctZone := binary.LittleEndian.Uint32(pt[148:152])
	if ctZone == wantDword {
		t.Fatalf("неинформативно: ct-зона pt[148:152]=%08x совпала с dword", ctZone)
	}
	// tail в blob — ИЗ ХВОСТА (последние 47Б blob == split-tail)
	if !bytes.Equal(blob[len(blob)-len(tail):], tail) {
		t.Fatal("blob: tail не из хвоста SplitLogin")
	}
}

// Тест 2 (К-2/P0-2): переходы состояний и unknown-политика.
func TestStateMachineTransitions(t *testing.T) {
	srv, f, cl, sess := newFlowSrv(t)
	_ = srv
	if sess.State != stConnected {
		t.Fatalf("старт: state=%s want CONNECTED", stateName(sess.State))
	}

	// unknown op (0x99, короткий фрейм → вне length-эвристик) = лог + raw relay, НЕ cc45, соединение живо
	sendPt(cl, sess, append([]byte{0x99}, make([]byte, 7)...)) // pt 8Б → ECB 16
	waitFor(t, func() bool { f.mu.Lock(); defer f.mu.Unlock(); return len(f.packets) == 1 })
	f.mu.Lock()
	if !bytes.Equal(f.packets[0], append([]byte{0x99}, make([]byte, 7)...)) {
		t.Fatalf("unknown: relayed %x", f.packets[0])
	}
	f.mu.Unlock()
	cl.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	if _, err := asRead(cl, make([]byte, 8)); err == nil {
		t.Fatal("unknown: пришёл ответ (cc45?) — не должен")
	}
	cl.SetReadDeadline(time.Time{})
	if sess.State != stConnected {
		t.Fatalf("unknown сменил state: %s", stateName(sess.State))
	}

	// CONNECTED 0x07 → AUTHED_GG
	authgg07(t, cl, sess)

	// AUTHED_GG: op=0x05 (24Б pt) — НЕ login → лог + raw relay, state не меняется
	pt05 := make([]byte, 24)
	pt05[0] = 0x05
	sendPt(cl, sess, pt05)
	waitFor(t, func() bool { f.mu.Lock(); defer f.mu.Unlock(); return len(f.packets) == 2 })
	if sess.State != stAuthedGG {
		t.Fatalf("op=0x05 в AUTHED_GG сменил state: %s", stateName(sess.State))
	}

	// AUTHED_GG 0x0B → login → AUTHED_LOGIN
	m := make([]byte, 128)
	copy(m[94:106], "statemachine")
	copy(m[108:114], "pw1234")
	binary.LittleEndian.PutUint32(m[124:128], 0xFFFFFFFF)
	tail := make([]byte, 55)
	binary.LittleEndian.PutUint32(tail[0:4], sess.ID)
	tail[20] = 0x20
	pt := append([]byte{0x00}, rsaEnc(sess, m)...) // live 7.7 EU: CM_LOGIN op=0x00 в AUTHED_GG
	pt = append(pt, tail...)
	sendPt(cl, sess, pt)
	waitFor(t, func() bool { f.mu.Lock(); defer f.mu.Unlock(); return len(f.packets) == 3 })
	if sess.State != stAuthedLogin {
		t.Fatalf("после login state=%s want AUTHED_LOGIN", stateName(sess.State))
	}
	f.mu.Lock()
	blob := f.packets[2]
	f.mu.Unlock()
	if len(blob) != 1+34+4+32 || blob[0] != 0x00 {
		t.Fatalf("login blob: len=%d b0=%02x (want 71/00)", len(blob), blob[0])
	}

	// AUTHED_LOGIN: unknown op (короткий) → лог + raw relay, соединение живо
	sendPt(cl, sess, append([]byte{0x77}, make([]byte, 7)...))
	waitFor(t, func() bool { f.mu.Lock(); defer f.mu.Unlock(); return len(f.packets) == 4 })
}

// Тест 5 (К-5/P1-5): " SteLGeN " → "stelgen" в decbuf перед релеем.
func TestUsernameNormalize(t *testing.T) {
	srv, f, cl, sess := newFlowSrv(t)
	_ = srv
	authgg07(t, cl, sess)

	m := make([]byte, 128)
	copy(m[94:103], " SteLGeN ") // 9 символов + нули
	copy(m[108:110], "pw")
	binary.LittleEndian.PutUint32(m[124:128], 0xFFFFFFFF)
	tail := make([]byte, 55)
	binary.LittleEndian.PutUint32(tail[0:4], sess.ID)
	tail[20] = 0x20
	pt := append([]byte{0x00}, rsaEnc(sess, m)...) // live 7.7 EU: CM_LOGIN op=0x00 в AUTHED_GG
	pt = append(pt, tail...)
	sendPt(cl, sess, pt)
	waitFor(t, func() bool { f.mu.Lock(); defer f.mu.Unlock(); return len(f.packets) == 1 })
	f.mu.Lock()
	blob := f.packets[0]
	f.mu.Unlock()

	decbuf := proto.BuildLoginDecbuf("stelgen", "pw", 0xFFFFFFFF, 34) // нормализованный
	dword := binary.LittleEndian.Uint32(tail[19:23])                  // asm-позиция k=1: data+148 = tail+19
	wantBlob := proto.Assemble("cbdb", byte(0), decbuf, dword, tail[23:])
	if !bytes.Equal(blob, wantBlob) {
		t.Fatalf("normalize blob:\ngot  %x\nwant %x", blob, wantBlob)
	}
	if !bytes.Equal(blob[1:15], append([]byte("stelgen"), make([]byte, 7)...)) {
		t.Fatalf("decbuf user: %q (want stelgen)", blob[1:15])
	}
}

// Тест 6 (К-6/P2-6): два логина в эмуляции дают разные playOk1/playOk2 (crypto/rand, не хардкод).
// Эмуляция 26b срабатывает только БЕЗ authd (с authd — raw relay, как ориг).
func TestSessionKeyRnd(t *testing.T) {
	srv, cl, sess := newFlowSrvRaw(t)
	_ = srv
	sess.State = stAuthedLogin // эмуляция: состояние после логина

	read26 := func() (pk1, pk2 uint32) {
		sendPt(cl, sess, append([]byte{0x02}, make([]byte, 23)...)) // CM_PLAY-форма 24Б pt
		rl := make([]byte, 2)
		cl.SetReadDeadline(time.Now().Add(3 * time.Second))
		if _, err := asRead(cl, rl); err != nil {
			t.Fatalf("26b reply: %v", err)
		}
		cl.SetReadDeadline(time.Time{})
		if binary.LittleEndian.Uint16(rl) != 26 {
			t.Fatalf("26b reply len: %d want 26", binary.LittleEndian.Uint16(rl))
		}
		body := make([]byte, 24)
		if _, err := asRead(cl, body); err != nil {
			t.Fatal(err)
		}
		pt := make([]byte, 24)
		for off := 0; off < 24; off += 8 {
			sess.BF2.Decrypt(pt[off:off+8], body[off:off+8])
		}
		if pt[0] != 0x07 {
			t.Fatalf("play-ok op=%02x want 07", pt[0])
		}
		return binary.LittleEndian.Uint32(pt[1:5]), binary.LittleEndian.Uint32(pt[5:9])
	}
	pk1a, pk2a := read26()
	pk1b, pk2b := read26()
	if pk1a == 1 || pk2a == 1010 {
		t.Fatalf("хардкод вернулся: pk1=%d pk2=%d", pk1a, pk2a)
	}
	if pk1a == pk1b && pk2a == pk2b {
		t.Fatalf("playOk повторился: %d/%d == %d/%d", pk1a, pk2a, pk1b, pk2b)
	}
	if sess.PlayOk1 != pk1b || sess.PlayOk2 != pk2b {
		t.Fatalf("SessionKey не сохранён: sess=%08x/%08x wire=%08x/%08x", sess.PlayOk1, sess.PlayOk2, pk1b, pk2b)
	}
}
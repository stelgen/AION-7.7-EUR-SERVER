// Тесты T1 «фейлы как эталон» (сорс: reference/Mobius_AionEmu 7.7, Aion-Lightning):
//  1. TestAuthFailFrames     — SM_LOGIN_FAIL(0x01)/SM_PLAY_FAIL(0x06): pt [op][D messageId]
//     → wire 18 (live-форма 06.10 11:08), roundtrip DecryptSecondary;
//  2. TestLoginTimeoutFail   — authd молчит после blob → 18Б [01][D loginFailCode] за loginTimeoutSec;
//  3. TestPlayTimeoutFail    — релей [05] без ответа authd → 18Б [06][D playFailCode];
//  4. TestReloginOnlineCache — relogin моложе onlineTtlSec → немедленный [01][D 7],
//     blob всё равно релеить (kick-семантика AccountController.login);
//  5. TestAuthFailCancel     — type=3 пришёл → таймаут НЕ стреляет, онлайн-кэш обновился.
package server

import (
	"encoding/binary"
	"net"
	"testing"
	"time"

	"aion-gate/internal/authdclient"
	"aion-gate/internal/config"
	"aion-gate/internal/proto"
)

// newFailSrv — сервер с конфигом + fake-authd; возвращает листенер (для второго клиента).
func newFailSrv(t *testing.T, cfg config.Gate) (*Server, *fakeAuthd, net.Listener) {
	t.Helper()
	srv, err := New(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go srv.Serve(ln)
	ac, as := net.Pipe()
	f := runFakeAuthd(t, as)
	srv.SetAuthd(authdclient.DialConn(ac, srv.AuthdHandler()))
	return srv, f, ln
}

// dialClient — новый клиент: dial + чтение welcome 194.
func dialClient(t *testing.T, ln net.Listener) net.Conn {
	t.Helper()
	cl, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cl.Close() })
	wl := make([]byte, 2)
	cl.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := asRead(cl, wl); err != nil {
		t.Fatal(err)
	}
	ecb := make([]byte, 192)
	if _, err := asRead(cl, ecb); err != nil {
		t.Fatal(err)
	}
	cl.SetReadDeadline(time.Time{})
	return cl
}

// doLogin — AUTH_GG + CM_LOGIN (op 0x0B, k=1, user@94/pwd@108/otp@124) как в TestE2ESkeleton.
func doLogin(t *testing.T, cl net.Conn, sess *Session, user string) {
	t.Helper()
	authgg07(t, cl, sess)
	m := make([]byte, 128)
	copy(m[94:94+len(user)], user)
	copy(m[108:115], "pw12345")
	binary.LittleEndian.PutUint32(m[124:128], 0xFFFFFFFF)
	data := make([]byte, 0, 184)
	data = append(data, 0x0B)
	data = append(data, rsaEnc(sess, m)...)
	data = append(data, make([]byte, 55)...)
	sendPt(cl, sess, data)
}

// pushType3 — authd→gate [02]-фрейм type=3 (login-ok, payload 52Б как в capture 74b).
func pushType3(t *testing.T, f *fakeAuthd, sess *Session) {
	t.Helper()
	payload := make([]byte, 52)
	binary.LittleEndian.PutUint32(payload[0:4], 42) // accId — условный
	push := make([]byte, 0, 7+52)
	push = append(push, 0x02)
	push = append(push, byte(sess.ID), byte(sess.ID>>8), byte(sess.ID>>16), byte(sess.ID>>24))
	push = append(push, byte(len(payload)+3), 0x00, 0x03)
	push = append(push, payload...)
	if _, err := f.conn.Write(push); err != nil {
		t.Fatal(err)
	}
}

// readFrame — читает один фрейм с дедлайном; возвращает РАСШИФРОВАННЫЙ key2-payload.
func readFrame(t *testing.T, cl net.Conn, sess *Session, timeout time.Duration) []byte {
	t.Helper()
	cl.SetReadDeadline(time.Now().Add(timeout))
	l := make([]byte, 2)
	if _, err := asRead(cl, l); err != nil {
		t.Fatalf("read frame len: %v", err)
	}
	n := int(binary.LittleEndian.Uint16(l))
	body := make([]byte, n-2)
	if _, err := asRead(cl, body); err != nil {
		t.Fatalf("read frame body: %v", err)
	}
	cl.SetReadDeadline(time.Time{})
	dec, err := proto.DecryptSecondary(sess.BF2, body)
	if err != nil {
		t.Fatalf("DecryptSecondary: %v", err)
	}
	return dec
}

// expectSilence — в течение d от клиента НЕ должно прийти ни одного фрейма.
func expectSilence(t *testing.T, cl net.Conn, d time.Duration) {
	t.Helper()
	cl.SetReadDeadline(time.Now().Add(d))
	l := make([]byte, 2)
	if _, err := asRead(cl, l); err == nil {
		t.Fatal("ожидали тишину, пришёл фрейм")
	}
	cl.SetReadDeadline(time.Time{})
}

// otherSess — вторая сессия сервера (map-порядок недетерминирован — перебор).
func otherSess(t *testing.T, srv *Server, not *Session) *Session {
	t.Helper()
	for i := 0; i < 8; i++ {
		s := srv.SessionByIndex(i)
		if s == nil {
			break
		}
		if s != not {
			return s
		}
	}
	t.Fatal("вторая сессия не найдена")
	return nil
}

// Тест 1: формы SM_LOGIN_FAIL/SM_PLAY_FAIL — wire 18, pt [op][D messageId] (эталон:
// SM_LOGIN_FAIL.java super(0x01)+writeD, SM_PLAY_FAIL.java super(0x06)+writeD).
func TestAuthFailFrames(t *testing.T) {
	srv, cl, sess := newFlowSrvRaw(t)

	srv.sendAuthFail(sess, opLoginFail, RespAlreadyLoggedIn, "test")
	dec := readFrame(t, cl, sess, 3*time.Second)
	if len(dec) != 8 || dec[0] != opLoginFail || binary.LittleEndian.Uint32(dec[1:5]) != RespAlreadyLoggedIn {
		t.Fatalf("SM_LOGIN_FAIL: len=%d dec=%x", len(dec), dec)
	}

	srv.sendAuthFail(sess, opPlayFail, RespServerDown, "test")
	dec = readFrame(t, cl, sess, 3*time.Second)
	if len(dec) != 8 || dec[0] != opPlayFail || binary.LittleEndian.Uint32(dec[1:5]) != RespServerDown {
		t.Fatalf("SM_PLAY_FAIL: len=%d dec=%x", len(dec), dec)
	}
}

// Тест 2: authd молчит после blob → SM_LOGIN_FAIL(loginFailCode) за loginTimeoutSec.
func TestLoginTimeoutFail(t *testing.T) {
	srv, f, ln := newFailSrv(t, config.Gate{AuthdTimeoutSec: 1, OnlineTtlSec: 60})
	cl := dialClient(t, ln)
	sess := srv.SessionByIndex(0)
	if sess == nil {
		t.Fatal("нет сессии")
	}
	doLogin(t, cl, sess, "timeout01")
	waitFor(t, func() bool { f.mu.Lock(); defer f.mu.Unlock(); return len(f.packets) == 1 })
	dec := readFrame(t, cl, sess, 4*time.Second)
	if dec[0] != opLoginFail || binary.LittleEndian.Uint32(dec[1:5]) != uint32(srv.Cfg.LoginFailCode) {
		t.Fatalf("login timeout: op=%02x id=%d want [01][%d]", dec[0], binary.LittleEndian.Uint32(dec[1:5]), srv.Cfg.LoginFailCode)
	}
	// соединение живо (план T1: НЕ рвать)
	if _, err := cl.Write([]byte{0}); err != nil {
		t.Fatalf("соединение закрыто после LoginFail: %v", err)
	}
}

// Тест 3: [05]-релей без ответа authd → SM_PLAY_FAIL(playFailCode) за playTimeoutSec.
func TestPlayTimeoutFail(t *testing.T) {
	srv, f, ln := newFailSrv(t, config.Gate{AuthdTimeoutSec: 1})
	cl := dialClient(t, ln)
	sess := srv.SessionByIndex(0)
	if sess == nil {
		t.Fatal("нет сессии")
	}
	doLogin(t, cl, sess, "play01")
	waitFor(t, func() bool { f.mu.Lock(); defer f.mu.Unlock(); return len(f.packets) == 1 })
	pushType3(t, f, sess) // login-ok → state AUTHED_LOGIN + таймер отменён
	if fr := readFrame(t, cl, sess, 3*time.Second); len(fr) < 1 || fr[0] != 0x03 {
		t.Fatalf("login-ok relay: op=%02x want 03", fr[0])
	}
	// CM_SERVER_LIST [05] → relay → тишина authd → PlayFail
	sendPt(cl, sess, []byte{0x05, 1, 0, 0, 0})
	dec := readFrame(t, cl, sess, 4*time.Second)
	if dec[0] != opPlayFail || binary.LittleEndian.Uint32(dec[1:5]) != uint32(srv.Cfg.PlayFailCode) {
		t.Fatalf("play timeout: op=%02x id=%d want [06][%d]", dec[0], binary.LittleEndian.Uint32(dec[1:5]), srv.Cfg.PlayFailCode)
	}
}

// Тест 4: relogin онлайн-акка → немедленный SM_LOGIN_FAIL(7), blob всё равно релеить.
func TestReloginOnlineCache(t *testing.T) {
	srv, f, ln := newFailSrv(t, config.Gate{OnlineTtlSec: 60})
	cl1 := dialClient(t, ln)
	sess1 := srv.SessionByIndex(0)
	if sess1 == nil {
		t.Fatal("нет сессии")
	}
	doLogin(t, cl1, sess1, "relogin01")
	waitFor(t, func() bool { f.mu.Lock(); defer f.mu.Unlock(); return len(f.packets) == 1 })
	pushType3(t, f, sess1)
	if fr := readFrame(t, cl1, sess1, 3*time.Second); fr[0] != 0x03 {
		t.Fatalf("login-ok relay: op=%02x", fr[0])
	}
	// relogin: второй клиент, тот же юзер, в пределах TTL
	cl2 := dialClient(t, ln)
	sess2 := otherSess(t, srv, sess1)
	doLogin(t, cl2, sess2, "relogin01")
	dec := readFrame(t, cl2, sess2, 3*time.Second) // НЕМЕДЛЕННО (не ждём таймаут)
	if dec[0] != opLoginFail || binary.LittleEndian.Uint32(dec[1:5]) != RespAlreadyLoggedIn {
		t.Fatalf("relogin: op=%02x id=%d want [01][7]", dec[0], binary.LittleEndian.Uint32(dec[1:5]))
	}
	// blob всё равно релеится (kick-семантика эталона: следующая попытка проходит)
	waitFor(t, func() bool { f.mu.Lock(); defer f.mu.Unlock(); return len(f.packets) == 2 })
	// таймер для sess2 не заармлен: больше фреймов нет
	expectSilence(t, cl2, 400*time.Millisecond)
}

// Тест 5: type=3 пришёл → таймаут не стреляет; онлайн-кэш получил user (relogin → 7).
func TestAuthFailCancel(t *testing.T) {
	srv, f, ln := newFailSrv(t, config.Gate{AuthdTimeoutSec: 1, OnlineTtlSec: 60})
	cl1 := dialClient(t, ln)
	sess1 := srv.SessionByIndex(0)
	if sess1 == nil {
		t.Fatal("нет сессии")
	}
	doLogin(t, cl1, sess1, "cancel01")
	waitFor(t, func() bool { f.mu.Lock(); defer f.mu.Unlock(); return len(f.packets) == 1 })
	pushType3(t, f, sess1)
	if fr := readFrame(t, cl1, sess1, 3*time.Second); fr[0] != 0x03 {
		t.Fatalf("login-ok relay: op=%02x", fr[0])
	}
	// таймаут (1с) прошёл — фейла НЕТ
	expectSilence(t, cl1, 1500*time.Millisecond)
	// онлайн-кэш: relogin cancel01 → немедленный 7
	cl2 := dialClient(t, ln)
	sess2 := otherSess(t, srv, sess1)
	doLogin(t, cl2, sess2, "cancel01")
	dec := readFrame(t, cl2, sess2, 3*time.Second)
	if dec[0] != opLoginFail || binary.LittleEndian.Uint32(dec[1:5]) != RespAlreadyLoggedIn {
		t.Fatalf("relogin after ok: op=%02x id=%d", dec[0], binary.LittleEndian.Uint32(dec[1:5]))
	}
}

// Тест 6 (запрос юзера 08.10): loginTestFail=-1 CYCLE — каждый логин следующий код
// (1,2,...), blob НЕ релеится (акк не лочится), соединение живо.
func TestLoginTestFailCycle(t *testing.T) {
	srv, f, ln := newFailSrv(t, config.Gate{LoginTestFail: -1})
	cl1 := dialClient(t, ln)
	sess1 := srv.SessionByIndex(0)
	if sess1 == nil {
		t.Fatal("нет сессии")
	}
	doLogin(t, cl1, sess1, "testfail01")
	dec := readFrame(t, cl1, sess1, 3*time.Second)
	if dec[0] != opLoginFail || binary.LittleEndian.Uint32(dec[1:5]) != 1 {
		t.Fatalf("cycle #1: op=%02x id=%d want [01][1]", dec[0], binary.LittleEndian.Uint32(dec[1:5]))
	}
	cl2 := dialClient(t, ln)
	sess2 := otherSess(t, srv, sess1)
	doLogin(t, cl2, sess2, "testfail01")
	dec = readFrame(t, cl2, sess2, 3*time.Second)
	if dec[0] != opLoginFail || binary.LittleEndian.Uint32(dec[1:5]) != 2 {
		t.Fatalf("cycle #2: op=%02x id=%d want [01][2]", dec[0], binary.LittleEndian.Uint32(dec[1:5]))
	}
	f.mu.Lock()
	n := len(f.packets)
	f.mu.Unlock()
	if n != 0 {
		t.Fatalf("test-режим: blob релеится (%d), должен НЕ релеиться", n)
	}
}

// Тест 7: playTestFail=-1 — после успешного логина [05] -> SM_PLAY_FAIL(1), authd НЕ релеится.
func TestPlayTestFail(t *testing.T) {
	srv, f, ln := newFailSrv(t, config.Gate{PlayTestFail: -1})
	cl := dialClient(t, ln)
	sess := srv.SessionByIndex(0)
	if sess == nil {
		t.Fatal("нет сессии")
	}
	doLogin(t, cl, sess, "playtest01")
	waitFor(t, func() bool { f.mu.Lock(); defer f.mu.Unlock(); return len(f.packets) == 1 })
	pushType3(t, f, sess)
	if fr := readFrame(t, cl, sess, 3*time.Second); fr[0] != 0x03 {
		t.Fatalf("login-ok relay: op=%02x", fr[0])
	}
	sendPt(cl, sess, []byte{0x05, 1, 0, 0, 0})
	dec := readFrame(t, cl, sess, 3*time.Second)
	if dec[0] != opPlayFail || binary.LittleEndian.Uint32(dec[1:5]) != 1 {
		t.Fatalf("play test: op=%02x id=%d want [06][1]", dec[0], binary.LittleEndian.Uint32(dec[1:5]))
	}
	f.mu.Lock()
	n := len(f.packets)
	f.mu.Unlock()
	if n != 1 { // login-блоб урёл ился, [05] — нет
		t.Fatalf("play test: [05] релеится (%d пакетов)", n)
	}
}

// Тест 8 (юзер 08.10): живые тексты клиента есть для ВСЕХ выдаваемых кодов 1..22+45.
func TestAuthFailTexts(t *testing.T) {
	for id := uint32(1); id <= 22; id++ {
		if authFailTextOf(id) == "" {
			t.Fatalf("нет текста клиента для messageId=%d", id)
		}
	}
	if authFailTextOf(45) == "" {
		t.Fatal("нет текста для 45 (authgate-спец)")
	}
	if authFailTextOf(0) != "" {
		t.Fatal("0 AUTHED не шлётся клиенту — текст не нужен")
	}
}

// ЭКСПЕРИМЕНТ charcount (09.10): байт [63] в 74Б login-ok = счётчик чаров акка.
func TestPadLoginOKCharCount(t *testing.T) {
	payload := make([]byte, 52) // authd type=3 payload 52Б
	pt := padLoginOK(3, payload, 7)
	if len(pt) != 64 {
		t.Fatalf("len=%d, want 64", len(pt))
	}
	if pt[0] != 3 || pt[63] != 7 {
		t.Fatalf("op=%d pt[63]=%d, want 3/7", pt[0], pt[63])
	}
	if zero := padLoginOK(3, payload, 0); zero[63] != 0 {
		t.Fatalf("off-режим: pt[63]=%d, want 0", zero[63])
	}
	if four := padLoginOK(4, payload, 7); len(four) != 53 {
		t.Fatalf("type=4 не падится: len=%d", len(four))
	}
}

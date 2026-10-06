// Package server — гейт 2106: сессии, welcome, релей в authd.
// Скелет (05.10 ночь): открытые позиции помечены TODO §5.x (см. docs/authgate-protocol-20261005.md).
package server

import (
	crand "crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"aion-gate/internal/authdclient"
	"aion-gate/internal/config"
	"aion-gate/internal/proto"
	"aion-gate/internal/ship"
)

// Session — клиент 2106 (частичное зеркало CClientSocket, §0 дока).
type Session struct {
	ID   uint32
	IP   [4]byte
	Key2 [16]byte
	BF2  *proto.Blowfish
	RSA  *proto.RSAKey

	// State-машина (К-2/P0-2, эталон AionPacketHandlerFactory): CONNECTED→AUTHED_GG
	// (после эха AUTH_GG) →AUTHED_LOGIN (после успешного login-decode). Общая для
	// classic (П4) и authgate-режимов; константы stConnected/stAuthedGG/stAuthedLogin в classic.go.
	State   uint8
	AccID   uint32
	LoginOk uint32
	PlayOk1 uint32
	PlayOk2 uint32

	mu       sync.Mutex
	conn     net.Conn
	authdReg bool // authd назначил сессию ([03])
}

func (s *Session) write(b []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil {
		return ErrClosed
	}
	_, err := s.conn.Write(b)
	return err
}

func (s *Session) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
}

var ErrClosed = errClosed{}

type errClosed struct{}

func (errClosed) Error() string { return "session closed" }

// Server — гейт.
type Server struct {
	Cfg config.Gate
	sh  *ship.S

	pool  *proto.KeyPool
	brute *Brute
	ips   *IPList
	key1  *proto.Blowfish

	mu       sync.Mutex
	authd    *authdclient.Client
	sess     map[uint32]*Session
	counter    uint32 // TODO §5.6: генератор sid @0x4041b8 (время+база) — сейчас счётчик с 1
	assigned   uint32 // [authd_sock+0xa0] — sid, назначенный authd ([03])
	variantIdx uint32   // round-robin привет-вариантов (welcomeProbe)
	variantAt  time.Time // момент последней смены варианта
}

func New(cfg config.Gate, sh *ship.S) (*Server, error) {
	cfg.FillDefaults()
	pool, err := buildPool(&cfg)
	if err != nil {
		return nil, err
	}
	k1 := proto.GenerateInitialKey(0x04bd)
	key1, err := proto.NewBlowfish(k1[:])
	if err != nil {
		return nil, err
	}
	return &Server{
		Cfg:   cfg,
		sh:    sh,
		pool:  pool,
		brute: NewBrute(cfg.TryCount, cfg.TryIntervalSec, cfg.TryBlockIntervalSec),
		ips:   LoadIPList(cfg.BlockIPsFile),
		key1:  key1,
		sess:  map[uint32]*Session{},
	}, nil
}

// buildPool — фиксированная пара (rsaFixedN/D) или обычный пул с экспонентой из конфига (П2).
func buildPool(cfg *config.Gate) (*proto.KeyPool, error) {
	log.Printf("rsa: exponent=%d (конфиг rsaExponent; альтернативы: 17 / 65537=F4 гита)", cfg.RsaExponent)
	if cfg.RsaFixedN != "" && cfg.RsaFixedD != "" {
		k, err := proto.RSAKeyFromHex(cfg.RsaFixedN, cfg.RsaFixedD, int64(cfg.RsaExponent))
		if err != nil {
			return nil, err
		}
		log.Printf("rsa: FIXED key N[:8]=%s... (Pub.key-эксперимент)", cfg.RsaFixedN[:16])
		return proto.NewKeyPoolFromKey(k), nil
	}
	return proto.NewKeyPool(int64(cfg.RsaExponent))
}

// AuthdHandler — колбеки для authdclient (DialAuthd или инъекция в тестах).
func (s *Server) AuthdHandler() authdclient.Handler {
	return authdclient.Handler{OnPacket: s.onAuthdPacket, OnClosed: s.onAuthdClosed, OnAssigned: s.onAuthdAssigned, OnRaw: s.onAuthdRaw}
}

// DialAuthd — одна коннекция к authd (оригинал держит одну, готча p1–p5).
func (s *Server) DialAuthd() error {
	addr := net.JoinHostPort(s.Cfg.AuthAddr, strconv.Itoa(s.Cfg.AuthPort))
	c, err := authdclient.Dial(addr, s.AuthdHandler())
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.authd = c
	s.mu.Unlock()
	s.send(ship.Event{Ev: ship.EvConnUp, Svc: "authd", Remote: addr})
	return nil
}

// send — телеметрия (TELEMETRY-SPEC: ship не критичный путь, nil-safe).
func (s *Server) send(ev ship.Event) {
	if s.sh != nil {
		s.sh.Send(ev)
	}
}

// SetAuthd — ручная инъекция готового клиента (тесты/внешний реконнект).
func (s *Server) SetAuthd(c *authdclient.Client) {
	s.mu.Lock()
	s.authd = c
	s.mu.Unlock()
}

// Serve — accept-цикл 2106 (main делает net.Listen).
func (s *Server) Serve(ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go s.handleConn(conn)
	}
}

// nextSID @0x4041b8: fc = rand32() + base (CRT rand детерминирован от старта —
// поэтому welcome dword0 повторяется после рестартов; у нас честный crypto/rand).
func (s *Server) nextSID() uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	return s.counter
}

// onAuthdAssigned — [03] от authd: payload[0:4] → [global+0xa0] (V в welcome).
func (s *Server) onAuthdAssigned(sid uint32) {
	s.mu.Lock()
	s.assigned = sid
	s.mu.Unlock()
}

// currentVariant — текущий пробный вариант welcome; ротация НЕ чаще VariantHoldSec
// (минимум 3 минуты на вариант — пауза на логин юзера).
func (s *Server) currentVariant() int {
	if s.Cfg.WelcomeForceVariant >= 0 {
		return s.Cfg.WelcomeForceVariant // закреплённый вариант (генератор реакции)
	}
	if !s.Cfg.WelcomeProbe {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	hold := time.Duration(s.Cfg.VariantHoldSec) * time.Second
	if hold < 3*time.Minute {
		hold = 3 * time.Minute // жёсткий минимум по требованию юзера
	}
	if s.variantAt.IsZero() {
		s.variantAt = time.Now()
	}
	if time.Since(s.variantAt) >= hold {
		s.variantIdx = (s.variantIdx + 1) % 10
		s.variantAt = time.Now()
	}
	return int(s.variantIdx)
}

func (s *Server) authdSession() uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.assigned
}

func (s *Server) withAuthd(f func(*authdclient.Client)) {
	s.mu.Lock()
	a := s.authd
	s.mu.Unlock()
	if a != nil {
		f(a)
	}
}

func (s *Server) track(sess *Session) {
	s.mu.Lock()
	s.sess[sess.ID] = sess
	s.mu.Unlock()
}

func (s *Server) drop(sess *Session) {
	s.mu.Lock()
	delete(s.sess, sess.ID)
	s.mu.Unlock()
	sess.close()
}

// handleConn — OnCreate: fork → прозрачный прокси с сравнением; иначе cc-22 для
// блок-листа → authd CltConnect → welcome → read-loop.
func (s *Server) handleConn(conn net.Conn) {
	if s.Cfg.Mode == "fork" {
		s.handleConnFork(conn)
		return
	}
	defer conn.Close()
	var ip [4]byte
	if ta, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		copy(ip[:], ta.IP.To4())
	}

	remote0 := net.IP(ip[:]).String()
	if s.ips.Blocked(ip) {
		sendCC(conn, 22) // §3: blocked IP → cc 22
		s.send(ship.Event{Ev: "cc", Svc: "blockip", Remote: remote0, Data: map[string]any{"code": 22}})
		return
	}
	sid := s.nextSID()
	key2 := proto.Key2FromLUT(byte(rand.Intn(256))) // @0x4075d0
	bf2, err := proto.NewBlowfish(key2[:])
	if err != nil {
		return
	}
	sess := &Session{ID: sid, IP: ip, Key2: key2, BF2: bf2, RSA: s.pool.Get(), conn: conn}
	s.track(sess)
	remote := net.IP(ip[:]).String()
	defer s.send(ship.Event{Ev: ship.EvConnDown, Remote: remote, Data: map[string]any{"sid": sid}})
	defer func() {
		s.drop(sess)
		s.withAuthd(func(a *authdclient.Client) { _ = a.SendDisconnect(sid) })
	}()
	s.send(ship.Event{Ev: ship.EvConnUp, Remote: remote, Data: map[string]any{"sid": sid}})

	s.withAuthd(func(a *authdclient.Client) { _ = a.SendConnect(sid, ip) }) // CltConnect @0x406000

	// Гипотеза V=0: клиент отвергает welcome с нулевым authd-session → жддаём [03].
	if ms := s.Cfg.WelcomeWaitAuthdMs; ms > 0 && s.authdSession() == 0 {
		deadline := time.Now().Add(time.Duration(ms) * time.Millisecond)
		for s.authdSession() == 0 && time.Now().Before(deadline) {
			time.Sleep(20 * time.Millisecond)
		}
		log.Printf("welcome wait-authd: V=%d after %dms", s.authdSession(), ms)
	}
	var w []byte
	if s.Cfg.Mode == "classic" {
		// П4а (байон-48): SM_INIT гита — plaintext 192B, static-ключ, encXORPass → wire 210.
		var sk [16]byte
		if _, err := rand.Read(sk[:]); err != nil {
			return
		}
		skBF, err := proto.NewBlowfish(sk[:])
		if err != nil {
			return
		}
		sess.BF2 = skBF // сессионный ключ доставлен клиенту в [153:169] (гит-схема)
		w = proto.BuildClassicWelcome(sid, sess.RSA.Modulus128(), sk, s.key1)
		log.Printf("welcome CLASSIC: sid=%d wire=%d sessionKey=random → [153:169]", sid, len(w))
	}
	vi := s.currentVariant()
	if vi == 4 {
		log.Printf("welcome variant=4 RAW: математически СЛОМАН (unscramble(raw)!=N, байон-48 §3) — только для A/B-логов")
	}
	if w == nil {
		if fx, ferr := hex.DecodeString(strings.TrimSpace(s.Cfg.WelcomeFixture)); ferr == nil && len(fx) >= 4 {
			w = fx // ФИКСТУРА: байт-в-байт реплей дампа оригинала (dumpPacket-лог)
			log.Printf("welcome FIXTURE replay: %d bytes", len(w))
		}
	}
	if w == nil {
		wargs := &proto.WelcomeArgs{
			SessionID:    sid, // fc = rand32 ([fc] @0x4041b8)
			AuthdSession: s.authdSession(), // V = authd [03] ([global+0xa0])
			Modulus:      sess.RSA.Modulus128(),
			Key2:         key2,
		}
		w = proto.BuildWelcomeVariant(wargs, s.key1, vi)
		log.Printf("welcome fields: variant=%d sid=%d(0x%08x) V=%d(0x%08x) mod8=%s key2=%s", vi, sid, sid, wargs.AuthdSession, wargs.AuthdSession, hex.EncodeToString(wargs.Modulus[:8]), hex.EncodeToString(key2[:]))
		log.Printf("welcome variant=%d (probe, hold=%ds)", vi, s.Cfg.VariantHoldSec)
	}
	if s.Cfg.DumpPacket {
		n := len(w)
		if n > 64 {
			n = 64
		}
		log.Printf("welcome len=%d hex=%s", len(w), hex.EncodeToString(w))
	}
	if _, err := conn.Write(w); err != nil {
		return
	}

	timeout := time.Duration(s.Cfg.SessionTimeoutMin) * time.Minute
	for {
		_ = conn.SetReadDeadline(time.Now().Add(timeout)) // sessionTimeout=5м
		payload, err := proto.ReadFrame(conn)
		if err != nil {
			return
		}
		if s.Cfg.DumpPacket {
			n := len(payload)
			if n > 64 {
				n = 64
			}
			log.Printf("frame len=%d hex=%s", len(payload), hex.EncodeToString(payload))
		}
		if s.Cfg.Mode == "classic" { // П4b: диспетчер по (op,state)
			if err := s.dispatchClassic(sess, payload); err != nil {
				log.Printf("classic: dispatch err sid=%d: %v", sess.ID, err)
				return
			}
			continue
		}
		// К-2 (P0-2): диспетчер по (state, op) как AionPacketHandlerFactory эталона;
		// длины 32/24/≥184 — fallback-эвристика (leak-клиент 312/314b).
		if err := s.dispatchAuthgate(sess, payload); err != nil {
			s.send(ship.Event{Ev: ship.EvParseErr, Remote: remote, Err: err.Error(), Data: map[string]any{"stage": "dispatch"}})
			return
		}
	}
}

// stateName — имя состояния для логов (К-2/P0-2).
func stateName(st uint8) string {
	switch st {
	case stConnected:
		return "CONNECTED"
	case stAuthedGG:
		return "AUTHED_GG"
	case stAuthedLogin:
		return "AUTHED_LOGIN"
	}
	return fmt.Sprintf("state%d", st)
}

// dispatchAuthgate — диспетчер (state, op) по эталону AionPacketHandlerFactory (К-2/P0-2):
//   CONNECTED{0x07→authgg, 0x08→UPDATE_SESSION}, AUTHED_GG{0x0B→login},
//   AUTHED_LOGIN{0x05,0x02→relay}; прочее = лог "unknown packet state=... op=..." (НЕ cc45).
// Длины 32/24/≥184 оставлены fallback-эвристикой (leak-клиент шлёт 312/314b).
func (s *Server) dispatchAuthgate(sess *Session, payload []byte) error {
	pt, err := proto.DecryptSecondary(sess.BF2, payload)
	if err != nil {
		// нерасшифрованный фрейм — только length-fallback (как раньше)
		log.Printf("dispatch: key2-decrypt FAIL sid=%d len=%d: %v — length-fallback", sess.ID, len(payload), err)
		return s.dispatchByLen(sess, payload)
	}
	if len(pt) < 1 {
		return nil
	}
	op := pt[0]
	switch sess.State {
	case stConnected:
		switch op {
		case 0x07: // CM_AUTH_GG
			return s.handleAuthGG(sess, payload) // эхо → State=AUTHED_GG
		case 0x08: // CM_UPDATE_SESSION (эталон: accountId/loginOk/reconnectKey → authReconnectingAccount)
			s.handleUpdateSession(sess, pt)
			return nil
		}
	case stAuthedGG:
		if op == 0x0B { // К-3 (P1-3): вход в login ТОЛЬКО по op=0x0B (эталон 7.7)
			return s.handleLogin(sess, payload)
		}
		if len(payload) == 24 { // фолбэк длин: 26b-пинги (эмуляция/relay как раньше)
			s.handle26(sess, payload)
			return nil
		}
		// прочее в AUTHED_GG: лог + raw-relay (НЕ cc45, НЕ рвать)
		log.Printf("login: op=0x%02x ≠ 0x0B (эталон 7.7) в AUTHED_GG — лог+raw-relay", op)
		s.withAuthd(func(a *authdclient.Client) { _ = a.SendPacket(sess.ID, pt) })
		return nil
	case stAuthedLogin:
		if op == 0x05 || op == 0x02 { // CM_SERVER_LIST / CM_PLAY → relay/эмуляция
			s.handle26pt(sess, pt)
			return nil
		}
	}
	// fallback: эвристики длин (старое поведение)
	return s.dispatchByLen(sess, payload)
}

// dispatchByLen — прежний length-диспетчер (fallback эвристика, К-2: оставить).
func (s *Server) dispatchByLen(sess *Session, payload []byte) error {
	switch {
	case len(payload) == 32: // клиент 34b: AUTH_GG (дизasm ночь-4: echo=[sid][28×0], RSA НЕ участвует)
		return s.handleAuthGG(sess, payload)
	case len(payload) >= 184: // логин 186/314
		return s.handleLogin(sess, payload)
	case len(payload) == 24: // пинги/запросы клиента (wire 26 = 2+len + 24 ECB) — эмуляция ответов оригинала
		s.handle26(sess, payload)
		return nil
	default:
		// unknown (К-2): лог state/op, соединение живо, raw-relay — НЕ cc45
		pt2, derr := proto.DecryptSecondary(sess.BF2, payload)
		if derr != nil {
			log.Printf("relay: key2-decrypt FAIL sid=%d len=%d: %v", sess.ID, len(payload), derr)
			return nil
		}
		op := byte(0)
		if len(pt2) > 0 {
			op = pt2[0]
		}
		log.Printf("unknown packet state=%s op=0x%02x len=%d — raw relay (НЕ cc45)", stateName(sess.State), op, len(pt2))
		s.withAuthd(func(a *authdclient.Client) { _ = a.SendPacket(sess.ID, pt2) })
		return nil
	}
}

// handleUpdateSession — CM_UPDATE_SESSION (0x08, CONNECTED, К-2/P0-2): релогин-флоу
// эталона (accountId, loginOk, reconnectKey → authReconnectingAccount). Релеим в authd
// с type=0x08 (клиентский опкод = тип по relay-контракту); при неоднозначности — лог+raw.
func (s *Server) handleUpdateSession(sess *Session, pt []byte) {
	if len(pt) >= 13 {
		log.Printf("update-session: sid=%d acc=%d loginOk=%d reconnectKey=%08x — relay type=0x08",
			sess.ID, binary.LittleEndian.Uint32(pt[1:5]), binary.LittleEndian.Uint32(pt[5:9]), binary.LittleEndian.Uint32(pt[9:13]))
	} else {
		log.Printf("update-session: sid=%d короткий pt=%d — relay-raw type=0x08", sess.ID, len(pt))
	}
	s.withAuthd(func(a *authdclient.Client) { _ = a.SendPacket(sess.ID, pt) })
}

// handleAuthGG: клиент 34b = [len][32Б blob] — CM_AUTH_GG, blob =
// EncryptSecondary([sid 4B][20B]) (classic). Ответ 42b = EncryptSecondary([sid][28×0]) —
// SM_AUTH_GG: capture-структура cipher [P][Q][Q][Q][P] = plaintext [A][28×0][A][pad0]
// (ночь-4, docs/session-20261005-authgate-disasm407d50.md §4). RSA в обмене НЕ участвует.
// Расшифровку клиентского blob делаем best-effort (лог расхождения sid), связь НЕ рвём:
// канонический ответ зависит только от нашего sid.
func (s *Server) handleAuthGG(sess *Session, blob []byte) error {
	if data, err := proto.DecryptSecondary(sess.BF2, blob); err == nil {
		if len(data) >= 4 {
			log.Printf("auth-gg: dec sid=%d blob=%s", sess.ID, hex.EncodeToString(data))
			if csid := binary.LittleEndian.Uint32(data[1:5]); csid != sess.ID {
				log.Printf("auth-gg: sid mismatch: client %d != session %d", csid, sess.ID)
				s.send(ship.Event{Ev: "authgg.mismatch", Data: map[string]any{"sid": sess.ID, "client": csid}})
			}
		}
	} else {
		log.Printf("auth-gg: blob не расшифровался как EncryptSecondary (len=%d): %v", len(blob), err)
		s.send(ship.Event{Ev: "authgg.blob", Data: map[string]any{"sid": sess.ID, "len": len(blob)}})
	}
	// Тест cc-сообщений: welcomeTestCC > 0 → в ответ на AUTH_GG шлём cc-код
	// (проверка отображения клиентом: 22 = «аккаунт заблокирован» и т.п.).
	if tc := s.Cfg.WelcomeTestCC; tc > 0 {
		log.Printf("auth-gg: TEST cc=%d", tc)
		sendCCSess(sess, byte(tc))
		return nil
	}
	// П3: форма SM_AUTH_GG — live 42b (default, live-принят — НЕ ТРОГАТЬ) или эталонная
	// 50b ([0b][sid][35×0], smAuthGgWire=50, A/B против орига в fork).
	var reply []byte
	if s.Cfg.SmAuthGgWire == 50 {
		reply = proto.BuildClassicAuthGG77(sess.ID)
		log.Printf("auth-gg: эталонная форма smAuthGgWire=50 (pt=%d → wire 50)", len(reply))
	} else {
		reply = make([]byte, 32)
		reply[0] = 0x0b // SM_AUTH_GG opcode (LE-дамп 06.10: [0b][sid][27×0])
		binary.LittleEndian.PutUint32(reply[1:5], sess.ID)
	}
	sess.State = stAuthedGG // К-2 (P0-2): эхо AUTH_GG отправлено → AUTHED_GG
	fr := proto.WriteFrame(proto.EncryptSecondary(sess.BF2, reply))
	dumpRaw(fmt.Sprintf("G>C authgg-reply sid=%d", sess.ID), fr)
	return sess.write(fr)
}

// handleLogin: релей логина в authd (§5.3, дизasm 05.10 + 0x417b60):
// первые 128Б = RSA-блок (BE-число с ведущими нулями, модуль = наш ключ из
// welcome); rsapricrt → decbuf (32Б, BE, выровнен к началу); dword@148
// (флаг 0x80000000 — TODO); блок @152 = len-152 байт ("len-24" после RSA).
// TODO(§5.3): длина decbuf — 32Б (математика) или 34Б (arg3=0x22 у оригинала) —
// верифицировать живым клиентом/authd.
// handleLogin: CM_LOGIN (op 0x00) — парс по гиту байон-48 (П3): pt=[op][ct 128×k][tail 55],
// чанки → RSA nopadding → полный m 128B BE; user@94:108 / pwd@108:124 / otp LE @124:128
// (loginex k>1: user@78 чанк1, pwd/otp чанк2). Валидно (user printable, otp=FFFFFFFF) →
// blob "cbdb" с РЕАЛЬНЫМИ кредами (decbuf = m[128-loginDecbufLen:]; asm ориг arg3=0x22=34).
// Невалидно → лог "login decode FAIL (exp=...)" + legacy-релей (полный m) — не рвать.
func (s *Server) handleLogin(sess *Session, data []byte) error {
	// КОРНЕВАЯ ГОТЧА 06.10 №2: data = RAW ECB(key2) — ОБЯЗАТЕЛЬНО DecryptSecondary
	// до SplitLogin/RSA (handleAuthGG это делал, login/26b — НЕТ; этим объясняются
	// и все старые «decbuf мусор»: RSA глушили ещё зашифрованные байты).
	pt, err := proto.DecryptSecondary(sess.BF2, data)
	if err != nil {
		log.Printf("login: key2-decrypt FAIL sid=%d len=%d: %v", sess.ID, len(data), err)
		return err
	}
	op, chunks, tail, shapeOK := proto.SplitLogin(pt)
	if !shapeOK {
		log.Printf("login: sid=%d op=0x%02x форма НЕ по гиту (pt=%d) — legacy-релей", sess.ID, op, len(pt))
		return s.handleLoginLegacy(sess, pt)
	}
	// К-3 (P1-3): op логируется всегда (эталон 7.7: CM_LOGIN = 0x0B в AUTHED_GG);
	// диспетчер пускает сюда только 0x0B — тут страховка для length-fallback путей.
	if op != 0x0B {
		log.Printf("login: op=0x%02x ≠ 0x0B (эталон 7.7) sid=%d k=%d — обрабатываю по форме, сверить логом", op, sess.ID, len(chunks))
	}
	ms := make([][]byte, 0, len(chunks))
	for i, ct := range chunks {
		m, err := sess.RSA.DecryptBlock(ct)
		if err != nil {
			log.Printf("login: RSA decrypt FAIL chunk %d/%d len=%d: %v", i+1, len(chunks), len(ct), err)
			return s.handleLoginLegacy(sess, pt)
		}
		ms = append(ms, m)
	}
	dec, ok := proto.DecodeLoginPlain(ms)
	if !ok {
		// К-4: логируем ОБЕ гипотезы k=1 (4.8 и 7.7-эталон) — первый живой прогон покажет раскладку
		log.Printf("login decode FAIL (exp=%d, chunks=%d): 4.8-user=%q 7.7-user=%q — legacy-релей (не рвать)",
			s.Cfg.RsaExponent, len(ms), dec.User, dec.Alt77User)
		s.send(ship.Event{Ev: "login.decodefail", Data: map[string]any{"sid": sess.ID, "exp": s.Cfg.RsaExponent, "chunks": len(ms)}})
		return s.handleLoginLegacy(sess, pt)
	}
	log.Printf("login OK: sid=%d chunks=%d layout=%s loginex=%v user=%q pwd=%q pwdHex=%s otp=%08x",
		sess.ID, len(ms), dec.Layout, dec.Ex, dec.User, dec.Pwd, dec.PwdHex, dec.Otp)
	s.send(ship.Event{Ev: "login", Remote: net.IP(sess.IP[:]).String(), Data: map[string]any{
		"sid": sess.ID, "user": dec.User, "otp": dec.Otp, "chunks": len(ms), "layout": dec.Layout,
	}})
	// К-5 (P1-5): эталон .trim().toLowerCase() — authd-автосоздание аккаунта чувствительно
	// к регистру/пробелам («StelGeN» vs «stelgen» = разные аккаунты).
	user := strings.TrimSpace(strings.ToLower(dec.User))
	if user != dec.User {
		log.Printf("login: username normalize: %q → %q (эталон trim+toLowerCase)", dec.User, user)
	}
	decbuf := proto.BuildLoginDecbuf(user, dec.Pwd, dec.Otp, s.Cfg.LoginDecbufLen)
	// К-1 (P0-1): k=1 — asm-позиции оригинала (dword = data+148, tail-блоб с data+152);
	// k≥2 (loginex) pt[148:152]/pt[152:] — ЗОНА ШИФРТЕКСТА (ct=pt[1:1+k*128]) →
	// dword/tail берём из SplitLogin-хвоста. При сомнении НЕ рвать — relay.
	var dword uint32
	var blobTail []byte
	if len(chunks) == 1 && len(pt) >= 156 {
		dword = binary.LittleEndian.Uint32(pt[148:152])
		blobTail = pt[152:]
	} else {
		dword = s.loginDwordFromTail(tail)
		blobTail = tail
		if len(chunks) == 1 {
			log.Printf("login: короткий k=1 (pt=%d) — dword/tail из SplitLogin-хвоста (defensive)", len(pt))
		}
	}
	blob := proto.Assemble("cbdb", byte(0), decbuf, dword, blobTail)
	s.withAuthd(func(a *authdclient.Client) { _ = a.SendPacket(sess.ID, blob) })
	log.Printf("login: relay authd sid=%d decbufLen=%d blobLen=%d k=%d dword=%08x taillen=%d",
		sess.ID, len(decbuf), len(blob), len(chunks), dword, len(blobTail))
	sess.State = stAuthedLogin // К-2 (P0-2): успешный login-decode → AUTHED_LOGIN
	return nil
}

// loginDwordFromTail — К-1 (P0-1): loginex-хвост → dword для blob "cbdb".
// Live-хвост 47Б: [sid LE][нули][0x20][7×0][68ffdab3e2fda892][2d9cc7baa87e0d49][dword].
// Ищем 0x20-блок (0x20 + ≥7 нулей); кандидаты ЛОГИРУЮТСЯ ВСЕ (первый прогон живого
// loginex покажет верный): (а) dword после полного 0x20-блока с magic-ами,
// (б) dword сразу после байта 0x20, (в) asm-эквивалент k=1 (на байт раньше 0x20).
// При неоднозначности/отсутствии блока → константный 0 (НЕ рвать — relay продолжается).
func (s *Server) loginDwordFromTail(tail []byte) uint32 {
	p := -1
	for i := 0; i+8 <= len(tail); i++ {
		if tail[i] != 0x20 {
			continue
		}
		z := true
		for _, c := range tail[i+1 : i+8] {
			if c != 0 {
				z = false
				break
			}
		}
		if z {
			p = i
			break
		}
	}
	if p < 0 {
		log.Printf("loginex: 0x20-блок в хвосте (%dБ) не найден → dword=0 (tail=%s)", len(tail), hex.EncodeToString(tail))
		return 0
	}
	var afterBlock, after20, straddle uint32
	hasAfterBlock := p+28 <= len(tail)
	if hasAfterBlock {
		afterBlock = binary.LittleEndian.Uint32(tail[p+24 : p+28])
	}
	if p+5 <= len(tail) {
		after20 = binary.LittleEndian.Uint32(tail[p+1 : p+5])
	}
	if p >= 1 {
		straddle = binary.LittleEndian.Uint32(tail[p-1 : p+3])
	}
	chosen := uint32(0)
	if hasAfterBlock {
		chosen = afterBlock // хвост заканчивается dword'ом после 0x20-блока — единственная чистая позиция
	}
	log.Printf("loginex: dword-кандидаты (0x20@%d/%d): после-блока=%08x после-0x20=%08x asm-эквив=%08x → выбор=%08x",
		p, len(tail), afterBlock, after20, straddle, chosen)
	return chosen
}

// handleLoginLegacy — старый релей (до П3): полный m 128Б + dword148 + хвост.
// Вход — УЖЕ расшифрованный pt (DecryptSecondary сделан в handleLogin).
func (s *Server) handleLoginLegacy(sess *Session, data []byte) error {
	const rsaLen = 128
	if len(data) < rsaLen+24 {
		// К-2 (P0-2): кривой/короткий логин — лог + raw-relay, соединение живо (НЕ cc45)
		log.Printf("login legacy: sid=%d короткий pt=%d (<%d) — лог+raw-relay (НЕ cc45)", sess.ID, len(data), rsaLen+24)
		s.withAuthd(func(a *authdclient.Client) { _ = a.SendPacket(sess.ID, data) })
		return nil
	}
	decbuf, err := sess.RSA.DecryptBlock(data[:rsaLen])
	if err != nil {
		log.Printf("login: RSA decrypt FAIL len=%d: %v", len(data), err)
		return err
	}
	dword148 := binary.LittleEndian.Uint32(data[148:152])
	tail := data[152:]
	log.Printf("login legacy: sid=%d decbuf=%s dword148=%08x taillen=%d tail=%s", sess.ID, hex.EncodeToString(decbuf), dword148, len(tail), hex.EncodeToString(tail))
	blob := proto.Assemble("cbdb", byte(0), decbuf, dword148, tail)
	s.withAuthd(func(a *authdclient.Client) { _ = a.SendPacket(sess.ID, blob) })
	return nil
}

// onAuthdPacket — push от authd: тип 4 = serverlist (payload из capture, §3).
// TODO §5.5: точный формат 74b/26b ответов клиенту; сейчас шлём payload зашифрованным key2 всем.
// dumpRaw — RAW-дамп всего трафика в тот же gate-лог (требование 06.10: полный raw).
func dumpRaw(tag string, b []byte) {
	log.Printf("RAW %s len=%d hex=%s", tag, len(b), hex.EncodeToString(b))
}

func (s *Server) onAuthdRaw(dir string, b []byte) {
	dumpRaw("AUTHD "+dir, b)
}

// parseServerList — [02]-type-4 (serverlist): полный дамп + эвристический парс
// (count-byte, UTF-16 имена серверов; точная структура — по живому дампу).
func parseServerList(typ byte, payload []byte) {
	if typ != 4 || len(payload) < 1 {
		return
	}
	cnt := int(payload[0])
	log.Printf("serverlist: len=%d count-byte=%d rest=%d", len(payload), cnt, len(payload)-1)
	head := len(payload)
	if head > 24 {
		head = 24
	}
	log.Printf("serverlist: head=%s", hex.EncodeToString(payload[:head]))
	for i := 0; i+1 < len(payload); i++ {
		if payload[i] >= 0x20 && payload[i] < 0x7f && payload[i+1] == 0 {
			j := i
			for j+1 < len(payload) && payload[j] >= 0x20 && payload[j] < 0x7f && payload[j+1] == 0 {
				j += 2
			}
			if j-i >= 8 {
				nm := make([]rune, 0, (j-i)/2)
				for k := i; k < j; k += 2 {
					nm = append(nm, rune(payload[k]))
				}
				log.Printf("serverlist: utf16-name@%d: %q", i, string(nm))
				i = j
			}
		}
	}
}

func (s *Server) onAuthdPacket(id uint32, typ byte, payload []byte) {
	log.Printf("RAW AUTHD pkt id=%d type=%d len=%d hex=%s", id, typ, len(payload), hex.EncodeToString(payload))
	parseServerList(typ, payload)
	// ЛЮБОЙ тип authd -> конкретной сессии по id.
	// КОНТРАКТ (live 06.10): клиентский ОПКОД = ТИП от authd — гейт ОБЯЗАН добавить
	// его в начало pt (capture: authd type=3 + payload 52Б → ориг шлёт клиенту
	// [03]+payload+пад до 64Б = wire 74 login-ok; type=4 → [04] 42b; type=7 → [07] 26b).
	// Без типа клиент получает неизвестный опкод (07...) и молча висит.
	s.mu.Lock()
	sess := s.sess[id]
	s.mu.Unlock()
	if sess == nil {
		log.Printf("authd pkt: нет сессии id=%d type=%d", id, typ)
		return
	}
	pt := append([]byte{typ}, payload...)
	if typ == 3 && len(pt) < 64 { // login-ok: паритет с оригом (pt 64Б → wire 74)
		pt = append(pt, make([]byte, 64-len(pt))...)
	}
	fr := proto.WriteFrame(proto.EncryptSecondary(sess.BF2, pt))
	dumpRaw(fmt.Sprintf("G>C authd-pkt type=%d sid=%d", typ, sess.ID), fr)
	_ = sess.write(fr)
	s.send(ship.Event{Ev: "serverlist", Svc: "authd", Data: map[string]any{"type": typ, "len": len(payload)}})
}

func (s *Server) onAuthdClosed(err error) {
	s.mu.Lock()
	s.authd = nil
	s.mu.Unlock()
	log.Printf("authd lost: %v (authReconnectInterval=%d)", err, s.Cfg.AuthReconnectInterval)
	// Наш гейт МОЖЕТ реконнектить (в отличие от оригинала) — аккуратно, только если задан интервал.
	if s.Cfg.AuthReconnectInterval > 0 {
		go func() {
			time.Sleep(time.Duration(s.Cfg.AuthReconnectInterval) * time.Second)
			if derr := s.DialAuthd(); derr != nil {
				log.Printf("authd reconnect: %v", derr)
			}
		}()
	}
}

// sendCC — cc-отказ @0x407e80: Assemble("cc", 1, code) → [len u16][01][code] plaintext.
// Клиент отображает код как сообщение: 22 = «аккаунт заблокирован» (наблюдено живым
// клиентом 06.10!), 45 = гейт не готов; полный реестр кодов — TODO §5.5.
func sendCC(conn net.Conn, code byte) {
	log.Printf("cc: send code=%d", code) // было невидимо — «blocked» без следа в логе!
	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	ccf := proto.WriteFrame(proto.Assemble("cc", byte(1), code))
	dumpRaw(fmt.Sprintf("G>C cc code=%d", code), ccf)
	_, _ = conn.Write(ccf)
}

func sendCCSess(sess *Session, code byte) {
	_ = sess.write(proto.WriteFrame(proto.Assemble("cc", byte(1), code)))
}

// handle26 — обёртка: payload = RAW ECB(key2) → расшифровка → handle26pt
// (та же готча, что и в login).
func (s *Server) handle26(sess *Session, payload []byte) {
	ptIn, err := proto.DecryptSecondary(sess.BF2, payload)
	if err != nil {
		log.Printf("26b: key2-decrypt FAIL sid=%d len=%d: %v", sess.ID, len(payload), err)
		return
	}
	s.handle26pt(sess, ptIn)
}

// handle26pt — эмуляция 26b-пингов (по capture 06.10: 42b server-info и 26b ack):
// op=0x05 → 42b: [04][01 01 01][IP][port 7777][00 00][f4 01 01 01][00 00 00 02][01 00 01][12x0]
// op=0x02 → 26b: [07][pk1 Rnd][pk2 Rnd][serverID][6x0] (хвост ориг = резидуум, клиент толерантен)
func (s *Server) handle26pt(sess *Session, ptIn []byte) {
	if len(ptIn) < 1 {
		return
	}
	op := ptIn[0]
	// ОРИГ релеит [05]/[02] в authd (ответы-типы 4/7 идут ОТ AUTHD); наша эмуляция
	// 42b/26b — только фолбэк, когда authd недоступен.
	relayed := false
	s.withAuthd(func(a *authdclient.Client) { relayed = a.SendPacket(sess.ID, ptIn) == nil })
	if relayed {
		log.Printf("26b: relay authd sid=%d op=%02x len=%d", sess.ID, op, len(ptIn))
		return
	}
	pt := s.build26ReplyPt(op, sess)
	if pt == nil {
		return
	}
	fr := proto.WriteFrame(proto.EncryptSecondary(sess.BF2, pt))
	dumpRaw(fmt.Sprintf("G>C 26b-reply op=%02x sid=%d", op, sess.ID), fr)
	_ = sess.write(fr)
}

// build26ReplyPt — сборка plaintext-ответа на 26b (op=0x05 → 42b server-info,
// op=0x02 → 26b play-ok); nil = ответа нет. Реюз: handle26 + fork-shadow (sess=nil).
// К-6 (P2-6): playOk1/playOk2 = crypto/rand u32 (эталон SessionKey: playOk1=Rnd,
// playOk2=Rnd — НЕ хардкод 1/1010); serverId — из конфига (serverID); значения
// сохраняются в сессию (checkLogin-семантика эталона).
func (s *Server) build26ReplyPt(op byte, sess *Session) []byte {
	var pt []byte
	switch op {
	case 0x05:
		ip := net.ParseIP(s.Cfg.WorldIP).To4()
		if ip == nil {
			ip = net.IPv4(192, 168, 0, 125)
		}
		pt = append([]byte{0x04, 0x01, 0x01, 0x01}, ip...)
		var port [2]byte
		binary.LittleEndian.PutUint16(port[:], uint16(s.Cfg.WorldPort)) // 7777 = 61 1e
		pt = append(pt, port[0], port[1], 0x00, 0x00, 0x00, 0x00, 0x00, 0x00)
		pt = append(pt, 0xf4, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x02, 0x01, 0x00, 0x01)
		pt = append(pt, make([]byte, 12)...) // 32
	case 0x02:
		var pk1, pk2 [4]byte
		_, _ = crand.Read(pk1[:])
		_, _ = crand.Read(pk2[:])
		if sess != nil {
			sess.PlayOk1 = binary.LittleEndian.Uint32(pk1[:])
			sess.PlayOk2 = binary.LittleEndian.Uint32(pk2[:])
		}
		pt = append([]byte{0x07}, pk1[:]...)
		pt = append(pt, pk2[:]...)
		pt = append(pt, byte(s.Cfg.ServerID))
		pt = append(pt, make([]byte, 6)...) // 16
	default:
		log.Printf("26b: op=%02x — без ответа (TODO)", op)
		return nil
	}
	return pt
}

// SessionByIndex — тест-хук.
func (s *Server) SessionByIndex(i int) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sess := range s.sess {
		if i == 0 {
			return sess
		}
		i--
	}
	return nil
}

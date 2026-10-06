// Package server — гейт 2106: сессии, welcome, релей в authd.
// Скелет (05.10 ночь): открытые позиции помечены TODO §5.x (см. docs/authgate-protocol-20261005.md).
package server

import (
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

	// classic-режим (П4): state-машина CONNECTED→AUTHED_GG→AUTHED_LOGIN
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
		switch {
		case len(payload) == 32: // клиент 34b: AUTH_GG (дизasm ночь-4: echo=[sid][28×0], RSA НЕ участвует)
			if err := s.handleAuthGG(sess, payload); err != nil {
				s.send(ship.Event{Ev: ship.EvParseErr, Remote: remote, Err: err.Error(), Data: map[string]any{"stage": "authgg"}})
				return
			}
		case len(payload) >= 184: // логин 186/314
			if err := s.handleLogin(sess, payload); err != nil {
				s.send(ship.Event{Ev: ship.EvParseErr, Remote: remote, Err: err.Error(), Data: map[string]any{"stage": "login"}})
				return
			}
		case len(payload) == 24: // пинги/запросы клиента (wire 26 = 2+len + 24 ECB) — эмуляция ответов оригинала
			s.handle26(sess, payload)
		default: // релей в authd (тип = первый байт payload)
			s.withAuthd(func(a *authdclient.Client) { _ = a.SendPacket(sess.ID, payload) })
			log.Printf("relay: sid=%d len=%d op=%02x", sess.ID, len(payload), payload[0])
		}
	}
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
	reply := make([]byte, 32)
	reply[0] = 0x0b // SM_AUTH_GG opcode (LE-дамп 06.10: [0b][sid][27×0])
	binary.LittleEndian.PutUint32(reply[1:5], sess.ID)
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
	chunks, _, shapeOK := proto.SplitLogin(pt)
	if !shapeOK {
		log.Printf("login: sid=%d форма НЕ по гиту (pt=%d) — legacy-релей", sess.ID, len(pt))
		return s.handleLoginLegacy(sess, pt)
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
		log.Printf("login decode FAIL (exp=%d, chunks=%d): user=%q — legacy-релей (не рвать)",
			s.Cfg.RsaExponent, len(ms), dec.User)
		s.send(ship.Event{Ev: "login.decodefail", Data: map[string]any{"sid": sess.ID, "exp": s.Cfg.RsaExponent, "chunks": len(ms)}})
		return s.handleLoginLegacy(sess, pt)
	}
	log.Printf("login OK: sid=%d chunks=%d loginex=%v user=%q pwd=%q pwdHex=%s otp=%08x",
		sess.ID, len(ms), dec.Ex, dec.User, dec.Pwd, dec.PwdHex, dec.Otp)
	s.send(ship.Event{Ev: "login", Remote: net.IP(sess.IP[:]).String(), Data: map[string]any{
		"sid": sess.ID, "user": dec.User, "otp": dec.Otp, "chunks": len(ms),
	}})
	decbuf := proto.BuildLoginDecbuf(dec.User, dec.Pwd, dec.Otp, s.Cfg.LoginDecbufLen)
	dword148 := binary.LittleEndian.Uint32(pt[148:152])
	blob := proto.Assemble("cbdb", byte(0), decbuf, dword148, pt[152:])
	s.withAuthd(func(a *authdclient.Client) { _ = a.SendPacket(sess.ID, blob) })
	log.Printf("login: relay authd sid=%d decbufLen=%d blobLen=%d", sess.ID, len(decbuf), len(blob))
	return nil
}

// handleLoginLegacy — старый релей (до П3): полный m 128Б + dword148 + хвост.
// Вход — УЖЕ расшифрованный pt (DecryptSecondary сделан в handleLogin).
func (s *Server) handleLoginLegacy(sess *Session, data []byte) error {
	const rsaLen = 128
	if len(data) < rsaLen+24 {
		sendCCSess(sess, 45) // кривой логин — гасим (TODO §5.2: точная семантика)
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
	// ЛЮБОЙ тип authd -> конкретной сессии по id (74b/42b/26b = EncryptSecondary(payload)):
	// payload 64->74b (serverlist), 32->42b (server-info IP:7777), 16->26b (пинги).
	s.mu.Lock()
	sess := s.sess[id]
	s.mu.Unlock()
	if sess == nil {
		log.Printf("authd pkt: нет сессии id=%d type=%d", id, typ)
		return
	}
	fr := proto.WriteFrame(proto.EncryptSecondary(sess.BF2, payload))
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

// handle26 — эмуляция 26b-пингов (по capture 06.10: 42b server-info и 26b ack):
// op=0x05 → 42b: [04][01 01 01][IP][port 7777][00 00][f4 01 01 01][00 00 00 02][01 00 01][12x0]
// op=0x02 → 26b: [07][01 00 00 00][1010][01][5x0] (хвост ориг = резидуум, клиент толерантен)
func (s *Server) handle26(sess *Session, payload []byte) {
	// payload = RAW ECB(key2) — расшифровываем (та же готча, что и в login)
	ptIn, err := proto.DecryptSecondary(sess.BF2, payload)
	if err != nil {
		log.Printf("26b: key2-decrypt FAIL sid=%d len=%d: %v", sess.ID, len(payload), err)
		return
	}
	op := ptIn[0]
	pt := s.build26ReplyPt(op)
	if pt == nil {
		return
	}
	fr := proto.WriteFrame(proto.EncryptSecondary(sess.BF2, pt))
	dumpRaw(fmt.Sprintf("G>C 26b-reply op=%02x sid=%d", op, sess.ID), fr)
	_ = sess.write(fr)
}

// build26ReplyPt — сборка plaintext-ответа на 26b (op=0x05 → 42b server-info,
// op=0x02 → 26b play-ok); nil = ответа нет. Реюз: handle26 + fork-shadow.
func (s *Server) build26ReplyPt(op byte) []byte {
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
		pt = append([]byte{0x07, 0x01, 0x00, 0x00, 0x00, 0xf2, 0x03, 0x00, 0x00, 0x01}, make([]byte, 6)...) // 16
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

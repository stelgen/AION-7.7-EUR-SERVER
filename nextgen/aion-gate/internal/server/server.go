// Package server — гейт 2106: сессии, welcome, релей в authd.
// Скелет (05.10 ночь): открытые позиции помечены TODO §5.x (см. docs/authgate-protocol-20261005.md).
package server

import (
	"encoding/binary"
	"log"
	"math/rand"
	"net"
	"strconv"
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
	counter  uint32 // TODO §5.6: генератор sid @0x4041b8 (время+база) — сейчас счётчик с 1
	assigned uint32 // [authd_sock+0xa0] — sid, назначенный authd ([03])
}

func New(cfg config.Gate, sh *ship.S) (*Server, error) {
	cfg.FillDefaults()
	pool, err := proto.NewKeyPool()
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

// AuthdHandler — колбеки для authdclient (DialAuthd или инъекция в тестах).
func (s *Server) AuthdHandler() authdclient.Handler {
	return authdclient.Handler{OnPacket: s.onAuthdPacket, OnClosed: s.onAuthdClosed}
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

func (s *Server) nextSID() uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	return s.counter
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

// handleConn — OnCreate: cc-22 для блок-листа → authd CltConnect → welcome → read-loop.
func (s *Server) handleConn(conn net.Conn) {
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

	wargs := &proto.WelcomeArgs{
		PlainByte:    byte(s.Cfg.WelcomePlainByte),
		SessionID:    sid,
		AuthdSession: s.authdSession(),
		Modulus:      sess.RSA.Modulus128(),
		Key2:         key2,
		LoginType:    byte(s.Cfg.LoginType),
		B0:           byte(s.Cfg.WelcomeB0),
		B1:           byte(s.Cfg.WelcomeB1),
		B2:           byte(s.Cfg.WelcomeB2),
	}
	if _, err := conn.Write(proto.BuildWelcome(wargs, s.key1)); err != nil {
		return
	}

	timeout := time.Duration(s.Cfg.SessionTimeoutMin) * time.Minute
	for {
		_ = conn.SetReadDeadline(time.Now().Add(timeout)) // sessionTimeout=5м
		payload, err := proto.ReadFrame(conn)
		if err != nil {
			return
		}
		switch {
		case len(payload) == 32: // клиент 34b: RSA-обмен
			if err := s.handleRSAExchange(sess, payload); err != nil {
				s.send(ship.Event{Ev: ship.EvParseErr, Remote: remote, Err: err.Error(), Data: map[string]any{"stage": "rsa"}})
				return
			}
		case len(payload) >= 184: // логин 186/314
			if err := s.handleLogin(sess, payload); err != nil {
				s.send(ship.Event{Ev: ship.EvParseErr, Remote: remote, Err: err.Error(), Data: map[string]any{"stage": "login"}})
				return
			}
		default: // §5.2: dispatch по type — не вскрыт; кривой размер гасим
			s.send(ship.Event{Ev: ship.EvParseErr, Remote: remote, Data: map[string]any{"len": len(payload)}})
			sendCC(conn, 45)
			return
		}
	}
}

// handleRSAExchange: RSA-dec 32Б → X; ответ 42b = [frame][ECB(key2): X+pad+XOR-csum]
// (инверсия DecryptSecondary; точный состав 42b — TODO §5.5).
func (s *Server) handleRSAExchange(sess *Session, payload []byte) error {
	x, err := sess.RSA.DecryptBlock(payload)
	if err != nil {
		return err
	}
	return sess.write(proto.WriteFrame(encryptToClient(sess.BF2, x)))
}

// handleLogin: релей логина в authd (§5.3, дизasm 05.10 + 0x417b60):
// первые 128Б = RSA-блок (BE-число с ведущими нулями, модуль = наш ключ из
// welcome); rsapricrt → decbuf (32Б, BE, выровнен к началу); dword@148
// (флаг 0x80000000 — TODO); блок @152 = len-152 байт ("len-24" после RSA).
// TODO(§5.3): длина decbuf — 32Б (математика) или 34Б (arg3=0x22 у оригинала) —
// верифицировать живым клиентом/authd.
func (s *Server) handleLogin(sess *Session, data []byte) error {
	const rsaLen = 128
	if len(data) < rsaLen+24 {
		sendCCSess(sess, 45) // кривой логин — гасим (TODO §5.2: точная семантика)
		return nil
	}
	decbuf, err := sess.RSA.DecryptBlock(data[:rsaLen])
	if err != nil {
		return err
	}
	dword148 := binary.LittleEndian.Uint32(data[148:152])
	tail := data[152:]
	blob := proto.Assemble("cbdb", byte(0), decbuf, dword148, tail)
	s.withAuthd(func(a *authdclient.Client) { _ = a.SendPacket(sess.ID, blob) })
	s.send(ship.Event{Ev: "login", Remote: net.IP(sess.IP[:]).String(), Data: map[string]any{
		"sid": sess.ID, "decbuf": len(decbuf), "tail": len(tail),
	}})
	return nil
}

// onAuthdPacket — push от authd: тип 4 = serverlist (payload из capture, §3).
// TODO §5.5: точный формат 74b/26b ответов клиенту; сейчас шлём payload зашифрованным key2 всем.
func (s *Server) onAuthdPacket(id uint32, typ byte, payload []byte) {
	if typ != 4 {
		log.Printf("authd packet: id=%d type=%d len=%d (не serverlist — TODO §5.5)", id, typ, len(payload))
		s.send(ship.Event{Ev: "authd.pkt", Svc: "authd", Data: map[string]any{"id": id, "type": typ, "len": len(payload)}})
		return
	}
	s.mu.Lock()
	all := make([]*Session, 0, len(s.sess))
	for _, sess := range s.sess {
		all = append(all, sess)
	}
	s.mu.Unlock()
	for _, sess := range all {
		_ = sess.write(proto.WriteFrame(encryptToClient(sess.BF2, payload)))
	}
	s.send(ship.Event{Ev: "serverlist", Svc: "authd", Data: map[string]any{"sessions": len(all), "len": len(payload)}})
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

// encryptToClient — инверсия proto.DecryptSecondary: [payload + паддинг + XOR-csum] ECB(key2).
func encryptToClient(bf2 *proto.Blowfish, payload []byte) []byte {
	n := (len(payload) + 4 + 7) &^ 7
	buf := make([]byte, n)
	copy(buf, payload)
	nd := n / 4
	var x uint32
	for k := 0; k < nd-1; k++ {
		x ^= binary.LittleEndian.Uint32(buf[k*4:])
	}
	binary.LittleEndian.PutUint32(buf[(nd-1)*4:], x)
	out := make([]byte, n)
	for off := 0; off < n; off += 8 {
		bf2.Encrypt(out[off:off+8], buf[off:off+8])
	}
	return out
}

// sendCC — cc-отказ @0x407e80: Assemble("cc", 1, code) → [01][code] plaintext
// (22 = blocked IP, 45 = гейт не готов; точный wire — TODO §5.5).
func sendCC(conn net.Conn, code byte) {
	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, _ = conn.Write(proto.WriteFrame(proto.Assemble("cc", byte(1), code)))
}

func sendCCSess(sess *Session, code byte) {
	_ = sess.write(proto.WriteFrame(proto.Assemble("cc", byte(1), code)))
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

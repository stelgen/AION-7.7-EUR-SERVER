// Package world — листенер serverPort (2104): канал Server64 (мир).
//
// Фрейминг — из C1-сорцов WorldSrvSocket (L2Auth-chaospaladin, НЕ дизasm):
//   m_packetSize = buf[0] + buf[1]<<8 + 1 - packetSizeType (packetSizeType=2),
//   3-й байт = type; wire = [u16 X LE][type][payload], X = total-1.
//
// Живые факты (packet-лог L2Authd 09.10, authd-ref/logs-2104/, live-логин юзера):
//   - greeting при accept: [03][authVersion u32][1 u32] — C1 OnCreate Send("cdd",3,build,1);
//     live: Server64 логирует "Protocol Version authVersion:2017012601, protocolVersion:1";
//   - heartbeat 60с: authd→мир [02] (пусто), мир→authd [05][users u16][limit u16]
//     (0000f401: users=0, limit=500; 0100f401 после логина: users=1);
//   - релей логина: authd→мир [00][uid u32][account 16Б null-pad][maxUsers=2000]
//     [16×00]["0000000\0"][tail 59Б из live-корпуса]; ack мира [00][uid][02000000];
//   - события мира: 2 (userLoggedToGs, uid+10×00), 3 (uid+char_id+lev), 9 (updateUserData),
//     35 (uid+01+char_id+lev+unk), 27/24/25/38/39/40 (uid-пинги); C1-таблица хендлеров 0-21.
//
// Дисциплина: наш мир-канал НЕ шлёт квитанции по умолчанию (gsAcks=false) —
// включаются после fork-диффа 2104. Прод не трогаем: листенер выключен без gsPort.
package world

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"aion-authd/internal/ship"
)

// DefaultRelayTail — байты [49:107] type-0 релея из живого корпуса (09.10, логин юзера):
// unk dword 554da0b8 + zeros6 + ff×24 + zeros4 + 2×dword 50c2366b + zeros12 (58Б).
const DefaultRelayTail = "554da0b8" +
	"000000000000" +
	"ffffffffffffffffffffffffffffffffffffffffffffffff" +
	"00000000" + "50c2366b" + "50c2366b" +
	"000000000000000000000000"

// Cfg — параметры мир-канала (нулевые = безопасные дефолты).
type Cfg struct {
	Port         int    // gsPort: порт листенера (0 = выкл)
	AuthVersion  uint32 // greeting [03][V][1] (live 2017012601; для 2110-канала свой 0xc621)
	MaxUsers     uint32 // поле релея (live 2000)
	HeartbeatSec int    // период ping (live 60)
	Acks         bool   // шлать ли квитанции на события мира (T2, дефолт false)
	RelayTailHex string // tail type-0 релея [48:107] (default = живой корпус)
}

// S — мир-канал (одно живое подключение Server64 — netstat-факт).
type S struct {
	Cfg Cfg
	sh  *ship.S

	mu      sync.Mutex
	conn    net.Conn
	users   int // из type-5 (world status)
	limit   int
	online  bool
	chPong  chan struct{} // сигналы чтения (для тестов/heartbeat)
	OnLogin func(uid uint32, account string) // хук логина (сверсии сверху)
}

// New — создать канал (не слушает до Serve).
func New(cfg Cfg, sh *ship.S) *S {
	if cfg.HeartbeatSec <= 0 {
		cfg.HeartbeatSec = 60
	}
	if cfg.RelayTailHex == "" {
		cfg.RelayTailHex = DefaultRelayTail
	}
	return &S{Cfg: cfg, sh: sh, chPong: make(chan struct{}, 8)}
}

// Enabled — включён ли листенер.
func (s *S) Enabled() bool { return s.Cfg.Port > 0 }

// Serve — accept-цикл (блокирует). Смерть коннекта → ожидание нового (мир сам реконнектится).
func (s *S) Serve(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		prev := s.conn
		s.conn = conn
		s.online = true
		s.users, s.limit = 0, 0
		s.mu.Unlock()
		if prev != nil {
			_ = prev.Close()
		}
		log.Printf("world: connected %s (greeting authVersion=%d)", conn.RemoteAddr(), s.Cfg.AuthVersion)
		s.ev("world.conn.up", conn.RemoteAddr().String(), nil)
		go s.loop(conn)
	}
}

// NotifyLogin — релей логина в мир (type 0): вызывается при успешном type=3 на 2110.
func (s *S) NotifyLogin(uid uint32, account string) {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		log.Printf("world: relay uid=%d %q — мир не подключён (тишина)", uid, account)
		return
	}
	fr, err := RelayLogin(s.Cfg, uid, account)
	if err != nil {
		log.Printf("world: relay uid=%d: %v", uid, err)
		return
	}
	if _, err := conn.Write(fr); err != nil {
		log.Printf("world: relay write: %v", err)
		return
	}
	log.Printf("world: relay type=0 uid=%d %q (%dБ)", uid, account, len(fr))
	s.ev("world.relay", strconv.Itoa(int(uid)), map[string]any{"user": account, "len": len(fr)})
}

// Status — текущий снапшот (users/limit из type-5).
func (s *S) Status() (online bool, users, limit int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.online, s.users, s.limit
}

// --- wire ---

// Encode — [u16 X LE][type][payload], X = total-1 (C1 WorldSrvSocket).
func Encode(typ byte, payload []byte) []byte {
	x := 2 + len(payload) // X = total_frame - 1 = type+payload+1 (C1: assembled+packetSizeType-1)
	out := make([]byte, 0, x+1)
	var h [2]byte
	binary.LittleEndian.PutUint16(h[:], uint16(x))
	out = append(out, h[0], h[1], typ)
	out = append(out, payload...)
	return out
}

// Greeting — [03][authVersion u32 LE][1 u32 LE] (C1: Send("cdd", 3, build, 1)).
func Greeting(authVersion uint32) []byte {
	p := make([]byte, 8)
	binary.LittleEndian.PutUint32(p[0:4], authVersion)
	binary.LittleEndian.PutUint32(p[4:8], 1)
	return Encode(3, p)
}

// Ping — heartbeat authd→мир: [02], payload пустой (live).
func Ping() []byte { return Encode(2, nil) }

// RelayLogin — type-0 релей логина (107Б payload): [uid][account 16Б][maxUsers][17×00]
// ["0000000\0"][unk dword][6×00][ff×24][4×00][2×dword][12×00] — live-корпус 09.10.
func RelayLogin(cfg Cfg, uid uint32, account string) ([]byte, error) {
	tailHex := cfg.RelayTailHex
	if tailHex == "" {
		tailHex = DefaultRelayTail // live-корпус 09.10
	}
	tail, err := parseHex(tailHex)
	if err != nil {
		return nil, fmt.Errorf("relay tail: %w", err)
	}
	if len(account) > 16 {
		account = account[:16]
	}
	p := make([]byte, 0, 4+16+4+16+8+len(tail))
	var u [4]byte
	binary.LittleEndian.PutUint32(u[:], uid)
	p = append(p, u[:]...)
	name := make([]byte, 16)
	copy(name, account)
	p = append(p, name...)
	var m [4]byte
	binary.LittleEndian.PutUint32(m[:], cfg.MaxUsers)
	p = append(p, m[:]...)
	p = append(p, make([]byte, 17)...)
	p = append(p, '0', '0', '0', '0', '0', '0', '0', 0)
	p = append(p, tail...)
	return Encode(0, p), nil
}

func parseHex(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		return nil, fmt.Errorf("нечётная длина hex")
	}
	out := make([]byte, len(s)/2)
	for i := 0; i < len(out); i++ {
		hi := hexVal(s[2*i])
		lo := hexVal(s[2*i+1])
		if hi < 0 || lo < 0 {
			return nil, fmt.Errorf("bad hex at %d", 2*i)
		}
		out[i] = byte(hi<<4 | lo)
	}
	return out, nil
}

func hexVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

// ReadFrame — читает один фрейм [u16 X][type][payload].
func ReadFrame(conn net.Conn) (typ byte, payload []byte, err error) {
	var h [2]byte
	if _, err = readFull(conn, h[:]); err != nil {
		return
	}
	x := int(binary.LittleEndian.Uint16(h[:]))
	if x < 2 || x > 0x2000 { // минимум: type+1; лимит = C1 BUFFER_SIZE
		return 0, nil, fmt.Errorf("world: bad frame size %d", x)
	}
	body := make([]byte, x-1)
	if _, err = readFull(conn, body); err != nil {
		return
	}
	return body[0], body[1:], nil
}

func readFull(conn net.Conn, b []byte) (int, error) {
	n := 0
	for n < len(b) {
		nn, err := conn.Read(b[n:])
		if err != nil {
			return n, err
		}
		n += nn
	}
	return n, nil
}

// --- внутреннее ---

func (s *S) loop(conn net.Conn) {
	defer func() {
		s.mu.Lock()
		if s.conn == conn {
			s.conn = nil
			s.online = false
		}
		s.mu.Unlock()
		_ = conn.Close()
		s.ev("world.conn.down", conn.RemoteAddr().String(), nil) // C1: worldstatus status=0
		log.Printf("world: disconnected %s (мир сам реконнектится — live)", conn.RemoteAddr())
	}()

	go s.heartbeat(conn)
	if _, err := conn.Write(Greeting(s.Cfg.AuthVersion)); err != nil {
		return
	}
	for {
		typ, payload, err := ReadFrame(conn)
		if err != nil {
			return
		}
		s.dispatch(conn, typ, payload)
	}
}

// heartbeat — ping каждые HeartbeatSec (live 60с).
func (s *S) heartbeat(conn net.Conn) {
	t := time.NewTicker(time.Duration(s.Cfg.HeartbeatSec) * time.Second)
	defer t.Stop()
	for range t.C {
		s.mu.Lock()
		live := s.conn == conn
		s.mu.Unlock()
		if !live {
			return
		}
		if _, err := conn.Write(Ping()); err != nil {
			return
		}
	}
}

// dispatch — события мира (таблица C1 0-21 + живые 7.7 27-44).
func (s *S) dispatch(conn net.Conn, typ byte, payload []byte) {
	switch typ {
	case 0: // play-ok ack мира: [uid][02000000]
		log.Printf("world: play-ok ack (%dБ)", len(payload))
		s.ev("world.ack", "0", map[string]any{"len": len(payload)})
	case 5: // world status: [users u16][limit u16]
		if len(payload) >= 4 {
			s.mu.Lock()
			s.users = int(binary.LittleEndian.Uint16(payload[0:2]))
			s.limit = int(binary.LittleEndian.Uint16(payload[2:4]))
			s.mu.Unlock()
			log.Printf("world: status users=%d limit=%d", s.users, s.limit)
		}
		select { case s.chPong <- struct{}{}: default: }
	case 2: // userLoggedToGs: [uid]...
		if len(payload) >= 4 {
			uid := binary.LittleEndian.Uint32(payload[0:4])
			log.Printf("world: user logged to GS uid=%d", uid)
			s.ev("world.user.logged", strconv.Itoa(int(uid)), nil)
		}
	default: // 3/9/13-44/… — события/квитанции: лог; ответ по флагу Acks (T2)
		log.Printf("world: event type=%d len=%d — лог%s", typ, len(payload), ackSuffix(s.Cfg.Acks))
		if s.Cfg.Acks && len(payload) >= 4 {
			uid := payload[0:4]
			frame := Encode(ackType(typ), append([]byte{}, uid...))
			_, _ = conn.Write(frame)
		}
		s.ev("world.event", strconv.Itoa(int(typ)), map[string]any{"len": len(payload)})
	}
}

// ackType — тип квитанции authd по живому корпусу (W→A → A→W): 35→31, 3→14, 27→19, 25→16, 24→13, 38/39/40→44.
func ackType(w byte) byte {
	switch w {
	case 35:
		return 31
	case 3:
		return 14
	case 27:
		return 19
	case 25:
		return 16
	case 24:
		return 13
	default:
		return 44
	}
}

func ackSuffix(acks bool) string {
	if acks {
		return " +квитанция"
	}
	return " (квитанции off)"
}

func (s *S) ev(name, remote string, data map[string]any) {
	if s.sh != nil {
		s.sh.Send(ship.Event{Ev: name, Svc: "world", Remote: remote, Data: data})
	}
}

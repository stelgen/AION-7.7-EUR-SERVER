// Package world — листенер serverPort (2104): канал Server64 (мир).
//
// Фрейминг — из C1-сорцов WorldSrvSocket (L2Auth-chaospaladin, НЕ дизasm):
//
//	m_packetSize = buf[0] + buf[1]<<8 + 1 - packetSizeType (packetSizeType=2),
//	3-й байт = type; wire = [u16 X LE][type][payload], X = total-1.
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

// DefaultRelayTail — байты [53:107] type-0 релея (после IP-дворда!): 04.10/09.10 корпуса 1-в-1:
// zeros2 + ff×24 + zeros4 + 2×константа 50c2366b + zeros16 (54Б).
// IP-дворд [49:53] = u32 LE от inet_addr(клиента) (04.10: 7f000001=127.0.0.1; 09.10: 554da0b8).
const DefaultRelayTail = "000000000000" +
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
	RawLog       bool   // RAW-hex лог мира (диаг; вкл. gsRawLog)
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
	chPong  chan struct{}                    // сигналы чтения (для тестов/heartbeat)
	OnLogin func(uid uint32, account string) // хук (резерв)
	// OnPlayAck — мир подтвердил play (W→A type=0 ack [uid][N]): pk1=N для type=7
	// (канон 09.10: три логина 2→2/4→4/8→8 — точное эхо ack-dword).
	OnPlayAck func(uid, pk1 uint32)
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

// RelayPlay — релей ВХОДА В МИР (type 0, 107Б): вызывается на CM_PLAY (op=0x02),
// НЕ на логин! (live 09.10: A→W type=0 идёт после Gate→Auth type=2/play; ack мира → type=7.)
func (s *S) RelayPlay(uid uint32, account, clientIP string) {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		log.Printf("world: relay uid=%d %q — мир не подключён (тишина)", uid, account)
		return
	}
	fr, err := RelayLogin(s.Cfg, uid, account, clientIP)
	if err != nil {
		log.Printf("world: relay uid=%d: %v", uid, err)
		return
	}
	if _, err := conn.Write(fr); err != nil {
		log.Printf("world: relay write: %v", err)
		return
	}
	log.Printf("world: relay type=0 uid=%d %q ip=%s (%dБ)", uid, account, clientIP, len(fr))
	s.ev("world.relay", strconv.Itoa(int(uid)), map[string]any{"user": account, "len": len(fr)})
}

// Status — текущий снапшот (users/limit из type-5).
func (s *S) Status() (online bool, users, limit int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.online, s.users, s.limit
}

// --- wire ---

// Encode — [u16 X LE][type][payload], X = body+2 (= type+payload+2) — ТА ЖЕ формула, что
// на 2110 (самоинклюзивный len=body+2); C1: m_packetSize = X+1-packetSizeType, packetSizeType=3.
func Encode(typ byte, payload []byte) []byte {
	x := 3 + len(payload) // X = type+payload+2
	out := make([]byte, 0, x+1)
	var h [2]byte
	binary.LittleEndian.PutUint16(h[:], uint16(x))
	out = append(out, h[0], h[1], typ)
	out = append(out, payload...)
	return out
}

// Greeting — канон ОРИГ-лога 04.10 (Auth->World,3: 792b39780100000000 — 9Б payload):
// [03][authVersion u32 LE][1 u32 LE][0x00] — хвостовой ноль ОБЯЗАТЕЛЕН (без него Server64
// молчит: его парсер ждёт 9-й байт — корень отказа R6-логина 11:07).
func Greeting(authVersion uint32) []byte {
	p := make([]byte, 9)
	binary.LittleEndian.PutUint32(p[0:4], authVersion)
	binary.LittleEndian.PutUint32(p[4:8], 1)
	p[8] = 0x00
	return Encode(3, p)
}

// Ping — heartbeat authd→мир: [02], payload пустой (live).
func Ping() []byte { return Encode(2, nil) }

// RelayLogin — type-0 релей play (107Б payload, канон 04.10/09.10 корпус 1-в-1):
// [uid][account 16Б][maxUsers=2000][17×00]["0000000\0"][IP-дворд = u32 LE от
// inet_addr(клиента)][zeros2][ff×24][zeros4][2×конст 50c2366b][zeros12].
func RelayLogin(cfg Cfg, uid uint32, account, clientIP string) ([]byte, error) {
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
	p := make([]byte, 0, 107)
	var u [4]byte
	binary.LittleEndian.PutUint32(u[:], uid)
	p = append(p, u[:]...)
	name := make([]byte, 20)
	copy(name, account)
	p = append(p, name...)
	var m [4]byte
	binary.LittleEndian.PutUint32(m[:], cfg.MaxUsers)
	p = append(p, m[:]...)
	p = append(p, make([]byte, 13)...)
	p = append(p, '0', '0', '0', '0', '0', '0', '0', 0)
	// IP-дворд [49:53]: реверс-октеты клиента (09.10 корпус: клиент 184.160.77.85 → 554da0b8).
	// ⚠ Противоречие с 04.10 (7f000001 «direct» для 127.0.0.1) — T2: сверить last_ip в БД на живом логине.
	var ipb [4]byte
	if a, bb, c, d := parseIP(clientIP); a >= 0 {
		ipb = [4]byte{byte(d), byte(c), byte(bb), byte(a)}
	} else {
		ipb = [4]byte{0x01, 0x00, 0x00, 0x7f} // fallback 127.0.0.1 (реверс)
	}
	p = append(p, ipb[0], ipb[1], ipb[2], ipb[3])
	p = append(p, tail...)
	return Encode(0, p), nil
}

// parseIP — «a.b.c.d» → октеты; -1 если не распарсилось.
func parseIP(s string) (int, int, int, int) {
	var a, b, c, d int
	if _, err := fmt.Sscanf(s, "%d.%d.%d.%d", &a, &b, &c, &d); err != nil {
		return -1, -1, -1, -1
	}
	return a, b, c, d
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
	if x < 3 || x > 0x2000 { // минимум: type+2; лимит = C1 BUFFER_SIZE
		return 0, nil, fmt.Errorf("world: bad frame size %d", x)
	}
	body := make([]byte, x-2) // type+payload = X-2
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

	greet := Greeting(s.Cfg.AuthVersion)
	if s.Cfg.RawLog {
		log.Printf("world RAW > %x", greet)
	}
	go s.heartbeat(conn)
	if _, err := conn.Write(greet); err != nil {
		return
	}
	buf := make([]byte, 0, 8192)
	tmp := make([]byte, 4096)
	for {
		n, err := conn.Read(tmp)
		if n > 0 {
			if s.Cfg.RawLog {
				log.Printf("world RAW < %x", tmp[:n])
			}
			buf = append(buf, tmp[:n]...)
			for len(buf) >= 3 {
				x := int(binary.LittleEndian.Uint16(buf[:2]))
				if x < 3 || x > 0x2000 {
					log.Printf("world: BAD SIZE %d (buf %dБ): %x", x, len(buf), buf[:min(64, len(buf))])
					buf = nil
					break
				}
				if len(buf) < x+2 {
					break
				}
				typ := buf[2]
				payload := append([]byte{}, buf[3:x+2]...)
				buf = buf[x+2:]
				s.dispatch(conn, typ, payload)
			}
			if buf == nil {
				buf = make([]byte, 0, 8192)
			}
		}
		if err != nil {
			return
		}
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
	case 0: // play-ok ack мира: [uid][N u32] → type=7 клиенту с pk1=N (канон 09.10)
		log.Printf("world: play-ok ack (%dБ)", len(payload))
		s.ev("world.ack", "0", map[string]any{"len": len(payload)})
		if len(payload) >= 8 && s.OnPlayAck != nil {
			uid := binary.LittleEndian.Uint32(payload[0:4])
			pk1 := binary.LittleEndian.Uint32(payload[4:8])
			log.Printf("world: ack uid=%d pk1=%d → type=7", uid, pk1)
			go s.OnPlayAck(uid, pk1)
		}
	case 5: // world status: [users u16][limit u16]
		if len(payload) >= 4 {
			s.mu.Lock()
			s.users = int(binary.LittleEndian.Uint16(payload[0:2]))
			s.limit = int(binary.LittleEndian.Uint16(payload[2:4]))
			s.mu.Unlock()
			log.Printf("world: status users=%d limit=%d", s.users, s.limit)
		}
		select {
		case s.chPong <- struct{}{}:
		default:
		}
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

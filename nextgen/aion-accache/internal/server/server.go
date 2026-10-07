// Package server: accept-loop + per-conn state-машина (порт AC_Socket::OnRead)
// + dispatch по таблицам (cmd -> handler). Каркас R2: БЕЗ capture —
// payload-раскладки TBD, ответы-ACP: номера TBD (эхо-cmd, помечено).
package server

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"aion-accache/internal/cache"
	"aion-accache/internal/config"
	"aion-accache/internal/db"
	"aion-accache/internal/dispatch"
	"aion-accache/internal/proto"
	"aion-accache/internal/ship"
)

// Handler — обработчик команды (payload TBD до R1).
type Handler func(s *S, c *Conn, cmd uint16, payload []byte)

// S — сервер.
type S struct {
	cfg    *config.Cfg
	cache  *cache.Store
	exec   db.Executor // nil если db.enabled=false
	sh     *ship.S
	handlers map[uint16]Handler
	mu     sync.Mutex
	ln     net.Listener
}

func New(cfg *config.Cfg, st *cache.Store, exec db.Executor, sh *ship.S) *S {
	s := &S{cfg: cfg, cache: st, exec: exec, sh: sh, handlers: map[uint16]Handler{}}
	s.registerDefaults()
	return s
}

// SetHandler — замена/инъекция обработчика (для тестов и доработки).
func (s *S) SetHandler(cmd uint16, h Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[cmd] = h
}

func (s *S) registerDefaults() {
	s.SetHandler(0x01, handleVersion)          // ACQ_VERSION_PACKET
	s.SetHandler(0x04, handleFirstLoad)        // ACQ_FIRST_LOAD_ACCOUNT_INFO
	s.SetHandler(0x09, handleSyncTest)         // ACQ_SYNC_PACKET_TEST (эхо)
	s.SetHandler(0x19, handleUpdateFatigue)    // ACQ_UPDATE_HIDDEN_FATIGUE
}

func (s *S) send(sh *ship.S, ev, svc, msg string) {
	if sh != nil {
		sh.Send(ship.Event{Ev: ev, Svc: svc, Msg: msg})
	}
}

// Serve — блокирующий accept-loop.
func (s *S) Serve() error {
	ln, err := net.Listen("tcp", s.cfg.Server.Listen)
	if err != nil {
		return err
	}
	s.ln = ln
	log.Printf("aion-accache: skeleton R2 listening %s (dispatcher T1 cmds 0..39, DB=%v)", s.cfg.Server.Listen, s.cfg.DB.Enabled)
	s.send(s.sh, "start", "accache", "listen "+s.cfg.Server.Listen)
	return s.serveOn(ln)
}

// SetListenerForTest / ServeOnTest — тестовый вход (уже открытый listener).
func (s *S) SetListenerForTest(ln net.Listener) { s.ln = ln }

func (s *S) ServeOnTest() error { return s.serveOn(s.ln) }

func (s *S) serveOn(ln net.Listener) error {
	for {
		c, err := ln.Accept()
		if err != nil {
			return err
		}
		go s.handleConn(c)
	}
}

func (s *S) Close() {
	if s.ln != nil {
		_ = s.ln.Close()
	}
}

// Conn — соединение с клиентом (Server64).
type Conn struct {
	net.Conn
	s    *S
	t1   bool // true = основной канал (T1)
}

func (s *S) handleConn(nc net.Conn) {
	defer nc.Close()
	c := &Conn{Conn: nc, s: s, t1: true}
	defer s.send(s.sh, "conn.down", "accache", nc.RemoteAddr().String())
	s.send(s.sh, "conn.up", "accache", nc.RemoteAddr().String())
	rd := proto.NewReader(nc)
	for {
		_ = nc.SetReadDeadline(time.Now().Add(10 * time.Minute))
		f, err := rd.ReadFrame()
		if err != nil {
			if err != io.EOF && !isTimeout(err) {
				s.logf("parse.err cmd=? err=%v", err)
			}
			return
		}
		s.dispatch(c, f)
	}
}

func isTimeout(err error) bool {
	ne, ok := err.(net.Error)
	return ok && ne.Timeout()
}

func (s *S) logf(format string, a ...any) {
	if s.cfg.Server.Verbose {
		log.Printf("accache: "+format, a...)
	}
	if s.cfg.Server.CaptureAll && s.cfg.Server.BaseDir != "" {
		// сырое hex — как logd (WriteIO); каркас: только в лог
	}
}

// dispatch — маршрутизация по T1 (каркас: один канал; T2-семантика = R1).
func (s *S) dispatch(c *Conn, f *proto.Frame) {
	name := dispatch.Name(true, f.Cmd)
	if s.cfg.Server.Verbose && len(f.Payload) > 0 {
		log.Printf("accache: recv %s (%#x) payload=%d %s", name, f.Cmd, len(f.Payload), hexPreview(f.Payload))
	}
	s.mu.Lock()
	h, ok := s.handlers[f.Cmd]
	s.mu.Unlock()
	if !ok {
		// Дефолт-поведение оригинала: дроп-хендлер 0x14006c050 (без ответа) — каркас так же.
		s.logf("unhandled %s (%#x) len=%d — dropped (default handler)", name, f.Cmd, len(f.Payload))
		s.send(s.sh, "unhandled", "accache", fmt.Sprintf("%s cmd=%#x len=%d", name, f.Cmd, len(f.Payload)))
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = ctx
	h(s, c, f.Cmd, f.Payload)
}

func hexPreview(b []byte) string {
	const max = 32
	if len(b) > max {
		b = b[:max]
	}
	return hex.EncodeToString(b)
}

// Respond — ответ клиенту. ⚠ ACP-номера TBD (R1 capture): каркас эхо-cmd с пометкой.
func (s *S) Respond(c *Conn, cmd uint16, payload []byte) {
	if err := proto.WriteFrame(c.Conn, cmd, payload); err != nil {
		s.logf("respond.err cmd=%#x err=%v", cmd, err)
	}
}

// --- каркасные хендлеры (payload-раскладка TBD; семантика из dispatch-77.md) ---

func handleVersion(s *S, c *Conn, cmd uint16, payload []byte) {
	// Ориг: SendIOBuffer + CheckGlobalUserInServer. ACP-cmd TBD; каркас отвечает эхо.
	s.Respond(c, cmd, nil)
}

func handleFirstLoad(s *S, c *Conn, cmd uint16, payload []byte) {
	// payload TBD (accountId первым dword — вероятная раскладка; R1 подтвердит).
	acc := readIntLE(payload, 0)
	var fat *db.Fatigue
	if s.exec != nil {
		f, err := db.GetAccountData(context.Background(), s.exec, acc)
		if err != nil {
			s.logf("firstload db.err acc=%d err=%v", acc, err)
		}
		fat = f
	}
	if fat == nil {
		if e, ok := s.cache.Get(acc); ok {
			fat = &db.Fatigue{Point: e.Fatigue.Point, UpdateTime: e.Fatigue.UpdateTime, NpcKill: e.Fatigue.NpcKill, LimitReset: e.Fatigue.LimitReset, LimitAccum: e.Fatigue.LimitAccum}
		} else {
			fat = &db.Fatigue{} // пустой result-set как у ориг при отсутствии записи
		}
	}
	// Ответ: EncodeFirstLoadAccountInfo_AddArg = 5×int (та же пятерка полей) — ACP-cmd TBD.
	buf := make([]byte, 20)
	putIntLE(buf, 0, fat.Point)
	putIntLE(buf, 4, fat.UpdateTime)
	putIntLE(buf, 8, fat.NpcKill)
	putIntLE(buf, 12, fat.LimitReset)
	putIntLE(buf, 16, fat.LimitAccum)
	s.Respond(c, cmd, buf) // TODO(R1): реальный ACP-cmd
}

func handleSyncTest(s *S, c *Conn, cmd uint16, payload []byte) {
	s.Respond(c, cmd, payload) // эхо-тест
}

func handleUpdateFatigue(s *S, c *Conn, cmd uint16, payload []byte) {
	// DecodeUpdateHiddenFatigue(..., I(accId), H, I, I, I) — каркас: accId@0, point/upd/npckill.
	acc := readIntLE(payload, 0)
	s.cache.UpdateFatigue(acc, int32(readIntLE(payload, 4)), int32(readIntLE(payload, 8)), int32(readIntLE(payload, 12)))
	s.Respond(c, cmd, nil) // TODO(R1): реальный ACP-cmd
}

func readIntLE(b []byte, off int) int {
	if len(b) < off+4 {
		return 0
	}
	return int(b[off]) | int(b[off+1])<<8 | int(b[off+2])<<16 | int(b[off+3])<<24
}

func putIntLE(b []byte, off int, v int32) {
	if off+4 > len(b) {
		return
	}
	b[off] = byte(v)
	b[off+1] = byte(v >> 8)
	b[off+2] = byte(v >> 16)
	b[off+3] = byte(v >> 24)
}

// Package server — authd: листенер 2110, коннекты гейта, диспетчер wire.
//
// Живой контракт (aion-gate internal/authdclient + probe 07.10):
//   - greeting [03][V] при accept;
//   - [00] регистрирует сессию; [01] закрывает (флаг онлайн НЕ снимается — live);
//   - [02][sid][len][blob]: blob[0] = опкод (0x00 login, 0x05 serverlist, 0x02 play,
//     0x08 update-session); для НЕизвестного sid → [01][sid] (negative-ack);
//   - ответы: [02][id][len][type][payload]; незнакомые опкоды = лог+тишина (НЕ рвать —
//     гейт шлёт только известные, а unknown-классы ждём на R5-диффе).
package server

import (
	"context"
	"errors"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"aion-authd/internal/config"
	"aion-authd/internal/logic"
	"aion-authd/internal/ship"
	"aion-authd/internal/store"
	"aion-authd/internal/wire"
	"aion-authd/internal/world"
)

// Server — authd.
type Server struct {
	Cfg *config.Config
	sh  *ship.S
	st  store.Store
	L   *logic.Deps
	W   *world.S // мир-канал 2104 (nil/выкл = не слать релеи)

	mu    sync.Mutex
	gates map[net.Conn]*gateConn
}

// gateConn — один коннект гейта (ориг держит мульти-гейт — живой факт fork-режима).
type gateConn struct {
	srv    *Server
	conn   net.Conn
	remote string

	wmu sync.Mutex
	ses map[uint32]*logic.GateSession
}

// New — создать сервер (Store закрывает вызывающий).
func New(cfg *config.Config, sh *ship.S, st store.Store) *Server {
	return &Server{Cfg: cfg, sh: sh, st: st, L: logic.New(cfg, st), gates: map[net.Conn]*gateConn{}}
}

// Serve — accept-цикл (блокирует).
func (s *Server) Serve(ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		g := &gateConn{srv: s, conn: conn, remote: conn.RemoteAddr().String(), ses: map[uint32]*logic.GateSession{}}
		s.mu.Lock()
		s.gates[conn] = g
		s.mu.Unlock()
		s.send(ship.Event{Ev: ship.EvConnUp, Svc: "gate", Remote: g.remote})
		log.Printf("gate: connected %s (greeting V=%#x)", g.remote, s.Cfg.AuthVersion)
		if _, err := conn.Write(wire.Greeting(s.Cfg.AuthVersion)); err != nil {
			s.dropGate(g)
			continue
		}
		go g.readLoop()
	}
}

// Close — закрыть все коннекты гейтов.
func (s *Server) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for c := range s.gates {
		_ = c.Close()
	}
}

// SendPlayOK — колбэк от мира: ack (uid, pk1) → type=7 сессии с этим AccID (канон 09.10: pk1 = эхо ack).
func (s *Server) SendPlayOK(uid, pk1 uint32) {
	s.mu.Lock()
	var target *gateConn
	var ses *logic.GateSession
	for _, g := range s.gates {
		for _, ss := range g.ses {
			if ss.AccID == uid {
				target, ses = g, ss
			}
		}
	}
	s.mu.Unlock()
	if target == nil {
		log.Printf("play-ok: сессия uid=%d не найдена (гейт отвалился?)", uid)
		return
	}
	target.reply(ses, &logic.Reply{Typ: 7, Payload: logic.BuildType7Pk1(pk1, ses.AccID, s.Cfg.ServerID)})
}

// dropGate — один коннект гейта отвалился.
func (s *Server) dropGate(g *gateConn) {
	s.mu.Lock()
	if _, ok := s.gates[g.conn]; ok {
		delete(s.gates, g.conn)
	}
	s.mu.Unlock()
	_ = g.conn.Close()
	s.send(ship.Event{Ev: ship.EvConnDown, Svc: "gate", Remote: g.remote})
	log.Printf("gate: disconnected %s", g.remote)
}

func (s *Server) send(ev ship.Event) {
	if s.sh != nil {
		s.sh.Send(ev)
	}
}

// RunSweeper — чистка онлайн-флагов по sweepSec (до ctx.Done).
func (s *Server) RunSweeper(ctx context.Context) {
	t := time.NewTicker(time.Duration(s.Cfg.SweepSec) * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if n := s.L.SweepOnline(); n > 0 {
				log.Printf("sweep: снято онлайн-флагов: %d (осталось %d)", n, s.L.OnlineCount())
				s.send(ship.Event{Ev: ship.EvSweep, Svc: "online", Msg: strconv.Itoa(n),
					Data: map[string]any{"expired": n, "online": s.L.OnlineCount()}})
			}
		}
	}
}

// readLoop — парсер потока гейта.
func (g *gateConn) readLoop() {
	defer g.srv.dropGate(g)
	for {
		f, err := wire.ReadFrame(g.conn)
		if err != nil {
			log.Printf("wire: %s read: %v", g.remote, err)
			if !isNetTimeout(err) {
				g.srv.send(ship.Event{Ev: ship.EvParseErr, Svc: "gate", Remote: g.remote, Err: err.Error()})
			}
			return
		}
		g.dispatch(f)
	}
}

func isNetTimeout(err error) bool {
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

// dispatch — маршрутизация фрейма.
func (g *gateConn) dispatch(f wire.Frame) {
	switch f.Type {
	case wire.FConnect:
		g.ses[f.Sid] = &logic.GateSession{Sid: f.Sid, IP: f.IP}
		log.Printf("connect: sid=%d ip=%d.%d.%d.%d (сессий %d)", f.Sid, f.IP[0], f.IP[1], f.IP[2], f.IP[3], len(g.ses))
	case wire.FDisconnect:
		if ses, ok := g.ses[f.Sid]; ok {
			if ses.User != "" && g.srv.Cfg.ClearOnlineOnDisconnect {
				g.srv.L.ClearOnline(ses.User)
			}
			delete(g.ses, f.Sid)
			log.Printf("disconnect: sid=%d user=%q (live: онлайн-флаг НЕ снимается, TTL)", f.Sid, ses.User)
		} else {
			log.Printf("disconnect: sid=%d — неизвестная сессия", f.Sid)
		}
	case wire.FPacket:
		ses, ok := g.ses[f.Sid]
		if !ok {
			log.Printf("packet: sid=%d БЕЗ CltConnect → [01][sid] (negative-ack, probe-live)", f.Sid)
			g.write(wire.UnknownSession(f.Sid))
			return
		}
		g.dispatchBlob(ses, f.Blob)
	default:
		log.Printf("wire: неожиданный тип %02x — игнор", f.Type)
	}
}

// dispatchBlob — опкод = blob[0] (клиентский опкод релеится гейтом 1-в-1).
func (g *gateConn) dispatchBlob(ses *logic.GateSession, blob []byte) {
	if len(blob) < 1 {
		log.Printf("packet: sid=%d пустой blob", ses.Sid)
		return
	}
	op := blob[0]
	var rep *logic.Reply
	switch op {
	case 0x00: // CM_LOGIN-релей ("cbdb"-blob)
		rep = g.srv.L.Login(ses, blob, net.IP(ses.IP[:]).String())
	case 0x05: // CM_SERVER_LIST
		rep = g.srv.L.ServerList(ses)
	case 0x02: // CM_PLAY
		if g.srv.W != nil && g.srv.W.Enabled() { // R6-путь: relay в мир (type0 107Б) → ack → type=7 pk1=ack (канон 09.10)
			g.srv.W.RelayPlay(ses.AccID, ses.User, net.IP(ses.IP[:]).String())
			g.srv.send(ship.Event{Ev: "world.play.relay", Svc: "authd", Data: map[string]any{"uid": ses.AccID}})
			return // type=7 придёт через OnPlayAck
		}
		rep = g.srv.L.Play(ses)
	case 0x08: // CM_UPDATE_SESSION — T3, контракт ответа не снят: лог+тишина
		log.Printf("update-session: sid=%d len=%d — лог+тишина (T3)", ses.Sid, len(blob))
		return
	default: // live-паритет: неизвестное = лог, НЕ рвать
		log.Printf("unknown: sid=%d op=0x%02x len=%d — лог+тишина", ses.Sid, op, len(blob))
		g.srv.send(ship.Event{Ev: "authd.unknown", Svc: "gate",
			Data: map[string]any{"sid": ses.Sid, "op": op, "len": len(blob)}})
		return
	}
	if rep == nil {
		g.srv.send(ship.Event{Ev: "authd.silent", Svc: "gate",
			Data: map[string]any{"sid": ses.Sid, "op": op}})
		return
	}
	g.reply(ses, rep)
}

func (g *gateConn) reply(ses *logic.GateSession, rep *logic.Reply) {
	fr, err := wire.ReplyPkt(ses.Sid, rep.Typ, rep.Payload)
	if err != nil {
		log.Printf("reply: sid=%d type=%d: %v", ses.Sid, rep.Typ, err)
		return
	}
	g.write(fr)
	if rep.Close { // ориг после фейла шлёт [01][sid] (закрытие сессии) — R5-дифф 07.10
		g.write(wire.UnknownSession(ses.Sid))
		delete(g.ses, ses.Sid)
	}
	g.srv.send(ship.Event{Ev: "authd.reply", Svc: "gate",
		Data: map[string]any{"sid": ses.Sid, "type": rep.Typ, "len": len(rep.Payload)}})
}

func (g *gateConn) write(b []byte) {
	g.wmu.Lock()
	defer g.wmu.Unlock()
	_ = g.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if _, err := g.conn.Write(b); err != nil {
		log.Printf("wire: write %s: %v", g.remote, err)
	}
}

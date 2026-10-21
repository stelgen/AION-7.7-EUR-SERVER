// Package server: accept-loop + per-conn state-машина (порт AC_Socket::OnRead)
// + dispatch по T1. R3 (21.10): хендлеры наблюдённых cmds с раскладками payload
// (internal/payload) + SQL-вызовы по proc-телам (internal/db).
// Политика ответов (capture-канон): ответ ТОЛЬКО там, где пара доказана capture
// (1↔1, 4↔4, 5↔5, 16↔16, 26↔26+27); остальные — тишина (как ориг).
// Ненаблюдённые cmds — дроп+лог (дефолт-хендлер 0x14006c050 у ориг тоже дропает).
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
	"aion-accache/internal/payload"
	"aion-accache/internal/proto"
	"aion-accache/internal/ship"
)

// Handler — обработчик команды.
type Handler func(s *S, c *Conn, cmd uint16, pl []byte)

// S — сервер.
type S struct {
	cfg      *config.Cfg
	cache    *cache.Store
	exec     db.Executor // nil если db.enabled=false
	sh       *ship.S
	handlers map[uint16]Handler
	mu       sync.Mutex
	ln       net.Listener
}

func New(cfg *config.Cfg, st *cache.Store, exec db.Executor, sh *ship.S) *S {
	s := &S{cfg: cfg, cache: st, exec: exec, sh: sh, handlers: map[uint16]Handler{}}
	s.registerDefaults()
	return s
}

// SetHandler — замена/инъекция обработчика (тесты/доработка).
func (s *S) SetHandler(cmd uint16, h Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[cmd] = h
}

func (s *S) registerDefaults() {
	s.SetHandler(0x01, hVersion)         // ACQ_VERSION_PACKET
	s.SetHandler(0x02, hTestInsert)      // ACQ_TEST_INSERT -> aion_insert_test
	s.SetHandler(0x03, hTestSelect)      // ACQ_TEST_SELECT -> aion_select_test
	s.SetHandler(0x04, hFirstLoad)       // ACQ_FIRST_LOAD_ACCOUNT_INFO
	s.SetHandler(0x05, hLoadBMPack)      // ACQ_LOAD_BM_PACK_LIST
	s.SetHandler(0x07, hLoadTrial)       // ACQ_LOAD_TRIAL_ACCOUNT_DATA
	s.SetHandler(0x08, hUpdateTrial)     // ACQ_UPDATE_TRIAL_ACCOUNT_DATA
	s.SetHandler(0x09, hSyncTest)        // ACQ_SYNC_PACKET_TEST (эхо)
	s.SetHandler(0x0e, hLogInfo)         // ACQ_LOG_INFO
	s.SetHandler(0x10, hCharLogin)       // ACQ_CHAR_LOGIN
	s.SetHandler(0x11, hCharLogout)      // ACQ_CHAR_LOGOUT
	s.SetHandler(0x19, hUpdateFatigue)   // ACQ_UPDATE_HIDDEN_FATIGUE
	s.SetHandler(0x1a, hLoadLuna)        // ACQ_LOAD_LUNA
	s.SetHandler(0x1f, hAskJumping)      // ACQ_ASK_CAN_MAKE_JUMPING_CHARACTER
	s.SetHandler(0x25, hTransform)       // ACQ_TRANSFORM_OPERATION
}

func (s *S) send(ev, svc, msg string) {
	if s.sh != nil {
		s.sh.Send(ship.Event{Ev: ev, Svc: svc, Msg: msg})
	}
}

// Serve — блокирующий accept-loop.
func (s *S) Serve() error {
	ln, err := net.Listen("tcp", s.cfg.Server.Listen)
	if err != nil {
		return err
	}
	s.ln = ln
	log.Printf("aion-accache: R3 listening %s (T1 handlers 16, DB=%v)", s.cfg.Server.Listen, s.cfg.DB.Enabled)
	s.send("start", "accache", "listen "+s.cfg.Server.Listen+" db="+fmt.Sprint(s.cfg.DB.Enabled))
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
	s *S
}

func (s *S) handleConn(nc net.Conn) {
	defer nc.Close()
	c := &Conn{Conn: nc, s: s}
	defer s.send("conn.down", "accache", nc.RemoteAddr().String())
	s.send("conn.up", "accache", nc.RemoteAddr().String())
	rd := proto.NewReader(nc)
	for {
		_ = nc.SetReadDeadline(time.Now().Add(10 * time.Minute))
		f, err := rd.ReadFrame()
		if err != nil {
			if err != io.EOF && !isTimeout(err) {
				s.logf("parse.err cmd=? err=%v", err)
				s.send("parse.err", "accache", err.Error())
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
}

// dispatch — маршрутизация по T1.
func (s *S) dispatch(c *Conn, f *proto.Frame) {
	name := dispatch.Name(true, f.Cmd)
	if s.cfg.Server.Verbose && len(f.Payload) > 0 {
		log.Printf("accache: recv %s (%#x) payload=%d %s", name, f.Cmd, len(f.Payload), hexPreview(f.Payload))
	}
	s.mu.Lock()
	h, ok := s.handlers[f.Cmd]
	s.mu.Unlock()
	if !ok {
		// Дефолт-поведение оригинала: дроп-хендлер 0x14006c050 (без ответа) + лог.
		s.logf("unhandled %s (%#x) len=%d — dropped (default handler)", name, f.Cmd, len(f.Payload))
		s.send("unhandled", "accache", fmt.Sprintf("%s cmd=%#x len=%d", name, f.Cmd, len(f.Payload)))
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

// respond — ответ клиенту (маркер 0xEC).
func (s *S) respond(c *Conn, cmd uint16, pl []byte) {
	if err := proto.WriteFrameS2C(c.Conn, cmd, pl); err != nil {
		s.logf("respond.err cmd=%#x err=%v", cmd, err)
	}
}

// --- хендлеры R3 (раскладки = capture-канон; SQL = proc-тела REF58) ---

// 0x01 VERSION: C2S {3,1} -> S2C {3}. Пара доказана capture.
func hVersion(s *S, c *Conn, cmd uint16, pl []byte) {
	if r, err := payload.ParseVersionReq(pl); err == nil {
		s.logf("version A=%d B=%d", r.A, r.B)
	}
	s.respond(c, cmd, payload.BuildVersionResp())
}

// 0x02/0x03 TEST_INSERT/SELECT: aion_insert_test / aion_select_test (тела REF58).
func hTestInsert(s *S, c *Conn, cmd uint16, pl []byte) {
	if s.exec == nil {
		return
	}
	if _, _, err := db.CallProc(context.Background(), s.exec, "aion_insert_test"); err != nil {
		s.logf("testinsert db.err %v", err)
		s.send("db.err", "accache", err.Error())
	}
}

func hTestSelect(s *S, c *Conn, cmd uint16, pl []byte) {
	if s.exec == nil {
		return
	}
	rows, _, err := db.CallProc(context.Background(), s.exec, "aion_select_test")
	if err != nil {
		s.logf("testselect db.err %v", err)
		return
	}
	if rows != nil {
		rows.Close()
	}
}

// 0x04 FIRST_LOAD: C2S [accId] -> S2C [accId][point][npckill][limitReset u8][ts][limitAccum u16].
// Пара доказана capture; роли полей = гипотеза (журнал теорий ROADMAP, арбитр R4).
func hFirstLoad(s *S, c *Conn, cmd uint16, pl []byte) {
	req, err := payload.ParseFirstLoadReq(pl)
	if err != nil {
		s.logf("firstload parse.err %v", err)
		return
	}
	var fat *db.Fatigue
	if s.exec != nil {
		f, err := db.GetAccountData(context.Background(), s.exec, int32(req.AccountID))
		if err != nil {
			s.logf("firstload db.err acc=%d err=%v", req.AccountID, err)
			s.send("db.err", "accache", err.Error())
		} else {
			fat = f
		}
	}
	if fat == nil {
		if e, ok := s.cache.Get(int(req.AccountID)); ok {
			fat = &db.Fatigue{Point: e.Fatigue.Point, UpdateTime: e.Fatigue.UpdateTime,
				NpcKill: e.Fatigue.NpcKill, LimitReset: e.Fatigue.LimitReset, LimitAccum: e.Fatigue.LimitAccum}
		} else {
			fat = &db.Fatigue{} // пустой result-set как у ориг при отсутствии записи
		}
	}
	resp := payload.BuildFirstLoadResp(req.AccountID,
		uint32(fat.Point), uint32(fat.NpcKill),
		byte(fat.LimitReset), uint32(fat.UpdateTime), uint16(fat.LimitAccum))
	s.respond(c, cmd, resp)
	s.logf("firstload acc=%d point=%d npckill=%d upd=%d reset=%d accum=%d",
		req.AccountID, fat.Point, fat.NpcKill, fat.UpdateTime, fat.LimitReset, fat.LimitAccum)
}

// 0x05 LOAD_BM_PACK: C2S [accId] -> S2C [accId][count u16].
// Пара доказана capture (count=0 при пустом списке). Записи при count>0 — R4.
func hLoadBMPack(s *S, c *Conn, cmd uint16, pl []byte) {
	req, err := payload.ParseFirstLoadReq(pl)
	if err != nil {
		s.logf("bmpack parse.err %v", err)
		return
	}
	count := uint16(0)
	if s.exec != nil {
		rows, err := db.GetAccountPackList(context.Background(), s.exec, int32(req.AccountID))
		if err != nil {
			s.logf("bmpack db.err acc=%d err=%v", req.AccountID, err)
			s.send("db.err", "accache", err.Error())
		} else {
			if len(rows) > 0 {
				// Формат записей не доказан — не рискуем байт-паритетом, логируем.
				s.logf("bmpack acc=%d rows=%d — entry-format TBD R4, отвечаю count=0 (capture-канон пустого списка)", req.AccountID, len(rows))
				s.send("bmpack.rows", "accache", fmt.Sprintf("acc=%d rows=%d TBD-R4", req.AccountID, len(rows)))
			}
		}
	}
	s.respond(c, cmd, payload.BuildBMPackResp(req.AccountID, count))
}

// 0x07 LOAD_TRIAL: C2S [accId][charRef] -> тишина (в capture ответа нет).
// SQL: aion_GetTrialAccountData (значения в лог; ответная раскладка = R4).
func hLoadTrial(s *S, c *Conn, cmd uint16, pl []byte) {
	acc, err := payload.AccountID(pl)
	if err != nil {
		s.logf("loadtrial parse.err %v", err)
		return
	}
	if s.exec != nil {
		t, err := db.GetTrialAccountData(context.Background(), s.exec, int32(acc))
		if err != nil {
			s.logf("loadtrial db.err acc=%d err=%v", acc, err)
		} else if t != nil {
			s.logf("loadtrial acc=%d upd=%d reset=%d sell=%d trade=%d dec=%d gath=%d ext=%d",
				acc, t.UpdateTime, t.ResetTime, t.SellGold, t.TradeGold, t.DecomposeSum, t.GatherSum, t.ExtractGatherSum)
		}
	}
}

// 0x08 UPDATE_TRIAL: C2S [accId][P1..P9] -> тишина. SQL: aion_SetTrialAccountData
// (роли полей = гипотеза, журнал теорий; арбитр R4).
func hUpdateTrial(s *S, c *Conn, cmd uint16, pl []byte) {
	r, err := payload.ParseUpdateTrialReq(pl)
	if err != nil {
		s.logf("updatetrial parse.err %v", err)
		return
	}
	if s.exec != nil {
		args := &db.UpdateTrialArgs{
			UpdateTime: int32(r.P1), ResetTime: int32(r.P2),
			SellGold: int64(r.P3), TradeGold: int64(r.P4),
			DecomposeSum: int32(r.P5), GatherSum: int32(r.P6), ExtractGatherSum: int32(r.P7),
		}
		if err := db.SetTrialAccountData(context.Background(), s.exec, int32(r.AccountID), args); err != nil {
			s.logf("updatetrial db.err acc=%d err=%v", r.AccountID, err)
			s.send("db.err", "accache", err.Error())
		}
	}
	s.logf("updatetrial acc=%d P=%d/%d/%d/%d/%d/%d/%d/%d/%d", r.AccountID, r.P1, r.P2, r.P3, r.P4, r.P5, r.P6, r.P7, r.P8, r.P9)
}

// 0x09 SYNC_TEST: эхо (SendIOBuffer у ориг).
func hSyncTest(s *S, c *Conn, cmd uint16, pl []byte) {
	s.respond(c, cmd, pl)
}

// 0x0e LOG_INFO: UTF-16 лог-строки Server64 — пишем в наш лог (raw), ответа нет.
func hLogInfo(s *S, c *Conn, cmd uint16, pl []byte) {
	s.logf("loginfo len=%d hex=%s", len(pl), hex.EncodeToString(pl))
}

// 0x10 CHAR_LOGIN: C2S -> S2C [accId][8×FF][12×0] (пара доказана capture).
// SQL (SetLoginUser_20121206) — R4: раскладка аргументов proc не доказана.
func hCharLogin(s *S, c *Conn, cmd uint16, pl []byte) {
	r, err := payload.ParseCharLoginReq(pl)
	if err != nil {
		s.logf("charlogin parse.err %v", err)
		return
	}
	s.respond(c, cmd, payload.BuildCharLoginResp(r.AccountID))
	s.logf("charlogin %s", r.Debug())
	s.send("char.login", "accache", fmt.Sprintf("acc=%d charRef=%#x stamp=%s", r.AccountID, r.CharRef, r.Stamp))
}

// 0x11 CHAR_LOGOUT: тишина (capture). SQL — R4.
func hCharLogout(s *S, c *Conn, cmd uint16, pl []byte) {
	r, err := payload.ParseCharLogoutReq(pl)
	if err != nil {
		s.logf("charlogout parse.err %v", err)
		return
	}
	s.logf("charlogout %s", r.Debug())
	s.send("char.logout", "accache", fmt.Sprintf("acc=%d charRef=%#x stamp=%s", r.AccountID, r.CharRef, r.Stamp))
}

// 0x19 UPDATE_HIDDEN_FATIGUE: C2S -> тишина (capture).
// SQL: aion_UpdateHiddenFatigueInfo_20170428(accId, point, npckill, updatetime, limit_reset, limit_play)
// — раскладка сходится с сигнатурой 1-в-1 (6 dwords).
func hUpdateFatigue(s *S, c *Conn, cmd uint16, pl []byte) {
	r, err := payload.ParseFatigueReq(pl)
	if err != nil {
		s.logf("fatigue parse.err %v", err)
		return
	}
	if s.exec != nil {
		if err := db.UpdateHiddenFatigueInfo(context.Background(), s.exec,
			int32(r.AccountID), int32(r.Point), int32(r.NpcKill),
			int32(r.UpdateTime), int32(r.LimitReset), int32(r.LimitPlay)); err != nil {
			s.logf("fatigue db.err acc=%d err=%v", r.AccountID, err)
			s.send("db.err", "accache", err.Error())
		}
	}
	s.cache.UpdateFatigue(int(r.AccountID), int32(r.Point), int32(r.NpcKill), int32(r.UpdateTime), int32(r.LimitReset), int32(r.LimitPlay))
	s.logf("fatigue acc=%d point=%d npc=%d upd=%d reset=%d play=%d",
		r.AccountID, r.Point, r.NpcKill, r.UpdateTime, r.LimitReset, r.LimitPlay)
}

// 0x1a LOAD_LUNA: C2S [accId] -> ДВА ответа (capture): cmd 27 [F2 00 01 00][accId][u16 0],
// затем cmd 26 [accId][ts unix][u16 0]. SQL: aion_loadLuna (значение в лог).
func hLoadLuna(s *S, c *Conn, cmd uint16, pl []byte) {
	req, err := payload.ParseFirstLoadReq(pl)
	if err != nil {
		s.logf("loadluna parse.err %v", err)
		return
	}
	if s.exec != nil {
		if l, err := db.GetLuna(context.Background(), s.exec, int32(req.AccountID)); err != nil {
			s.logf("loadluna db.err acc=%d err=%v", req.AccountID, err)
		} else {
			s.logf("loadluna acc=%d luna=%d", req.AccountID, l)
		}
	}
	s.respond(c, 0x1b, payload.BuildLunaUpdateResp(req.AccountID))
	s.respond(c, cmd, payload.BuildLunaLoadResp(req.AccountID, uint32(time.Now().Unix())))
}

// 0x1f ASK_CAN_MAKE_JUMPING_CHARACTER: [u8 01][accId][u16 0001] -> тишина (capture).
// SQL: aion_QueryCanCreateJumpingCharacter_20170428 — output-параметры, раскладка ответа = R4.
func hAskJumping(s *S, c *Conn, cmd uint16, pl []byte) {
	r, err := payload.ParseJumpingReq(pl)
	if err != nil {
		s.logf("jumping parse.err %v", err)
		return
	}
	s.logf("jumping flag=%d acc=%d val=%d", r.Flag, r.AccountID, r.Value)
}

// 0x25 TRANSFORM_OPERATION: [u32 subtype][accId][u8] -> тишина (capture).
// SQL: aion_GetTransformList / aion_PutTransform — раскладка = R4.
func hTransform(s *S, c *Conn, cmd uint16, pl []byte) {
	r, err := payload.ParseTransformReq(pl)
	if err != nil {
		s.logf("transform parse.err %v", err)
		return
	}
	s.logf("transform subtype=%#x acc=%d flag=%d", r.SubType, r.AccountID, r.Flag)
}


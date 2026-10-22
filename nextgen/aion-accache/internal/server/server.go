// Package server: accept-loop + per-conn state-машина (порт AC_Socket::OnRead)
// + dispatch по T1. R3 (21.10): хендлеры наблюдённых cmds с раскладками payload
// (internal/payload) + SQL-вызовы по proc-телам (internal/db).
// Политика ответов (capture-канон): ответ ТОЛЬКО там, где пара доказана capture
// (1↔1, 4↔4, 5↔5, 16↔16, 26↔26+27); R4-финал 22.10: добавлены ПУШИ ориг
// (0x0D после VERSION; 0x0F/0x14/0x1C при CHAR_LOGIN — capture-20261022,
// 2 чара × 3 логина: триггеры однозначны, порядок в кластере = 15→20→28, между
// ними ответы 16/27 — ACS многопоточен, наш порядок детерминированный).
// Остальные — тишина (как ориг).
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
	"encoding/binary"
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
	// R3-хвост (кросс-пульс 21.10): хендлеры с proc-параметрами из accountcache-ref/db-procs-77-ref58.rpt.
	s.SetHandler(0x06, hUpdateBMPack)        // ACQ_UPDATE_BM_PACK -> aion_SetAccountPack (3 int, 1 tinyint)
	s.SetHandler(0x0f, hCharCreated)         // ACQ_CHAR_CREATED -> aion_SetCreateUser_20121206 (7 int, 1 datetime)
	s.SetHandler(0x12, hCharDelete)          // ACQ_CHAR_DELETE -> aion_SetDeleteUser_20120703 (4 int)
	s.SetHandler(0x13, hCharDeleteCompleted) // ACQ_CHAR_DELETE_COMPLETED -> aion_SetDeleteCompletedUser_20120703 (4 int)
	s.SetHandler(0x15, hCharLevelChanged)    // ACQ_CHAR_LEVEL_CHANGED -> aion_SetUserLevelChanged (3 int, 1 bigint)
	// R3.6 — луна/борд/опросы/монстркоры/логин-евенты (handlers_r3x.go, тишина: пары не доказаны capture):
	s.SetHandler(0x17, hUpdateLoginEventRecord)  // ACQ_UPDATE_LOGIN_EVENT_RECORD -> aion_UpdateLoginEventInfo
	s.SetHandler(0x18, hDeleteLoginEventRecord)  // ACQ_DELETE_LOGIN_EVENT_RECORD -> aion_DeleteLoginEventInfo
	s.SetHandler(0x1c, hConfirmLunaReward)       // ACQ_CONFIRM_LUNA_REWARD -> aion_ConfirmLunaReward
	s.SetHandler(0x1d, hDecreaseLunaKey)         // ACQ_DECREASE_LUNA_KEY -> aion_DecreaseLunaKey
	s.SetHandler(0x1e, hUpdateBoardBMState)      // ACQ_UPDATE_BOARD_BM_STATE -> aion_UpdateBoardBMState_2017
	s.SetHandler(0x20, hLoadPlayTimesForPolls)   // ACQ_LOAD_PREVIOUS_PLAYTIMES_FOR_POLLS -> aion_GetPlayTimesForPolls
	s.SetHandler(0x21, hLoadPlayTimeForOnePoll)  // ACQ_LOAD_PREVIOUS_PLAYTIME_FOR_ONE_POLL -> aion_GetPlayTimeForPoll
	s.SetHandler(0x22, hUpdatePlayTimeForOnePoll)// ACQ_UPDATE_PREVIOUS_PLAYTIME_FOR_ONE_POLL -> aion_UpdatePlayTimeForPoll
	s.SetHandler(0x23, hUpdatePlayTimesForPolls) // ACQ_UPDATE_PREVIOUS_PLAYTIMES_FOR_POLLS -> aion_UpdatePlayTimesForPolls
	s.SetHandler(0x24, hResetLunaReward)         // ACQ_RESET_LUNA_REWARD -> aion_ResetLunaReward
	s.SetHandler(0x26, hMonsterCoreUpdate)       // ACQ_MONSTER_CORE_UPDATE_VALUE -> aion_MonsterCoreUpdate
	s.SetHandler(0x27, hMonsterCoreUpgrade)      // ACQ_MONSTER_CORE_UPGRADE -> aion_MonsterCoreUpgradeAndDecreaseCount
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

// 0x01 VERSION: C2S {3,1} -> S2C {3} + ПУШ ACP_13 (R4-финал 22.10).
// Пара доказана capture В ОБОИХ corpus 1:1:1 (10.07: 1 VERSION → 1+1;
// 22.10: оба соединения c VERSION получили VERSION+ACP_13; соединения без
// VERSION — без пуша). payload = [ts] = константа 0x640831AE (Т7 ⏳).
func hVersion(s *S, c *Conn, cmd uint16, pl []byte) {
	if r, err := payload.ParseVersionReq(pl); err == nil {
		s.logf("version A=%d B=%d", r.A, r.B)
	}
	s.respond(c, cmd, payload.BuildVersionResp())
	s.respond(c, 0x0d, payload.BuildCharCustomSavedPush(payload.CharCustomSavedTsDefault))
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
			fat = &db.Fatigue{Point: 1} // R4-хвост 21.10: пустая строка -> point=1
			// (эмпирика golden acc=7: строки account_data НЕТ, ответ point=1 —
			// журнал теорий: дефолт ACS RAM-кэша для нового аккаунта)
		}
	}
	// R4-live 22.10 (Т6 live): point = RAM-sticky (не с DB/wire — ориг ответил 1 при DB 0);
	// первая наблюдённая точка клампится в 1 (Т4).
	point := fat.Point
	if e, ok := s.cache.Get(int(req.AccountID)); ok {
		point = e.Fatigue.Point
	} else if point < 1 {
		point = 1
	}
	resp := payload.BuildFirstLoadResp(req.AccountID,
		uint32(point), uint32(fat.NpcKill),
		// R4-хвост 21.10: u8@12 и u16@17 НЕ byte(limit_play_reset_time) —
		// прод limit_reset=0x6AC5DFE0 дал бы 0xE0, а golden везде 0x00.
		// Гипотеза (журнал теорий): X/Y = производные limit_play_accum_time
		// (для acc=1010 accum=0 → 0/0 — все golden байт-паритетны).
		byte(fat.LimitAccum), uint32(fat.UpdateTime), uint16(fat.LimitAccum>>16))
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
	// ⚠ Т2 «push 0x15 после BM» ОПРОВЕРЖЕНА live 22.10 (R4 A/B): все 5 кадров O-0x15
	// идут непосредственно за C-0x1F ASK_JUMPING (5/5), после BM ориг не пушит —
	// утренняя корреляция была по позиции. Ответ на 0x1F = в hAskJumping.
}

// 0x07 LOAD_TRIAL: C2S [accId][charRef] -> тишина (в capture ответа нет).
// SQL: aion_GetTrialAccountData (значения в лог; ответная раскладка = R4).
func hLoadTrial(s *S, c *Conn, cmd uint16, pl []byte) {
	acc, err := payload.AccountID(pl)
	if err != nil {
		s.logf("loadtrial parse.err %v", err)
		return
	}
	xEcho := uint32(0)
	if len(pl) >= 8 {
		xEcho = binary.LittleEndian.Uint32(pl[4:]) // второй dword C2S (эхо в ответе)
	}
	reset, sell, trade, decompose, gather, extract := uint32(0), uint32(0), uint32(0), uint32(0), uint32(0), uint32(0)
	if s.exec != nil {
		t, err := db.GetTrialAccountData(context.Background(), s.exec, int32(acc))
		if err != nil {
			s.logf("loadtrial db.err acc=%d err=%v", acc, err)
		} else if t != nil {
			reset, sell, trade = uint32(t.ResetTime), uint32(t.SellGold), uint32(t.TradeGold)
			decompose, gather, extract = uint32(t.DecomposeSum), uint32(t.GatherSum), uint32(t.ExtractGatherSum)
			s.logf("loadtrial acc=%d reset=%d sell=%d trade=%d dec=%d gath=%d ext=%d",
				acc, t.ResetTime, t.SellGold, t.TradeGold, t.DecomposeSum, t.GatherSum, t.ExtractGatherSum)
		}
	}
	// R4-хвост 21.10: C2S 0x07 -> S2C 0x06 (capture-канон: кадр после batch1,
	// payload содержит sell=917/trade=70 из trial_account_data acc=1010).
	s.respond(c, 0x06, payload.BuildTrialResp(acc, xEcho, reset, sell, trade, decompose, gather, extract))
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

// 0x10 CHAR_LOGIN: C2S -> S2C [accId][8×FF][12×0] + ТРИ ПУША ориг (R4-финал 22.10,
// capture-20261022: кластер при каждом логине ×3, оба чара):
//   0x0F CHAR_CREATED [charRef][0x000A07EA][3];
//   0x14 CHAR_INFO_REFRESH [charRef][..][lastLogin][..][1][..][flag][maxLevel];
//   0x1C CONFIRM_LUNA_REWARD [accId][charId=charRef&0xFFFF][u16 0].
// SQL (SetLoginUser_20121206) — R3: раскладка аргументов proc не доказана.
func hCharLogin(s *S, c *Conn, cmd uint16, pl []byte) {
	r, err := payload.ParseCharLoginReq(pl)
	if err != nil {
		s.logf("charlogin parse.err %v", err)
		return
	}
	// R3 SQLStore: срез global_user_data ДБ ПЕРВЫМ ходом (пуш должен показать
	// pre-login значение — наблюдение Т10; SetLoginUser идёт ПОСЛЕ пушей).
	charID := int32(r.CharRef & 0xFFFF)
	lastLogin, maxLevel := uint32(0), uint32(0)
	if ui, ok := s.cache.UserInfo(int(r.AccountID)); ok {
		lastLogin, maxLevel = ui.LastLogin, ui.MaxLevel
	}
	if s.exec != nil {
		if ui, ok, err := db.GetUserInfo(context.Background(), s.exec, int32(r.AccountID), charID); err != nil {
			s.logf("charlogin userinfo db.err acc=%d err=%v", r.AccountID, err)
			s.send("db.err", "accache", err.Error())
		} else if ok {
			// строка есть — RAM-кэш обновляем (пустой result-set кэш НЕ трогает)
			s.cache.SetUserInfo(int(r.AccountID), cache.UserInfo{LastLogin: uint32(ui.LastLogin), MaxLevel: uint32(ui.MaxLevel)})
			lastLogin, maxLevel = uint32(ui.LastLogin), uint32(ui.MaxLevel)
		}
	}
	s.respond(c, cmd, payload.BuildCharLoginResp(r.AccountID))
	s.respond(c, 0x0f, payload.BuildCharCreatedPush(r.CharRef))
	s.respond(c, 0x14, payload.BuildCharInfoRefreshPush(r.CharRef, lastLogin, 0, maxLevel))
	s.respond(c, 0x1c, payload.BuildConfirmLunaRewardPush(r.AccountID, uint32(r.CharRef&0xFFFF)))
	s.logf("charlogin %s lastLogin=%d maxLevel=%d", r.Debug(), lastLogin, maxLevel)
	s.send("char.login", "accache", fmt.Sprintf("acc=%d charRef=%#x stamp=%s", r.AccountID, r.CharRef, r.Stamp))
	if s.exec != nil {
		if lt, err := parseStamp(r.Stamp); err != nil {
			s.logf("charlogin stamp parse.err %q: %v — SetLoginUser пропущен", r.Stamp, err)
		} else if err := db.SetLoginUser(context.Background(), s.exec, charID, s.dbServerID(), s.dbServerID(), lt); err != nil {
			s.logf("charlogin setlogin db.err acc=%d err=%v", r.AccountID, err)
			s.send("db.err", "accache", err.Error())
		} else {
			s.logf("charlogin setlogin char=%d server=%d t=%s", charID, s.cfg.DB.ServerID, lt.Format(time.RFC3339))
		}
	}
}

// parseStamp — штамп Server64 из C2S CHAR_LOGIN/CHAR_LOGOUT
// ("2026-10-22T06:53:11.620", формат ISO с 3 знаками мс). Таймзона штампа
// не маркирована — парсим как UTC (арбитр live A/B SQL-дифф, Т16 ⏳).
func parseStamp(s string) (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05.000", s)
}

// dbServerID — server_id/cur_server_id для global_user_data-процедур (дефолт 1
// при unset/0 — конфиг может не содержать ключ; REF58-БД: server_id=1).
func (s *S) dbServerID() int32 {
	if s.cfg.DB.ServerID <= 0 {
		return 1
	}
	return int32(s.cfg.DB.ServerID)
}

// 0x11 CHAR_LOGOUT: тишина (capture) + SQL aion_SetLogoutUser_20120703 (R3:
// char_id/server_id/cur_server_id/logout_time — UPDATE last_logout_time).
func hCharLogout(s *S, c *Conn, cmd uint16, pl []byte) {
	r, err := payload.ParseCharLogoutReq(pl)
	if err != nil {
		s.logf("charlogout parse.err %v", err)
		return
	}
	s.logf("charlogout %s", r.Debug())
	s.send("char.logout", "accache", fmt.Sprintf("acc=%d charRef=%#x stamp=%s", r.AccountID, r.CharRef, r.Stamp))
	if s.exec != nil {
		charID := int32(r.CharRef & 0xFFFF)
		if lt, err := parseStamp(r.Stamp); err != nil {
			s.logf("charlogout stamp parse.err %q: %v — SetLogoutUser пропущен", r.Stamp, err)
		} else if err := db.SetLogoutUser(context.Background(), s.exec, charID, s.dbServerID(), s.dbServerID(), lt); err != nil {
			s.logf("charlogout setlogout db.err char=%d err=%v", charID, err)
			s.send("db.err", "accache", err.Error())
		} else {
			s.logf("charlogout setlogout char=%d server=%d t=%s", charID, s.cfg.DB.ServerID, lt.Format(time.RFC3339))
		}
	}
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
	s.cache.UpdateFatigue(int(r.AccountID), int32(r.NpcKill), int32(r.UpdateTime), int32(r.LimitReset), int32(r.LimitPlay))
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
	ts := uint32(0)
	if s.exec != nil {
		if l, err := db.GetLuna(context.Background(), s.exec, int32(req.AccountID)); err != nil {
			s.logf("loadluna db.err acc=%d err=%v", req.AccountID, err)
		} else {
			s.logf("loadluna acc=%d luna=%d", req.AccountID, l)
		}
		// R4-хвост (21.10, ПРОД-ФАКТ acc=1010): ts в ответе = limit_play_reset_time
		// из account_data — capture 0x6AC5DFE0 совпал с колонкой байт-в-байт;
		// НЕ account_luna.updatedate (строки может не быть) и НЕ time.Now().
		if f, err := db.GetAccountData(context.Background(), s.exec, int32(req.AccountID)); err != nil {
			s.logf("loadluna accountdata db.err acc=%d err=%v", req.AccountID, err)
		} else if f != nil {
			ts = uint32(f.LimitReset)
		}
	}
	s.respond(c, 0x1b, payload.BuildLunaUpdateResp(req.AccountID))
	s.respond(c, cmd, payload.BuildLunaLoadResp(req.AccountID, ts))
}

// 0x1f ASK_CAN_MAKE_JUMPING_CHARACTER: [u8 01][accId][u16 0001] -> тишина (capture).
// SQL: aion_QueryCanCreateJumpingCharacter_20170428 — output-параметры, раскладка ответа = R4.
func hAskJumping(s *S, c *Conn, cmd uint16, pl []byte) {
	r, err := payload.ParseJumpingReq(pl)
	if err != nil {
		s.logf("jumping parse.err %v", err)
		return
	}
	// R4-live 22.10 канон: C2S 0x1F -> S2C 0x15 [acc][02][01][8×0] (5/5 кадров live,
	// следующий кадр за запросом).
	s.respond(c, 0x15, payload.BuildBM21Push(r.AccountID))
	s.logf("jumping flag=%d acc=%d val=%d -> 0x15", r.Flag, r.AccountID, r.Value)
}

// --- R3-хвост: зарегистрированные по proc-телам REF58 ---

// 0x06 UPDATE_BM_PACK -> aion_SetAccountPack(accountId INT, charId INT, itemPackId INT, active TINYINT)
// C2S [accId][charId][packId][active u8]. Ответ не требуется (capture: тишина).
func hUpdateBMPack(s *S, c *Conn, cmd uint16, pl []byte) {
	if len(pl) < 13 { // 4+4+4+1
		s.logf("bmpack parse.err len=%d", len(pl))
		return
	}
	acc := int32(binary.LittleEndian.Uint32(pl[0:]))
	charId := int32(binary.LittleEndian.Uint32(pl[4:]))
	packId := int32(binary.LittleEndian.Uint32(pl[8:]))
	active := int32(pl[12])
	if s.exec != nil {
		_, _, err := db.CallProc(context.Background(), s.exec, "aion_SetAccountPack", acc, charId, packId, active)
		if err != nil {
			s.logf("bmpack db.err acc=%d char=%d err=%v", acc, charId, err)
			s.send("db.err", "accache", err.Error())
		}
	}
	s.logf("bmpack acc=%d char=%d pack=%d active=%d", acc, charId, packId, active)
}

// 0x0f CHAR_CREATED (C2S-уведомление Server64 при создании чара) — R4-live 22.10:
// раскладка = сессионная [u32 1][charRef][utf16z stamp][tail] (как CHAR_LOGOUT).
// Гипотеза R3.6 [acc][char][slot][level][gender][race][class][date] ОПРОВЕРГНУТА live
// (парс давал мусор: slot=байты UTF-16). proc aion_SetCreateUser_20121206 требует
// acc/slot/level/gender/race/class — источник аргументов = другой канал (⏳ Т21):
// лог + ship, SQL НЕ зовём.
func hCharCreated(s *S, c *Conn, cmd uint16, pl []byte) {
	r, err := payload.ParseCharLogoutReq(pl)
	if err != nil {
		s.logf("charcreated parse.err len=%d", len(pl))
		return
	}
	s.logf("charcreated charRef=%#x stamp=%q tail=%x len=%d", r.CharRef, r.Stamp, r.Tail, len(pl))
	s.send("char.created", "accache", fmt.Sprintf("charRef=%#x stamp=%s", r.CharRef, r.Stamp))
}

// 0x12 CHAR_DELETE -> aion_SetDeleteUser_20120703(accountId, charId, deleteTime, flag INT)
// C2S [accId][charId][deleteTime][flag]. Ответ: тишина.
func hCharDelete(s *S, c *Conn, cmd uint16, pl []byte) {
	if len(pl) < 16 {
		s.logf("chardelete parse.err len=%d", len(pl))
		return
	}
	acc := int32(binary.LittleEndian.Uint32(pl[0:]))
	charId := int32(binary.LittleEndian.Uint32(pl[4:]))
	deleteTime := int32(binary.LittleEndian.Uint32(pl[8:]))
	flag := int32(binary.LittleEndian.Uint32(pl[12:]))
	if s.exec != nil {
		_, _, err := db.CallProc(context.Background(), s.exec, "aion_SetDeleteUser_20120703", acc, charId, deleteTime, flag)
		if err != nil {
			s.logf("chardelete db.err acc=%d char=%d err=%v", acc, charId, err)
			s.send("db.err", "accache", err.Error())
		}
	}
	s.logf("chardelete acc=%d char=%d time=%d flag=%d", acc, charId, deleteTime, flag)
}

// 0x13 CHAR_DELETE_COMPLETED -> aion_SetDeleteCompletedUser_20120703(accountId, charId, completeTime, flag INT)
// C2S [accId][charId][completeTime][flag]. Ответ: тишина.
func hCharDeleteCompleted(s *S, c *Conn, cmd uint16, pl []byte) {
	if len(pl) < 16 {
		s.logf("chardeletecompleted parse.err len=%d", len(pl))
		return
	}
	acc := int32(binary.LittleEndian.Uint32(pl[0:]))
	charId := int32(binary.LittleEndian.Uint32(pl[4:]))
	completeTime := int32(binary.LittleEndian.Uint32(pl[8:]))
	flag := int32(binary.LittleEndian.Uint32(pl[12:]))
	if s.exec != nil {
		_, _, err := db.CallProc(context.Background(), s.exec, "aion_SetDeleteCompletedUser_20120703", acc, charId, completeTime, flag)
		if err != nil {
			s.logf("chardeletecompleted db.err acc=%d char=%d err=%v", acc, charId, err)
			s.send("db.err", "accache", err.Error())
		}
	}
	s.logf("chardeletecompleted acc=%d char=%d time=%d flag=%d", acc, charId, completeTime, flag)
}

// 0x15 CHAR_LEVEL_CHANGED -> aion_SetUserLevelChanged(accountId, charId, level INT, exp BIGINT)
// C2S [accId][charId][level][exp u64]. Ответ: тишина.
func hCharLevelChanged(s *S, c *Conn, cmd uint16, pl []byte) {
	if len(pl) < 20 { // 3*4+8
		s.logf("charlevelchanged parse.err len=%d", len(pl))
		return
	}
	acc := int32(binary.LittleEndian.Uint32(pl[0:]))
	charId := int32(binary.LittleEndian.Uint32(pl[4:]))
	level := int32(binary.LittleEndian.Uint32(pl[8:]))
	exp := int64(binary.LittleEndian.Uint64(pl[12:]))
	if s.exec != nil {
		_, _, err := db.CallProc(context.Background(), s.exec, "aion_SetUserLevelChanged", acc, charId, level, exp)
		if err != nil {
			s.logf("charlevelchanged db.err acc=%d char=%d err=%v", acc, charId, err)
			s.send("db.err", "accache", err.Error())
		}
	}
	s.logf("charlevelchanged acc=%d char=%d level=%d exp=%d", acc, charId, level, exp)
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


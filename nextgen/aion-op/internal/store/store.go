// Package store — SQLite WAL: события логов, метрики процессов, алерты.
// Phase 0.5. Retention по умолчанию 30 дней. Один писатель (канал), читатели — те же конны.
package store

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS events(
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ts INTEGER NOT NULL, svc TEXT NOT NULL, kind TEXT NOT NULL,
  sev INTEGER NOT NULL, text TEXT NOT NULL);
CREATE INDEX IF NOT EXISTS ix_events_ts ON events(ts);
CREATE TABLE IF NOT EXISTS proc_metrics(
  ts INTEGER NOT NULL, name TEXT NOT NULL, pid TEXT NOT NULL,
  mem_mb INTEGER NOT NULL, handles INTEGER NOT NULL, handles_dpm INTEGER NOT NULL DEFAULT 0);
CREATE INDEX IF NOT EXISTS ix_proc_ts ON proc_metrics(ts);
CREATE INDEX IF NOT EXISTS ix_proc_name ON proc_metrics(name, ts);
CREATE TABLE IF NOT EXISTS sys_mem(
  ts INTEGER NOT NULL, free_phys_mb INTEGER NOT NULL, free_commit_mb INTEGER NOT NULL);
CREATE INDEX IF NOT EXISTS ix_sys_ts ON sys_mem(ts);
CREATE TABLE IF NOT EXISTS alerts(
  idem TEXT PRIMARY KEY, opened_ts INTEGER NOT NULL, closed_ts INTEGER NOT NULL DEFAULT 0,
  sev INTEGER NOT NULL, text TEXT NOT NULL, active INTEGER NOT NULL DEFAULT 1);
CREATE INDEX IF NOT EXISTS ix_alerts_active ON alerts(active);
CREATE TABLE IF NOT EXISTS actions(
  id INTEGER PRIMARY KEY AUTOINCREMENT, ts INTEGER NOT NULL,
  action TEXT NOT NULL, target TEXT NOT NULL, ok INTEGER NOT NULL,
  executed INTEGER NOT NULL DEFAULT 0, detail TEXT NOT NULL);
CREATE INDEX IF NOT EXISTS ix_actions_ts ON actions(ts);
CREATE TABLE IF NOT EXISTS ccu_world(
  ts INTEGER NOT NULL, world INTEGER NOT NULL, light INTEGER NOT NULL,
  dark INTEGER NOT NULL, npc INTEGER NOT NULL);
CREATE INDEX IF NOT EXISTS ix_ccuw_ts ON ccu_world(ts);
CREATE TABLE IF NOT EXISTS ccu_auth(
  ts INTEGER NOT NULL, server_id INTEGER NOT NULL, world_user INTEGER NOT NULL,
  limit_user INTEGER NOT NULL, auth_user INTEGER NOT NULL, wait_user INTEGER NOT NULL);
CREATE INDEX IF NOT EXISTS ix_ccua_ts ON ccu_auth(ts);
CREATE TABLE IF NOT EXISTS sql_gauge(
  ts INTEGER NOT NULL, blocked INTEGER NOT NULL, resq_depth INTEGER NOT NULL,
  resq_wait_ms INTEGER NOT NULL);
CREATE INDEX IF NOT EXISTS ix_sga_ts ON sql_gauge(ts);
CREATE TABLE IF NOT EXISTS sql_waits(
  ts INTEGER NOT NULL, wait_type TEXT NOT NULL, dms INTEGER NOT NULL, dcnt INTEGER NOT NULL);
CREATE INDEX IF NOT EXISTS ix_swa_ts ON sql_waits(ts);
`

// EventRow — событие в хранилище (JSON-готовое).
type EventRow struct {
	Ts   int64  `json:"ts"`
	Svc  string `json:"svc"`
	Kind string `json:"kind"`
	Sev  int    `json:"sev"`
	Text string `json:"text"`
}

// AlertRow — алерт.
type AlertRow struct {
	Idem     string `json:"idem"`
	Sev      int    `json:"sev"`
	Text     string `json:"text"`
	OpenedTs int64  `json:"opened_ts"`
}

type MetricRow struct {
	Name    string
	PID     string
	MemMB   uint64
	Handles uint64
	DPM     int64
}

type Store struct {
	db   *sql.DB
	ops  chan func()
	quit chan struct{}
	done chan struct{}
}

// Open — создать/открыть БД (WAL), запустить писателя с часовой чисткой.
func Open(path string, retentionDays int) (*Store, error) {
	if retentionDays <= 0 {
		retentionDays = 30
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("sqlite open: %w", err)
	}
	db.SetMaxOpenConns(1) // один писатель; читатели сериализуются — объёмы малые
	for _, p := range []string{
		"PRAGMA journal_mode=WAL", "PRAGMA busy_timeout=5000", "PRAGMA synchronous=NORMAL",
	} {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("%s: %w", p, err)
		}
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("schema: %w", err)
	}
	s := &Store{db: db, ops: make(chan func(), 1024), quit: make(chan struct{}), done: make(chan struct{})}
	go s.writer(retentionDays)
	return s, nil
}

func (s *Store) writer(retentionDays int) {
	purge := time.NewTicker(time.Hour)
	defer purge.Stop()
	for {
		select {
		case f, ok := <-s.ops:
			if !ok {
				close(s.done)
				return
			}
			f()
		case <-purge.C:
			cut := time.Now().AddDate(0, 0, -retentionDays).Unix()
			for _, q := range []string{
				`DELETE FROM events WHERE ts < ?`,
				`DELETE FROM proc_metrics WHERE ts < ?`,
				`DELETE FROM sys_mem WHERE ts < ?`,
				`DELETE FROM ccu_world WHERE ts < ?`,
				`DELETE FROM ccu_auth WHERE ts < ?`,
				`DELETE FROM sql_gauge WHERE ts < ?`,
				`DELETE FROM sql_waits WHERE ts < ?`,
				`DELETE FROM alerts WHERE active=0 AND closed_ts < ?`,
			} {
				_, _ = s.db.Exec(q, cut)
			}
		case <-s.quit:
			for {
				select {
				case f, ok := <-s.ops:
					if !ok {
						close(s.done)
						return
					}
					f()
				default:
					close(s.done)
					return
				}
			}
		}
	}
}

// enqueue — не блокирует продюсеров (дроп с логом лучше зависшего оператора).
func (s *Store) enqueue(f func()) {
	select {
	case s.ops <- f:
	default:
		log.Println("store: очередь полна, запись дропнута")
	}
}

// Flush — дождаться применения всех записей в очереди (детерминизм тестов).
func (s *Store) Flush() {
	done := make(chan struct{})
	select {
	case s.ops <- func() { close(done) }:
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
	default:
	}
}

// PurgeNow — чистка старше cutoff (для тестов/ручного прогона).
func (s *Store) PurgeNow(cutoff time.Time) {
	s.enqueue(func() {
		c := cutoff.Unix()
		for _, q := range []string{
			`DELETE FROM events WHERE ts < ?`,
			`DELETE FROM proc_metrics WHERE ts < ?`,
			`DELETE FROM sys_mem WHERE ts < ?`,
			`DELETE FROM ccu_world WHERE ts < ?`,
			`DELETE FROM ccu_auth WHERE ts < ?`,
			`DELETE FROM sql_gauge WHERE ts < ?`,
			`DELETE FROM sql_waits WHERE ts < ?`,
			`DELETE FROM alerts WHERE active=0 AND closed_ts < ?`,
		} {
			_, _ = s.db.Exec(q, c)
		}
	})
}

// AddEvent — событие лога.
func (s *Store) AddEvent(ev EventRow) {
	s.enqueue(func() {
		_, _ = s.db.Exec(`INSERT INTO events(ts,svc,kind,sev,text) VALUES(?,?,?,?,?)`,
			ev.Ts, ev.Svc, ev.Kind, ev.Sev, ev.Text)
	})
}

// AddMetrics — срез метрик процессов.
func (s *Store) AddMetrics(when time.Time, rows []MetricRow) {
	s.enqueue(func() {
		ts := when.Unix()
		for _, r := range rows {
			_, _ = s.db.Exec(
				`INSERT INTO proc_metrics(ts,name,pid,mem_mb,handles,handles_dpm) VALUES(?,?,?,?,?,?)`,
				ts, r.Name, r.PID, r.MemMB, r.Handles, r.DPM)
		}
	})
}

// AddSys — системная память.
func (s *Store) AddSys(when time.Time, freePhysMB, freeCommitMB uint64) {
	s.enqueue(func() {
		_, _ = s.db.Exec(`INSERT INTO sys_mem(ts,free_phys_mb,free_commit_mb) VALUES(?,?,?)`,
			when.Unix(), freePhysMB, freeCommitMB)
	})
}

// ActionRow — запись audit-journal (кто/что/когда/исполнялось ли).
type ActionRow struct {
	Ts       int64  `json:"ts"`
	Action   string `json:"action"`
	Target   string `json:"target"`
	OK       bool   `json:"ok"`
	Executed bool   `json:"executed"`
	Detail   string `json:"detail"`
}

// AddAction — запись в audit (не блокирует при переполнении очереди).
func (s *Store) AddAction(a ActionRow) {
	s.enqueue(func() {
		_, _ = s.db.Exec(`INSERT INTO actions(ts,action,target,ok,executed,detail) VALUES(?,?,?,?,?,?)`,
			a.Ts, a.Action, a.Target, boolInt(a.OK), boolInt(a.Executed), a.Detail)
	})
}

// RecentActions — последние записи audit (новые сверху).
func (s *Store) RecentActions(limit int) []ActionRow {
	rows, err := s.db.Query(
		`SELECT ts,action,target,ok,executed,detail FROM actions ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var res []ActionRow
	for rows.Next() {
		var a ActionRow
		var ok, ex int
		if err := rows.Scan(&a.Ts, &a.Action, &a.Target, &ok, &ex, &a.Detail); err == nil {
			a.OK, a.Executed = ok == 1, ex == 1
			res = append(res, a)
		}
	}
	return res
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// CcuWorldRow — CCU по миру (TBL_GAME_WORLD_INFO zone0).
type CcuWorldRow struct{ World, Light, Dark, Npc int }

// CcuAuthRow — CCU по серверам (AionAccounts.user_count).
type CcuAuthRow struct{ ServerID, WorldUser, LimitUser, AuthUser, WaitUser int }

// WaitRow — дельта wait-типа.
type WaitRow struct {
	Type string
	DMS  int64
	DCnt int64
}

// AddCcuWorld — CCU по мирам.
func (s *Store) AddCcuWorld(when time.Time, rows []CcuWorldRow) {
	s.enqueue(func() {
		ts := when.Unix()
		for _, r := range rows {
			_, _ = s.db.Exec(`INSERT INTO ccu_world(ts,world,light,dark,npc) VALUES(?,?,?,?,?)`,
				ts, r.World, r.Light, r.Dark, r.Npc)
		}
	})
}

// AddCcuAuth — CCU по серверам.
func (s *Store) AddCcuAuth(when time.Time, rows []CcuAuthRow) {
	s.enqueue(func() {
		ts := when.Unix()
		for _, r := range rows {
			_, _ = s.db.Exec(
				`INSERT INTO ccu_auth(ts,server_id,world_user,limit_user,auth_user,wait_user) VALUES(?,?,?,?,?,?)`,
				ts, r.ServerID, r.WorldUser, r.LimitUser, r.AuthUser, r.WaitUser)
		}
	})
}

// AddSqlGauge — blocked/resq срез.
func (s *Store) AddSqlGauge(when time.Time, blocked, resqDepth int, resqWaitMs int64) {
	s.enqueue(func() {
		_, _ = s.db.Exec(`INSERT INTO sql_gauge(ts,blocked,resq_depth,resq_wait_ms) VALUES(?,?,?,?)`,
			when.Unix(), blocked, resqDepth, resqWaitMs)
	})
}

// AddSqlWaits — дельты waits за окно.
func (s *Store) AddSqlWaits(when time.Time, waits []WaitRow) {
	s.enqueue(func() {
		ts := when.Unix()
		for _, w := range waits {
			_, _ = s.db.Exec(`INSERT INTO sql_waits(ts,wait_type,dms,dcnt) VALUES(?,?,?,?)`,
				ts, w.Type, w.DMS, w.DCnt)
		}
	})
}

// GaugePoint — точка истории gauge.
type GaugePoint struct {
	Ts        int64 `json:"ts"`
	Blocked   int   `json:"blocked"`
	ResqDepth int   `json:"resq_depth"`
}

func (s *Store) GaugeHistory(minutes int) []GaugePoint {
	rows, err := s.db.Query(
		`SELECT ts,blocked,resq_depth FROM sql_gauge WHERE ts>? ORDER BY ts`,
		time.Now().Add(-time.Duration(minutes)*time.Minute).Unix())
	if err != nil {
		return nil
	}
	defer rows.Close()
	var res []GaugePoint
	for rows.Next() {
		var p GaugePoint
		if err := rows.Scan(&p.Ts, &p.Blocked, &p.ResqDepth); err == nil {
			res = append(res, p)
		}
	}
	return res
}

// WaitSum — агрегат дельт по типу за окно (для таблицы SQL-вкладки).
type WaitSum struct {
	Type string `json:"type"`
	DMS  int64  `json:"dms"`
	DCnt int64  `json:"dcnt"`
}

func (s *Store) WaitSums(minutes, limit int) []WaitSum {
	rows, err := s.db.Query(
		`SELECT wait_type, SUM(dms), SUM(dcnt) FROM sql_waits WHERE ts>? GROUP BY wait_type
		 ORDER BY SUM(dms) DESC LIMIT ?`,
		time.Now().Add(-time.Duration(minutes)*time.Minute).Unix(), limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var res []WaitSum
	for rows.Next() {
		var w WaitSum
		if err := rows.Scan(&w.Type, &w.DMS, &w.DCnt); err == nil {
			res = append(res, w)
		}
	}
	return res
}

// LatestWorldAuth — последние строки CCU (world+auth).
func (s *Store) LatestWorldAuth() (world []map[string]any, auth []map[string]any) {
	world = []map[string]any{}
	rows, err := s.db.Query(
		`SELECT ts,world,light,dark,npc FROM ccu_world WHERE ts=(SELECT MAX(ts) FROM ccu_world)`)
	if err == nil {
		for rows.Next() {
			var ts, w, l, d, n int64
			if err := rows.Scan(&ts, &w, &l, &d, &n); err == nil {
				world = append(world, map[string]any{"ts": ts, "world": w, "light": l, "dark": d, "npc": n})
			}
		}
		rows.Close()
	}
	auth = []map[string]any{}
	rows, err = s.db.Query(
		`SELECT ts,server_id,world_user,limit_user,auth_user,wait_user FROM ccu_auth
		 WHERE ts=(SELECT MAX(ts) FROM ccu_auth) ORDER BY server_id`)
	if err == nil {
		for rows.Next() {
			var ts, sid, wu, lu, au, wq int64
			if err := rows.Scan(&ts, &sid, &wu, &lu, &au, &wq); err == nil {
				auth = append(auth, map[string]any{"ts": ts, "server_id": sid,
					"world_user": wu, "limit_user": lu, "auth_user": au, "wait_user": wq})
			}
		}
		rows.Close()
	}
	return world, auth
}

// OpenAlert — записать открытый алерт (движок сам держит active-set).
// INSERT OR IGNORE + UPDATE: переоткрытие обновляет opened_ts, повторы — нет.
func (s *Store) OpenAlert(idem string, sev int, text string) {
	s.enqueue(func() {
		ts := time.Now().Unix()
		_, _ = s.db.Exec(
			`INSERT OR IGNORE INTO alerts(idem,opened_ts,sev,text,active) VALUES(?,?,?,?,'1')`,
			idem, ts, sev, text)
		_, _ = s.db.Exec(
			`UPDATE alerts SET active=1, closed_ts=0, sev=?, text=? ,
			 opened_ts=CASE WHEN active=0 THEN ? ELSE opened_ts END WHERE idem=?`,
			sev, text, ts, idem)
	})
}

// CloseAlert — закрыть активный алерт.
func (s *Store) CloseAlert(idem string) {
	s.enqueue(func() {
		_, _ = s.db.Exec(`UPDATE alerts SET active=0, closed_ts=? WHERE idem=? AND active=1`,
			time.Now().Unix(), idem)
	})
}

// RecentEvents — последние события (новые сначала).
func (s *Store) RecentEvents(limit int) []EventRow {
	rows, err := s.db.Query(`SELECT ts,svc,kind,sev,text FROM events ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var res []EventRow
	for rows.Next() {
		var e EventRow
		if err := rows.Scan(&e.Ts, &e.Svc, &e.Kind, &e.Sev, &e.Text); err == nil {
			res = append(res, e)
		}
	}
	return res
}

// CountEvents — число событий kind за окно (для рейт-алертов).
func (s *Store) CountEvents(kind string, window time.Duration) int {
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM events WHERE kind=? AND ts>?`,
		kind, time.Now().Add(-window).Unix()).Scan(&n)
	return n
}

// ActiveAlerts — активные алерты (серьёзные сверху).
func (s *Store) ActiveAlerts() []AlertRow {
	rows, err := s.db.Query(
		`SELECT idem,sev,text,opened_ts FROM alerts WHERE active=1 ORDER BY sev DESC, opened_ts ASC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var res []AlertRow
	for rows.Next() {
		var a AlertRow
		if err := rows.Scan(&a.Idem, &a.Sev, &a.Text, &a.OpenedTs); err == nil {
			res = append(res, a)
		}
	}
	return res
}

// Series — точки метрики процесса (для спарклайна UI).
type Point struct {
	Ts      int64  `json:"ts"`
	MemMB   uint64 `json:"mem_mb"`
	Handles uint64 `json:"handles"`
}

func (s *Store) Series(name string, minutes int) []Point {
	rows, err := s.db.Query(
		`SELECT ts,mem_mb,handles FROM proc_metrics WHERE name=? AND ts>? ORDER BY ts`,
		name, time.Now().Add(-time.Duration(minutes)*time.Minute).Unix())
	if err != nil {
		return nil
	}
	defer rows.Close()
	var res []Point
	for rows.Next() {
		var p Point
		if err := rows.Scan(&p.Ts, &p.MemMB, &p.Handles); err == nil {
			res = append(res, p)
		}
	}
	return res
}

// Close — слить очередь и закрыть БД.
func (s *Store) Close() {
	close(s.quit)
	close(s.ops)
	<-s.done
	_ = s.db.Close()
}

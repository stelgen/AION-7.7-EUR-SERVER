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
			_, _ = s.db.Exec(`DELETE FROM events WHERE ts < ?`, cut)
			_, _ = s.db.Exec(`DELETE FROM proc_metrics WHERE ts < ?`, cut)
			_, _ = s.db.Exec(`DELETE FROM sys_mem WHERE ts < ?`, cut)
			_, _ = s.db.Exec(`DELETE FROM alerts WHERE active=0 AND closed_ts < ?`, cut)
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
		_, _ = s.db.Exec(`DELETE FROM events WHERE ts < ?`, c)
		_, _ = s.db.Exec(`DELETE FROM proc_metrics WHERE ts < ?`, c)
		_, _ = s.db.Exec(`DELETE FROM sys_mem WHERE ts < ?`, c)
		_, _ = s.db.Exec(`DELETE FROM alerts WHERE active=0 AND closed_ts < ?`, c)
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

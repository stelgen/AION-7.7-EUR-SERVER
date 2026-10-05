// Package sqlmon — read-only SQL-наблюдение: CCU (AionAccounts.user_count + Aion_log
// TBL_GAME_WORLD_INFO zone0), waits (deltas), blocking/compile-queue, TBL_GAME_SERVER_INFO.
// Логин aionop_ro: db_datareader + VIEW SERVER STATE, больше ничего.
package sqlmon

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"sync"
	"time"

	_ "github.com/microsoft/go-mssqldb"
)

type Ccu struct {
	World int `json:"world"`
	Light int `json:"light"`
	Dark  int `json:"dark"`
	Npc   int `json:"npc"`
}

type AuthCcu struct {
	ServerID  int `json:"server_id"`
	WorldUser int `json:"world_user"`
	LimitUser int `json:"limit_user"`
	AuthUser  int `json:"auth_user"`
	WaitUser  int `json:"wait_user"`
}

type Gauge struct {
	Blocked    int   `json:"blocked"`
	ResqDepth  int   `json:"resq_depth"`
	ResqWaitMs int64 `json:"resq_wait_ms"`
}

type WaitDelta struct {
	Type string `json:"type"`
	DMS  int64  `json:"dms"`
	DCnt int64  `json:"dcnt"`
}

type ServerInfo struct {
	ServerID int           `json:"server_id"`
	Status   int           `json:"status"`
	FreeDisk sql.NullInt64 `json:"free_disk"`
}

type Snap struct {
	When    time.Time    `json:"when"`
	Err     string       `json:"err,omitempty"`
	World   []Ccu        `json:"world"`
	Auth    []AuthCcu    `json:"auth"`
	Gauge   Gauge        `json:"gauge"`
	Waits   []WaitDelta  `json:"waits"`
	SrvInfo []ServerInfo `json:"srv_info"`
}

type Collector interface {
	Collect(ctx context.Context) Snap
}

// Holder — слот последнего среза.
type Holder struct {
	mu sync.RWMutex
	s  Snap
}

func (h *Holder) Set(s Snap) { h.mu.Lock(); h.s = s; h.mu.Unlock() }
func (h *Holder) Get() Snap  { h.mu.RLock(); defer h.mu.RUnlock(); return h.s }

// --- Реальный коллектор (mode=local: SQL на той же VM) ---

type DB struct {
	db   *sql.DB
	prev map[string][2]int64 // wait_type → [cnt, ms]
}

func New(db *sql.DB) *DB { return &DB{db: db, prev: map[string][2]int64{}} }

func (c *DB) Collect(ctx context.Context) Snap {
	s := Snap{When: time.Now()}

	rows, err := c.db.QueryContext(ctx,
		`SELECT WORLD_ID, LIGHT_USERS, DARK_USERS, NPC_COUNT
		 FROM Aion_log.aiongm_ur.TBL_GAME_WORLD_INFO WHERE ZONE_ID=0 ORDER BY WORLD_ID`)
	if err != nil {
		return Snap{When: time.Now(), Err: "ccu_world: " + err.Error()}
	}
	for rows.Next() {
		var w Ccu
		if err := rows.Scan(&w.World, &w.Light, &w.Dark, &w.Npc); err == nil {
			s.World = append(s.World, w)
		}
	}
	rows.Close()

	rows, err = c.db.QueryContext(ctx,
		`SELECT server_id, world_user, limit_user, auth_user, wait_user FROM (
		   SELECT server_id, world_user, limit_user, auth_user, wait_user,
		          ROW_NUMBER() OVER (PARTITION BY server_id ORDER BY record_time DESC) rn
		   FROM AionAccounts.dbo.user_count) t WHERE rn=1 ORDER BY server_id`)
	if err != nil {
		return Snap{When: time.Now(), Err: "ccu_auth: " + err.Error()}
	}
	for rows.Next() {
		var a AuthCcu
		if err := rows.Scan(&a.ServerID, &a.WorldUser, &a.LimitUser, &a.AuthUser, &a.WaitUser); err == nil {
			s.Auth = append(s.Auth, a)
		}
	}
	rows.Close()

	var g Gauge
	err = c.db.QueryRowContext(ctx, `SELECT
		(SELECT COUNT(*) FROM sys.dm_exec_requests WHERE blocking_session_id<>0),
		(SELECT COUNT(*) FROM sys.dm_exec_requests WHERE wait_type='RESOURCE_SEMAPHORE_QUERY_COMPILE'),
		(SELECT ISNULL(SUM(wait_time_ms),0) FROM sys.dm_exec_requests WHERE wait_type='RESOURCE_SEMAPHORE_QUERY_COMPILE')`).
		Scan(&g.Blocked, &g.ResqDepth, &g.ResqWaitMs)
	if err != nil {
		return Snap{When: time.Now(), Err: "gauge: " + err.Error()}
	}
	s.Gauge = g

	rows, err = c.db.QueryContext(ctx, `SELECT wait_type, waiting_tasks_count, wait_time_ms
		FROM sys.dm_os_wait_stats
		WHERE waiting_tasks_count>0 AND wait_type NOT LIKE 'SLEEP%' AND wait_type NOT LIKE 'XE%'
		  AND wait_type NOT LIKE 'BROKER%' AND wait_type NOT LIKE 'SQLTRACE%'
		  AND wait_type NOT IN ('WAITFOR','DIRTY_PAGE_POLL','HADR_FILESTREAM_IOMGR_IOCOMPLETION')
		ORDER BY wait_time_ms DESC`)
	if err != nil {
		return Snap{When: time.Now(), Err: "waits: " + err.Error()}
	}
	type cur struct{ cnt, ms int64 }
	curMap := map[string]cur{}
	for rows.Next() {
		var wt string
		var c2 cur
		if err := rows.Scan(&wt, &c2.cnt, &c2.ms); err == nil {
			curMap[wt] = c2
		}
	}
	rows.Close()
	for wt, c2 := range curMap {
		if p, ok := c.prev[wt]; ok {
			dms, dcnt := c2.ms-p[1], c2.cnt-p[0]
			if dms > 0 || dcnt > 0 {
				s.Waits = append(s.Waits, WaitDelta{Type: wt, DMS: dms, DCnt: dcnt})
			}
		}
		c.prev[wt] = [2]int64{c2.cnt, c2.ms}
	}
	sort.Slice(s.Waits, func(i, j int) bool { return s.Waits[i].DMS > s.Waits[j].DMS })
	if len(s.Waits) > 8 {
		s.Waits = s.Waits[:8]
	}

	rows, err = c.db.QueryContext(ctx,
		`SELECT SERVER_ID, SERVER_STATUS, FREE_DISK FROM Aion_log.aiongm_ur.TBL_GAME_SERVER_INFO ORDER BY SERVER_ID`)
	if err == nil {
		for rows.Next() {
			var si ServerInfo
			if err := rows.Scan(&si.ServerID, &si.Status, &si.FreeDisk); err == nil {
				s.SrvInfo = append(s.SrvInfo, si)
			}
		}
		rows.Close()
	}
	return s
}

// --- Mock (dev/демо) ---

type Mock struct{}

func NewMock() *Mock { return &Mock{} }

func (m *Mock) Collect(ctx context.Context) Snap {
	return Snap{
		When:  time.Now(),
		World: []Ccu{{World: 1, Light: 1, Dark: 0, Npc: 300}},
		Auth:  []AuthCcu{{ServerID: 1, WorldUser: 1, LimitUser: 500, AuthUser: 1, WaitUser: 0}},
		Gauge: Gauge{Blocked: 0, ResqDepth: 0, ResqWaitMs: 0},
		Waits: []WaitDelta{
			{Type: "WRITELOG", DMS: 120, DCnt: 3},
			{Type: "PAGEIOLATCH_SH", DMS: 60, DCnt: 2},
		},
		SrvInfo: []ServerInfo{{ServerID: 2, Status: 1, FreeDisk: sql.NullInt64{Int64: 42, Valid: true}}},
	}
}

var _ = fmt.Sprintf // резерв

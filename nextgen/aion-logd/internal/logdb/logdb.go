// Package logdb — DB-слой logd: воспроизведение ровно тех SQL-вызовов,
// которые делает оригинальный LogServer64 (строки взяты из его exe):
//
//	{call Log_TblGameServerInfo_UpdateLogfreedisk(%d,%d)}
//	{call Log_TblGameServerInfo_UpdateServerstatus(%d,%d,%d)}
//	{call Log_TblGameWorldInfo_InitializeCount(%d)}
//
// Текстовые логи в БД НЕ пишем (aion_BulkInsertWide — отдельный этап, по сверке).
// Интерфейс Executor → юнит-тесты на фейке без реальной БД.
package logdb

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// Executor — минимум от *sql.DB (для фейка в тестах).
type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type DB struct {
	Ex            Executor
	WorldID       int           // в TBL-процах оригинала = 1
	ServerID      int           // SERVER_ID=2 зафиксирован внутри процы freedisk; тут для serverstatus
	FreediskEvery time.Duration // 0 = выкл (оригинал ~5 мин)
	StatusEvery   time.Duration // 0 = выкл
	LogDir        string        // диск, free-space которого репортим
	FreeMB        func(dir string) (int, error)
	// Notify — хук телеметрии (ship), вызывается на действия/ошибки таймеров. Может быть nil.
	Notify func(ev, msg string, kv map[string]any)
}

func (d *DB) notify(ev, msg string, kv map[string]any) {
	if d.Notify != nil {
		d.Notify(ev, msg, kv)
	}
}

type Cfg struct {
	Enabled     bool   `yaml:"enabled"`
	Conn        string `yaml:"conn"`         // sqlserver://user:pass@host?database=master
	WorldID     int    `yaml:"world_id"`     // 1
	ServerID    int    `yaml:"server_id"`    // 2 (в проце freedisk зафиксирован)
	FreediskSec int    `yaml:"freedisk_sec"` // 0 = выкл
	StatusSec   int    `yaml:"status_sec"`   // 0 = выкл
	LogDir      string `yaml:"log_dir"`      // диск для free-space
}

// Open — реальная БД (go-mssqldb, "sqlserver"). Тесты используют Executor-фейк.
func (c Cfg) Open() (*DB, error) {
	db, err := sql.Open("sqlserver", c.Conn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(2)
	return &DB{
		Ex:            db,
		WorldID:       c.WorldID,
		ServerID:      c.ServerID,
		FreediskEvery: time.Duration(c.FreediskSec) * time.Second,
		StatusEvery:   time.Duration(c.StatusSec) * time.Second,
		LogDir:        c.LogDir,
		FreeMB:        FreeMB,
	}, nil
}

const ()

func (d *DB) UpdateLogfreedisk(ctx context.Context, freeDiskGB int) error {
	q := fmt.Sprintf("exec Aion_log.dbo.Log_TblGameServerInfo_UpdateLogfreedisk @free_disk=%d, @world_id=%d",
		freeDiskGB, d.WorldID)
	_, err := d.Ex.ExecContext(ctx, q)
	return err
}

func (d *DB) UpdateServerstatus(ctx context.Context, status int) error {
	q := fmt.Sprintf("exec Aion_log.dbo.Log_TblGameServerInfo_UpdateServerstatus @server_status=%d, @world_id=%d, @server_id=%d",
		status, d.WorldID, d.ServerID)
	_, err := d.Ex.ExecContext(ctx, q)
	return err
}

// InitializeCount — вызывается logd'ом один раз при старте мира (ServerStarted svc=3, Л3).
func (d *DB) InitializeCount(ctx context.Context) error {
	q := fmt.Sprintf("exec Aion_log.dbo.Log_TblGameWorldInfo_InitializeCount @world_id=%d", d.WorldID)
	_, err := d.Ex.ExecContext(ctx, q)
	return err
}

// UpdateMainStatus — Л2: зона-агрегат оригинала (свет/тьма/NPC/СТОРО по зоне).
// ПОЗИЦИОННЫЙ {call} — имена параметров не завязаны на sp_helptext.
// СЕМАНТИКА АРГУМЕНТОВ (какой metric type-5 = light/dark/npc) — после DB-сверки;
// вызывать только при включённом mainstatus_sec и подтверждённом маппинге.
func (d *DB) UpdateMainStatus(ctx context.Context, light, dark, npc, pcL, pcD, world, zone int) error {
	q := fmt.Sprintf("{call Aion_log.dbo.Log_TblGameWorldInfo_UpdateMainStatus(%d,%d,%d,%d,%d,%d,%d)}",
		light, dark, npc, pcL, pcD, world, zone)
	_, err := d.Ex.ExecContext(ctx, q)
	return err
}

// UpdateTotalMainStatus — Л2: тотал-строка зоны 0 (SET БЕЗ PC_STORE_DARK_USERS, REF58).
func (d *DB) UpdateTotalMainStatus(ctx context.Context, light, dark, npc, pcL, world, zone int) error {
	q := fmt.Sprintf("{call Aion_log.dbo.Log_TblGameWorldInfo_UpdateTotalMainStatus(%d,%d,%d,%d,%d,%d)}",
		light, dark, npc, pcL, world, zone)
	_, err := d.Ex.ExecContext(ctx, q)
	return err
}

// RunTimers — два тикера оригинала (freedisk/status). Блокирует до ctx.Done.
func (d *DB) RunTimers(ctx context.Context) {
	if d.FreediskEvery > 0 {
		go d.ticker(ctx, d.FreediskEvery, "freedisk", func() error {
			mb, err := d.FreeMB(d.LogDir)
			if err != nil {
				return err
			}
			// оригинал шлёт маленькое число (56) — это ГБ, столбец/параметр tinyint
			gb := mb / 1024
			if gb > 255 {
				gb = 255
			}
			return d.UpdateLogfreedisk(ctx, gb)
		})
	}
	if d.StatusEvery > 0 {
		go d.ticker(ctx, d.StatusEvery, "serverstatus", func() error {
			return d.UpdateServerstatus(ctx, 1) // 1 = working (как в TBL_GAME_SERVER_INFO живого профиля)
		})
	}
}

func (d *DB) ticker(ctx context.Context, every time.Duration, name string, fn func() error) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := fn(); err != nil {
				log.Printf("[logdb] %s: %v", name, err)
				d.notify("db.err", name, map[string]any{"err": err.Error()})
			} else {
				d.notify("db", name, nil)
			}
		}
	}
}

var _ = fmt.Sprintf

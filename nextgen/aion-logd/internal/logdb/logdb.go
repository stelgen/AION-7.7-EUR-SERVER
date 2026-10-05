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

const (
	qFreedisk  = `{call Log_TblGameServerInfo_UpdateLogfreedisk(?,?)}`
	qStatus    = `{call Log_TblGameServerInfo_UpdateServerstatus(?,?,?)}`
	qInitCount = `{call Log_TblGameWorldInfo_InitializeCount(?)}`
)

func (d *DB) UpdateLogfreedisk(ctx context.Context, freeDiskMB int) error {
	_, err := d.Ex.ExecContext(ctx, qFreedisk, freeDiskMB, d.WorldID)
	return err
}

func (d *DB) UpdateServerstatus(ctx context.Context, status int) error {
	_, err := d.Ex.ExecContext(ctx, qStatus, status, d.WorldID, d.ServerID)
	return err
}

// InitializeCount — вызывается logd'ом один раз при появлении мира (ServerStarted).
func (d *DB) InitializeCount(ctx context.Context) error {
	_, err := d.Ex.ExecContext(ctx, qInitCount, d.WorldID)
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
			return d.UpdateLogfreedisk(ctx, mb)
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
			}
		}
	}
}

var _ = fmt.Sprintf

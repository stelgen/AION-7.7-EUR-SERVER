// world_store.go — Этап 5 wiring: SQLStore для мир-БД (_AionWorldNew114_rc).
package db

import (
	"context"
	"database/sql"
)

// WorldStore — обёртка SQL для мир-БД (использует тот же Executor interface).
type WorldStore struct{ db *sql.DB }

// NewWorldStore — подключение к мир-БД.
func NewWorldStore(conn string) (*WorldStore, error) {
	db, err := sql.Open("sqlserver", conn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return &WorldStore{db: db}, nil
}

// Exec — реализация Executor.
func (ws *WorldStore) Exec(ctx context.Context, query string, args ...any) (Result, error) {
	res, err := ws.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	aff, _ := res.RowsAffected()
	return &worldResult{affected: aff}, nil
}

// Query — реализация Executor.
func (ws *WorldStore) Query(ctx context.Context, query string, args ...any) (Rows, error) {
	rows, err := ws.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &worldRows{rows: rows}, nil
}

// worldRows — адаптер sql.Rows → db.Rows.
type worldRows struct{ rows *sql.Rows }

func (wr *worldRows) Next() bool       { return wr.rows.Next() }
func (wr *worldRows) Scan(dest ...any) error { return wr.rows.Scan(dest...) }
func (wr *worldRows) Close() error     { return wr.rows.Close() }
func (wr *worldRows) Err() error       { return wr.rows.Err() }

// worldResult — адаптер sql.Result → db.Result.
type worldResult struct{ affected int64 }

func (wr *worldResult) RowsAffected() (int64, error) { return wr.affected, nil }

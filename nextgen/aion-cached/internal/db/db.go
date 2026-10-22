// Package db — DB-слой мира CacheD64 (781 procs, каркас R3).
// Тела procs сняты в cached-ref/db-procs-world-781.txt (REF58-сверка);
// multi-version dispatch: 155 versioned (_20XXXXXX) + 140 exe-only missing.
//
// SQLStore подключается после capture-раскладок (R4); сейчас каркас:
// Executor-интерфейс (фейк в тестах) + CallWorldProc с ранним ErrProcMissing
// (exe зовёт 140 procs, которых в прод-БД нет — поведение-маркер для A/B).
// connStr = СЕКРЕТ: только на VM/env (S6).
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Ошибки DB-слоя.
var (
	// ErrProcMissing — exe ссылается на proc, которой в прод-БД НЕТ (из 140).
	// Ориг в этой ситуации держит multi-version dispatch и, вероятно, фейлится
	// на SQL-уровне; наш маркер = ранняя детекция для A/B-сравнения (журнал #12).
	ErrProcMissing = errors.New("db: proc missing in prod DB (exe-only)")
	// ErrProcUnknown — proc не из списков (ни 781, ни 140) — новая/опечатка.
	ErrProcUnknown = errors.New("db: proc unknown (not in 781+140 inventory)")
)

// Rows/Result — интерфейсы (фейк в тестах без sqlmock; как accache).
type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Close() error
	Err() error
}

type Result interface {
	RowsAffected() (int64, error)
}

// Executor — абстракция БД (фейк в тестах / SQLStore в проде).
type Executor interface {
	Exec(ctx context.Context, query string, args ...any) (Result, error)
	Query(ctx context.Context, query string, args ...any) (Rows, error)
}

// SQLStore — реальная БД (подключение после capture-раскладок R4).
type SQLStore struct{ db *sql.DB }

type sqlResult struct{ r sql.Result }

func (s sqlResult) RowsAffected() (int64, error) { return s.r.RowsAffected() }

type sqlRows struct{ r *sql.Rows }

func (s sqlRows) Next() bool             { return s.r.Next() }
func (s sqlRows) Scan(d ...any) error    { return s.r.Scan(d...) }
func (s sqlRows) Close() error           { return s.r.Close() }
func (s sqlRows) Err() error             { return s.r.Err() }

// NewSQLStore — открытие (connStr = СЕКРЕТ на VM/env, S6).
func NewSQLStore(conn string) (*SQLStore, error) {
	d, err := sql.Open("sqlserver", conn)
	if err != nil {
		return nil, fmt.Errorf("cached db: %w", err)
	}
	d.SetMaxOpenConns(10) // common.xml numberOfDBThreads=10
	if err := d.Ping(); err != nil {
		d.Close()
		return nil, fmt.Errorf("cached db ping: %w", err)
	}
	return &SQLStore{db: d}, nil
}

func (s *SQLStore) Exec(ctx context.Context, q string, args ...any) (Result, error) {
	r, err := s.db.ExecContext(ctx, q, args...)
	return sqlResult{r}, err
}

func (s *SQLStore) Query(ctx context.Context, q string, args ...any) (Rows, error) {
	r, err := s.db.QueryContext(ctx, q, args...)
	return sqlRows{r}, err
}

// CallWorldProc — универсальный вызов proc мира с проверкой инвентаря:
//   - IsKnownMissing(name) → ErrProcMissing БЕЗ SQL (A/B маркер, ж#12);
//   - !ProcExists(name)    → ErrProcUnknown;
//   - иначе {call dbo.<name>(?,...)} позиционно; proc без result-set → Result.
//
// Формат {call} = тот же, что accache (ODBC-канон NCsoft).
func CallWorldProc(ctx context.Context, ex Executor, name string, args ...any) (Rows, Result, error) {
	if IsKnownMissing(name) {
		return nil, nil, fmt.Errorf("%w: %s", ErrProcMissing, name)
	}
	if !ProcExists(name) {
		return nil, nil, fmt.Errorf("%w: %s", ErrProcUnknown, name)
	}
	q := "{call dbo." + name
	if len(args) > 0 {
		q += "(" + strings.Repeat("?,", len(args))
		q = q[:len(q)-1] + ")}"
	} else {
		q += ")}"
	}
	rows, err := ex.Query(ctx, q, args...)
	if err != nil {
		if res, e2 := ex.Exec(ctx, q, args...); e2 == nil {
			return nil, res, nil
		}
		return nil, nil, fmt.Errorf("cached db %s: %w", name, err)
	}
	return rows, nil, nil
}

// Deadline — стандартный таймаут SQL-вызова (10с, как accache).
func Deadline() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}
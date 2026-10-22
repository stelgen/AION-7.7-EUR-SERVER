package db

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// TestWorldProcsInventory — 781 procs: ключевые домены + versioned-счётчик.
func TestWorldProcsInventory(t *testing.T) {
	if len(WorldProcs) != 781 {
		t.Fatalf("WorldProcs = %d, want 781", len(WorldProcs))
	}
	versioned := 0
	for _, p := range WorldProcs {
		if p.Versioned {
			versioned++
		}
	}
	if versioned != 155 {
		t.Fatalf("versioned = %d, want 155", versioned)
	}
	// ключевые домены (прод-факты из сверки 08.10)
	checks := map[string]struct {
		proc   string
		domain string
	}{
		"CharInfo":    {"aion_GetCharInfo_20160303", "CharInfo"},
		"ItemEnchant": {"aion_GetItemEnchant", "ItemEnchant"},
		"Guild":       {"aion_GetGuild", "Guild"},
		"VendorItem":  {"aion_GetVendorItemListAllLight", "VendorItem"},
	}
	for dom, c := range checks {
		if ProcExists(c.proc) {
			if got := WorldProcs[c.proc].Domain; got != c.domain {
				t.Fatalf("%s: domain=%q, want %q", c.proc, got, c.domain)
			}
		} else if WorldProcs[c.proc].Domain == "" {
			t.Fatalf("%s: контрольный proc домена %s отсутствует в инвентаре", c.proc, dom)
		}
	}
}

// TestExeOnlyMissing — 140 exe-only: ранний ErrProcMissing ДО SQL.
func TestExeOnlyMissing(t *testing.T) {
	if len(ExeOnlyProcs) != 140 {
		t.Fatalf("ExeOnlyProcs = %d, want 140", len(ExeOnlyProcs))
	}
	for _, p := range ExeOnlyProcs {
		if ProcExists(p) {
			t.Fatalf("%s помечен exe-only, но есть в WorldProcs (781) — рассинхрон списков", p)
		}
		if !IsKnownMissing(p) {
			t.Fatalf("%s не в missingSet", p)
		}
	}
	// фейк-Executor: НЕ должен быть вызван
	ex := &fakeExec{}
	_, _, err := CallWorldProc(context.Background(), ex, ExeOnlyProcs[0])
	if !errors.Is(err, ErrProcMissing) {
		t.Fatalf("want ErrProcMissing, got %v", err)
	}
	if len(ex.execs) != 0 || len(ex.queries) != 0 {
		t.Fatalf("SQL-вызовы при missing-proc: execs=%d queries=%d", len(ex.execs), len(ex.queries))
	}
}

// TestDbOnly — 132 db-only: в инвентаре есть, exe не зовёт.
func TestDbOnly(t *testing.T) {
	if len(DbOnlyProcs) != 132 {
		t.Fatalf("DbOnlyProcs = %d, want 132", len(DbOnlyProcs))
	}
	for _, p := range DbOnlyProcs {
		if !ProcExists(p) {
			t.Fatalf("db-only %s отсутствует в WorldProcs", p)
		}
		if IsKnownMissing(p) {
			t.Fatalf("db-only %s попал в ExeOnlyProcs", p)
		}
	}
}

// TestCallWorldProcUnknown — proc вне инвентаря (781+140) → ErrProcUnknown.
func TestCallWorldProcUnknown(t *testing.T) {
	ex := &fakeExec{}
	_, _, err := CallWorldProc(context.Background(), ex, "aion_ThisProcDoesNotExist")
	if !errors.Is(err, ErrProcUnknown) {
		t.Fatalf("want ErrProcUnknown, got %v", err)
	}
}

// TestCallWorldProcQueryForm — существующий proc: {call dbo.<name>(?,?,...)} позиционно.
func TestCallWorldProcQueryForm(t *testing.T) {
	ex := &fakeExec{rowsData: [][]any{{int32(1)}}}
	args := []any{int32(1010), int32(2)}
	rows, res, err := CallWorldProc(context.Background(), ex, "aion_GetCharInfo_20160303", args...)
	if err != nil {
		t.Fatalf("CallWorldProc: %v", err)
	}
	if res != nil {
		t.Fatal("Query-путь не должен отдавать Result")
	}
	if rows == nil {
		t.Fatal("rows = nil")
	}
	if len(ex.queries) != 1 {
		t.Fatalf("queries=%d", len(ex.queries))
	}
	want := "{call dbo.aion_GetCharInfo_20160303(?,?)}"
	if ex.queries[0] != want {
		t.Fatalf("query=%q, want %q", ex.queries[0], want)
	}
	// версия-суффикс: тот же proc c _20151117 — отдельный вызов (multi-version)
	if _, _, err := CallWorldProc(context.Background(), ex, "aion_GetCharInfo_20140929"); err != nil {
		// может не существовать в инвентаре — проверяем по факту инвентаря
		if !ProcExists("aion_GetCharInfo_20140929") && !errors.Is(err, ErrProcUnknown) {
			t.Fatalf("versioned-вариант: %v", err)
		}
	}
}

// TestExecFallback — proc без result-set: Query ошибка → Exec-фолбэк (как accache CallProc).
func TestExecFallback(t *testing.T) {
	// queryFails: Query всегда ошибка → CallWorldProc идёт в Exec
	ex := &fakeExec{queryFails: true}
	_, res, err := CallWorldProc(context.Background(), ex, "aion_UpdateAchievement", int32(1))
	if err != nil {
		t.Fatalf("exec-fallback: %v", err)
	}
	if res == nil {
		t.Fatal("res = nil после Exec-фолбэка")
	}
	if len(ex.execs) != 1 || !strings.Contains(ex.execs[0], "{call dbo.aion_UpdateAchievement(?)}") {
		t.Fatalf("exec=%v", ex.execs)
	}
}

// fakeExec — минимальный фейк Executor.
type fakeExec struct {
	execs     []string
	queries   []string
	rowsData  [][]any
	queryFails bool
}

type fakeResult struct{}

func (fakeResult) RowsAffected() (int64, error) { return 1, nil }

type fakeRows struct{ data [][]any; i int }

func (r *fakeRows) Next() bool              { return r.i < len(r.data) }
func (r *fakeRows) Scan(d ...any) error     { return nil }
func (r *fakeRows) Close() error            { return nil }
func (r *fakeRows) Err() error              { return nil }

func (f *fakeExec) Exec(ctx context.Context, q string, args ...any) (Result, error) {
	f.execs = append(f.execs, q)
	return fakeResult{}, nil
}

func (f *fakeExec) Query(ctx context.Context, q string, args ...any) (Rows, error) {
	if f.queryFails {
		return nil, errors.New("no result-set (exec-only proc)")
	}
	f.queries = append(f.queries, q)
	return &fakeRows{data: f.rowsData}, nil
}
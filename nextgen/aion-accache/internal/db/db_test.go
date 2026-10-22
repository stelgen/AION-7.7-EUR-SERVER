// R3-тесты db-слоя: формы запросов и раскладки аргументов (без живой БД —
// арбитр live A/B). Тела procs = accountcache-ref/db-procs-77-ref58.rpt.
package db

import (
	"context"
	"strings"
	"testing"
	"time"
)

type capExec struct {
	queries []string
	qArgs   [][]any
	execs   []string
	eArgs   [][]any
	rows    [][]any
}

type capResult struct{}

func (capResult) RowsAffected() (int64, error) { return 1, nil }

type capRows struct {
	data [][]any
	i    int
}

func (r *capRows) Next() bool { return r.i < len(r.data) }

// Scan — честный (через указатели; урок replayRows: copy(dest,row) без deref
// оставляет колонки нулями).
func (r *capRows) Scan(dest ...any) error {
	row := r.data[r.i]
	for j, d := range dest {
		if j >= len(row) {
			break
		}
		switch dv := d.(type) {
		case *int64:
			switch x := row[j].(type) {
			case int64:
				*dv = x
			case int32:
				*dv = int64(x)
			case int:
				*dv = int64(x)
			}
		case *int32:
			switch x := row[j].(type) {
			case int32:
				*dv = x
			case int64:
				*dv = int32(x)
			case int:
				*dv = int32(x)
			}
		}
	}
	r.i++
	return nil
}
func (r *capRows) Close() error { return nil }
func (r *capRows) Err() error             { return nil }

func (c *capExec) Query(ctx context.Context, q string, args ...any) (Rows, error) {
	c.queries = append(c.queries, q)
	c.qArgs = append(c.qArgs, args)
	return &capRows{data: c.rows}, nil
}

func (c *capExec) Exec(ctx context.Context, q string, args ...any) (Result, error) {
	c.execs = append(c.execs, q)
	c.eArgs = append(c.eArgs, args)
	return capResult{}, nil
}

// TestGetUserInfo — SELECT last_login_time + MAX(user_level) (acc, acc, char).
func TestGetUserInfo(t *testing.T) {
	ex := &capExec{rows: [][]any{{int64(0x6AC5E7B9), int32(80)}}}
	ui, ok, err := GetUserInfo(context.Background(), ex, 1010, 1002)
	if err != nil || !ok {
		t.Fatalf("GetUserInfo: ok=%v err=%v", ok, err)
	}
	if ui.LastLogin != 0x6AC5E7B9 || ui.MaxLevel != 80 {
		t.Fatalf("UserInfo = %+v", ui)
	}
	q := ex.queries[0]
	if !strings.Contains(q, "global_user_data") || !strings.Contains(q, "MAX(user_level)") ||
		!strings.Contains(q, "is_special_server = 0") || !strings.Contains(q, "delete_date = 0") {
		t.Fatalf("GetUserInfo query = %s", q)
	}
	if len(ex.qArgs[0]) != 3 || ex.qArgs[0][0] != int32(1010) || ex.qArgs[0][1] != int32(1010) || ex.qArgs[0][2] != int32(1002) {
		t.Fatalf("GetUserInfo args = %v", ex.qArgs[0])
	}
}

// TestGetUserInfoEmpty — нет строки => ok=false, нули, ошибки нет.
func TestGetUserInfoEmpty(t *testing.T) {
	ex := &capExec{}
	ui, ok, err := GetUserInfo(context.Background(), ex, 7, 999)
	if err != nil || ok || ui.LastLogin != 0 || ui.MaxLevel != 0 {
		t.Fatalf("empty = %+v ok=%v err=%v", ui, ok, err)
	}
}

// TestSetLoginUser — {call dbo.aion_SetLoginUser_20120703(?,?,?,?)} позиционно.
func TestSetLoginUser(t *testing.T) {
	ex := &capExec{}
	lt := time.Date(2026, 10, 22, 6, 53, 11, 620e6, time.UTC)
	if err := SetLoginUser(context.Background(), ex, 1002, 1, 1, lt); err != nil {
		t.Fatal(err)
	}
	if ex.execs[0] != "EXEC dbo.aion_SetLoginUser_20120703 @char_id=@p1, @server_id=@p2, @cur_server_id=@p3, @login_time=@p4" {
		t.Fatalf("query = %s", ex.execs[0])
	}
	a := ex.eArgs[0]
	if len(a) != 4 || a[0] != int32(1002) || a[1] != int32(1) || a[2] != int32(1) || !a[3].(time.Time).Equal(lt) {
		t.Fatalf("args = %v", a)
	}
}

// TestSetLogoutUser — {call dbo.aion_SetLogoutUser_20120703(?,?,?,?)} позиционно.
func TestSetLogoutUser(t *testing.T) {
	ex := &capExec{}
	lt := time.Date(2026, 10, 22, 6, 56, 17, 617e6, time.UTC)
	if err := SetLogoutUser(context.Background(), ex, 1003, 1, 1, lt); err != nil {
		t.Fatal(err)
	}
	if ex.execs[0] != "EXEC dbo.aion_SetLogoutUser_20120703 @char_id=@p1, @server_id=@p2, @cur_server_id=@p3, @logout_time=@p4" {
		t.Fatalf("query = %s", ex.execs[0])
	}
	a := ex.eArgs[0]
	if len(a) != 4 || a[0] != int32(1003) || a[1] != int32(1) || a[2] != int32(1) || !a[3].(time.Time).Equal(lt) {
		t.Fatalf("args = %v", a)
	}
}

// TestReadOnly — обёртка тени A/B: Exec глушится (nop, без ошибки), Query проходит.
func TestReadOnly(t *testing.T) {
	ex := &capExec{rows: [][]any{{int64(1), int32(80)}}}
	ro := NewReadOnly(ex)
	res, err := ro.Exec(context.Background(), "{call dbo.aion_SetLoginUser_20120703(?,?,?,?)}", 1, 1, 1, time.Now())
	if err != nil || res == nil {
		t.Fatalf("ro Exec: err=%v res=%v", err, res)
	}
	if len(ex.execs) != 0 {
		t.Fatalf("Exec просочился в нижний слой: %v", ex.execs)
	}
	rows, err := ro.Query(context.Background(), "SELECT 1")
	if err != nil || rows == nil {
		t.Fatalf("ro Query: %v", err)
	}
	if len(ex.queries) != 1 || ex.queries[0] != "SELECT 1" {
		t.Fatalf("queries = %v", ex.queries)
	}
}

package logdb

import (
	"context"
	"database/sql"
	"testing"
)

type fakeExec struct {
	calls []struct {
		q    string
		args []any
	}
	err error
}

func (f *fakeExec) ExecContext(_ context.Context, q string, args ...any) (sql.Result, error) {
	f.calls = append(f.calls, struct {
		q    string
		args []any
	}{q, args})
	if f.err != nil {
		return nil, f.err
	}
	return fakeResult{}, nil
}

type fakeResult struct{}

func (fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (fakeResult) RowsAffected() (int64, error) { return 0, nil }

func newTestDB() (*DB, *fakeExec) {
	fe := &fakeExec{}
	return &DB{Ex: fe, WorldID: 1, ServerID: 2}, fe
}

func TestUpdateLogfreedisk(t *testing.T) {
	db, fe := newTestDB()
	if err := db.UpdateLogfreedisk(context.Background(), 56); err != nil {
		t.Fatal(err)
	}
	want := "exec Aion_log.dbo.Log_TblGameServerInfo_UpdateLogfreedisk @free_disk=56, @world_id=1"
	if len(fe.calls) != 1 || fe.calls[0].q != want {
		t.Fatalf("calls: %+v", fe.calls)
	}
	if len(fe.calls[0].args) != 0 {
		t.Fatalf("args: %+v", fe.calls[0].args)
	}
}

func TestUpdateServerstatus(t *testing.T) {
	db, fe := newTestDB()
	if err := db.UpdateServerstatus(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	want := "exec Aion_log.dbo.Log_TblGameServerInfo_UpdateServerstatus @server_status=1, @world_id=1, @server_id=2"
	if len(fe.calls) != 1 || fe.calls[0].q != want {
		t.Fatalf("calls: %+v", fe.calls)
	}
}

func TestInitializeCount(t *testing.T) {
	db, fe := newTestDB()
	if err := db.InitializeCount(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := "exec Aion_log.dbo.Log_TblGameWorldInfo_InitializeCount @world_id=1"
	if len(fe.calls) != 1 || fe.calls[0].q != want {
		t.Fatalf("calls: %+v", fe.calls)
	}
}

func TestDBErrorPropagates(t *testing.T) {
	db, fe := newTestDB()
	fe.err = context.DeadlineExceeded
	if err := db.UpdateLogfreedisk(context.Background(), 1); err == nil {
		t.Fatal("ошибка executor должна возвращаться")
	}
}

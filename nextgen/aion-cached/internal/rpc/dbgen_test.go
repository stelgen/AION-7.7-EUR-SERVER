// DB-генератор тесты: RP cmd=1 из captured-шаблона + DB-подстановка.
package rpc

import (
	"encoding/binary"
	"testing"
)

// fakeExec для DB-генератора.
type dbgenExec struct {
	rows [][]any
}

func (e *dbgenExec) Next() bool                           { return false }
func (e *dbgenExec) Scan(dst ...any) error                { return nil }
func (e *dbgenExec) Close() error                         { return nil }
func (e *dbgenExec) Err() error                           { return nil }
func (e *dbgenExec) RowsAffected() (int64, error)         { return 0, nil }

func TestDBGenRPLoad(t *testing.T) {
	tpl := loadTemplate("rp1_char_struct.bin")
	if tpl == nil {
		t.Skip("no template")
	}
	// без Executor — чистый replay с X-подстановкой
	dbg := &DBGen{}
	out := dbg.GenerateRPLoadDB(nil, 0xBEEF, 1002, 1010)
	if len(out) != len(tpl) {
		t.Fatalf("size mismatch: %d vs %d", len(out), len(tpl))
	}
	x := binary.LittleEndian.Uint32(out[7:11])
	if x != 0xBEEF {
		t.Fatalf("X = %#x", x)
	}
	// charRef подставлен
	charRef := binary.LittleEndian.Uint32(out[28:32])
	if charRef&0xFFFF != 1002 {
		t.Fatalf("charRef = %#x, charId low = %d", charRef, charRef&0xFFFF)
	}
	// уровень (шаблон: 0x50 = 80)
	if out[54] != 80 {
		t.Logf("level byte = %d (template value, DB not available)", out[54])
	}
}

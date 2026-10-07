package world

import (
	"bytes"
	"testing"
)

// TestApplyStatUpdate — канон capture: SM_STATUPDATE_HP = [HP u32 @0][MP u32 @4]; подмена из сессии.
func TestApplyStatUpdate(t *testing.T) {
	raw := []byte{0x10, 0x27, 0, 0, 0x64, 0, 0, 0} // HP=10000 MP=100 из прод-раскладки
	p := Patch{Name: "SM_STATUPDATE_HP", Fields: map[int]string{0: "hp", 4: "mp"}}
	out := Apply(raw, p, map[string]uint32{"hp": 12345, "mp": 678})
	if !bytes.Equal(out[:4], []byte{0x39, 0x30, 0, 0}) { // 12345 = 0x3039 LE
		t.Fatalf("hp: % X (want 39 30 00 00)", out[:4])
	}
	if !bytes.Equal(out[4:8], []byte{0xA6, 0x02, 0, 0}) { // 678 = 0x2A6 LE
		t.Fatalf("mp: % X (want A6 02 00 00)", out[4:8])
	}
	// исходная раскладка не мутируется
	if raw[0] != 0x10 {
		t.Fatalf("layout mutated")
	}
}

// TestApplyTime — директива time подставляет актуальный unix (не из прошлого).
func TestApplyTime(t *testing.T) {
	raw := make([]byte, 4)
	p := Patch{Name: "SM_TIME_CHECK", Fields: map[int]string{0: "time"}}
	out := Apply(raw, p, nil)
	ts := int(out[0]) | int(out[1])<<8 | int(out[2])<<16 | int(out[3])<<24
	if ts < 1790000000 { // 2026-10
		t.Fatalf("time: %d", ts)
	}
}

// TestLoadPatchesYAML — реальный layouts.yaml проекта.
func TestLoadPatchesYAML(t *testing.T) {
	patches, err := LoadPatches("testdata/layouts.yaml")
	if err != nil {
		t.Skipf("layouts.yaml недоступен: %v", err)
	}
	if len(patches) < 2 {
		t.Fatalf("patches: %d", len(patches))
	}
	if _, ok := patches["SM_STATUPDATE_HP"].Fields[0]; !ok {
		t.Fatalf("SM_STATUPDATE_HP field 0 отсутствует")
	}
}

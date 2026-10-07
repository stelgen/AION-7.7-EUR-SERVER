package world

import (
	"bytes"
	"testing"
)

// TestLoadRealLayouts — РЕАЛЬНЫЕ прод-раскладки из capture 08.10 (инвентарь 3167Б, скиллы 2878Б...).
func TestLoadRealLayouts(t *testing.T) {
	ls, err := New("testdata/layouts")
	if err != nil {
		t.Skipf("layouts недоступны: %v", err)
	}
	if n := ls.Count(); n < 10 {
		t.Fatalf("мало раскладок: %d", n)
	}
	inv, ok := ls.Get("SM_INVENTORY_INFO")
	if !ok || len(inv) != 3167 {
		t.Fatalf("SM_INVENTORY_INFO: ok=%v len=%d (want 3167)", ok, len(inv))
	}
	if sk, ok := ls.Get("SM_SKILL_LIST"); !ok || len(sk) != 2878 {
		t.Fatalf("SM_SKILL_LIST: ok=%v len=%d (want 2878)", ok, len(sk))
	}
	seq := ls.InitSequence()
	if len(seq) < 7 {
		t.Fatalf("InitSequence короткая: %d", len(seq))
	}
	if seq[0][0] != "SM_ABNORMAL_STATE" || seq[3][0] != "SM_INVENTORY_INFO" {
		t.Fatalf("порядок инициализации нарушен: %v", seq)
	}
}

// TestReplaceOID — R3.7: подмена всех вхождений OID-константы capture-сессии.
func TestReplaceOID(t *testing.T) {
	// синтетика: payload с двумя вхождениями OID 0x443E29B5 (LE b5 29 3e 44)
	raw := []byte{0xB5, 0x29, 0x3E, 0x44, 0xFF, 0xB5, 0x29, 0x3E, 0x44}
	out := ReplaceOID(raw, 0x443E29B5, 0x1001)
	want := []byte{0x01, 0x10, 0, 0, 0xFF, 0x01, 0x10, 0, 0}
	if !bytesEqual(out, want) {
		t.Fatalf("ReplaceOID: got % X want % X", out, want)
	}
	// факт capture: в 13 раскладках OID отсутствует
	if inv, ok := GetLayout(t); ok {
		if bytesContains(inv, []byte{0xB5, 0x29, 0x3E, 0x44}) {
			t.Fatalf("OID найден в SM_INVENTORY_INFO — факт персоно-агностичности нарушен")
		}
	}
}

func bytesEqual(a, b []byte) bool { return bytes.Equal(a, b) }

func bytesContains(hay, needle []byte) bool { return bytes.Contains(hay, needle) }

func GetLayout(t *testing.T) ([]byte, bool) {
	t.Helper()
	ls, err := New("testdata/layouts")
	if err != nil {
		return nil, false
	}
	return ls.Get("SM_INVENTORY_INFO")
}

package world

import "testing"

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
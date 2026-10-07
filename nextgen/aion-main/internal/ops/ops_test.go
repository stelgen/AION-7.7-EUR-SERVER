package ops

import "testing"

// TestLoadRealYAML — реальный ops.yaml проекта (688 имён, канарейка SM_KEY 0x48).
func TestLoadRealYAML(t *testing.T) {
	r, err := Load("../../ops.yaml")
	if err != nil {
		t.Skipf("ops.yaml недоступен из этого каталога: %v", err)
	}
	if r.Count() < 600 {
		t.Fatalf("реестр слишком мал: %d", r.Count())
	}
	p, ok := r.Lookup(0x48, "SM")
	if !ok || p.Name != "SM_KEY" {
		t.Fatalf("SM_KEY lookup: %v %v", p, ok)
	}
	if p, ok := r.Lookup(0x0106, "CM"); !ok || p.Name != "CM_MOVE" {
		t.Fatalf("CM_MOVE lookup: %v %v", p, ok)
	}
	if _, ok := r.Lookup(0x00D6, "CM"); !ok {
		t.Fatalf("CM_VERSION_CHECK отсутствует")
	}
}

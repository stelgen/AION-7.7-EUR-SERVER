package store

import "testing"

func TestMapGetOrCreate(t *testing.T) {
	s := NewMap()
	a1, created, err := s.GetOrCreate("stelgen", true)
	if err != nil || !created || a1.UID != 1 {
		t.Fatalf("first: %+v created=%v err=%v", a1, created, err)
	}
	a2, created, err := s.GetOrCreate("stelgen", true)
	if err != nil || created || a2.UID != 1 {
		t.Fatalf("second: %+v created=%v", a2, created)
	}
	if _, _, err := s.GetOrCreate("ghost", false); err != ErrNotFound {
		t.Fatalf("no-autocreate: want ErrNotFound, got %v", err)
	}
	if _, _, err := s.GetOrCreate("ghost", true); err != nil {
		t.Fatalf("autocreate: %v", err)
	}
}

func TestMapSeeded(t *testing.T) {
	// live: акк "1" = uid 7, "stelgen" = 1010
	s := NewMapSeeded(map[string]uint32{"1": 7, "stelgen": 1010})
	a, created, _ := s.GetOrCreate("1", false)
	if created || a.UID != 7 {
		t.Fatalf("seeded: %+v created=%v", a, created)
	}
	a, _, _ = s.GetOrCreate("new", true)
	if a.UID != 1011 {
		t.Fatalf("nextUID: %d want 1011", a.UID)
	}
}

func TestMapBlocks(t *testing.T) {
	s := NewMap()
	a, _, _ := s.GetOrCreate("bad", false)
	if a.Blocked() {
		t.Fatal("не заблокирован")
	}
	s.AddBlock(a.UID, 3, "причина")
	rs, err := s.Blocks(a.UID)
	if err != nil || len(rs) != 1 || rs[0].Code != 3 {
		t.Fatalf("blocks: %+v err=%v", rs, err)
	}
}

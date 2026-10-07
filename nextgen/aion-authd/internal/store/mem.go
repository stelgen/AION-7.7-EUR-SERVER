package store

import "sync"

// MapStore — in-memory Store (дефолт для тестов/стенда/standalone).
// uid растёт с 1 (live: акк "1" = uid 7, "stelgen" = 1010 — счётчик БД).
type MapStore struct {
	mu      sync.Mutex
	byName  map[string]*Account
	blocks  map[uint32][]BlockReason
	nextUID uint32
}

// NewMap — пустой стор.
func NewMap() *MapStore {
	return &MapStore{byName: map[string]*Account{}, blocks: map[uint32][]BlockReason{}, nextUID: 1}
}

// NewMapSeeded — стор с предзаданными uid (воспроизведение live-состояния).
func NewMapSeeded(seed map[string]uint32) *MapStore {
	s := NewMap()
	for name, uid := range seed {
		n := name
		s.byName[n] = &Account{UID: uid, Name: n}
		if uid >= s.nextUID {
			s.nextUID = uid + 1
		}
	}
	return s
}

// GetOrCreate — найти или создать. autoCreate=false + нет акка → ErrNotFound.
func (s *MapStore) GetOrCreate(name string, autoCreate bool) (Account, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a, ok := s.byName[name]; ok {
		return *a, false, nil
	}
	if !autoCreate {
		return Account{}, false, ErrNotFound
	}
	a := &Account{UID: s.nextUID, Name: name}
	s.nextUID++
	s.byName[name] = a
	return *a, true, nil
}

// Blocks — блок-строки по uid.
func (s *MapStore) Blocks(uid uint32) ([]BlockReason, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rs := s.blocks[uid]
	out := make([]BlockReason, len(rs))
	copy(out, rs)
	return out, nil
}

// AddBlock — тест-хелпер: заблокировать акк.
func (s *MapStore) AddBlock(uid uint32, code int32, msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blocks[uid] = append(s.blocks[uid], BlockReason{Code: code, Msg: msg})
}

// LogLogin — no-op in-memory (last_login живёт только в БД).
func (s *MapStore) LogLogin(a Account, ip string) error { return nil }

// SetFlag — тест-хелпер: править акк напрямую.
func (s *MapStore) SetFlag(name string, fn func(*Account)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.byName[name]
	if !ok {
		return ErrNotFound
	}
	fn(a)
	return nil
}

// Close — no-op.
func (s *MapStore) Close() error { return nil }

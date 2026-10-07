// Package cache: RAM-хранилище аккаунт-данных (семантика ACS: RAM-кэш + писатель в БД).
package cache

import (
	"sync"
)

// HiddenFatigue — то, что читает aion_GetAccountData_20170428.
type HiddenFatigue struct {
	Point      int32
	UpdateTime int32
	NpcKill    int32 // isnull(0)
	LimitReset int32
	LimitAccum int32
}

// AccountPack — aion_GetAccountPackList.
type AccountPack struct {
	PackType   int
	ExpireDate string // DATETIME как строка/аргумент
}

// Entry — RAM-запись аккаунта (каркас: минимум для FIRST_LOAD).
type Entry struct {
	AccountID int
	Fatigue   HiddenFatigue
	Packs     []AccountPack
}

// Store — потокобезопасный RAM-кэш.
type Store struct {
	mu   sync.RWMutex
	accs map[int]*Entry
}

func New() *Store {
	return &Store{accs: map[int]*Entry{}}
}

func (s *Store) Get(accountID int) (*Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.accs[accountID]
	return e, ok
}

func (s *Store) Put(e *Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accs[e.AccountID] = e
}

// UpdateFatigue — ACQ_UPDATE_HIDDEN_FATIGUE (3 приращения, каркасная семантика).
func (s *Store) UpdateFatigue(accountID int, point, upd, npckill int32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.accs[accountID]
	if !ok {
		e = &Entry{AccountID: accountID}
		s.accs[accountID] = e
	}
	e.Fatigue.Point = point
	e.Fatigue.UpdateTime = upd
	e.Fatigue.NpcKill = npckill
}

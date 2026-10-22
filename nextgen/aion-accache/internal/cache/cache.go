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

// UserInfo — срез global_user_data для пуша CHAR_INFO_REFRESH (R4-финал 22.10):
// last_login_time + MAX(user_level) по норм-серверам (capture: 0x50=80 у обоих чаров,
// при level=1 у чара 1003 в БД — значит не per-char level).
type UserInfo struct {
	LastLogin uint32
	MaxLevel  uint32
}

// Store — потокобезопасный RAM-кэш.
type Store struct {
	mu    sync.RWMutex
	accs  map[int]*Entry
	users map[int]UserInfo
}

func New() *Store {
	return &Store{accs: map[int]*Entry{}, users: map[int]UserInfo{}}
}

// SetUserInfo — посев RAM-среза global_user_data (SQLStore R3: SELECT из
// global_user_data по account_id; в тестах — прод-факты capture).
func (s *Store) SetUserInfo(accountID int, u UserInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.users == nil {
		s.users = map[int]UserInfo{}
	}
	s.users[accountID] = u
}

// UserInfo — чтение RAM-среза (ok=false = нет записи: ориг бы читал свою RAM-модель;
// эмиттер тогда шлёт нули — журнал теорий Т8).
func (s *Store) UserInfo(accountID int) (UserInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[accountID]
	return u, ok
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

// UpdateFatigue — ACQ_UPDATE_HIDDEN_FATIGUE (полная семантика R3:
// point/npckill/updatetime/limit_reset/limit_play по proc aion_UpdateHiddenFatigueInfo_20170428).
func (s *Store) UpdateFatigue(accountID int, point, npckill, updatetime, limitReset, limitPlay int32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.accs[accountID]
	if !ok {
		e = &Entry{AccountID: accountID}
		s.accs[accountID] = e
	}
	e.Fatigue.Point = point
	e.Fatigue.NpcKill = npckill
	e.Fatigue.UpdateTime = updatetime
	e.Fatigue.LimitReset = limitReset
	e.Fatigue.LimitAccum = limitPlay
}

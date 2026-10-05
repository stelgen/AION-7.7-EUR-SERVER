package server

import (
	"sync"
	"time"
)

// Brute — менеджер неудачных логинов (CBadUser; конфиг: tryCount=20 / tryInterval=60с
// → блок 120с, §4). Ключ — аккаунт (или IP), точную привязку уточнить при живом клиенте.
type Brute struct {
	mu     sync.Mutex
	fails  map[string]*failRec
	try    int
	window time.Duration
	block  time.Duration
}

type failRec struct {
	n       int
	first   time.Time
	blocked time.Time // zero = не заблокирован
}

func NewBrute(tryCount, intervalSec, blockIntervalSec int) *Brute {
	return &Brute{
		fails:  map[string]*failRec{},
		try:    tryCount,
		window: time.Duration(intervalSec) * time.Second,
		block:  time.Duration(blockIntervalSec) * time.Second,
	}
}

// Fail фиксирует неудачу; возвращает true, если теперь ключ заблокирован.
func (b *Brute) Fail(key string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	r := b.fails[key]
	if r == nil {
		r = &failRec{}
		b.fails[key] = r
	}
	if !r.blocked.IsZero() {
		return true
	}
	if now.Sub(r.first) > b.window {
		r.n, r.first = 0, now
	}
	r.n++
	if r.n >= b.try {
		r.blocked = now.Add(b.block)
		return true
	}
	return false
}

// Blocked — ключ заблокирован и окно блока не истекло.
func (b *Brute) Blocked(key string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	r := b.fails[key]
	if r == nil || r.blocked.IsZero() {
		return false
	}
	if time.Now().After(r.blocked) {
		delete(b.fails, key)
		return false
	}
	return true
}

// Reset — удачный логин сбрасывает счётчик.
func (b *Brute) Reset(key string) {
	b.mu.Lock()
	delete(b.fails, key)
	b.mu.Unlock()
}

// Package rpc — мир-init replay (R5): полные цепочки ответов CacheD→Server64/NPCSvr
// из живого capture подъёма мира 22.10. Покрывает 27 уникальных RQ→RP пар.
// Fire-and-forget (277/179/170/200/224) — БЕЗ ответа (канон capture).
// Login-path cmds (11/14/288/175/225/278) — уже покрыты отдельно, НЕ переопределяются.
package rpc

import (
	"embed"
	"log"
	"sync"
)

//go:embed testdata/world_init_*.bin
var worldInitFS embed.FS

// WorldInitReplay — stateful replay мир-init цепочек (per-conn состояние).
type WorldInitReplay struct {
	mu    sync.Mutex
	count map[uint16]int // per-cmd счётчик вызовов
}

// NewWorldInitReplay — создать per-conn replay.
func NewWorldInitReplay() *WorldInitReplay {
	return &WorldInitReplay{count: map[uint16]int{}}
}

// worldInitFile — имя файла цепочки по cmd (hex3, без счётчика — capture-канон: одна цепочка на cmd).
func worldInitFile(cmd uint16) string {
	return "testdata/world_init_" + hex3(cmd) + ".bin"
}

// worldInitExists — есть ли captured-цепочка для этого cmd.
func worldInitExists(cmd uint16) bool {
	data, err := worldInitFS.ReadFile(worldInitFile(cmd))
	return err == nil && len(data) > 0
}

// EmitWorldInit — ответ CacheD на мир-init запрос (captured цепочка целиком).
// Возвращает nil если нет captured-цепочки (тишина — политика).
func (r *WorldInitReplay) EmitWorldInit(cmd uint16) []byte {
	r.mu.Lock()
	r.count[cmd]++
	r.mu.Unlock()
	data, err := worldInitFS.ReadFile(worldInitFile(cmd))
	if err != nil || len(data) == 0 {
		return nil
	}
	log.Printf("[cached] world-init: cmd=%d → replay %dB", cmd, len(data))
	return data
}

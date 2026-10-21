// Package ship — телеметрия (JSON-лог событий) для aion-cached, R1.
// Канон логов: raw-first, ship-событие = метрика факта, НЕ догадка.
// R2 TODO: расширит формат при появлении PDB-канона attribute-схемы CacheD64.
package ship

import (
	"encoding/json"
	"log"
	"time"
)

// Event — ship-событие (структурный факт: канал, опкод, действие).
type Event struct {
	Channel string `json:"channel"` // server | admin | npcDb
	Opcode  uint16 `json:"opcode"`
	Name    string `json:"name"` // хандлер (например "server:CacheVersion(000)")
	Bytes   int    `json:"bytes,omitempty"`
	Status  string `json:"status"` // ok | reject | unknown
	At      string `json:"at"`
}

// Emit — логирует ship-событие (R1 = JSON в stdout; R2 — вынос в_UDP/файл).
func Emit(e Event) {
	if e.At == "" {
		e.At = time.Now().UTC().Format(time.RFC3339Nano)
	}
	b, err := json.Marshal(e)
	if err != nil {
		log.Printf("[ship] marshal error: %v", err)
		return
	}
	log.Printf("[ship] %s", string(b))
}

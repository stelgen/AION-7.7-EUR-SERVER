// Package rpc — NPCSvr-флоу (R4.7.1): полные цепочки ответов CacheD→NPCSvr из
// живого capture подъёма мира 22.10 (последний блок gp-tail: C/O пары копия-2016).
// NPCSvr-коннект определяется первым кадром cmd=201 (0x0C9); далее — replay цепочек
// per-cmd (stateful: 0x0D3/0x0E9 отвечают только со 2-го вызова — канон capture).
// Тикеры (0x0B3/0x115/0x0AA/0x0C8/0x0E0/0x12B) — fire-and-forget, ответа нет (канон).
package rpc

import (
	"embed"
	"log"
	"sync"
)

//go:embed testdata/npcsrv_*.bin
var npcFS embed.FS

// NpcSrvReplay — stateful replay цепочек NPCSvr-канала (per-conn состояние).
type NpcSrvReplay struct {
	mu    sync.Mutex
	count map[uint16]int // per-cmd счётчик вызовов
}

// NewNpcSrvReplay — создать per-conn replay.
func NewNpcSrvReplay() *NpcSrvReplay {
	return &NpcSrvReplay{count: map[uint16]int{}}
}

// npcChainFile — имя файла цепочки по cmd и номеру вызова (0-based).
func npcChainFile(cmd uint16, idx int) string {
	return "testdata/npcsrv_" + hex3(cmd) + "_" + itoa10(idx) + ".bin"
}

// EmitNpcSrv — ответ CacheD на кадр NPCSvr (captured цепочка целиком).
// R4.6.2: сначала W30-бёрст (батчи по occurrence, capture 23.10), затем
// legacy-файлы 22.10. nil = канон-тишина.
func (r *NpcSrvReplay) EmitNpcSrv(cmd uint16) []byte {
	r.mu.Lock()
	idx := r.count[cmd]
	r.count[cmd]++
	r.mu.Unlock()
	if w := EmitNpcSrvW30(cmd, idx); w != nil {
		log.Printf("[cached] npcsrv: cmd=%d call#%d → W30 батч %dБ", cmd, idx+1, len(w))
		return w
	}
	if NpcW30BatchCount(cmd) > 0 {
		// cmd есть в W30, но occurrence исчерпан — канон-тишина (fire-and-forget)
		log.Printf("[cached] npcsrv: cmd=%d call#%d — W30 occurrence исчерпан, silent", cmd, idx+1)
		return nil
	}
	data, err := npcFS.ReadFile(npcChainFile(cmd, idx))
	if err != nil {
		// нет capture-цепочки для этого вызова — тишина (политика «ответ только
		// где канон доказан»; capture = эталон, отсутствие = тишина)
		log.Printf("[cached] npcsrv: cmd=%d call#%d — нет capture-цепочки, silent", cmd, idx+1)
		return nil
	}
	log.Printf("[cached] npcsrv: cmd=%d call#%d → chain replay %dB", cmd, idx+1, len(data))
	return data
}

func hex3(v uint16) string {
	const d = "0123456789abcdef"
	return string([]byte{d[v>>8&0xf], d[v>>4&0xf], d[v&0xf]})
}

func itoa10(v int) string {
	if v == 0 {
		return "0"
	}
	s := ""
	for v > 0 {
		s = string(rune('0'+v%10)) + s
		v /= 10
	}
	return s
}

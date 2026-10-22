// Package rpc — ChainReplay: полные response chains из live capture (R4).
// Вместо cherry-picking отдельных кадров — replay ENTIRE response chain
// для каждого запроса (RQ_LOAD / RQ_LOAD_ITEM). Server64 получает ТОЧНО
// то, что ожидает (capture = ориг-эталон).
package rpc

import (
	"embed"
	"log"
	"sync"
)

//go:embed testdata/chain_*.bin
var chainFS embed.FS

// ChainReplay — stateful replay: отслеживает счётчик RQ_LOAD_ITEM запросов
// (page=00 → echo chain, page=01 → data chain) и RQ_LOAD счётчик.
type ChainReplay struct {
	mu           sync.Mutex
	loadCount    int // счётчик RQ_LOAD запросов (для чередования шаблонов)
	itemCount    int // счётчик RQ_LOAD_ITEM запросов (page=00 → echo, page=01 → data)
	itemParity   int // 0=echo chain, 1=data chain (чередование)
}

var globalChain = &ChainReplay{}

// LoadChain — загружает бинарную цепочку и возвращает конкатенированные кадры.
func LoadChain(name string) []byte {
	data, err := chainFS.ReadFile("testdata/" + name)
	if err != nil {
		log.Printf("[cached] chain %s not found: %v", name, err)
		return nil
	}
	return data
}

// GetLoadChain — полная цепочка ответа на RQ_LOAD (RP cmd=1 1156Б).
func GetLoadChain() []byte {
	globalChain.mu.Lock()
	defer globalChain.mu.Unlock()
	idx := globalChain.loadCount % 2
	globalChain.loadCount++
	name := "chain_rq_load_" + itoa(idx) + ".bin"
	data := LoadChain(name)
	if data == nil {
		data = LoadChain("chain_rq_load_0.bin")
	}
	return data
}

// GetItemChain — полная цепочка ответа на RQ_LOAD_ITEM.
// page=00 → echo chain (~2.8КБ), page=01 → data chain (~91КБ).
// Чередование: первый запрос → echo, второй → data.
func GetItemChain(page byte) []byte {
	globalChain.mu.Lock()
	defer globalChain.mu.Unlock()
	var name string
	if page == 0 {
		// page=00 → echo chain (краткий ack)
		idx := 0 // всегда echo для page=00
		name = "chain_rq_load_item_" + itoa(idx*2) + ".bin"
	} else {
		// page=01 → data chain (полный item data)
		idx := 1
		name = "chain_rq_load_item_" + itoa(idx*2+1) + ".bin"
	}
	data := LoadChain(name)
	if data == nil {
		data = LoadChain("chain_rq_load_item_0.bin")
	}
	return data
}

// ResetChains — сброс счётчиков (новая Server64-сессия).
func ResetChains() {
	globalChain.mu.Lock()
	defer globalChain.mu.Unlock()
	globalChain.loadCount = 0
	globalChain.itemCount = 0
	globalChain.itemParity = 0
}

func itoa(i int) string {
	return string(rune('0' + i))
}

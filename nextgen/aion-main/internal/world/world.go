// Package world — мир-стейт R3.5: раскладки (layouts) = байт-в-байт прод-ответы из capture 08.10
// (S4-паритет: сервер шлёт клиенту ТО, что шлёт ориг), канонический порядок инициализации мира.
// Источник: расшифрованные payload'ы capture#3b (юзер-сессия, 0 invalid).
package world

import (
	"fmt"
	"os"
	"path/filepath"
)

// InitOrder — канонический порядок SM-пакетов при входе в мир (кадры capture#3b: 16→40→86→87→101→106→172→[174-186 strongholds]→427).
var InitOrder = []string{
	"SM_ABNORMAL_STATE",
	"SM_SKILL_LIST",
	"SM_WAREHOUSE_INFO",
	"SM_INVENTORY_INFO",
	"SM_STATS_INFO",
	"SM_LUNA_SYSTEM_INFO",
	"SM_SIEGE_LOCATION_INFO",
	"SM_STATUPDATE_HP",
}

// LayoutStore — хранилище раскладок (имя SM-пакета → payload байт-в-байт из прод-capture).
type LayoutStore struct {
	dir        string
	layouts    map[string][]byte
	loadedSize int
}

// New — загружает все *.bin из dir (имя файла = имя SM-пакета).
func New(dir string) (*LayoutStore, error) {
	ls := &LayoutStore{dir: dir, layouts: map[string][]byte{}}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("world: readdir %s: %w", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".bin" {
			continue
		}
		name := e.Name()[:len(e.Name())-4]
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("world: read %s: %w", e.Name(), err)
		}
		ls.layouts[name] = data
		ls.loadedSize += len(data)
	}
	return ls, nil
}

// Get — payload раскладки.
func (ls *LayoutStore) Get(name string) ([]byte, bool) {
	b, ok := ls.layouts[name]
	return b, ok
}

// Count / Bytes — для канарейки (S7).
func (ls *LayoutStore) Count() int { return len(ls.layouts) }
func (ls *LayoutStore) Bytes() int { return ls.loadedSize }

// InitSequence — каноничная последовательность (имя, payload) входа в мир;
// пропускает отсутствующие раскладки (backlog R3.5+: STRONGHOLDS и др.).
func (ls *LayoutStore) InitSequence() (seq [][2]string) {
	for _, name := range InitOrder {
		if b, ok := ls.Get(name); ok {
			seq = append(seq, [2]string{name, string(b)})
		}
	}
	return seq
}
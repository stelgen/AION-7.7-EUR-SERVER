// Т30 settings store — per-charId хранилище zlib-XML настроек клиента.
// CacheD хранит клиентские настройки (options/quickbar/favorites) как zlib XML.
// 3 страницы: page0=main options (28КБ), page1=quickbar/chainskill (3КБ), page2=favorites (744Б).
// Поток: LOGIN → Server64 спрашивает CacheD → CacheD отдаёт страницы.
//        SAVE → Server64 шлёт обновлённые → CacheD хранит в RAM + БД.
package server

import (
	"fmt"
	"log"
	"sync"
)

// settingsPage — одна страница настроек (zlib-сжатый кадр целиком, маркер 0xAF).
type settingsPage struct {
	Wire []byte // полный S2C-кадр (с заголовком), готовый к отправке
}

// SettingsStore — per-charId хранилище страниц настроек.
type SettingsStore struct {
	mu    sync.RWMutex
	pages map[uint32][]settingsPage // charId → [3] страницы (0=main, 1=quickbar, 2=favorites)
}

var globalSettings = &SettingsStore{pages: make(map[uint32][]settingsPage)}

// Get — возвращает страницы настроек для charId (nil = нет сохранённых).
func (s *SettingsStore) Get(charId uint32) [][]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pages, ok := s.pages[charId]
	if !ok {
		return nil
	}
	out := make([][]byte, len(pages))
	for i, p := range pages {
		out[i] = p.Wire
	}
	return out
}

// Put — сохраняет страницы для charId.
func (s *SettingsStore) Put(charId uint32, wires [][]byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pages := make([]settingsPage, len(wires))
	for i, w := range wires {
		pages[i] = settingsPage{Wire: w}
	}
	s.pages[charId] = pages
	log.Printf("[cached] settings store: char=%d saved %d pages", charId, len(wires))
}

// Has — проверка наличия настроек.
func (s *SettingsStore) Has(charId uint32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.pages[charId]
	return ok
}

// Stats — строка статистики для мониторинга.
func (s *SettingsStore) Stats() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return fmt.Sprintf("chars=%d", len(s.pages))
}

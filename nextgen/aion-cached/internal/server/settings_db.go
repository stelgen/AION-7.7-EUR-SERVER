// Т30: DB-персист настроек клиента — zlib-XML страницы per-charId.
// SettingsStore + Executor → прозрачная persistence: RAM кэш + DB чтение/запись.
// Таблица: character_settings (char_id, page, settings_data varbinary(max)).
package server

import (
	"context"
	"encoding/binary"
	"log"

	"aion-cached/internal/db"
)

// SettingsDB — DB-операции для settings store.
type SettingsDB struct {
	Ex db.Executor
}

// LoadFromDB — загрузка страниц из БД (если есть) → в RAM store.
func (sdb *SettingsDB) LoadFromDB(ctx context.Context, charId uint32) int {
	if sdb.Ex == nil {
		return 0
	}
	q := "SELECT page, settings_data FROM character_settings WHERE char_id = @p1 ORDER BY page"
	rows, err := sdb.Ex.Query(ctx, q, charId)
	if err != nil {
		log.Printf("[cached] settings DB load err char=%d: %v", charId, err)
		return 0
	}
	defer rows.Close()
	type pageRow struct{ page int32; data []byte }
	var pageRows []pageRow
	for rows.Next() {
		var pr pageRow
		if err := rows.Scan(&pr.page, &pr.data); err != nil { continue }
		pageRows = append(pageRows, pr)
	}
	if len(pageRows) == 0 { return 0 }
	var wires [][]byte
	for _, pr := range pageRows {
		total := len(pr.data) + 7
		frame := make([]byte, total)
		binary.LittleEndian.PutUint16(frame[0:], uint16(total))
		binary.LittleEndian.PutUint16(frame[2:], 14)
		frame[4] = 0xAF
		binary.LittleEndian.PutUint16(frame[5:], ^uint16(14))
		copy(frame[7:], pr.data)
		wires = append(wires, frame)
	}
	globalSettings.Put(charId, wires)
	log.Printf("[cached] settings DB loaded char=%d pages=%d", charId, len(wires))
	return len(wires)
}

// SaveToDB — DELETE старых + INSERT новых.
func (sdb *SettingsDB) SaveToDB(ctx context.Context, charId uint32, wires [][]byte) {
	if sdb.Ex == nil { return }
	_, _ = sdb.Ex.Exec(ctx, "DELETE FROM character_settings WHERE char_id = @p1", charId)
	for i, w := range wires {
		if len(w) < 7 { continue }
		data := w[7:]
		_, err := sdb.Ex.Exec(ctx,
			"INSERT INTO character_settings (char_id, page, settings_data, data_len) VALUES (@p1, @p2, @p3, @p4)",
			charId, i, data, len(data))
		if err != nil {
			log.Printf("[cached] settings DB save err char=%d page=%d: %v", charId, i, err)
		}
	}
	log.Printf("[cached] settings DB saved char=%d pages=%d", charId, len(wires))
}

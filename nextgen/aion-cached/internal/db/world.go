// world.go — Этап 5: запросы к мир-БД (_AionWorldNew114_rc) для DB-генератора.
// user_data: level/координаты/статы; user_item: инвентарь.
package db

import (
	"context"
	"fmt"
)

// CharWorldData — данные чара из мир-БД (user_data) для RP cmd=1.
type CharWorldData struct {
	Lev         int32
	LastNormalX float32
	LastNormalY float32
	LastNormalZ float32
	ResX        float32
	ResY        float32
	ResZ        float32
}

// WorldItemRow — строка user_item для O109.
type WorldItemRow struct {
	NameID    uint32
	SlotID    int32
	Amount    uint32
	Slot      uint32
	Warehouse uint32
}

// GetCharWorldData — SELECT lev + координаты из user_data (мир-БД).
func GetCharWorldData(ctx context.Context, ex Executor, charId uint32) (CharWorldData, error) {
	var w CharWorldData
	q := `SELECT lev, last_normal_xlocation, last_normal_ylocation, last_normal_zlocation, ` +
		`resurrect_xlocation, resurrect_ylocation, resurrect_zlocation ` +
		`FROM user_data WHERE char_id = @p1`
	rows, err := ex.Query(ctx, q, charId)
	if err != nil {
		return w, fmt.Errorf("world user_data: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return w, fmt.Errorf("user_data char=%d: not found", charId)
	}
	var lnx, lny, lnz, rx, ry, rz float64
	if err := rows.Scan(&w.Lev, &lnx, &lny, &lnz, &rx, &ry, &rz); err != nil {
		return w, fmt.Errorf("world user_data scan: %w", err)
	}
	w.LastNormalX = float32(lnx)
	w.LastNormalY = float32(lny)
	w.LastNormalZ = float32(lnz)
	w.ResX = float32(rx)
	w.ResY = float32(ry)
	w.ResZ = float32(rz)
	return w, nil
}

// GetUserItems — SELECT инвентарь из user_item (мир-БД), warehouse=0 only.
func GetUserItems(ctx context.Context, ex Executor, charId uint32) ([]WorldItemRow, error) {
	q := `SELECT name_id, slot_id, amount, slot, warehouse ` +
		`FROM user_item WHERE char_id = @p1 AND warehouse = 0 ORDER BY id`
	rows, err := ex.Query(ctx, q, charId)
	if err != nil {
		return nil, fmt.Errorf("world user_item: %w", err)
	}
	defer rows.Close()
	var out []WorldItemRow
	for rows.Next() {
		var r WorldItemRow
		var slotId, amount int64
		if err := rows.Scan(&r.NameID, &slotId, &amount, &r.Slot, &r.Warehouse); err != nil {
			return nil, fmt.Errorf("world user_item scan: %w", err)
		}
		r.SlotID = int32(slotId)
		r.Amount = uint32(amount)
		out = append(out, r)
	}
	return out, nil
}

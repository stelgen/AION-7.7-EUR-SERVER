// Package rpc — DB-генератор read-пути (R4→R5): RP cmd=1 из global_user_data
// + item XML кэш (capture-данные, общие для всех чаров) + item blocks (replay).
// Эволюция от capture-replay к DB-генерации: RP cmd=1 первым (критический путь).
package rpc

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"

	"aion-cached/internal/db"
)

// DBGen — DB-генератор read-пути.
type DBGen struct {
	Ex db.Executor // SQL Executor (nil = replay-only)
}

// CharDBInfo — данные чара из global_user_data.
type CharDBInfo struct {
	CharID   uint32
	AccID    uint32
	Level    uint32
	ServerID uint32
	ClassType uint32
	RaceType  uint32
	Exists   bool
}

// GetCharInfo — SELECT из global_user_data.
func GetCharInfo(ctx context.Context, ex db.Executor, accID, charID uint32) (CharDBInfo, error) {
	var info CharDBInfo
	info.CharID = charID
	info.AccID = accID
	q := "SELECT user_level, cur_server_id, ISNULL(class_type,0), ISNULL(race_type,0) FROM global_user_data WHERE account_id = @p1 AND char_id = @p2 AND delete_date = 0"
	rows, err := ex.Query(ctx, q, accID, charID)
	if err != nil {
		return info, fmt.Errorf("GetCharInfo: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return info, nil // нет строки = Exists=false
	}
	var level, serverID int32
	var classType, raceType int32
	if err := rows.Scan(&level, &serverID, &classType, &raceType); err != nil {
		return info, fmt.Errorf("GetCharInfo scan: %w", err)
	}
	info.Level = uint32(level)
	info.ServerID = uint32(serverID)
	info.ClassType = uint32(classType)
	info.RaceType = uint32(raceType)
	info.Exists = true
	return info, nil
}

// GenerateRPLoadDB — строит RP cmd=1 из captured-шаблона + DB-подстановкой
// (level, charRef, position). Неизвестные поля сохраняются из шаблона.
func (g *DBGen) GenerateRPLoadDB(ctx context.Context, sessionX uint32, charId, accId uint32) []byte {
	tpl := loadTemplate("rp1_char_struct.bin")
	if tpl == nil {
		log.Printf("[cached] dbgen: RP cmd=1 template missing")
		return GetRPLoad(sessionX) // фолбэк на чистый replay
	}
	out := make([]byte, len(tpl))
	copy(out, tpl)

	// X (payload[0:4] = frame[7:11])
	binary.LittleEndian.PutUint32(out[7:11], sessionX)

	// charRef (payload[21:25] = frame[28:32]) — [charId u16][slot u16]
	charRef := charId | 0x0002<<16 // slot=2 (из capture; слот ⏳)
	binary.LittleEndian.PutUint32(out[28:32], charRef)

	// Если Executor доступен — подставить level из БД
	if g.Ex != nil {
		info, err := GetCharInfo(ctx, g.Ex, accId, charId)
		if err != nil {
			log.Printf("[cached] dbgen GetCharInfo err: %v — использую шаблон", err)
		} else if info.Exists {
			// level: payload[47] (frame[54]) — 0x50 = 80 в capture
			if len(out) > 54 {
				out[54] = byte(info.Level)
				log.Printf("[cached] dbgen level=%d из БД (char=%d)", info.Level, charId)
			}
		}
	}
	return out
}

// GenerateCharListDB — RP cmd=90 (CharacterList) из шаблона + DB.
// MVP: замена X только (полный charCount/charName генератор = следующий шаг).
func (g *DBGen) GenerateCharListDB(sessionX uint32) []byte {
	tpl := loadTemplate("rp90_charlist.bin")
	if tpl == nil {
		return GetRPCharList(sessionX)
	}
	out := make([]byte, len(tpl))
	copy(out, tpl)
	binary.LittleEndian.PutUint32(out[7:11], sessionX)
	return out
}

// GenerateItemPages — RP cmd=0x0E ×3 (zlib XML): ШТАТНЫЕ данные сервера (общие для
// всех чаров — item/skill определения), кэшируются из capture.
func (g *DBGen) GenerateItemPages(sessionX uint32) [][]byte {
	return GetRPLoadItemPages(sessionX)
}

// GenerateItemBlocks — RP cmd=109 ×12 (per-char items): replay из capture.
// Полный DB-генератор item-инвентаря = следующий шаг (бинарный формат item-записей).
func (g *DBGen) GenerateItemBlocks(sessionX uint32) [][]byte {
	return GetRPItemBlocks(sessionX)
}

// EmitReadPathDB — полная цепочка read-пути с DB-генерацией.
func (g *DBGen) EmitReadPathDB(ctx context.Context, sc SessionContext) [][]byte {
	var frames [][]byte
	if cl := g.GenerateCharListDB(sc.X); cl != nil {
		frames = append(frames, cl)
	}
	if rp := g.GenerateRPLoadDB(ctx, sc.X, sc.CharID, sc.AccID); rp != nil {
		frames = append(frames, rp)
	}
	for _, pg := range g.GenerateItemPages(sc.X) {
		frames = append(frames, pg)
	}
	for _, blk := range g.GenerateItemBlocks(sc.X) {
		frames = append(frames, blk)
	}
	return frames
}

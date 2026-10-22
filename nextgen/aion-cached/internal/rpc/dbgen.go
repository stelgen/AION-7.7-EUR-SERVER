// Package rpc — DB-генератор read-пути (R4→R5): RP cmd=1 из global_user_data
// (Т31 byte-level map) + charName + item XML кэш + item blocks (replay).
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
	CharID    uint32
	AccID     uint32
	Level     uint32
	ServerID  uint32
	ClassType uint32
	RaceType  uint32
	Exists    bool
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
		return info, nil
	}
	var level, serverID, classType, raceType int32
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

// buildCharRef — Т31: charRef = [charId u16 LE][0x00][0x02] → u32 = charId | 0x02000000.
func buildCharRef(charId uint32) uint32 {
	return charId | 0x02000000
}

// GenerateRPLoadFull — RP cmd=1 из captured-шаблона с ПОЛНОЙ DB-подстановкой (Т31):
// [0:4]=X, [21:25]=charRef, [49]=level (single byte!), [903:928]=charName UTF-16.
// Остальные поля из шаблона (неизвестные сохраняются).
func (g *DBGen) GenerateRPLoadFull(ctx context.Context, sessionX uint32, charId, accId uint32, charName string) []byte {
	tpl := loadTemplate("rp1_char_struct.bin")
	if tpl == nil {
		log.Printf("[cached] dbgen: RP cmd=1 template missing")
		return GetRPLoad(sessionX)
	}
	out := make([]byte, len(tpl))
	copy(out, tpl)

	// [0:4] = X (payload offset 0 = frame offset 7)
	binary.LittleEndian.PutUint32(out[7:11], sessionX)

	// [21:25] = charRef (frame[28:32]) — ИСПРАВЛЕНО: charId | 0x02000000
	charRef := buildCharRef(charId)
	binary.LittleEndian.PutUint32(out[28:32], charRef)

	// DB-подстановка (level + доп. поля из БД)
	if g.Ex != nil {
		info, err := GetCharInfo(ctx, g.Ex, accId, charId)
		if err != nil {
			log.Printf("[cached] dbgen GetCharInfo err: %v — шаблон", err)
		} else if info.Exists {
			// [49] = level (SINGLE BYTE! frame[7+49]=frame[56])
			if len(out) > 56 {
				out[56] = byte(info.Level)
				log.Printf("[cached] dbgen level=%d char=%d из БД", info.Level, charId)
			}
		}
	}

	// [903:928] = charName UTF-16LE (если предоставлен)
	if charName != "" && len(out) > 928 {
		// кодировать charName в UTF-16LE + хэш-суффикс из шаблона
		// шаблон: [name UTF-16][- UTF-16][hash binary]
		// MVP: подставить только имя, хэш оставить из шаблона
		nameBytes := []byte(charName)
		u16 := make([]byte, 0, len(nameBytes)*2)
		for _, b := range nameBytes {
			u16 = append(u16, b, 0x00)
		}
		// перезаписать имя в шаблоне (начиная с offset 903 payload = 910 frame)
		nameStart := 7 + 903 // frame offset
		for i := 0; i < len(u16) && nameStart+i < len(out); i++ {
			out[nameStart+i] = u16[i]
		}
		log.Printf("[cached] dbgen charName='%s' подставлен @payload[903]", charName)
	}

	return out
}

// GenerateCharListFull — RP cmd=90 (CharacterList) из шаблона + X.
func (g *DBGen) GenerateCharListFull(sessionX uint32) []byte {
	tpl := loadTemplate("rp90_charlist.bin")
	if tpl == nil {
		return GetRPCharList(sessionX)
	}
	out := make([]byte, len(tpl))
	copy(out, tpl)
	binary.LittleEndian.PutUint32(out[7:11], sessionX)
	return out
}

// GenerateItemPagesFull — RP cmd=0x0E ×3 (zlib XML settings store).
// Общие для всех чаров (серверные определения) — replay из capture.
func (g *DBGen) GenerateItemPagesFull(sessionX uint32) [][]byte {
	return GetRPLoadItemPages(sessionX)
}

// GenerateItemBlocksFull — RP cmd=109 ×12 (per-char items): replay из capture.
func (g *DBGen) GenerateItemBlocksFull(sessionX uint32) [][]byte {
	return GetRPItemBlocks(sessionX)
}

// EmitReadPathFull — полная цепочка read-пути с DB-генерацией + charName.
func (g *DBGen) EmitReadPathFull(ctx context.Context, sc SessionContext, charName string) [][]byte {
	var frames [][]byte
	if cl := g.GenerateCharListFull(sc.X); cl != nil {
		frames = append(frames, cl)
	}
	if rp := g.GenerateRPLoadFull(ctx, sc.X, sc.CharID, sc.AccID, charName); rp != nil {
		frames = append(frames, rp)
	}
	for _, pg := range g.GenerateItemPagesFull(sc.X) {
		frames = append(frames, pg)
	}
	for _, blk := range g.GenerateItemBlocksFull(sc.X) {
		frames = append(frames, blk)
	}
	return frames
}

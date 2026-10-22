// Package rpc — RP-эмиттеры read-пути (R4, capture-replay MVP).
// Шаблоны из live capture (testdata rp*.bin) + X/charRef-подстановка.
// Стратегия (мимикрия-first, WORKFLOW §5): эмитим captured-ответы с подстановкой
// сессионного X — Server64 видит структурно валидные данные. Полный генератор
// из БД = следующий шаг (после A/B и разбора RP cmd=1 полей).
package rpc

import (
	"embed"
	"encoding/binary"
	"fmt"
	"log"
)

//go:embed testdata/rp*.bin
var replayFS embed.FS

// loadTemplate — загружает бинарный шаблон из testdata.
func loadTemplate(name string) []byte {
	data, err := replayFS.ReadFile("testdata/" + name)
	if err != nil {
		log.Printf("[cached] replay: template %s not found: %v", name, err)
		return nil
	}
	return data
}

// substituteX — подстановка сессионного хендла X в шаблон.
// X находится в payload[0:4] (после 7Б заголовка) для всех read-path ответов.
func substituteX(frame []byte, newX uint32) []byte {
	if len(frame) < wireHeaderLen+4 {
		return frame
	}
	out := make([]byte, len(frame))
	copy(out, frame)
	binary.LittleEndian.PutUint32(out[7:11], newX)
	return out
}

// substituteXCharRef — подстановка X и charRef (если charRef ≠ 0).
// X в payload[0:4], charRef ищется по паттерну [xx xx 00 02] в первых 32Б payload.
func substituteXCharRef(frame []byte, newX uint32, charRef uint32) []byte {
	out := substituteX(frame, newX)
	if charRef == 0 {
		return out
	}
	// ищем паттерн charRef (byte0 = low byte charId, byte2 = 0x02) в payload
	for i := wireHeaderLen + 4; i < len(out)-4 && i < wireHeaderLen+40; i++ {
		if out[i+2] == 0x02 && out[i+1] == byte(charRef>>8) && out[i] == byte(charRef&0xFF) && out[i+3] == 0 {
			binary.LittleEndian.PutUint32(out[i:], charRef)
			return out
		}
	}
	return out
}

const wireHeaderLen = 7

// GetRPLoad — RP cmd=1 (char struct 1156Б) с подстановкой X.
func GetRPLoad(sessionX uint32) []byte {
	tpl := loadTemplate("rp1_char_struct.bin")
	if tpl == nil {
		return nil
	}
	return substituteX(tpl, sessionX)
}

// GetRPCharList — RP cmd=90 (CharacterList 1571Б) с подстановкой X.
func GetRPCharList(sessionX uint32) []byte {
	tpl := loadTemplate("rp90_charlist.bin")
	if tpl == nil {
		return nil
	}
	return substituteX(tpl, sessionX)
}

// GetRPLoadItemPages — RP cmd=0x0E ×3 (zlib item XML pages) с подстановкой X.
// Возвращает 3 кадра (page 0/1/2).
func GetRPLoadItemPages(sessionX uint32) [][]byte {
	var out [][]byte
	for i := 0; i < 3; i++ {
		tpl := loadTemplate(fmt.Sprintf("rp0e_page%d.bin", i))
		if tpl == nil {
			continue
		}
		out = append(out, substituteX(tpl, sessionX))
	}
	return out
}

// GetRPItemBlocks — RP cmd=109 ×12 (item blocks 8021Б) с подстановкой X.
func GetRPItemBlocks(sessionX uint32) [][]byte {
	var out [][]byte
	for i := 0; i < 12; i++ {
		tpl := loadTemplate(fmt.Sprintf("rp109_block%d.bin", i))
		if tpl == nil {
			continue
		}
		out = append(out, substituteX(tpl, sessionX))
	}
	return out
}

// SessionContext — контекст текущей Server64-сессии (для X-подстановки).
type SessionContext struct {
	X      uint32 // сессионный хендл (из cmd=288)
	AccID  uint32
	CharID uint32
	CharRef uint32 // [charId u16][slot u16] = 0x020003EA
}

// EmitReadPath — полная цепочка read-пути: CharacterList + RP_LOAD + item pages + item blocks.
// Возвращает ВСЕ S2C-кадры для RQ_LOAD + RQ_LOAD_ITEM ×2 (как ориг).
func EmitReadPath(sc SessionContext) [][]byte {
	var frames [][]byte
	// 1. CharacterList (cmd=90)
	if cl := GetRPCharList(sc.X); cl != nil {
		frames = append(frames, cl)
	}
	// 2. RP cmd=1 (char struct)
	if rp := GetRPLoad(sc.X); rp != nil {
		frames = append(frames, rp)
	}
	// 3. RP cmd=0x0E ×3 (zlib item XML pages)
	for _, pg := range GetRPLoadItemPages(sc.X) {
		frames = append(frames, pg)
	}
	// 4. RP cmd=109 ×12 (item blocks)
	for _, blk := range GetRPItemBlocks(sc.X) {
		frames = append(frames, blk)
	}
	return frames
}

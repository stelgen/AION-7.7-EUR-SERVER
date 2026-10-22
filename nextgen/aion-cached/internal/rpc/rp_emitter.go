// Package rpc — RP cmd=1 полный эмиттер из БД (Т31 offsets) + RP cmd=109 item-парсер.
// Стратегия: captured-шаблон + DB-подстановка всех известных полей Т31.
// Неизвестные поля сохраняются из шаблона (безопаснее нулевого заполнения).
package rpc

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"

	"aion-cached/internal/db"
)

// Т31 offsets — RP cmd=1 payload (1149Б, НЕ dword-aligned!)
const (
	offX       = 0   // [0:4] session handle (u32)
	offA       = 4   // [4:8] a=4 (u32)
	offOne     = 8   // [8:10] 01 (u16)
	offSep     = 12  // [12] 00 (1 byte)
	offHash    = 13  // [13:17] ff40529f (4B)
	offCharRef = 21  // [21:25] charRef (4B)
	offPos     = 25  // [25:29] position/timestamp (4B)
	offLevel   = 49  // [49] level (1 BYTE!)
	offStat1   = 63  // [63:65] e35d (u16)
	offStat2   = 67  // [67:69] c72b (u16)
	offStat3   = 75  // [75:77] 3c=60 (u16)
	offStat4   = 83  // [83:85] 16=22 (u16)
	offKinah   = 863 // [863:866] a08601 = 100000 kinah (3Б LE)
	offCharNam = 903 // [903:928] charName UTF-16LE (25Б)
)

// GenerateRPLoadStruct — ПОЛНЫЙ эмиттер RP cmd=1 (1156Б) из captured-шаблона
// с DB-подстановкой: X, charRef, level (single byte), charName UTF-16.
// Все остальные поля — из шаблона (неизвестные сохраняются).
func GenerateRPLoadStruct(ctx context.Context, ex db.Executor, sessionX uint32, charId, accId uint32, charName string) []byte {
	tpl := loadTemplate("rp1_char_struct.bin")
	if tpl == nil {
		log.Printf("[cached] rp1 template missing — zero-fill")
		tpl = make([]byte, 1156)
	}
	out := make([]byte, len(tpl))
	copy(out, tpl)

	// [0:4] = X
	binary.LittleEndian.PutUint32(out[7+offX:11+offX], sessionX)

	// [21:25] = charRef
	charRef := buildCharRef(charId)
	binary.LittleEndian.PutUint32(out[7+offCharRef:11+offCharRef], charRef)

	// DB-подстановка
	if ex != nil {
		info, err := GetCharInfo(ctx, ex, accId, charId)
		if err != nil {
			log.Printf("[cached] rp1 dbgen err: %v", err)
		} else if info.Exists {
			// [49] = level (SINGLE BYTE)
			out[7+offLevel] = byte(info.Level)
			log.Printf("[cached] rp1 dbgen: level=%d char=%d", info.Level, charId)
		}
	}

	// [903:928] = charName UTF-16LE (перезаписать имя, хэш-суффикс из шаблона)
	if charName != "" && len(out) > 7+offCharNam+len(charName)*2 {
		nameStart := 7 + offCharNam
		for i, b := range []byte(charName) {
			out[nameStart+i*2] = b
			out[nameStart+i*2+1] = 0x00
		}
		log.Printf("[cached] rp1 dbgen: charName='%s' @payload[%d]", charName, offCharNam)
	}

	return out
}

// GenerateCharListStruct — RP cmd=90 (CharacterList) из шаблона + X.
func GenerateCharListStruct(sessionX uint32) []byte {
	return GenerateCharListStructX(sessionX)
}

// --- RP cmd=109 item block parser ---

// ItemRecord — одна item-запись из cmd=109 блока (Т33: ~470Б, [ffff] маркер).
type ItemRecord struct {
	Offset  int    // offset в payload
	Length  int    // длина записи
	Raw     []byte // сырые данные записи
	ItemID  uint32 // Aion item template ID (если найден)
	HasFFFF bool   // [ffff] маркер присутствует
}

// ParseItemBlocks — разбирает RP cmd=109 payload на item-записи
// (Т33: паттерн [ffff] каждые ~470Б, Aion item IDs в диапазоне 100-200М).
func ParseItemBlocks(payload []byte) []ItemRecord {
	var records []ItemRecord
	var ffffPos []int
	for i := 20; i < len(payload)-2; i++ {
		if payload[i] == 0xFF && payload[i+1] == 0xFF {
			ffffPos = append(ffffPos, i)
		}
	}
	if len(ffffPos) < 2 {
		records = append(records, ItemRecord{Offset: 0, Length: len(payload), Raw: payload})
		return records
	}
	start := 0
	for _, fp := range ffffPos {
		rec := ItemRecord{
			Offset:  start,
			Length:  fp - start,
			Raw:     payload[start:fp],
			HasFFFF: true,
		}
		for off := 0; off < len(rec.Raw)-4; off += 4 {
			v := binary.LittleEndian.Uint32(rec.Raw[off : off+4])
			if v >= 100000000 && v <= 199999999 {
				rec.ItemID = v
				break
			}
		}
		records = append(records, rec)
		start = fp
	}
	if start < len(payload) {
		records = append(records, ItemRecord{Offset: start, Length: len(payload) - start, Raw: payload[start:]})
	}
	return records
}

// CountItemRecords — счётчик item-записей в cmd=109 payload.
func CountItemRecords(payload []byte) int {
	return len(ParseItemBlocks(payload))
}

// GenerateCharListStructX — RP cmd=90 с X-подстановкой (internal helper).
func GenerateCharListStructX(sessionX uint32) []byte {
	tpl := loadTemplate("rp90_charlist.bin")
	if tpl == nil {
		return nil
	}
	out := make([]byte, len(tpl))
	copy(out, tpl)
	binary.LittleEndian.PutUint32(out[7:11], sessionX)
	return out
}

// fmtItemRecord — отладочная строка для item-записи.
func fmtItemRecord(r ItemRecord) string {
	if r.ItemID > 0 {
		return fmt.Sprintf("item=%d len=%d", r.ItemID, r.Length)
	}
	return fmt.Sprintf("len=%d", r.Length)
}

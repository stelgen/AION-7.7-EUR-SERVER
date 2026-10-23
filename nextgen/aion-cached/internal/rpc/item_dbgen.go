// item_dbgen.go — Этап 3: генератор O109 item-блоков из БД user_item.
//
// КАРТА записи 470Б (Этап 3, db-field-map §Этап 3, 23.10):
//   +0   item_id u32 LE (двойное вхождение: +0 и +22)
//   +4   amount u32 LE
//   +461 slot u8 (старший байт @462 = item-специфика: 0x43/0x46/0x4B/0x4C/0x4D)
//   +463 amount u32 LE (дубль)
//   остальное — опции/энчант (шаблон сохраняет)
//
// Фильтры: warehouse=1 (склад) НЕ шлётся в login-блоках (29 строк БД 1002).
// Порядок записей в capture ≠ БД-порядок — сверка по МНОЖЕСТВУ (id,amount,slot).
package rpc

import "encoding/binary"

// DBItemRow — строка user_item (мир-БД).
type DBItemRow struct {
	NameID  uint32
	SlotID  int32
	Amount  uint32
	Slot    uint32
	Warehouse uint32
}

const (
	recSize    = 470
	offRecID   = 0
	offRecAmt  = 4
	offRecSlot = 461
	recsPerBlockDefault = 16
)

// GenerateItemRecord — запись 470Б из шаблона + БД-поля.
func GenerateItemRecord(tplRec []byte, r DBItemRow) []byte {
	out := make([]byte, len(tplRec))
	copy(out, tplRec)
	if len(out) < 470 {
		return out
	}
	binary.LittleEndian.PutUint32(out[offRecID:offRecID+4], r.NameID)
	binary.LittleEndian.PutUint32(out[offRecAmt:offRecAmt+4], r.Amount)
	out[offRecSlot] = byte(r.Slot & 0xFF)
	// дубль item_id @+22 (внутренняя ссылка)
	if len(out) > 26 {
		binary.LittleEndian.PutUint32(out[22:26], r.NameID)
	}
	// дубль amount @+463
	if len(out) >= 467 {
		binary.LittleEndian.PutUint32(out[463:467], r.Amount)
	}
	return out
}

// GenerateItemBlocksFromDB — сборка блоков cmd=109 из БД-строк:
// заголовок из шаблонного блока (с SubstWorldFlow-подстановкой X/charRef),
// записи по recsPerBlock из БД (warehouse=0 only), последняя — хвост шаблона.
// tplBlock — capture-блок (шаблон записи = запись #2, idx 470..940).
func GenerateItemBlocksFromDB(tplBlock []byte, sessionX, charRef uint32, rows []DBItemRow, recsPerBlock int) [][]byte {
	if len(tplBlock) < 32+recSize*2 {
		return nil
	}
	// фильтр склада
	var inv []DBItemRow
	for _, r := range rows {
		if r.Warehouse == 0 {
			inv = append(inv, r)
		}
	}
	if recsPerBlock <= 0 {
		recsPerBlock = recsPerBlockDefault
	}
	var out [][]byte
	tplRec := tplBlock[32+recSize : 32+recSize*2] // запись #2 как шаблон полей
	for start := 0; start < len(inv); start += recsPerBlock {
		end := start + recsPerBlock
		if end > len(inv) {
			end = len(inv)
		}
		chunk := inv[start:end]
		total := 7 + 32 + len(chunk)*recSize + 22 // wire = hdr7 + payload(заголовок32 + записи + хвост22)
		blk := make([]byte, total)
		copy(blk[7:39], tplBlock[7:39]) // payload-заголовок 32Б (frame 7..39)
		for i, r := range chunk {
			copy(blk[39+i*recSize:39+(i+1)*recSize], GenerateItemRecord(tplRec, r)) // записи на payload[32]
		}
		// хвост из шаблона (последние 22Б)
		if len(tplBlock) >= len(blk) {
			copy(blk[39+len(chunk)*recSize:], tplBlock[39+len(chunk)*rec_size_guard(len(chunk), recsPerBlock):len(tplBlock)])
		}
		// сессионная подстановка заголовка: X-байты @2, charRef @10 (заголовок блока = [len2][cmd2][AF][~cmd2][X@2]... — X в payload @2)
		binary.LittleEndian.PutUint32(blk[9:13], sessionX) // payload[2:6]
		binary.LittleEndian.PutUint32(blk[17:21], charRef) // payload[10:14]
		// len-поле и ~cmd пересчитать
		fixLen(blk)
		out = append(out, blk)
	}
	return out
}

// rec_size_guard — хелпер для безопасного индекса хвоста.
func rec_size_guard(count, per int) int { return count * per }

// fixLen — пересчёт len-поля и ~cmd для собранного блока (accache-канон).
func fixLen(blk []byte) {
	total := uint16(len(blk))
	binary.LittleEndian.PutUint16(blk[0:2], total)
	cmd := binary.LittleEndian.Uint16(blk[2:4])
	binary.LittleEndian.PutUint16(blk[5:7], ^cmd)
}

// world_flow_db.go — Этап 5: интеграция DB-генератора в мир-init эмиссию.
//
// Стратегия: существующий chain (118 кадров) — базовый порядок кадров.
// O1 (RP_LOAD) и O109 (item-блоки) ЗАМЕНЯЮТСЯ на DB-генерированные версии.
// Остальные кадры (мир-кадры O108/O23/O32/O95/O97/O242/O244/O239/O245 и др.)
// остаются из chain с сессионными подстановками (template passthrough).
//
// Результат: данные чара из БД для ЛЮБОГО чара + мир-кадры из capture.
package rpc

import "encoding/binary"

// EmitWorldFlowDB — DB-версия мир-init load-сегмента.
// Принимает: chain-кадры (из GetWorldFlowChain), сессию, DB-данные.
// Возвращает: кадры с заменёнными O1/O109 + session-подстановками.
func EmitWorldFlowDB(frames [][]byte, sessionX, charRef uint32, w CharWorldData, items []DBItemRow) [][]byte {
	var out [][]byte
	itemIdx := 0 // индекс для O109-записей (по порядку БД)
	recsPerBlk := recsPerBlockDefault

	for _, f := range frames {
		if len(f) < 7 {
			out = append(out, f)
			continue
		}
		cmd := binary.LittleEndian.Uint16(f[2:4])

		switch cmd {
		case 1: // O1 RP_LOAD → DB-генератор
			db1 := GenerateRPLoadFromTemplate(f, sessionX, charRef, w)
			out = append(out, db1)

		case 0x6D: // O109 item-блок → DB-генератор (по N записей из items)
			if itemIdx < len(items) {
				end := itemIdx + recsPerBlk
				if end > len(items) {
					end = len(items)
				}
				chunk := items[itemIdx:end]
				itemIdx = end
				// строим записи
				tplRec := f[39 : 39+recSize] // запись-шаблон из chain
				n := len(chunk)
				blk := make([]byte, 7+32+n*recSize+22)
				copy(blk[:7+32], f[:7+32]) // заголовок из chain
				for i, r := range chunk {
					row := DBItemRow{NameID: r.NameID, Amount: r.Amount, Slot: r.Slot}
					copy(blk[39+i*recSize:39+(i+1)*recSize], GenerateItemRecord(tplRec, row))
				}
				// хвост из chain (последние 22Б)
				tailStart := 39 + n*recSize
				if tailStart+22 <= len(f) {
					copy(blk[tailStart:tailStart+22], f[len(f)-22:])
				} else if len(f) >= 22 {
					copy(blk[tailStart:tailStart+22], f[len(f)-22:])
				}
				fixLen(blk)
				out = append(out, blk)
			} else {
				out = append(out, f) // БД-айтемы кончились — passthrough
			}

		default: // мир-кадры + O14 + прочие — chain passthrough с session-подстановкой
			out = append(out, SubstituteWorldFlow(f, WorldFlowSess1X, sessionX, WorldFlowSess1CharRef, charRef, WorldFlowSess1A, 4))
		}
	}
	return out
}

// Package server — opcode-таблицы и диспетчер (R1 каркас).
// Каналы 1-в-1 к канонам L2 C1 (nextgen/cached-ref/l2-c1-cached/include/cached/network/packets/):
//
//	2006 server:      serverHandlers/  (223 файла: 000_CacheVersion, 001_LoadCharacter, 009_SaveCharacter, ...)
//	2007 admin:       adminHandlers/   (57 файлов GQ_*: 01_CheckCharacter, 03_SetBuilderCharacter, 12_AddItem, ...)
//	2009 npcDb:       npcDbHandlers/   (6 файлов: 01_Version, 02_LoadNpc, 03_SaveNpc, 04_UpdateBoss)
package server

import (
	"log"

	"aion-cached/internal/ship"
	"aion-cached/internal/wire"
)

// Handler — хандлер одного opcode (payload в, кадр-ответ out; nil = не отвечаем).
type Handler func(ch Channel, opcode uint16, payload []byte) []byte

var (
	// serverHandlers — 2006: RQ/RP (ServerToDb / DbToServer).
	serverHandlers = map[uint16]Handler{
		0x00: stub("server:CacheVersion", "000_CacheVersion"),
		0x01: stub("server:LoadCharacter", "001_LoadCharacter"),
		0x09: stub("server:SaveCharacter", "009_SaveCharacter"),
	}

	// adminHandlers — 2007: interactive GQ/GP (AdminToCache / CacheToAdmin).
	adminHandlers = map[uint16]Handler{
		0x01: stub("admin:GQ_MAKEBUILDER", "03_SetBuilderCharacter"), // GQ_MAKEBUILDER ↔ L2 03_SetBuilderCharacter
		0x12: stub("admin:GQ_ITEMADD", "12_AddItem"),                 // GQ_ITEMADD ↔ L2 12_AddItem
	}

	// npcDbHandlers — 2009: NPC-DB канал (NPCSvr64).
	npcDbHandlers = map[uint16]Handler{
		0x01: stub("npcDb:Version", "01_Version"),
		0x02: stub("npcDb:LoadNpc", "02_LoadNpc"),
		0x03: stub("npcDb:SaveNpc", "03_SaveNpc"),
		0x04: stub("npcDb:UpdateBoss", "04_UpdateBoss"),
	}
)

// Route — opcode-роутер по каналу. Возвращает ответ-кадр (wire.Build, НЕзашифрованный).
// Лог + ship-событие на каждый вызов (raw-first канон LOGGING-SPEC).
func Route(ch Channel, frame *wire.Frame) []byte {
	var table map[uint16]Handler
	var chanName string
	switch ch {
	case ChanServer:
		table, chanName = serverHandlers, "server"
	case ChanAdmin:
		table, chanName = adminHandlers, "admin"
	case ChanNpcDb:
		table, chanName = npcDbHandlers, "npcDb"
	default:
		return nil
	}

	h, ok := table[frame.Opcode]
	if !ok {
		log.Printf("[cached] %s: unknown protocol opcode 0x%04X (unknown-proto)", chanName, frame.Opcode)
		ship.Emit(ship.Event{Channel: chanName, Opcode: frame.Opcode, Name: "<unknown>", Status: "unknown"})
		return nil
	}
	log.Printf("[cached] %s dispatch opcode 0x%04X (%d bytes payload)", chanName, frame.Opcode, len(frame.Payload))
	return h(ch, frame.Opcode, frame.Payload)
}

// stub — заглушка: лог opname + ship-событие, кадр-ответ-«ок».
func stub(opName, refHeader string) Handler {
	return func(ch Channel, opcode uint16, payload []byte) []byte {
		log.Printf("[cached] stub handler %s (ref: packet%s.h) payload=%d bytes", opName, refHeader, len(payload))
		ship.Emit(ship.Event{Channel: chanName(ch), Opcode: opcode, Name: opName, Bytes: len(payload), Status: "ok"})
		return wire.Build(0xFF, []byte("ok"))
	}
}

// chanName — имя канала для ship.
func chanName(ch Channel) string {
	switch ch {
	case ChanServer:
		return "server"
	case ChanAdmin:
		return "admin"
	case ChanNpcDb:
		return "npcDb"
	}
	return "unknown"
}

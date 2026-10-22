// Package server — opcode-роутер и MVP-хандлеры (R3).
// Каналы 1-в-1 к канонам L2 C1 (nextgen/cached-ref/l2-c1-cached/.../packets/):
//
//	2006 server: ServerToDb (381 RQ) / DbToServer (238 RP) — нумерация из .profile
//	2007 admin:  AdminToCache (55 GQ) / CacheToAdmin (53 GP)
//	2009 npcDb:  NPC-DB канал (NPCSvr64)
//
// Политика ответов (capture-канон accache/cached): ответ ТОЛЬКО там, где форма
// доказана. Доказательства уровня «канон-эталон L2 C1» (сурсы MasterToma):
//   - RQ_PING → RP_PING not-found форма "cdddddddd" (packet144_RequestPing.cpp);
//   - RQ_VERSION: форма S2C "cd" доказана (packet000), значение прот-версии Aion
//     НЕизвестно ($Revision: 311 = L2) ⇒ НЕ отвечаем (журнал теорий #9).
// Остальные cmds — лог+ship, тишина (как unknown у ориг; арбитр = capture R4).
package server

import (
	"log"
	"time"

	"aion-cached/internal/rpc"
	"aion-cached/internal/ship"
	"aion-cached/internal/wire"
)

// Handler — хандлер одного opcode: payload в, ответ-кадр out (nil = тишина).
type Handler func(ch Channel, opcode uint16, payload []byte) []byte

// Константы топ-команд ServerToDb/DbToServer (из rpc.ServerToDbNames/DbToServerNames).
const (
	rqPing    = 108 // RQ_PING
	rqVersion = 215 // RQ_VERSION
	rqLoad    = 11  // RQ_LOAD (char login read-путь)
	rqLoadIte = 14  // RQ_LOAD_ITEM
	rqLeaveW  = 15  // RQ_LEAVE_WORLD
	rqUSave   = 16  // RQ_U_SAVE
	rpPing    = 45  // RP_PING
)

var (
	// serverHandlers — 2006: RQ (ServerToDb). MVP-хендлеры R3/R4.
	serverHandlers = map[uint16]Handler{
		rqPing:    hPing,    // доказанная форма (L2 C1 packet144)
		rqVersion: hVersion, // лог без ответа (форма доказана, значение — нет)
		rqLoad:    hRQLoad,           // R4 22.10: [X][a][charId][accId] (live capture)
		rqLoadIte: hRQLoadItem,      // R4 22.10: [X][a][charId][кат][accId][page]
		rqLeaveW:  hRQLeaveWorld,     // R4 22.10: [acc][char][pos×4][pad]
		rqUSave:   hRQUSave,          // R4 22.10: [acc][char][сейв...]
		rqSessionOpen: hSessionOpen,  // R4 22.10: первый кадр сессии [X][a][accId]
		rqSessionAcc:  hSessionAcc,   // R4 22.10: [X][a][accId][01 01 00]
	}

	// adminHandlers — 2007: GQ (interactive). Заглушки silent до capture.
	adminHandlers = map[uint16]Handler{}

	// npcDbHandlers — 2009: NPC-DB канал. Заглушки silent до capture.
	npcDbHandlers = map[uint16]Handler{}
)

// Route — opcode-роутер по каналу. Возвращает ответ-кадр (wire.Build, plaintext) или nil.
// Лог + ship-событие на каждый вызов (raw-first канон LOGGING-SPEC).
func Route(ch Channel, frame *wire.Frame) []byte {
	var table map[uint16]Handler
	var protoIdx int
	var chanName string
	switch ch {
	case ChanServer:
		table, protoIdx, chanName = serverHandlers, 0, "server"
	case ChanAdmin:
		table, protoIdx, chanName = adminHandlers, 6, "admin"
	case ChanNpcDb:
		table, protoIdx, chanName = npcDbHandlers, -1, "npcDb" // npcDb-словаря в .profile нет (Name -> <UNK>, R4)
	default:
		return nil
	}

	name := rpc.Name(protoIdx, frame.Cmd)
	h, ok := table[frame.Cmd]
	if !ok {
		log.Printf("[cached] %s unhandled %s (op=%d) len=%d — dropped (silent policy)",
			chanName, name, frame.Cmd, len(frame.Payload))
		ship.Emit(ship.Event{Channel: chanName, Opcode: frame.Cmd, Name: name,
			Bytes: len(frame.Payload), Status: "unhandled"})
		return nil
	}
	log.Printf("[cached] %s dispatch %s (op=%d, %d bytes payload)",
		chanName, name, frame.Cmd, len(frame.Payload))
	ship.Emit(ship.Event{Channel: chanName, Opcode: frame.Cmd, Name: name,
		Bytes: len(frame.Payload), Status: "dispatched"})
	return h(ch, frame.Cmd, frame.Payload)
}

// hPing — RP_PING (not-found форма, L2 C1 packet144_RequestPing.cpp канон):
// C2S "ddddd" [p1][charId][p2][p3][p4] →
// S2C "cdddddddd" [RP_PING][p2][p3][p4][p1][ts][0][0][0].
// Журнал теорий #10: Aion-форма ожидаемо идентична (арбитр = live capture R4).
func hPing(ch Channel, opcode uint16, payload []byte) []byte {
	if len(payload) < 20 {
		log.Printf("[cached] ping parse.err len=%d (<20) — drop", len(payload))
		ship.Emit(ship.Event{Channel: chanName(ch), Opcode: opcode, Name: "RP_PING",
			Bytes: len(payload), Status: "parse.err"})
		return nil
	}
	p1 := le32(payload[0:4])
	charId := le32(payload[4:8])
	p2 := le32(payload[8:12])
	p3 := le32(payload[12:16])
	p4 := le32(payload[16:20])
	// RAM-модели юзеров нет (R4) — always not-found ветка (штатная у ориг).
	resp33 := make([]byte, 32) // 8 dwords: [p2][p3][p4][p1][ts][0][0][0]
	put32(resp33[0:], p2)
	put32(resp33[4:], p3)
	put32(resp33[8:], p4)
	put32(resp33[12:], p1)
	put32(resp33[16:], uint32(time.Now().Unix())) // pingReceivedTimestamp
	// [20:32] = нули
	log.Printf("[cached] ping char=%d (not-found form, L2 C1 canon)", charId)
	ship.Emit(ship.Event{Channel: chanName(ch), Opcode: opcode, Name: "RP_PING",
		Bytes: len(payload), Status: "notfound"})
	return wire.BuildS2C(rpPing, resp33)
}

// hVersion — C2S "d" [clientProtocolVersion]; S2C форма "cd" доказана L2 C1
// (packet000_CacheVersionPacket.cpp: Send("cd", op, isEqual)), но cachedProtocolVersion
// Aion НЕизвестна ($Revision: 311 = L2 C1) ⇒ НЕ отвечаем (журнал теорий #9).
func hVersion(ch Channel, opcode uint16, payload []byte) []byte {
	if len(payload) < 4 {
		log.Printf("[cached] version parse.err len=%d — drop", len(payload))
		return nil
	}
	client := le32(payload[0:4])
	log.Printf("[cached] version clientProtocol=%d — no reply (cachedProtocolVersion Aion TBD, теории #9)",
		client)
	ship.Emit(ship.Event{Channel: chanName(ch), Opcode: opcode, Name: "RP_VERSION",
		Bytes: len(payload), Status: "no-reply-tbd"})
	return nil
}

// silent — хандлер топ-пути read/write: лог + ship, тишина (раскладки = R4/capture).
func silent(name string) Handler {
	return func(ch Channel, opcode uint16, payload []byte) []byte {
		log.Printf("[cached] %s: %d bytes — logged, silent (раскладка = capture R4)", name, len(payload))
		ship.Emit(ship.Event{Channel: chanName(ch), Opcode: opcode, Name: name,
			Bytes: len(payload), Status: "silent-tbd"})
		return nil
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

// le32/put32 — LE u32-хелперы хендлеров.
func le32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

func put32(b []byte, v uint32) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}


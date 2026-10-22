// relay_handlers.go — GQ/GP MVP (NPRelay-клиент на 2006).
//
// КАНОН (доказано 22.10):
//   - NPRelay64 common.xml: cacheServerPort=2006 — релей подключается к ТОМУ ЖЕ
//     2006-каналу, что и Server64 (второй тип клиента).
//   - GQ (NPRelay2Server, 55 ops) — запросы релея; GP (Server2NPRelay, 52) — ответы.
//   - id-пространства RQ (Server2DB) и GQ ПЕРЕСЕКАЮТСЯ (RQ_U_UPDATE_COUNT=2 vs
//     GQ_LOGON=2) ⇒ режим соединения переключается по handshake-кадру:
//     GQ_LOGON(2)/GQ_KICKOFF(1) → per-conn relay-режим (диспетчер GQ-имён).
//   - ProtocolVersion@RelaySocket (0x14008d940) = `xor eax,eax; ret` — NO-OP
//     заглушка (как DummyCrypt), handshakes в этой сборке упрощены.
//
// Политика MVP: все GQ-cmds silent-лог (раскладки = capture R4), GP-ответы НЕ
// эмитим. GQ-таблица: GQ_ITEMADD=3, GQ_MAKEBUILDER=7, GQ_CHECK_CHAR=20,
// GQ_SET_BUILDER_CHAR=22, GQ_KICK_CHAR=24, GQ_MAIL_SEND=34, GQ_BUILD_COMMAND=43.
package server

import (
	"log"

	"aion-cached/internal/rpc"
	"aion-cached/internal/ship"
)

// GQ-константы (handshake/ключевые).
const (
	gqKickoff = 1
	gqLogon   = 2
)

// connMode — режим соединения 2006.
type connMode int

const (
	modeServer connMode = iota // Server64: RQ/RP (дефолт)
	modeRelay                  // NPRelay64: GQ/GP (после handshake)
)

// relayGQHandler — хандлер GQ-опкода (silent MVP).
type relayGQHandler func(name string, opcode uint16, payload []byte) []byte

// relayHandlers — GQ-диспетчер (silent по умолчанию).
func relayGQDispatch(mode *connMode, ch Channel, opcode uint16, payload []byte, name string) []byte {
	log.Printf("[cached] relay GQ %s (op=%d, %d bytes) — silent (раскладки = capture R4)",
		name, opcode, len(payload))
	ship.Emit(ship.Event{Channel: "relay", Opcode: opcode, Name: name,
		Bytes: len(payload), Status: "silent-tbd"})
	return nil
}

// maybeRelayHandshake — если кадр = GQ_LOGON/GQ_KICKOFF (ServerToDb-имена при этом
// не меняются: RQ-таблица тоже содержит 1/2, но это RQ_UNUSED_0/RQ_U_UPDATE_COUNT —
// handshake релея различаем ПО ИМЕНИ-КЛАССУ ниже; MVP-эвристика: кадр cmd 1/2 в
// начале соединения переводит conn в relay-режим — Server64 первым кадром шлёт
// RQ_VERSION(215) или RQ_TEST_* (не 1/2), NPRelay64 — GQ_LOGON(2)/GQ_KICKOFF(1).
func maybeRelayHandshake(mode *connMode, opcode uint16) bool {
	if *mode == modeRelay {
		return true // уже релей
	}
	if opcode == gqLogon || opcode == gqKickoff {
		*mode = modeRelay
		log.Printf("[cached] conn switched to RELAY mode (handshake GQ op=%d)", opcode)
		ship.Emit(ship.Event{Channel: "relay", Opcode: opcode,
			Name: rpc.Name(6, opcode), Status: "handshake"})
		return true
	}
	return false
}
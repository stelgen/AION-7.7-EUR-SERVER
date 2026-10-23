// relay_handlers.go — GQ/GP MVP (NPRelay-клиент на 2006).
//
// КАНОН (дизasm 23.10, [cached-ref/gq-gp-wire-20261023.md] — supersedes 22.10-гипотезу):
//   - NPRelay64 common.xml: cacheServerPort=2006 — релей подключается к ТОМУ ЖЕ
//     2006-каналу, что и Server64 (второй тип клиента, НЕ отдельный порт).
//   - Binary relay-wire = ЗАГЛУШКА: RelaySocket::OnRead @0x14008e080 — ACS-каркас
//     (state@+0x1cc/len@+0x1c8/buf@+0x2008/handler-table@+0x1d8, лимит 0x2000),
//     кадр = [u16 total][type u8][0xBE const][payload], type u8 = 0..5
//     (≥6 → лог "Not supported operation" + close), NO cmd u16 / NO 0xEB-маркер.
//   - Таблица хендлеров @0x141556d50: slot0=ProtocolVersion (xor eax,eax;ret —
//     NO-OP), slot1-5=NoPacket (лог "not supported", ret 0, БЕЗ ответа).
//     OnCreate таблицу НЕ переустанавливает ⇒ по wire живут только type 0..5,
//     и все они silent (ориг отвечает NOTHING) — наш silent-MVP = байт-паритет.
//   - Реальный NPRelay-флоу (wallet/vendor/mail/escrow) = RQ-канон 2006 через
//     общий ServerToDb-диспетчер: 9 Decode*Relay(RelaySocket*) + 8
//     Encode*Result + Get/PutRelaySocketPtr — раскладки в доке §2/§3.
//   - GQ/GP-имена из .profile (55/52 ops) = словарь 2007-text admin-канала
//     (теория #14 подтверждена дизasmом), НЕ binary wire.
//
// Политика MVP: relay-кадры silent-лог (паритет NoPacket: ориг молчит), GP
// НЕ эмитим. Детект = на СЫРОМ буфере ДО accache-Feed (relay-кадр короче
// HeaderWire=7 и маркер чужой — Feed его reject'нул бы и закрыл conn).
package server

import (
	"encoding/binary"
	"log"

	"aion-cached/internal/ship"
)

// relayBe — константа протокола RelaySocket (GetCmd_WRQ: buf[1] обязан = 0xBE).
const relayBe = 0xBE

// relayTypeNames — имена wire-type 0..5 (слоты таблицы 0x141556d50:
// slot0=ProtocolVersion, slot1-5=NoPacket; GQ-семантика по .profile).
var relayTypeNames = [6]string{"VERSION", "GQ_KICKOFF", "GQ_LOGON", "GQ_ITEMADD", "GQ_ITEMDELETE", "GQ_ITEMMOVE"}

// connMode — режим соединения 2006.
type connMode int

const (
	modeServer connMode = iota // Server64: RQ/RP (дефолт)
	modeRelay                  // NPRelay64: relay-wire (после первого [type][0xBE]-кадра)
)

// relayFrame — детект relay-кадра на сыром буфере (дизasm-канон):
// [u16 total][type u8 ≤5][0xBE][payload]. total = полная длина кадра.
// Возвращает type и total; ok=false если это не relay-кадр.
// Безопасность: у accache-кадра b3 = cmd>>8; cmd с highbyte 0xBE = 0xBE00+ >
// лимита 0x2000 ⇒ протокол-кадр с b3==0xBE не существует — ложных срабатываний нет.
func relayFrame(b []byte) (typ byte, total int, ok bool) {
	if len(b) < 4 {
		return 0, 0, false
	}
	if b[2] > 5 || b[3] != relayBe {
		return 0, 0, false
	}
	total = int(binary.LittleEndian.Uint16(b[0:2]))
	if total < 4 || total > 0x2000 {
		return 0, 0, false
	}
	return b[2], total, true
}

// relayWireDispatch — MVP-хендлер relay-wire кадра (silent, паритет ориг:
// slot0=NO-OP VERSION, slot1-5=NoPacket "not supported operation" без ответа).
func relayWireDispatch(mode *connMode, typ byte, payload []byte) []byte {
	if *mode != modeRelay {
		*mode = modeRelay
		log.Printf("[cached] conn switched to RELAY mode (wire type=%d %s)", typ, relayTypeNames[typ])
		ship.Emit(ship.Event{Channel: "relay", Opcode: uint16(typ),
			Name: relayTypeNames[typ], Status: "handshake"})
	}
	name := relayTypeNames[typ]
	status := "noop"
	if typ >= 1 {
		status = "not-supported(silent-parity)"
	}
	log.Printf("[cached] relay wire %s (type=%d, %d bytes) — silent", name, typ, len(payload))
	ship.Emit(ship.Event{Channel: "relay", Opcode: uint16(typ), Name: name,
		Bytes: len(payload), Status: status})
	return nil
}

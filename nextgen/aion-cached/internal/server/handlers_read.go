// R4 read-путь MVP — раскладки из live capture 22.10 (ROADMAP «РАСКЛАДКИ READ-ПУТИ
// СНЯТЫ»; логин юзера через capture-стенд cdmirror). Все кадры marker 0xAE/0xAF (Т26),
// plaintext. Хендлеры: ПАРСИНГ + лог + ship; ответы S2C — следующий шаг (нужен полный
// разбор RP cmd=1/34/109 — orig-эталоны в capture).
package server

import (
	"fmt"
	"log"

		"aion-cached/internal/ship"
)

// rqSession — общий префикс сессионных кадров: [X u32][a u32][accId u32].
type rqSession struct {
	X     uint32 // хендл сессии (константа в рамках conn, 0x06F38F70 в capture)
	A     uint32 // константа 4 (роль ⏳)
	AccID uint32
}

func parseSession(pl []byte) (rqSession, bool) {
	if len(pl) < 12 {
		return rqSession{}, false
	}
	return rqSession{X: le32(pl[0:4]), A: le32(pl[4:8]), AccID: le32(pl[8:12])}, true
}

// Константы read-пути (cmd u16; rqLoad/rqLoadIte/rqLeaveW/rqUSave уже в handlers.go).
const (
	rqSessionOpen = 288 // первый кадр сессии Server64: [X][a][accId][pad]
	rqSessionAcc  = 175 // [X][a][accId][01 01 00]
)

// rqLoadFrame — cmd 11 (23Б): [X][a][charId][accId].
type rqLoadFrame struct {
	Session            rqSession
	CharID             uint32
	ExtraDwordAccIDDup uint32 // payload[12:16] = accId (дубль; роль ⏳)
}

// rqLoadItemFrame — cmd 14 (28Б): [X][a][charId][category][accId][page u8] + pad.
type rqLoadItemFrame struct {
	Session   rqSession
	CharID    uint32
	Category  uint32 // capture: 02, 03 (два запроса на логин)
	AccID     uint32
	Page      byte // capture: 00 / 01
}

// rqLeaveWorldFrame — cmd 15 (35Б): [accId][charId][x][y][z][heading][pad].
type rqLeaveWorldFrame struct {
	AccID    uint32
	CharID   uint32
	Pos      [4]uint32 // координаты/heading — роль ⏳ (float? xor? — арбитр RP cmd=1)
	Pad      uint32
}

// hSessionOpen — cmd 288.
func hSessionOpen(ch Channel, opcode uint16, pl []byte) []byte {
	r, ok := parseSession(pl)
	if !ok {
		log.Printf("[cached] session open parse.err len=%d", len(pl))
		return nil
	}
	log.Printf("[cached] session open X=%#x a=%d acc=%d", r.X, r.A, r.AccID)
	emitRead(ch, opcode, "RQ_SESSION_OPEN", fmt.Sprintf("X=%#x acc=%d", r.X, r.AccID))
	return nil
}

// hSessionAcc — cmd 175.
func hSessionAcc(ch Channel, opcode uint16, pl []byte) []byte {
	r, ok := parseSession(pl)
	if !ok || len(pl) < 16 {
		log.Printf("[cached] session acc parse.err len=%d", len(pl))
		return nil
	}
	log.Printf("[cached] session acc X=%#x a=%d acc=%d tail=%x", r.X, r.A, r.AccID, pl[12:16])
	emitRead(ch, opcode, "RQ_SESSION_ACC", fmt.Sprintf("X=%#x acc=%d", r.X, r.AccID))
	return nil
}

// hRQLoad — cmd 11: [X][a][charId][accId].
func hRQLoad(ch Channel, opcode uint16, pl []byte) []byte {
	if len(pl) < 16 {
		log.Printf("[cached] RQ_LOAD parse.err len=%d", len(pl))
		return nil
	}
	s, _ := parseSession(pl[:12])
	charID, accID := le32(pl[8:12]), le32(pl[12:16])
	log.Printf("[cached] RQ_LOAD X=%#x a=%d char=%d acc=%d", s.X, s.A, charID, accID)
	emitRead(ch, opcode, "RQ_LOAD", fmt.Sprintf("char=%d acc=%d", charID, accID))
	return nil
}

// hRQLoadItem — cmd 14: [X][a][charId][category][accId][page u8].
func hRQLoadItem(ch Channel, opcode uint16, pl []byte) []byte {
	if len(pl) < 21 {
		log.Printf("[cached] RQ_LOAD_ITEM parse.err len=%d", len(pl))
		return nil
	}
	s, _ := parseSession(pl[:12])
	charID := le32(pl[8:12])
	category := le32(pl[12:16])
	accID := le32(pl[16:20])
	page := pl[20]
	log.Printf("[cached] RQ_LOAD_ITEM X=%#x char=%d cat=%d acc=%d page=%d", s.X, charID, category, accID, page)
	emitRead(ch, opcode, "RQ_LOAD_ITEM", fmt.Sprintf("char=%d cat=%d page=%d", charID, category, page))
	return nil
}

// hRQLeaveWorld — cmd 15: [accId][charId][pos 4×u32][pad].
func hRQLeaveWorld(ch Channel, opcode uint16, pl []byte) []byte {
	if len(pl) < 28 {
		log.Printf("[cached] RQ_LEAVE_WORLD parse.err len=%d", len(pl))
		return nil
	}
	accID, charID := le32(pl[0:4]), le32(pl[4:8])
	var pos [4]uint32
	for i := range pos {
		pos[i] = le32(pl[8+i*4 : 12+i*4])
	}
	log.Printf("[cached] RQ_LEAVE_WORLD acc=%d char=%d pos=[%#x %#x %#x %#x]", accID, charID, pos[0], pos[1], pos[2], pos[3])
	emitRead(ch, opcode, "RQ_LEAVE_WORLD", fmt.Sprintf("char=%d", charID))
	return nil
}

// hRQUSave — cmd 16 (857Б в capture): [accId][charId][большой сейв...] — разбор = отдельный шаг.
func hRQUSave(ch Channel, opcode uint16, pl []byte) []byte {
	if len(pl) < 8 {
		log.Printf("[cached] RQ_U_SAVE parse.err len=%d", len(pl))
		return nil
	}
	accID, charID := le32(pl[0:4]), le32(pl[4:8])
	log.Printf("[cached] RQ_U_SAVE acc=%d char=%d saveBytes=%d — детальный разбор = отдельный шаг", accID, charID, len(pl)-8)
	emitRead(ch, opcode, "RQ_U_SAVE", fmt.Sprintf("char=%d bytes=%d", charID, len(pl)-8))
	return nil
}

// emitRead — ship-событие read-пути.
func emitRead(ch Channel, opcode uint16, name, msg string) {
	log.Printf("[cached] read-path %s: %s", name, msg)
	ship.Emit(ship.Event{Channel: chanName(ch), Opcode: opcode, Name: name,
		Bytes: 0, Status: "parsed:" + msg})
}

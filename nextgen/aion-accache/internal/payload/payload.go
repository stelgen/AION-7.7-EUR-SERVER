// Package payload: раскладки ACQ/ACP-payload — канон R3 (21.10).
// Источники: живой capture 10.10 (accountcache-ref/capture-20261007/frames.txt,
// 2 логина: accId 1010 и 7), proc-сигнатуры (accountcache-ref/db-procs-77-ref58.rpt),
// PDB Decode*/Encode* сигнатуры (pdb-publics-8035).
//
// АКСИОМА №1 (R3, доказана capture): ПЕРВЫЙ DWORD payload = ACCOUNT_ID — но ТОЛЬКО
// для cmds VERSION/FIRST_LOAD/BM/LUNA/JUMPING/TRANSFORM/TRIAL/FATIGUE.
// ⚠ ИСКЛЮЧЕНИЕ: CHAR_LOGIN (16) / CHAR_LOGOUT (17) — свой layout (см. ниже).
// Header = 7Б: [u16 len][u16 cmd][u8 m][u16 ~cmd]; payload = len-7.
// Все ответы строятся с маркером 0xEC (proto.BuildS2C).
package payload

import (
	"encoding/binary"
	"errors"
	"fmt"
	"unicode/utf16"
)

var ErrShort = errors.New("payload: short")

// AccountID читает первый dword payload (= accountId, аксиома №1).
func AccountID(p []byte) (uint32, error) {
	if len(p) < 4 {
		return 0, ErrShort
	}
	return binary.LittleEndian.Uint32(p), nil
}

// --- C2S-запросы ---

// VersionReq (cmd 1 C2S): {A=3, B=1}.
type VersionReq struct{ A, B uint32 }

func ParseVersionReq(p []byte) (*VersionReq, error) {
	if len(p) < 8 {
		return nil, ErrShort
	}
	return &VersionReq{A: binary.LittleEndian.Uint32(p), B: binary.LittleEndian.Uint32(p[4:])}, nil
}

// FirstLoadReq (cmd 4 C2S): [accId] — только accountId.
type FirstLoadReq struct{ AccountID uint32 }

func ParseFirstLoadReq(p []byte) (*FirstLoadReq, error) {
	if len(p) < 4 {
		return nil, ErrShort
	}
	return &FirstLoadReq{AccountID: binary.LittleEndian.Uint32(p)}, nil
}

// BMPackReq (cmd 5 C2S): [accId].
type BMPackReq = FirstLoadReq

// LunaLoadReq (cmd 26 C2S): [accId].
type LunaLoadReq = FirstLoadReq

// CharSessionReq (cmd 16 C2S логин / 17 C2S логаут).
// ⚠ ИСКЛЮЧЕНИЕ из аксиомы №1: здесь accId НЕ первый dword.
// Логин:  [u32 1][u32 0][charRef][accId][utf16z-stamp][u32 1][u32 2][u32 0]
// Логаут: [u32 1][charRef][utf16z-stamp][u32 1][u32 2][u32 0] (БЕЗ accId!)
// charRef = [charId u16][x u16] (0x020003EA = 1002+0002 у capture). Stamp — UTF-16LE
// z-строка локального времени Server64 ("2026-10-07T13:35:55.860").
type CharSessionReq struct {
	F1        uint32
	F2        uint32 // только у логина (=0)
	CharRef   uint32
	AccountID uint32 // только у логина
	Stamp     string
	Tail      []byte // 12Б: 01000000 02000000 00000000
	logout    bool
}

func ParseCharLoginReq(p []byte) (*CharSessionReq, error)  { return parseCharSession(p, false) }
func ParseCharLogoutReq(p []byte) (*CharSessionReq, error) { return parseCharSession(p, true) }

func parseCharSession(p []byte, logout bool) (*CharSessionReq, error) {
	off := 0
	r := &CharSessionReq{logout: logout}
	need := func(n int) error {
		if len(p) < off+n {
			return ErrShort
		}
		return nil
	}
	if err := need(4); err != nil {
		return nil, err
	}
	r.F1 = binary.LittleEndian.Uint32(p[off:])
	off += 4
	if !logout {
		if err := need(4); err != nil {
			return nil, err
		}
		r.F2 = binary.LittleEndian.Uint32(p[off:])
		off += 4
	}
	if err := need(4); err != nil {
		return nil, err
	}
	r.CharRef = binary.LittleEndian.Uint32(p[off:])
	off += 4
	if !logout {
		if err := need(4); err != nil {
			return nil, err
		}
		r.AccountID = binary.LittleEndian.Uint32(p[off:])
		off += 4
	}
	start := off
	for off+2 <= len(p) {
		if p[off] == 0 && p[off+1] == 0 {
			break
		}
		off += 2
	}
	if off+2 > len(p) {
		return nil, ErrShort
	}
	r.Stamp = decodeUTF16(p[start:off])
	off += 2
	if err := need(12); err != nil {
		return nil, err
	}
	r.Tail = append([]byte(nil), p[off:off+12]...)
	return r, nil
}

// FatigueReq (cmd 25 C2S): [accId][point][npckill][updatetime][limit_reset][limit_play]
// Роли полей = aion_UpdateHiddenFatigueInfo_20170428(accountId, hidden_fatigue_point,
// hidden_fatigue_npcKill, hidden_fatigue_updatetime, limit_reset_time, limit_play_time).
// Capture: accId=1010, point=0, npckill=0, updatetime=0x6AC62093, limit_reset=0x6AC5DFE0,
// limit_play=0. Гипотеза ролей ts1=updatetime/ts2=limit_reset — в журнале теорий ROADMAP.
type FatigueReq struct {
	AccountID  uint32
	Point      uint32
	NpcKill    uint32
	UpdateTime uint32
	LimitReset uint32
	LimitPlay  uint32
}

func ParseFatigueReq(p []byte) (*FatigueReq, error) {
	if len(p) < 24 {
		return nil, ErrShort
	}
	return &FatigueReq{
		AccountID:  binary.LittleEndian.Uint32(p),
		Point:      binary.LittleEndian.Uint32(p[4:]),
		NpcKill:    binary.LittleEndian.Uint32(p[8:]),
		UpdateTime: binary.LittleEndian.Uint32(p[12:]),
		LimitReset: binary.LittleEndian.Uint32(p[16:]),
		LimitPlay:  binary.LittleEndian.Uint32(p[20:]),
	}, nil
}

// UpdateTrialReq (cmd 8 C2S): [accId] + 9 u32.
// Capture: accId, ts(0x6AC620A3), 0, 0x395(917), 0, 0x46(70), 0, 0, 0, 0.
// Гипотеза ролей по aion_SetTrialAccountData(accId, updateTime, resetTime, sellGold,
// tradeGold, decomposeSum, gatherSum, extractGatherSum): updateTime=P1, resetTime=P2,
// sellGold=P3(917), tradeGold=P4, decomposeSum=P5(70), gatherSum=P6, extractGatherSum=P7,
// P8/P9 = pad — ЖУРНАЛ теорий, арбитр R4.
type UpdateTrialReq struct {
	AccountID                        uint32
	P1, P2, P3, P4, P5, P6, P7, P8, P9 uint32
}

func ParseUpdateTrialReq(p []byte) (*UpdateTrialReq, error) {
	if len(p) < 40 {
		return nil, ErrShort
	}
	r := &UpdateTrialReq{}
	vals := []*uint32{&r.AccountID, &r.P1, &r.P2, &r.P3, &r.P4, &r.P5, &r.P6, &r.P7, &r.P8, &r.P9}
	for i, v := range vals {
		*v = binary.LittleEndian.Uint32(p[i*4:])
	}
	return r, nil
}

// JumpingReq (cmd 31 C2S): [u8 flag=01][accId][u16]. Хвост-байты capture: 00 01
// — LE-чтение даёт 0x0100; семантика (LE vs BE u16) — ГИПОТЕЗА, арбитр R4.
type JumpingReq struct {
	Flag      byte
	AccountID uint32
	Value     uint16
}

func ParseJumpingReq(p []byte) (*JumpingReq, error) {
	if len(p) < 7 {
		return nil, ErrShort
	}
	return &JumpingReq{
		Flag:      p[0],
		AccountID: binary.LittleEndian.Uint32(p[1:]),
		Value:     binary.LittleEndian.Uint16(p[5:]),
	}, nil
}

// TransformReq (cmd 37 C2S): [u32 subtype=0xF2][accId][u8 00].
type TransformReq struct {
	SubType   uint32
	AccountID uint32
	Flag      byte
}

func ParseTransformReq(p []byte) (*TransformReq, error) {
	if len(p) < 9 {
		return nil, ErrShort
	}
	return &TransformReq{
		SubType:   binary.LittleEndian.Uint32(p),
		AccountID: binary.LittleEndian.Uint32(p[4:]),
		Flag:      p[8],
	}, nil
}

// --- S2C-ответы (каноны capture) ---

// BuildVersionResp (cmd 1 S2C): {3}.
func BuildVersionResp() []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, 3)
	return b
}

// BuildFirstLoadResp (cmd 4 S2C): [accId][point][npckill][limitReset u8][ts][limitAccum u16] = 19Б.
// Роли полей = гипотеза (журнал теорий ROADMAP, арбитр R4-дифф). Байты = capture-канон.
func BuildFirstLoadResp(acc, point, npckill uint32, limitReset byte, updatetime uint32, limitAccum uint16) []byte {
	b := make([]byte, 19)
	binary.LittleEndian.PutUint32(b[0:], acc)
	binary.LittleEndian.PutUint32(b[4:], point)
	binary.LittleEndian.PutUint32(b[8:], npckill)
	b[12] = limitReset
	binary.LittleEndian.PutUint32(b[13:], updatetime)
	binary.LittleEndian.PutUint16(b[17:], limitAccum)
	return b
}

// BuildBMPackResp (cmd 5 S2C): [accId][count u16] — count=0 у capture (паков нет).
// Формат записей при count>0 = TBD R4 (дозахват с активным паком).
func BuildBMPackResp(acc uint32, count uint16) []byte {
	b := make([]byte, 6)
	binary.LittleEndian.PutUint32(b[0:], acc)
	binary.LittleEndian.PutUint16(b[4:], count)
	return b
}

// BuildCharLoginResp (cmd 16 S2C): [accId][8×FF][12×0] = 24Б.
func BuildCharLoginResp(acc uint32) []byte {
	b := make([]byte, 24)
	binary.LittleEndian.PutUint32(b[0:], acc)
	for i := 4; i < 12; i++ {
		b[i] = 0xFF
	}
	return b
}

// LunaUpdatePrefix — фиксированный префикс cmd 27 S2C из capture (f2 00 01 00).
// Семантика TBD R4 (не accId, не timestamp; вероятнее luna-дескриптор).
var LunaUpdatePrefix = []byte{0xF2, 0x00, 0x01, 0x00}

// BuildLunaUpdateResp (cmd 27 S2C): [F2 00 01 00][accId][u16 0] = 10Б.
func BuildLunaUpdateResp(acc uint32) []byte {
	b := make([]byte, 10)
	copy(b, LunaUpdatePrefix)
	binary.LittleEndian.PutUint32(b[4:], acc)
	return b
}

// BuildLunaLoadResp (cmd 26 S2C): [accId][ts unix][u32 0] = 12Б (capture-канон N=19).
func BuildLunaLoadResp(acc, ts uint32) []byte {
	b := make([]byte, 12)
	binary.LittleEndian.PutUint32(b[0:], acc)
	binary.LittleEndian.PutUint32(b[4:], ts)
	return b
}

func u32(b []byte) uint32 { return binary.LittleEndian.Uint32(b) }

func decodeUTF16(b []byte) string {
	u := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		u = append(u, binary.LittleEndian.Uint16(b[i:]))
	}
	return string(utf16.Decode(u))
}

// Debug — строка для лога (не для прод-путей).
func (r *CharSessionReq) Debug() string {
	return fmt.Sprintf("charRef=%#x accId=%d stamp=%q tail=%x", r.CharRef, r.AccountID, r.Stamp, r.Tail)
}

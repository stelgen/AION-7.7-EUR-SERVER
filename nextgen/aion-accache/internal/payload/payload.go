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
	// R4 21.10: раскладка R2.5 ПОДТВЕРЖДЕНА прод-фактами (acc=1010): колонка
	// hidden_fatigue_updatetime = wire-off12 (0x6AC62093/0x6AC620A3 — записывается
	// каждым UPDATE), limit_play_reset_time = wire-off16 (0x6AC5DFE0),
	// limit_play_accum_time = wire-off20 (0). Роли НЕ трогать.
	return &FatigueReq{
		AccountID:  binary.LittleEndian.Uint32(p),
		Point:      binary.LittleEndian.Uint32(p[4:]),
		NpcKill:    binary.LittleEndian.Uint32(p[8:]),
		UpdateTime: binary.LittleEndian.Uint32(p[12:]),
		LimitReset: binary.LittleEndian.Uint32(p[16:]),
		LimitPlay:  binary.LittleEndian.Uint32(p[20:]),
	}, nil
}

// UpdateTrialReq (cmd 8 C2S): [accId] + 9 u32 (канон R5-live 22.10, Т23 — прод-факт):
// [accId][P1=updateTime][P2=resetTime][P3=sellGold][P4=РЕЗЕРВ=0][P5=tradeGold]
// [P6=decomposeSum][P7=gatherSum][P8=extractGatherSum][P9=pad=0].
// P4 = тот самый CONST0 из trial-ответа (S2C 0x06 dword4). Роль «P4=tradeGold»
// опровергнута прод-записью (затёрла trade 70→0). Проц-вызов пропускает P4/P9.
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

// BuildLunaUpdateResp (cmd 27 S2C): [acc&0xFF][0x00][0x01][0x00][accId][u16 0] = 10Б.
// R4-live 22.10: первый байт = accId&0xFF (1010→0xF2, 1023→0xFF) — «константа F2» из
// capture-20261007 была артефактом одного аккаунта. Остальные 3 байта префикса константы.
func BuildLunaUpdateResp(acc uint32) []byte {
	b := make([]byte, 10)
	b[0] = byte(acc & 0xFF)
	b[2] = 0x01
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

// BuildBM21Push (S2C cmd 0x15 = ОТВЕТ НА ASK_CAN_MAKE_JUMPING 0x1F — канон R4-live 22.10:
// все 5 кадров O-0x15 идут непосредственно за C-0x1F (5/5); после LOAD_BM_PACK ориг НЕ пушит
// — «push после BM» (Т2) = ложная корреляция по позиции, опровергнута live):
// [accId u32][0x02][0x01][8×0x00] = 14Б; байты 0x02/0x01 = константы.
func BuildBM21Push(acc uint32) []byte {
	b := make([]byte, 14)
	binary.LittleEndian.PutUint32(b[0:], acc)
	b[4], b[5] = 0x02, 0x01
	return b
}

// --- S2C-пуши R4-ФИНАЛ (22.10, capture-20261022: 2 чара × 3 логина + logout-кластер) ---
// Раскладки выведены сравнением полей между чарами (charRef 0x020003EA/0x020003EB)
// и логинами; константы подтверждены в ОБОИХ corpus (capture-20261007 + capture-20261022).

// CharCustomSavedTsDefault — константа ACP_13 (S2C 0x0D, пуш сразу после VERSION-ответа):
// 0x640831AE = 2023-03-08 06:56:46 UTC. Одинакова в обоих corpus и для обоих чаров.
// Роль поля ⏳ (журнал теорий Т7): не время события (2023 год); вероятно sentinel/
// дата сборки 7.7. Эмитим константой — capture-канон байт-в-байт.
const CharCustomSavedTsDefault uint32 = 0x640831AE

// BuildCharCustomSavedPush (S2C 0x0D): [ts u32] = 4Б.
func BuildCharCustomSavedPush(ts uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, ts)
	return b
}

// CharCreatedConst — константа ACP_15 (S2C 0x0F): 0x000A07EA = 657386.
// Одинакова во всех кадрах обоих corpus (2 чара). Роль ⏳ (Т8): не acc/char/ts
// (657386с от эпохи ≈ 1970-01-08). Эмитим константой — capture-канон.
const CharCreatedConst uint32 = 0x000A07EA

// BuildCharCreatedPush (S2C 0x0F, пуш при CHAR_LOGIN): [charRef][const][3] = 12Б.
// charRef = эхо charRef из C2S CHAR_LOGIN (0x020003EA/0x020003EB — 2 чара capture).
func BuildCharCreatedPush(charRef uint32) []byte {
	b := make([]byte, 12)
	binary.LittleEndian.PutUint32(b[0:], charRef)
	binary.LittleEndian.PutUint32(b[4:], CharCreatedConst)
	binary.LittleEndian.PutUint32(b[8:], 3)
	return b
}

// BuildCharInfoRefreshPush (S2C 0x14, пуш при CHAR_LOGIN): 14×u32 = 56Б:
// [charRef][16×0][lastLogin][12×0][1][8×0][flag][maxLevel].
//   lastLogin = global_user_data.last_login_time (прод-факты SQL AionAccountCacheD_rc:
//   колонки char_id/account_id/create_date/last_login_time/last_logout_time/delete_date/
//   delete_completed_date/user_level/server_id/...; 0x6AC5E7B9 у логина#1 после старта ACS,
//   0x6AD988A7 = секунда-в-секунду предыдущего логина у #2/#3 — RAM-race ориг, Т8);
//   maxLevel  = MAX(user_level) по норм-серверам аккаунта (0x50=80 у ОБОИХ чаров —
//   паттерн maxLevAmongNormalServers из proc user_board_bm; у чара 1003 в БД level=1);
//   flag (dword12): наблюдения 0(10.07)/1/1/0(22.10) — ⏳, в реплее/дефолте 0.
func BuildCharInfoRefreshPush(charRef, lastLogin, flag, maxLevel uint32) []byte {
	b := make([]byte, 56)
	binary.LittleEndian.PutUint32(b[0:], charRef)
	binary.LittleEndian.PutUint32(b[20:], lastLogin)
	binary.LittleEndian.PutUint32(b[36:], 1)
	binary.LittleEndian.PutUint32(b[48:], flag)
	binary.LittleEndian.PutUint32(b[52:], maxLevel)
	return b
}

// BuildConfirmLunaRewardPush (S2C 0x1C, пуш при CHAR_LOGIN): [accId][charId][u16 0] = 10Б.
// charId = младшее слово charRef (0x020003EA → 1002, 0x020003EB → 1003 — capture ×3+×3).
func BuildConfirmLunaRewardPush(acc, charId uint32) []byte {
	b := make([]byte, 10)
	binary.LittleEndian.PutUint32(b[0:], acc)
	binary.LittleEndian.PutUint32(b[4:], charId)
	return b
}

// BuildTrialResp (S2C cmd 0x06, ответ на C2S LOAD_TRIAL 0x07): 10×u32 = 40Б:
// [accId][X echo][resetTime][sellGold][CONST0][tradeGold][decompose][gather][extract][pad].
// R4-хвост 21.10 capture-канон (acc=1010, прод sell=917/trade=70):
// golden dwords после acc+X = [0][917][0][70][0][0][0][0] — dword3 = 0 (константа
// в единственном наблюдённом кадре, роль неясна — журнал теорий), dword4 = tradeGold.
func BuildTrialResp(acc, xEcho uint32, reset, sell, trade, decompose, gather, extract uint32) []byte {
	b := make([]byte, 40)
	binary.LittleEndian.PutUint32(b[0:], acc)
	binary.LittleEndian.PutUint32(b[4:], xEcho)
	binary.LittleEndian.PutUint32(b[8:], reset)
	binary.LittleEndian.PutUint32(b[12:], sell)
	binary.LittleEndian.PutUint32(b[16:], 0) // CONST0 (capture dword3, роль TBD)
	binary.LittleEndian.PutUint32(b[20:], trade)
	binary.LittleEndian.PutUint32(b[24:], decompose)
	binary.LittleEndian.PutUint32(b[28:], gather)
	binary.LittleEndian.PutUint32(b[32:], extract)
	// b[36:] = pad 0
	return b
}

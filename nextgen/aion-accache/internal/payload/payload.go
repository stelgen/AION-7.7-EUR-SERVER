// Package payload: раскладки ACQ/ACP-payload, снятые с ЖИВОГО capture 10.10
// (accountcache-ref/capture-20261007/frames.txt, 2 логина) — каркас R2.5.
//
// База почти всех payload: префикс u32 ChannelID = 0x000003F2 (1010) — «канал»
// сессии Server64<->ACS (наблюдён во всех кадрах кроме CHAR_LOGOUT/VERSION-tail).
// Семантика отдельных полей (арбитр = PDB Decode*/Encode* сигнатуры) — R3.
package payload

import (
	"encoding/binary"
	"errors"
	"fmt"
	"unicode/utf16"
)

// ChannelID — константа канала в префиксе payload (u32 LE f2 03 00 00).
const ChannelID = 0x000003F2

var ErrShort = errors.New("payload: short")

// SplitChannel отделяет префикс канала. ok=false если payload короче 4Б
// или префикс != ChannelID.
func SplitChannel(p []byte) (rest []byte, ok bool) {
	if len(p) < 4 {
		return p, false
	}
	if v := binary.LittleEndian.Uint32(p[:4]); v != ChannelID {
		return p, false
	}
	return p[4:], true
}

// VersionReq (cmd 1 C2S): наблюдено {3, 1}.
type VersionReq struct{ A, B uint32 }

func ParseVersionReq(p []byte) (*VersionReq, error) {
	if len(p) < 8 {
		return nil, ErrShort
	}
	return &VersionReq{A: u32(p[0:]), B: u32(p[4:])}, nil
}

// VersionResp (cmd 1 S2C): наблюдено {3}.
type VersionResp struct{ A uint32 }

func ParseVersionResp(p []byte) (*VersionResp, error) {
	if len(p) < 4 {
		return nil, ErrShort
	}
	return &VersionResp{A: u32(p)}, nil
}

// FirstLoadResp (cmd 4 S2C): ch + u32 + u32 + u8 + ts + u16.
type FirstLoadResp struct {
	Channel uint32
	A, B    uint32
	Flag    byte
	Ts      uint32
	Pad     uint16
}

func ParseFirstLoadResp(p []byte) (*FirstLoadResp, error) {
	if len(p) < 19 {
		return nil, ErrShort
	}
	return &FirstLoadResp{
		Channel: u32(p[0:]), A: u32(p[4:]), B: u32(p[8:]),
		Flag: p[12], Ts: u32(p[13:]), Pad: binary.LittleEndian.Uint16(p[17:]),
	}, nil
}

// BMPackResp (cmd 5 S2C): ch + u16 (статус).
type BMPackResp struct {
	Channel uint32
	Status  uint16
}

func ParseBMPackResp(p []byte) (*BMPackResp, error) {
	if len(p) < 6 {
		return nil, ErrShort
	}
	return &BMPackResp{Channel: u32(p[0:]), Status: binary.LittleEndian.Uint16(p[4:6])}, nil
}

// LunaResp — общий хвост луна-ответов (26/27/28 S2C): ch [+ u32 ts/u32 val] + u16/u32.
type LunaResp struct {
	Channel uint32
	Ts      uint32 // e0dfc56a у LOAD_LUNA; ea030000 у CONFIRM_LUNA_REWARD
	Pad     uint16
}

func ParseLunaResp(p []byte) (*LunaResp, error) {
	if len(p) < 10 {
		return nil, ErrShort
	}
	return &LunaResp{Channel: u32(p[0:]), Ts: u32(p[4:]), Pad: binary.LittleEndian.Uint16(p[8:])}, nil
}

// CharLoginReq (cmd 16 C2S) / CharLogoutReq (cmd 17 C2S) — одна раскладка.
// Логин:     u32 1 | u32 0 | charRef | ch | utf16z-stamp | u32 1 | u32 2 | u32 0
// Логаут:    u32 1 | charRef | utf16z-stamp | u32 1 | u32 2 | u32 0   (БЕЗ channel-префикса)
// Stamp — UTF-16LE z-строка локального времени Server64 ("2026-10-07T13:35:55.860").
type CharSessionReq struct {
	F1      uint32
	F2      uint32 // у логина=0, у логаута поля нет (сдвигается)
	CharRef uint32 // 0x020003EA — charRef (PDB: charId + SpecialSvrTypeEnum)
	Channel uint32 // только у логина
	Stamp   string // декодированный UTF-16 stamp
	Tail    []byte // 12Б: 01000000 02000000 00000000
	logout  bool
}

// ParseCharLoginReq разбирает payload cmd 16 (C2S) со stamp'ом.
func ParseCharLoginReq(p []byte) (*CharSessionReq, error) {
	return parseCharSession(p, false)
}

// ParseCharLogoutReq разбирает payload cmd 17 (C2S).
func ParseCharLogoutReq(p []byte) (*CharSessionReq, error) {
	return parseCharSession(p, true)
}

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
	r.F1 = u32(p[off:])
	off += 4
	if !logout {
		if err := need(4); err != nil {
			return nil, err
		}
		r.F2 = u32(p[off:])
		off += 4
	}
	if err := need(4); err != nil {
		return nil, err
	}
	r.CharRef = u32(p[off:])
	off += 4
	if !logout {
		if err := need(4); err != nil {
			return nil, err
		}
		r.Channel = u32(p[off:])
		off += 4
	}
	// UTF-16LE z-строка до двойного нуля
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
	off += 2 // терминатор
	if err := need(12); err != nil {
		return nil, err
	}
	r.Tail = append([]byte(nil), p[off:off+12]...)
	return r, nil
}

// FatigueReq (cmd 25 C2S): ch + u32 0 + u32 0 + u32 ts1 + u32 ts2 + u32 0.
// PDB-сигнатура DecodeUpdateHiddenFatigue(I,H,I,I,I) — точные роли полей = R3.
type FatigueReq struct {
	Channel uint32
	Zero1   uint32
	Zero2   uint32
	Ts1     uint32 // unix (0x6AC62093)
	Ts2     uint32 // файловое время lo (0x6AC6DFE0)
	Zero3   uint32
}

func ParseFatigueReq(p []byte) (*FatigueReq, error) {
	if len(p) < 24 {
		return nil, ErrShort
	}
	return &FatigueReq{
		Channel: u32(p[0:]), Zero1: u32(p[4:]), Zero2: u32(p[8:]),
		Ts1: u32(p[12:]), Ts2: u32(p[16:]), Zero3: u32(p[20:]),
	}, nil
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
	return fmt.Sprintf("charRef=%#x stamp=%q tail=%x", r.CharRef, r.Stamp, r.Tail)
}

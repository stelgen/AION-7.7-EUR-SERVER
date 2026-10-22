// Package rpc — session-aware подстановки (R4.5): RP-кадры цепочек содержат
// сессионные поля captured-логина (X/a/accId); при эмите живому Server64 они
// должны быть своими. Карта substMap построена из живого A/B 22.10 (тень v10 vs
// копия-2016, пары по cmd+len): KindKeep = рантайм-динамика (координаты/тайм-
// штампы/счётчики — оставляем capture-значения, арбитр = живой прогон).
// Т38: a-поле варьируется по сессии (3/4) — подставляется из cmd=288.
package rpc

import (
	"encoding/binary"
)

// Session — сессионные поля (из cmd=288: [X u32][a u32][accId u16][pad]).
type Session struct {
	X     uint32
	A     byte
	AccID uint16
}

// SubstKind — тип подстановки.
type SubstKind byte

const (
	SubstKeep SubstKind = iota // рантайм-динамика: оставить capture-байт
	SubstX                     // session X (u32)
	SubstA                     // session a (u8)
	SubstACC                   // session accId (u16) — Т36
)

// Subst — одна подстановка (frame-офсеты: payload = frame[7:]).
type Subst struct {
	Off  int
	Kind SubstKind
}

// substMap — per-cmd карта (frame-координаты), источник: живой A/B 22.10 вечер.
// Кадры без записи (0x0DC и пр.) идентичны captured — не трогаются.
var substMap = map[uint16][]Subst{
	0x001: { // RP cmd=1 char struct (1156)
		{7, SubstX}, {11, SubstA}, {24, SubstACC},
		{94, SubstKeep}, {95, SubstKeep}, {415, SubstKeep}, {416, SubstKeep},
		{417, SubstKeep}, {419, SubstKeep}, {420, SubstKeep}, {423, SubstKeep},
		{424, SubstKeep}, {427, SubstKeep}, {486, SubstKeep}, {487, SubstKeep},
		{488, SubstKeep}, {710, SubstKeep}, {711, SubstKeep}, {722, SubstKeep},
		{723, SubstKeep}, {724, SubstKeep}, {725, SubstKeep}, {962, SubstKeep},
		{963, SubstKeep}, {975, SubstKeep}, {1099, SubstKeep}, {1100, SubstKeep},
	},
	0x00B: {{9, SubstX}, {13, SubstA}, {21, SubstKeep}},
	0x00E: {{8, SubstX}, {12, SubstA}, {20, SubstKeep}}, // zlib pages
	0x017: {{9, SubstX}, {13, SubstA}, {21, SubstKeep}},
	0x020: {{9, SubstX}, {13, SubstA}, {21, SubstKeep}},
	0x022: {{9, SubstX}, {13, SubstA}, {21, SubstKeep}},
	0x02C: {{9, SubstX}, {13, SubstA}, {21, SubstKeep}},
	0x03D: {{9, SubstX}, {13, SubstA}, {21, SubstKeep}},
	0x03E: {{7, SubstX}, {11, SubstA}, {19, SubstKeep}},
	0x04D: {{9, SubstX}, {13, SubstA}, {21, SubstKeep}},
	0x05E: {{9, SubstX}, {13, SubstA}, {21, SubstKeep}},
	0x061: {{11, SubstX}, {15, SubstA}, {19, SubstKeep}},
	0x065: {{9, SubstX}, {13, SubstA}, {21, SubstKeep}},
	0x06C: {{9, SubstX}, {13, SubstA}, {21, SubstKeep}, {1043, SubstKeep}, {1044, SubstKeep}},
	0x06D: {{9, SubstX}, {13, SubstA}, {21, SubstKeep}}, // item blocks
	0x07F: {{9, SubstX}, {13, SubstA}, {21, SubstKeep}},
	0x0DB: {{9, SubstX}, {13, SubstA}, {21, SubstKeep}},
	0x0F0: {{11, SubstX}, {15, SubstA}, {23, SubstKeep}, {63, SubstKeep}, {64, SubstKeep}},
}

// SubstituteSession — применить session-подстановки к полному wire-кадру.
// Кадры без записи в substMap возвращаются как есть.
func SubstituteSession(frame []byte, s Session) []byte {
	if len(frame) < 7 {
		return frame
	}
	cmd := binary.LittleEndian.Uint16(frame[2:4])
	subs, ok := substMap[cmd]
	if !ok {
		return frame
	}
	out := make([]byte, len(frame))
	copy(out, frame)
	for _, sub := range subs {
		switch sub.Kind {
		case SubstX:
			if sub.Off+4 <= len(out) {
				binary.LittleEndian.PutUint32(out[sub.Off:sub.Off+4], s.X)
			}
		case SubstA:
			if sub.Off < len(out) {
				out[sub.Off] = s.A
			}
		case SubstACC:
			if sub.Off+2 <= len(out) {
				binary.LittleEndian.PutUint16(out[sub.Off:sub.Off+2], s.AccID)
			}
		}
	}
	return out
}

// ApplySessionToChain — chain = конкатенация wire-кадров; применить session-
// подстановки к каждому кадру и склеить обратно. Неразбираемый хвост — как есть.
func ApplySessionToChain(chain []byte, s Session) []byte {
	out := make([]byte, 0, len(chain))
	off := 0
	for off+2 <= len(chain) {
		ln := int(binary.LittleEndian.Uint16(chain[off : off+2]))
		if ln < 7 || off+ln > len(chain) {
			return chain
		}
		out = append(out, SubstituteSession(chain[off:off+ln], s)...)
		off += ln
	}
	if off != len(chain) {
		return chain
	}
	return out
}

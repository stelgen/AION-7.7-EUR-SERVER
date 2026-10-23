// world_flow.go — эмиттеры мир-init флоу Server64↔CacheD (R5, capture 2026-10-23).
//
// КАНОН (кросс-сессийный дифф login1 vs login2, 23.10):
//   - C-кадры (Server64→CacheD): X LE @payload[0:4] = frame[7:11] (Т28-канон,
//     подтверждён во ВСЕХ C-кадрах диффом [7:11)).
//   - O-кадры (CacheD→Server64) с сессионной шапкой: X BE @frame[9:13]
//     (payload[2:6]) + a @frame[13] (payload[6], варьируется 3/4 — Т38).
//     ПЕРВОЕ BE-поле в cached wire (доказано O61: X=0x508F3700 → 50 8F 37 00).
//   - O61 23Б канон: payload[0:2]=0x00EA const, [2:6]=X BE, [6]=a,
//     [10:14]=charRef LE (0x020003EA=1002|0x02000000, Т31), [14]=динамика,
//     [18]=0x66=102 const.
//   - O1 (RP_LOAD 1149Б): X LE @payload[0:4] (Т31) + рантайм-зоны (координаты,
//     timestamps) — остаются DIFF, это ок (как accache A/B ≤ рантайм-зоны).
//   - SAME-кадры (39 cmds) — эмитятся шаблоном без подстановок.
//   - LEN-DIFF кадры (O108, O244, O245, C353, C354) — динамический контент,
//     отдельная категория (не эмитим chain'ом).
package rpc

import (
	"embed"
	"encoding/binary"
)

// worldFlowFS — chain-файлы capture 2026-10-23 (sess1, юзер 1010/чар 1002).
//
//go:embed testdata/chain_worldinit_20261023.bin testdata/chain_charlist_20261023.bin
var worldFlowFS embed.FS

// Сессионные константы sess1 (capture 2026-10-23, юзер 1010/чар 1002):
// шаблоны chain_worldinit содержат ЭТИ значения — при эмиссии подставляем текущие.
const (
	WorldFlowSess1X uint32 = 0x508F3700
	WorldFlowSess1A byte   = 4
)

// GetWorldFlowChain — полный init-флоу O-кадров (118 кадров, 105КБ) sess1.
// Возвращает ПОЛНЫЕ wire-кадры, сконкатенированные (формат = chain_rq_load_*).
func GetWorldFlowChain() []byte {
	b, err := worldFlowFS.ReadFile("testdata/chain_worldinit_20261023.bin")
	if err != nil {
		return nil
	}
	return b
}

// GetCharListChain — CharacterList (O90, 1564Б) sess1.
func GetCharListChain() []byte {
	b, err := worldFlowFS.ReadFile("testdata/chain_charlist_20261023.bin")
	if err != nil {
		return nil
	}
	return b
}

// SubstituteWorldFlow — УСЛОВНЫЕ сессионные подстановки к O-кадру.
// РОБАСТНЫЙ поиск X в шапке (frame[5:25]): два формата шапки доказаны
// кросс-сессией 23.10:
//   - X LE @payload[0:4] (frame[7:11]) — O1 (Т31) и МЕЛКИЕ O-кадры (24/47/62/91/99/128/132/139/150: diff ровно 5Б в [7:11))
//   - X BE @payload[2:6] (frame[9:13]) — O61-семейство (X=0x508F3700 → 50 8F 37 00)
// Заменяем XFrom на XTo В ТОМ ЖЕ порядке байт, где нашли.
// a @frame[13] — изолированно (Т38: 4/3). Зона поиска = первые 25Б кадра
// (за пределами — X в данных встречается, не трогаем).
func SubstituteWorldFlow(frame []byte, xFrom, xTo uint32, aFrom, aTo byte) []byte {
	out := append([]byte(nil), frame...)
	leFrom := make([]byte, 4)
	binary.LittleEndian.PutUint32(leFrom, xFrom)
	leTo := make([]byte, 4)
	binary.LittleEndian.PutUint32(leTo, xTo)
	beFrom := make([]byte, 4)
	binary.BigEndian.PutUint32(beFrom, xFrom)
	beTo := make([]byte, 4)
	binary.BigEndian.PutUint32(beTo, xTo)
	// зона поиска: frame[5:25] (после заголовка, внутри начала payload)
	searchEnd := 25
	if len(out) < searchEnd {
		searchEnd = len(out)
	}
	head := out[:searchEnd]
	if idx := indexOf(head, leFrom); idx >= 0 {
		copy(out[idx:idx+4], leTo)
	} else if idx := indexOf(head, beFrom); idx >= 0 {
		copy(out[idx:idx+4], beTo)
	}
	if len(out) > 13 && out[13] == aFrom {
		out[13] = aTo
	}
	return out
}

func indexOf(hay, needle []byte) int {
	for i := 0; i+len(needle) <= len(hay); i++ {
		match := true
		for j := range needle {
			if hay[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// SplitWorldFlowPreload — разбивает chain на (preload, load):
// preload = кадры ДО первого O1 (RP_LOAD) — хронологически O196 ack + O90
// CharacterList (ответы на session 288/175); load = от O1 до конца.
func SplitWorldFlowPreload(blob []byte) (preload, load [][]byte) {
	frames := SplitWorldFlowFrames(blob)
	for i, f := range frames {
		if binary.LittleEndian.Uint16(f[2:4]) == 1 {
			return frames[:i], frames[i:]
		}
	}
	return frames, nil
}

// EmitWorldFlowPreload — сессионно-подставленный preload (O196+O90).
func EmitWorldFlowPreload(x uint32, a byte) []byte {
	preload, _ := SplitWorldFlowPreload(GetWorldFlowChain())
	var out []byte
	for _, f := range preload {
		out = append(out, SubstituteWorldFlow(f, WorldFlowSess1X, x, WorldFlowSess1A, a)...)
	}
	return out
}

// EmitWorldFlowLoad — сессионно-подставленный load-сегмент (O1..O109×7).
func EmitWorldFlowLoad(x uint32, a byte) []byte {
	_, load := SplitWorldFlowPreload(GetWorldFlowChain())
	var out []byte
	for _, f := range load {
		out = append(out, SubstituteWorldFlow(f, WorldFlowSess1X, x, WorldFlowSess1A, a)...)
	}
	return out
}

// SplitWorldFlowFrames — разбивает chain-блоб на отдельные wire-кадры
// по len-полю (каждый кадр: [u16 total][cmd u16][0xAF][~cmd u16][payload]).
func SplitWorldFlowFrames(blob []byte) [][]byte {
	var out [][]byte
	for i := 0; i+2 <= len(blob); {
		total := int(binary.LittleEndian.Uint16(blob[i : i+2]))
		if total < 7 || i+total > len(blob) {
			break // битый хвост — останавливаемся
		}
		out = append(out, blob[i:i+total])
		i += total
	}
	return out
}

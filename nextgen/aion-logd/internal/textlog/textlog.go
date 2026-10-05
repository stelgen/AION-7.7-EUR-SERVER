// Package textlog — разбор type-9 (текст-логи/онлайн-таблица NC-logd). Л1, 05.10.2026.
//
// Layout подтверждён живыми сэмплами (mirror-эталон 14 шт + боевой payload 05.10):
//
//	body = [u32 id][ {u32 key][wchar name NUL]... } [tail...] [SYSTEMTIME 8 WORD]
//	SYSTEMTIME = ВСЕГДА последние 16 байт пакета (проверено: 223b и 251b варианты).
//	id = 928 во всех сэмплах. entries = онлайн-сессии: (char_id,"Имя")(uid,"Аккаунт").
//	Пример: (1002,"SteLGeN")(1010,"Stelgen"); в 251b-варианте 2 сессии (4 записи).
//	Хвост после entries не декодирован до конца: там u32-подобные поля (в т.ч.
//	world-id 210010000/210040000), qword-тик, floats-координаты, блоки
//	0xFFFF8000/0x0000FFFF — отдаём как hex + в ship.raw для будущей раскладки (Л2-сосед).
package textlog

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// Entry — одна запись онлайн-списка (key = char_id/uid, name = имя).
type Entry struct {
	Key  uint32
	Name string
}

// Record — разобранная type-9 запись.
type Record struct {
	ID      uint32  // @0, в сэмплах 928
	Entries []Entry // пары [key][имя] до конца «прозрачной» зоны
	World   uint32  // мир-подобный u32 в хвосте (210010000/210040000), 0 = не найден
	Stamp   string  // SYSTEMTIME из последних 16 байт: "2006-01-02 15:04:05.000"
	TailHex string  // хвост после entries без stamps, hex (кап 400 символов)
	BodyLen int
}

// Parse — тело type-9 → Record. Ошибка только на откровенно коротком теле.
func Parse(body []byte) (*Record, error) {
	if len(body) < 8 {
		return nil, fmt.Errorf("textlog: body=%d, нужно ≥8", len(body))
	}
	r := &Record{ID: binary.LittleEndian.Uint32(body[0:4]), BodyLen: len(body)}
	off := 4
	for off+8 <= len(body) { // key u32 + имя ≥1 символ+NUL (4 байта)
		key := binary.LittleEndian.Uint32(body[off : off+4])
		name, next, ok := nameAt(body, off+4)
		if !ok {
			break
		}
		r.Entries = append(r.Entries, Entry{Key: key, Name: name})
		off = next
	}
	tail := body[off:]
	if len(tail) >= 16 {
		r.Stamp = stampString(tail[len(tail)-16:])
		core := tail[:len(tail)-16]
		r.TailHex = capHex(fmt.Sprintf("% X", core), 400)
	} else {
		r.TailHex = capHex(fmt.Sprintf("% X", tail), 400)
	}
	for i := 0; i+4 <= len(tail)-16 && i < 64; i += 4 {
		v := binary.LittleEndian.Uint32(tail[i : i+4])
		if v == 210010000 || v == 210040000 {
			r.World = v
			break
		}
	}
	return r, nil
}

// String — человекочитаемая строка для per-service .err/.log и ship.
func (r *Record) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "tlog id=%d n=%d", r.ID, len(r.Entries))
	for _, e := range r.Entries {
		fmt.Fprintf(&sb, " [%d:%q]", e.Key, e.Name)
	}
	if r.World != 0 {
		fmt.Fprintf(&sb, " world=%d", r.World)
	}
	if r.Stamp != "" {
		fmt.Fprintf(&sb, " time=%s", r.Stamp)
	}
	if len(r.Entries) == 0 && r.TailHex != "" {
		fmt.Fprintf(&sb, " tail=%s", r.TailHex)
	}
	return sb.String()
}

// nameAt — UTF-16LE строка с off до NUL. ok=false, если пустая/невалидные символы/
// не завершена NUL в разумном лимите — защита от «заглатывания» хвоста.
func nameAt(b []byte, off int) (string, int, bool) {
	var sb strings.Builder
	o, n := off, 0
	for o+2 <= len(b) && n < 64 {
		w := binary.LittleEndian.Uint16(b[o:])
		o += 2
		n++
		if w == 0 {
			if sb.Len() == 0 {
				return "", 0, false
			}
			return sb.String(), o, true
		}
		if !validRune(w) {
			return "", 0, false
		}
		sb.WriteRune(rune(w))
	}
	return "", 0, false
}

// validRune — печатные алфавиты, которые реально ходят в NC-логах:
// ASCII, кириллица, хангыль, CJK-пунктуация/кана. Всё остальное = не имя.
func validRune(w uint16) bool {
	switch {
	case w >= 0x20 && w <= 0x7E,
		w >= 0x400 && w <= 0x4FF,
		w >= 0xAC00 && w <= 0xD7A3,
		w >= 0x3000 && w <= 0x30FF:
		return true
	}
	return false
}

// stampString — 8 WORD → "YYYY-MM-DD HH:MM:SS.mmm" с санити-чеком; "" = не SYSTEMTIME.
func stampString(st []byte) string {
	w := make([]uint16, 8)
	for i := range w {
		w[i] = binary.LittleEndian.Uint16(st[i*2:])
	}
	if w[0] < 2020 || w[0] > 2035 || w[1] < 1 || w[1] > 12 || w[2] > 7 ||
		w[3] < 1 || w[3] > 31 || w[4] > 23 || w[5] > 59 || w[6] > 60 || w[7] > 999 {
		return ""
	}
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d.%03d",
		w[0], w[1], w[3], w[4], w[5], w[6], w[7])
}

func capHex(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
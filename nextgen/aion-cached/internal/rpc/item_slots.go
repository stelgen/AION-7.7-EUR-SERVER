// RP cmd=109 item slots — полный парсинг и генерация (Т33/Т35).
// Структура: [0:32] header → [32:300] item 0 (268Б) → [ffff + 468Б slot] × N → [ffff + partial].
// Пустые slot'ы = нули. Заполненные содержат Aion item template IDs.
package rpc

import (
	"encoding/binary"
	"fmt"
)

// ItemSlot — один item-slot из cmd=109 блока.
type ItemSlot struct {
	Index    int    // порядковый номер слота (0 = первый)
	Start    int    // offset в payload
	End      int    // конец слота
	Data     []byte // данные слота (включая [ffff] маркер если есть)
	HasFFFF  bool   // слот начинается с [ffff]
	ItemID   uint32 // первый найденный Aion item ID (0 = пустой)
	IsEmpty  bool   // все данные после [ffff] = нули
}

// ParseItemSlots — разбирает cmd=109 payload на слоты.
// Первый слот: [32:первый ffff] (header + item data).
// Остальные: [ffff:следующий ffff] (470Б, могут быть нулями).
func ParseItemSlots(payload []byte) []ItemSlot {
	var slots []ItemSlot
	var ffffPos []int
	for i := 28; i < len(payload)-2; i++ {
		if payload[i] == 0xFF && payload[i+1] == 0xFF && (i == 0 || payload[i-1] != 0xFF) {
			ffffPos = append(ffffPos, i)
		}
	}
	if len(ffffPos) == 0 {
		// нет [ffff] — весь payload один слот
		slots = append(slots, ItemSlot{Index: 0, Start: 0, End: len(payload), Data: payload, IsEmpty: isAllZero(payload[32:])})
		return slots
	}
	// слот 0: [0:первый ffff] — header + первый item
	slot0 := ItemSlot{Index: 0, Start: 0, End: ffffPos[0], Data: payload[:ffffPos[0]]}
	slot0.IsEmpty = isAllZero(payload[32:ffffPos[0]])
	slot0.ItemID = findItemId(payload[32:ffffPos[0]])
	slots = append(slots, slot0)
	// слоты 1..N: [ffff:следующий ffff]
	for j := 0; j < len(ffffPos); j++ {
		start := ffffPos[j]
		end := len(payload)
		if j+1 < len(ffffPos) {
			end = ffffPos[j+1]
		}
		data := payload[start:end]
		slot := ItemSlot{
			Index:   j + 1,
			Start:   start,
			End:     end,
			Data:    data,
			HasFFFF: true,
		}
		// данные после [ffff] (2Б)
		body := data[2:]
		slot.IsEmpty = isAllZero(body)
		slot.ItemID = findItemId(body)
		slots = append(slots, slot)
	}
	return slots
}

// isAllZero — все байты нули (кроме последних 2Б которые могут быть маркером).
func isAllZero(data []byte) bool {
	for i, b := range data {
		if b != 0 && i < len(data)-2 {
			return false
		}
	}
	return true
}

// findItemId — ищет первый Aion item template ID (100000000-199999999).
func findItemId(data []byte) uint32 {
	for off := 0; off < len(data)-4; off += 4 {
		v := binary.LittleEndian.Uint32(data[off : off+4])
		if v >= 100000000 && v <= 199999999 {
			return v
		}
	}
	return 0
}

// GenerateItemSlots — генерирует cmd=109 payload из слотов.
// Использует captured-шаблон + подстановку заполненных слотов.
func GenerateItemSlots(template []byte, slots []ItemSlot) []byte {
	if len(template) == 0 {
		return nil
	}
	out := make([]byte, len(template))
	copy(out, template)
	// перезаписать данные заполненных слотов
	for _, slot := range slots {
		if slot.IsEmpty || slot.Start >= len(out) {
			continue
		}
		end := slot.End
		if end > len(out) {
			end = len(out)
		}
		copy(out[slot.Start:end], slot.Data[:min(end-slot.Start, len(slot.Data))])
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// DescribeSlots — описание слотов для лога.
func DescribeSlots(slots []ItemSlot) string {
	filled, empty := 0, 0
	var itemIds []uint32
	for _, s := range slots {
		if s.IsEmpty {
			empty++
		} else {
			filled++
			if s.ItemID > 0 {
				itemIds = append(itemIds, s.ItemID)
			}
		}
	}
	result := fmt.Sprintf("slots=%d filled=%d empty=%d", len(slots), filled, empty)
	if len(itemIds) > 0 {
		result += fmt.Sprintf(" itemIds=%v", itemIds[:minInt(5, len(itemIds))])
	}
	return result
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Package rpc — ItemBlock parser/generator (Т33: cmd=109, 8021Б, слоты 470Б).
// Структура блока: header(~30Б: X+a+charRef+cat+flag+count) + 17 слотов × 470Б
// (каждый с [ffff] префиксом) + footer(194Б). Слот = item-запись или пустой.
package rpc

import (
	"encoding/binary"
	"fmt"
)

// ItemBlock — разобранный cmd=109 блок.
type ItemBlock struct {
	Header   []byte   // первые ~30Б (X+a+charRef+cat+flag+count)
	Slots    [][]byte // 17 слотов × 470Б (каждый начинается с [ffff])
	Footer   []byte   // оставшиеся байты
	Category uint32   // категория (3 из capture)
	Flag     uint32   // флаг (17 из capture)
	Count    uint32   // счётчик (197 из capture)
	Total    int      // полный размер payload
}

// ItemSlotSize — размер одного слота (470Б вкл. [ffff] маркер).
const ItemSlotSize = 470

// ParseItemBlock — разбирает cmd=109 payload на структуру ItemBlock.
// payload = frame[7:] (после wire-заголовка).
func ParseItemBlock(payload []byte) *ItemBlock {
	ib := &ItemBlock{Total: len(payload)}
	if len(payload) < 30 {
		return ib
	}
	// header: первые 30Б
	ib.Header = make([]byte, 30)
	copy(ib.Header, payload[:30])
	// поля из header
	ib.Category = binary.LittleEndian.Uint32(payload[14:18])
	ib.Flag = binary.LittleEndian.Uint32(payload[18:22])
	ib.Count = binary.LittleEndian.Uint32(payload[26:30])

	// найти все [ffff] маркеры (после header)
	var ffff []int
	for i := 28; i < len(payload)-1; i++ {
		if payload[i] == 0xFF && payload[i+1] == 0xFF {
			ffff = append(ffff, i)
		}
	}
	if len(ffff) == 0 {
		return ib
	}

	// header ends at first ffff
	ib.Header = make([]byte, ffff[0])
	copy(ib.Header, payload[:ffff[0]])
	// reparse fields from full header
	if len(ib.Header) >= 22 {
		ib.Category = binary.LittleEndian.Uint32(ib.Header[14:18])
		ib.Flag = binary.LittleEndian.Uint32(ib.Header[18:22])
	}
	if len(ib.Header) >= 30 {
		ib.Count = binary.LittleEndian.Uint32(ib.Header[26:30])
	}

	// slots: от каждого [ffff] до следующего
	for i, fp := range ffff {
		end := len(payload)
		if i+1 < len(ffff) {
			end = ffff[i+1]
		}
		slot := make([]byte, end-fp)
		copy(slot, payload[fp:end])
		ib.Slots = append(ib.Slots, slot)
	}

	// footer: после последнего полного слота
	lastEnd := ffff[len(ffff)-1] + ItemSlotSize
	if lastEnd < len(payload) {
		ib.Footer = make([]byte, len(payload)-lastEnd)
		copy(ib.Footer, payload[lastEnd:])
	}

	return ib
}

// CountNonEmptySlots — слоты с данными (не только [ffff] + нули).
func (ib *ItemBlock) CountNonEmptySlots() int {
	count := 0
	for _, slot := range ib.Slots {
		nonz := 0
		for _, b := range slot[2:] { // пропустить [ffff]
			if b != 0 {
				nonz++
			}
		}
		if nonz > 4 {
			count++
		}
	}
	return count
}

// BuildItemBlock — собирает cmd=109 payload из header + слотов + footer.
func BuildItemBlock(header []byte, slots [][]byte, footer []byte) []byte {
	out := make([]byte, 0, len(header)+len(slots)*ItemSlotSize+len(footer))
	out = append(out, header...)
	for _, slot := range slots {
		out = append(out, slot...)
	}
	out = append(out, footer...)
	return out
}

// Summary — человекочитаемая сводка блока.
func (ib *ItemBlock) Summary() string {
	return fmt.Sprintf("ItemBlock: total=%d header=%d slots=%d(nonEmpty=%d) footer=%d cat=%d flag=%d count=%d",
		ib.Total, len(ib.Header), len(ib.Slots), ib.CountNonEmptySlots(), len(ib.Footer),
		ib.Category, ib.Flag, ib.Count)
}

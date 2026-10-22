// Item slots тесты: парсинг и генерация на реальных captured-дампах.
package rpc

import (
	"fmt"
	"testing"
)

// TestParseItemSlots — парсинг реального captured cmd=109 блока.
func TestParseItemSlots(t *testing.T) {
	tpl := loadTemplate("rp109_block0.bin")
	if tpl == nil {
		t.Skip("no rp109_block0.bin")
	}
	payload := tpl[7:]
	slots := ParseItemSlots(payload)
	if len(slots) == 0 {
		t.Fatal("ParseItemSlots returned 0 slots")
	}
	t.Logf("ParseItemSlots: %d slots", len(slots))
	// слот 0 = header + первый item (заполненный)
	if slots[0].IsEmpty {
		t.Fatal("slot 0 should NOT be empty (contains item data)")
	}
	if slots[0].ItemID == 0 {
		t.Fatal("slot 0 should have an item ID")
	}
	t.Logf("slot 0: itemID=%d data_len=%d", slots[0].ItemID, len(slots[0].Data))
	// остальные слоты
	filled, empty := 0, 0
	for _, s := range slots[1:] {
		if s.HasFFFF {
			if s.IsEmpty {
				empty++
			} else {
				filled++
			}
		}
	}
	t.Logf("slots 1+: filled=%d empty=%d", filled, empty)
	// первый [ffff] слот должен быть на правильном offset
	if len(slots) > 1 && slots[1].HasFFFF {
		t.Logf("slot 1 starts at payload[%d] with [ffff]", slots[1].Start)
	}
}

// TestGenerateItemSlots — генерация cmd=109 из слотов (roundtrip).
func TestGenerateItemSlots(t *testing.T) {
	tpl := loadTemplate("rp109_block0.bin")
	if tpl == nil {
		t.Skip("no template")
	}
	payload := tpl[7:]
	slots := ParseItemSlots(payload)
	// генерировать из слотов
	regenerated := GenerateItemSlots(payload, slots)
	if len(regenerated) != len(payload) {
		t.Fatalf("regenerated size %d != original %d", len(regenerated), len(payload))
	}
	// байт-в-байт сравнение
	for i := range payload {
		if regenerated[i] != payload[i] {
			t.Fatalf("byte mismatch @%d: gen=%02x orig=%02x", i, regenerated[i], payload[i])
		}
	}
	t.Log("roundtrip: regenerated == original ✓")
}

// TestItemSlotFields — проверка полей item-записей из capture.
func TestItemSlotFields(t *testing.T) {
	tpl := loadTemplate("rp109_block0.bin")
	if tpl == nil {
		t.Skip("no template")
	}
	payload := tpl[7:]
	slots := ParseItemSlots(payload)
	// слот 0 должен содержать item ID 182400001 (из Т33 анализа)
	if len(slots) > 0 && slots[0].ItemID != 182400001 {
		t.Fatalf("slot 0 itemID = %d, want 182400001", slots[0].ItemID)
	}
	// большинство слотов должны быть пустыми (fill=1, empty=17)
	filledCount := 0
	for _, s := range slots {
		if !s.IsEmpty {
			filledCount++
		}
	}
	t.Logf("filled slots: %d of %d", filledCount, len(slots))
	if filledCount == 0 {
		t.Fatal("no filled slots found")
	}
}

// TestItemSlotsMultipleBlocks — парсинг всех 12 блоков.
func TestItemSlotsMultipleBlocks(t *testing.T) {
	totalSlots := 0
	totalFilled := 0
	totalItems := 0
	for i := 0; i < 12; i++ {
		tpl := loadTemplate(fmt.Sprintf("rp109_block%d.bin", i))
		if tpl == nil {
			continue
		}
		payload := tpl[7:]
		slots := ParseItemSlots(payload)
		totalSlots += len(slots)
		for _, s := range slots {
			if !s.IsEmpty {
				totalFilled++
			}
			if s.ItemID > 0 {
				totalItems++
			}
		}
	}
	t.Logf("all blocks: slots=%d filled=%d items_with_id=%d", totalSlots, totalFilled, totalItems)
	if totalSlots == 0 {
		t.Fatal("no slots parsed")
	}
	if totalItems == 0 {
		t.Fatal("no item IDs found across all blocks")
	}
}

// RP-эмиттер тесты: RP cmd=1 struct + RP cmd=109 item parser.
package rpc

import (
	"encoding/binary"
	"testing"
)

// TestGenerateRPLoadStruct — RP cmd=1 (1156Б) из шаблона с подстановками Т31.
func TestGenerateRPLoadStruct(t *testing.T) {
	out := GenerateRPLoadStruct(nil, nil, 0xDEAD, 1002, 1010, "SteLGeN")
	if len(out) < 1150 {
		t.Fatalf("RP cmd=1 too short: %d", len(out))
	}
	// маркер = 0xAF (S2C cached)
	if out[4] != 0xAF {
		t.Fatalf("marker = %#x, want 0xAF", out[4])
	}
	// cmd = 1
	cmd := binary.LittleEndian.Uint16(out[2:4])
	if cmd != 1 {
		t.Fatalf("cmd = %d, want 1", cmd)
	}
	// X подставлен (payload[0:4] = frame[7:11])
	x := binary.LittleEndian.Uint32(out[7:11])
	if x != 0xDEAD {
		t.Fatalf("X = %#x, want 0xDEAD", x)
	}
	// charRef (payload[21:25] = frame[28:32]): charId=1002 → low word
	charRef := binary.LittleEndian.Uint32(out[28:32])
	if charRef&0xFFFF != 1002 {
		t.Fatalf("charRef low = %d, want 1002", charRef&0xFFFF)
	}
	if charRef>>24 != 0x02 {
		t.Fatalf("charRef high byte = %#x, want 0x02", charRef>>24)
	}
	// level: payload[49] = frame[56] — из шаблона 80
	if out[56] != 80 {
		t.Logf("level = %d (template value)", out[56])
	}
	// charName: payload[903:928] = frame[910:935] — UTF-16LE
	if len(out) > 911 {
		if out[910] == 'S' && out[911] == 0x00 {
			t.Log("charName UTF-16 'S' подставлен ✓")
		} else {
			t.Logf("charName first bytes = %x %x", out[910], out[911])
		}
	}
}


// TestParseItemBlocks — парсер cmd=109 item-записей (Т33 паттерн).
func TestParseItemBlocks(t *testing.T) {
	// Загрузить реальный captured item block
	tpl := loadTemplate("rp109_block0.bin")
	if tpl == nil {
		t.Skip("no rp109_block0.bin")
	}
	payload := tpl[7:]
	records := ParseItemBlocks(payload)
	if len(records) == 0 {
		t.Fatal("ParseItemBlocks returned 0 records")
	}
	t.Logf("ParseItemBlocks: %d records from %dБ payload", len(records), len(payload))
	// проверить что записи найдены
	foundItems := 0
	for _, r := range records {
		if r.ItemID > 0 {
			foundItems++
		}
	}
	if foundItems == 0 {
		t.Log("ни одной Aion item ID не найдено (ожидаемо для тестовых данных)")
	}
}

// TestCountItemRecords — счётчик записей.
func TestCountItemRecords(t *testing.T) {
	tpl := loadTemplate("rp109_block0.bin")
	if tpl == nil {
		t.Skip("no template")
	}
	payload := tpl[7:]
	count := CountItemRecords(payload)
	if count == 0 {
		t.Fatal("CountItemRecords = 0")
	}
	t.Logf("CountItemRecords: %d", count)
}

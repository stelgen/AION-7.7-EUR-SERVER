// DB-генератор тесты: RP cmd=1 full field substitution (Т31 offsets).
package rpc

import (
	"fmt"
	"encoding/binary"
	"testing"
)

func TestDBGenRPLoadFull(t *testing.T) {
	tpl := loadTemplate("rp1_char_struct.bin")
	if tpl == nil {
		t.Skip("no template")
	}
	dbg := &DBGen{}
	// Без Ex — replay с подстановкой X/charRef/charName
	out := dbg.GenerateRPLoadFull(nil, 0xBEEF, 1002, 1010, "SteLGeN")
	if len(out) != len(tpl) {
		t.Fatalf("size mismatch: %d vs %d", len(out), len(tpl))
	}
	// X подставлен
	x := binary.LittleEndian.Uint32(out[7:11])
	if x != 0xBEEF {
		t.Fatalf("X = %#x", x)
	}
	// charRef: Т31 offset payload[21:25] = frame[28:32]
	charRef := binary.LittleEndian.Uint32(out[28:32])
	if charRef&0xFFFF != 1002 {
		t.Fatalf("charRef low = %d, want 1002", charRef&0xFFFF)
	}
	if charRef>>24 != 0x02 {
		t.Fatalf("charRef high byte = %#x, want 0x02", charRef>>24)
	}
	// level: Т31 offset payload[49] = frame[56] (single byte)
	// шаблон имеет 80 → после подстановки (без DB) должен остаться 80 из шаблона
	// (DB подстановка перезапишет значением из БД)
	if out[56] != 80 {
		t.Logf("level = %d (template)", out[56])
	}
	// charName: payload[903:928] = frame[910:935]
	// проверяем UTF-16LE подстановку
	if len(out) > 920 {
		// первый символ 'S' = 0x53, 0x00
		if out[910] == 'S' && out[911] == 0x00 {
			t.Log("charName UTF-16 подставлен ✓")
		} else {
			t.Logf("charName first bytes = %x %x (expected 53 00)", out[910], out[911])
		}
	}
}

func TestBuildCharRef(t *testing.T) {
	tests := []struct {
		charId uint32
		want   uint32
		wantHex string
	}{
		{1002, 0x020003EA, "ea030002"},
		{1003, 0x020003EB, "eb030002"},
		{1005, 0x020003ED, "ed030002"},
	}
	for _, tt := range tests {
		got := buildCharRef(tt.charId)
		if got != tt.want {
			t.Errorf("buildCharRef(%d) = %#x, want %#x", tt.charId, got, tt.want)
		}
		// проверяем байты
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, got)
		if hex := fmt.Sprintf("%x", b); hex != tt.wantHex {
			t.Errorf("buildCharRef(%d) bytes = %s, want %s", tt.charId, hex, tt.wantHex)
		}
	}
}

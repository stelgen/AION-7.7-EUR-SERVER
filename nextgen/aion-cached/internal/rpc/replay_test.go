// R4 replay-тесты: RP-эмиттеры возвращают captured-шаблоны с X-подстановкой.
package rpc

import (
	"encoding/binary"
	"testing"
)

func TestReplayTemplates(t *testing.T) {
	rp1 := GetRPLoad(0xDEADBEEF)
	if rp1 == nil {
		t.Fatal("RP cmd=1 template not found")
	}
	if len(rp1) < 1000 {
		t.Fatalf("RP cmd=1 too short: %d", len(rp1))
	}
	// маркер S2C = 0xAF
	if rp1[4] != 0xAF {
		t.Fatalf("RP cmd=1 marker = %#x, want 0xAF", rp1[4])
	}
	// X подставлен
	x := binary.LittleEndian.Uint32(rp1[7:11])
	if x != 0xDEADBEEF {
		t.Fatalf("RP cmd=1 X = %#x, want 0xDEADBEEF", x)
	}
	// cmd=1
	cmd := binary.LittleEndian.Uint16(rp1[2:4])
	if cmd != 1 {
		t.Fatalf("RP cmd = %d, want 1", cmd)
	}
}

func TestReplayCharList(t *testing.T) {
	rp := GetRPCharList(0xCAFEBABE)
	if rp == nil {
		t.Fatal("RP cmd=90 template not found")
	}
	x := binary.LittleEndian.Uint32(rp[7:11])
	if x != 0xCAFEBABE {
		t.Fatalf("CharList X = %#x", x)
	}
}

func TestReplayItemPages(t *testing.T) {
	pages := GetRPLoadItemPages(0x11111111)
	if len(pages) == 0 {
		t.Fatal("no item pages")
	}
	for i, pg := range pages {
		x := binary.LittleEndian.Uint32(pg[7:11])
		if x != 0x11111111 {
			t.Fatalf("page %d X = %#x", i, x)
		}
		// marker = 0xAF (S2C)
		if pg[4] != 0xAF {
			t.Fatalf("page %d marker = %#x", i, pg[4])
		}
	}
}

func TestReplayItemBlocks(t *testing.T) {
	blocks := GetRPItemBlocks(0x22222222)
	if len(blocks) == 0 {
		t.Fatal("no item blocks")
	}
	for i, blk := range blocks {
		x := binary.LittleEndian.Uint32(blk[7:11])
		if x != 0x22222222 {
			t.Fatalf("block %d X = %#x", i, x)
		}
	}
}

func TestEmitReadPath(t *testing.T) {
	sc := SessionContext{X: 0xAABBCCDD, AccID: 1010, CharID: 1002, CharRef: 0x020003EA}
	frames := EmitReadPath(sc)
	if len(frames) < 10 {
		t.Fatalf("EmitReadPath returned %d frames, want >= 10", len(frames))
	}
	// все кадры маркер 0xAF
	for i, f := range frames {
		if f[4] != 0xAF {
			t.Fatalf("frame %d marker = %#x, want 0xAF", i, f[4])
		}
	}
	// первый = CharacterList (cmd=90)
	cl := binary.LittleEndian.Uint16(frames[0][2:4])
	if cl != 90 {
		t.Fatalf("first frame cmd = %d, want 90 (CharacterList)", cl)
	}
}

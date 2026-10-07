package wire

import (
	"encoding/binary"
	"testing"
)

// TestSplit — канон: [u16 size LE self-inclusive][body]; реальная последовательность из capture.
func TestSplit(t *testing.T) {
	b1 := Frame([]byte{1, 2, 3, 4, 5}) // тело 5 (мин S2C-заголовок), фрейм 7
	b2 := Frame(make([]byte, 20))      // фрейм 22
	stream := append(append([]byte{}, b1...), b2...)
	bodies, tail := Split(stream)
	if len(bodies) != 2 || tail != 0 {
		t.Fatalf("split: %d bodies, tail=%d", len(bodies), tail)
	}
	if len(bodies[0]) != 5 || len(bodies[1]) != 20 {
		t.Fatalf("body sizes: %d %d", len(bodies[0]), len(bodies[1]))
	}
}

// TestSplitTail — обрезанный хвост остаётся в tail.
func TestSplitTail(t *testing.T) {
	stream := Frame([]byte{1, 2, 3})[:4]
	_, tail := Split(stream)
	if tail != 4 {
		t.Fatalf("tail: %d", tail)
	}
}

// TestBuildServerFrame — S2C: SM_KEY 0x48 -> E=0x01F9, code=0x56, ~E ok (канон capture).
func TestBuildServerFrame(t *testing.T) {
	f := BuildServerFrame(0x48, []byte{0xAA, 0xBB, 0xCC, 0xDD})
	if binary.LittleEndian.Uint16(f[0:2]) != 11 {
		t.Fatalf("size: %d", binary.LittleEndian.Uint16(f[0:2]))
	}
	if f[2] != 0xF9 || f[3] != 0x01 || f[4] != 0x56 {
		t.Fatalf("header: %X", f[2:5])
	}
}

// TestParseClientBody — [op][0x75][~op] валидация.
func TestParseClientBody(t *testing.T) {
	dec := []byte{0xD6, 0x00, 0x75, 0x29, 0xFF, 0x01, 0x02}
	op, payload, ok := ParseClientBody(dec)
	if !ok || op != 0x00D6 || payload[0] != 0x01 {
		t.Fatalf("parse: op=%04X ok=%v", op, ok)
	}
	// битый ~op
	dec[3] = 0x00
	if _, _, ok := ParseClientBody(dec); ok {
		t.Fatalf("invalid ~op accepted")
	}
}

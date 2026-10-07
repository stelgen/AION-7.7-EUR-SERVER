package proto

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

// Голдены из ЖИВОГО capture (accountcache-ref/capture-20261007, логин 13:35).
// len-поле = ПОЛНАЯ длина кадра (включая len-поле), тело = len-2.

func TestGoldenFromCapture(t *testing.T) {
	cases := []struct {
		name string
		wire string // hex полного кадра с провода
		cmd  uint16
		pay  string
	}{
		{"VERSION C2S", "0f000100ebfeff0300000001000000", 0x01, "0300000001000000"},
		{"FIRST_LOAD C2S", "0b000400ebfbfff2030000", 0x04, "f2030000"},
		{"LOAD_BM_PACK C2S", "0b000500ebfafff2030000", 0x05, "f2030000"},
		{"ASK_JUMPING_CHAR C2S", "0e001f00ebe0ff01f20300000001", 0x1F, "01f20300000001"},
		{"LOAD_TRIAL C2S", "0f000700ebf8fff2030000ea030002", 0x07, "f2030000ea030002"},
		{"LOAD_LUNA C2S", "0b001a00ebe5fff2030000", 0x1A, "f2030000"},
	}
	for _, c := range cases {
		wire := mustHex(t, c.wire)
		rd := NewReader(bytes.NewReader(wire))
		f, err := rd.ReadFrame()
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if f.Cmd != c.cmd {
			t.Fatalf("%s: cmd=%#x want %#x", c.name, f.Cmd, c.cmd)
		}
		if got := hexStr(f.Payload); got != c.pay {
			t.Fatalf("%s: payload=%s want %s", c.name, got, c.pay)
		}
	}
}

func TestGoldenS2CMarkerEC(t *testing.T) {
	// Ответы ACS инкрементят маркер: 0xEC (S2C). Кадры из capture O>.
	cases := []struct {
		wire string
		cmd  uint16
	}{
		{"0b000100ecfeff03000000", 0x01},                                       // VERSION resp
		{"1a000400ecfbfff2030000010000000000000000ca1fc66a0000", 0x04},          // FIRST_LOAD resp
		{"0d000500ecfafff203000000001500", 0x05},                                // LOAD_BM_PACK resp
	}
	for _, c := range cases {
		wire := mustHex(t, c.wire)
		f, err := ParseFrame(wire)
		if err != nil {
			t.Fatalf("S2C %#x: %v", c.cmd, err)
		}
		if f.Cmd != c.cmd {
			t.Fatalf("cmd=%#x want %#x", f.Cmd, c.cmd)
		}
		if f.Direction != DirS2C {
			t.Fatalf("want DirS2C")
		}
	}
}

func TestBuildMatchesCapture(t *testing.T) {
	// Build обязан давать байт-в-байт тот же кадр, что шлёт Server64.
	want := mustHex(t, "0b000400ebfbfff2030000") // FIRST_LOAD C2S из capture
	got, err := Build(0x04, mustHex(t, "f2030000"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("Build: got %x want %x", got, want)
	}
	if l := binary.LittleEndian.Uint16(got[0:2]); l != uint16(len(got)) {
		t.Fatalf("len field %d != wire len %d", l, len(got))
	}
}

func TestBuildGolden(t *testing.T) {
	payload := make([]byte, 20)
	wire, err := Build(0x04, payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(wire) != 27 {
		t.Fatalf("wire len = %d, want 27", len(wire))
	}
	if got := binary.LittleEndian.Uint16(wire[0:2]); got != 27 {
		t.Fatalf("len field = %d, want 27 (= полный размер)", got)
	}
	if got := binary.LittleEndian.Uint16(wire[2:4]); got != 0x04 {
		t.Fatalf("cmd = %#x", got)
	}
	if wire[4] != Marker {
		t.Fatalf("marker = %#x", wire[4])
	}
	if got := binary.LittleEndian.Uint16(wire[5:7]); got != ^uint16(0x04) {
		t.Fatalf("inv = %#x", got)
	}
}

func TestRoundtrip(t *testing.T) {
	cases := []struct {
		cmd uint16
		pl  []byte
	}{
		{0x01, []byte{0x01, 0x02, 0x03, 0x04}},
		{0x04, make([]byte, 20)},
		{0x27, []byte{0xAA}},
	}
	for _, c := range cases {
		wire, err := Build(c.cmd, c.pl)
		if err != nil {
			t.Fatal(err)
		}
		rd := NewReader(bytes.NewReader(wire))
		f, err := rd.ReadFrame()
		if err != nil {
			t.Fatalf("cmd %#x: %v", c.cmd, err)
		}
		if f.Cmd != c.cmd {
			t.Fatalf("cmd = %#x want %#x", f.Cmd, c.cmd)
		}
		if !reflect.DeepEqual(f.Payload, c.pl) {
			t.Fatalf("payload mismatch")
		}
	}
}

func TestBadInv(t *testing.T) {
	wire, _ := Build(0x04, nil)
	wire[5] ^= 0xFF // ломаем ~cmd
	rd := NewReader(bytes.NewReader(wire))
	if _, err := rd.ReadFrame(); err == nil {
		t.Fatal("want error for broken inv")
	}
}

func TestCmdRange(t *testing.T) {
	if _, err := Build(0x6C, nil); err != ErrCmdRange {
		t.Fatalf("want ErrCmdRange, got %v", err)
	}
}

func TestSplitFrame(t *testing.T) {
	wire, _ := Build(0x09, []byte{1, 2, 3, 4, 5, 6, 7, 8})
	slow := &oneByte{r: bytes.NewReader(wire)}
	rd2 := NewReader(slow)
	f, err := rd2.ReadFrame()
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Payload) != 8 {
		t.Fatalf("payload = %d", len(f.Payload))
	}
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hexDecode(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func hexStr(b []byte) string {
	const hexd = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hexd[v>>4]
		out[i*2+1] = hexd[v&0xF]
	}
	return string(out)
}

func hexDecode(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		return nil, ErrShort
	}
	out := make([]byte, len(s)/2)
	for i := 0; i < len(out); i++ {
		hi, ok1 := hexVal(s[i*2])
		lo, ok2 := hexVal(s[i*2+1])
		if !ok1 || !ok2 {
			return nil, ErrShort
		}
		out[i] = hi<<4 | lo
	}
	return out, nil
}

func hexVal(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

type oneByte struct{ r *bytes.Reader }

func (o *oneByte) Read(p []byte) (int, error) {
	if len(p) > 1 {
		p = p[:1]
	}
	return o.r.Read(p)
}

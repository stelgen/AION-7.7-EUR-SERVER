package proto

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

func TestBuildGolden(t *testing.T) {
	// cmd=0x04 FIRST_LOAD, payload = 20 байт (5xint)
	payload := make([]byte, 20)
	wire, err := Build(0x04, payload)
	if err != nil {
		t.Fatal(err)
	}
	// wire = [len=2+27=29][cmd=04 00][EB][~cmd=FB FB][20x0] = 29
	if len(wire) != 27 {
		t.Fatalf("wire len = %d, want 27", len(wire))
	}
	wantLen := uint16(25)
	if got := binary.LittleEndian.Uint16(wire[0:2]); got != wantLen {
		t.Fatalf("len field = %d, want %d", got, wantLen)
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

func TestBadMarker(t *testing.T) {
	wire, _ := Build(0x04, nil)
	wire[4] = 0x00 // ломаем маркер
	_, err := Validate(wire[2:])
	if err == nil || err != ErrShort && err != ErrMarker {
		// Validate получает buf без len-поля; маркер в body[2]
		t.Fatalf("want ErrMarker, got %v", err)
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
	// читаем по 1 байту — должен ресинкнуться
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

type oneByte struct{ r *bytes.Reader }

func (o *oneByte) Read(p []byte) (int, error) {
	if len(p) > 1 {
		p = p[:1]
	}
	return o.r.Read(p)
}

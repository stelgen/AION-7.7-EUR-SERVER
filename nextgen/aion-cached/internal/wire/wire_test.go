package wire

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

// buildFrame — сборка C2S-кадра канона: [len total][cmd u16][0xEB][~cmd][payload][len total].
func buildFrame(cmd uint16, payload []byte) []byte {
	return BuildC2S(cmd, payload)
}

// TestRoundtrip — BuildC2S → Feed → cmd/payload сохраняются.
func TestRoundtrip(t *testing.T) {
	payload := []byte("char:TestChar;skill:105;")
	raw := buildFrame(11, payload) // RQ_LOAD

	dec := NewDecoder()
	n, fr, err := dec.Feed(raw)
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	if n != len(raw) {
		t.Fatalf("consumed %d, want %d", n, len(raw))
	}
	if fr.Cmd != 11 || !bytes.Equal(fr.Payload, payload) {
		t.Fatalf("roundtrip: cmd=%d payload=%q", fr.Cmd, fr.Payload)
	}
	// инварианты: маркер, ~cmd, len-двойник
	if raw[4] != MarkerC2S {
		t.Fatalf("marker = %#x", raw[4])
	}
	total := binary.LittleEndian.Uint16(raw[0:2])
	if len(raw) != int(total) {
		t.Fatalf("len(raw)=%d != totalLen=%d", len(raw), total)
	}
}

// TestS2CMarker — BuildS2C: маркер 0xEC, ~cmd инверсия.
func TestS2CMarker(t *testing.T) {
	raw := BuildS2C(45, []byte("ping"))
	if raw[4] != MarkerS2C {
		t.Fatalf("marker = %#x, want 0xEC", raw[4])
	}
	cmd := binary.LittleEndian.Uint16(raw[2:4])
	if binary.LittleEndian.Uint16(raw[5:7]) != ^cmd {
		t.Fatal("~cmd != ^cmd")
	}
	dec := NewDecoder()
	// S2C-кадр НЕ парсится C2S-декодером (маркер 0xEB) → ErrBadMarker
	if _, _, err := dec.Feed(raw); !errors.Is(err, ErrBadMarker) {
		t.Fatalf("want ErrBadMarker, got %v", err)
	}
}

// TestGoldenAccacheFrame — golden-кадр capture (TestVersionGolden-формат):
// 0f00 0100 ebfe ff 03 00000001 00000000 — len=15, cmd=1, EB, ~cmd=0xFFFE.
func TestGoldenAccacheFrame(t *testing.T) {
	golden := []byte{
		0x0f, 0x00, // totalLen = 15
		0x01, 0x00, // cmd = 1
		0xEB,             // marker C2S
		0xfe, 0xff,       // ~cmd = 0xFFFE = ^0x0001
		0x03, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, // payload 8Б (15Б ровно, без хвоста)
	}
	dec := NewDecoder()
	n, fr, err := dec.Feed(golden)
	if err != nil {
		t.Fatalf("golden Feed: %v", err)
	}
	if n != len(golden) || fr.Cmd != 1 {
		t.Fatalf("golden: n=%d cmd=%d payload=% x", n, fr.Cmd, fr.Payload)
	}
	if !bytes.Equal(fr.Payload, golden[7:15]) {
		t.Fatalf("golden payload: % x", fr.Payload)
	}
}

// TestBadChecksumRejected — испорченное len-поле (контроль) → reject.
func TestBadChecksumRejected(t *testing.T) {
	raw := buildFrame(11, []byte("data"))
	raw[1] ^= 0xFF // len u16 старший байт
	_, _, err := NewDecoder().Feed(raw)
	if err == nil {
		t.Fatal("испорченный len-контроль должен дать ошибку")
	}
}

// TestBadMarkerRejected — маркер != 0xEB → reject.
func TestBadMarkerRejected(t *testing.T) {
	raw := buildFrame(11, []byte("data"))
	raw[4] = 0x00
	_, _, err := NewDecoder().Feed(raw)
	if !errors.Is(err, ErrBadMarker) {
		t.Fatalf("want ErrBadMarker, got %v", err)
	}
}

// TestBadSizeRejected — total < минимума → reject.
func TestBadSizeRejected(t *testing.T) {
	if _, _, err := NewDecoder().Feed([]byte{0x03, 0x00}); !errors.Is(err, ErrBadSize) {
		t.Fatalf("want ErrBadSize, got %v", err)
	}
}

// TestTruncated — обрезанный кадр → ErrShort.
func TestTruncated(t *testing.T) {
	full := buildFrame(11, []byte("payload-data"))
	for cut := 1; cut < len(full); cut += 3 {
		if _, _, err := NewDecoder().Feed(full[:cut]); !errors.Is(err, ErrShort) {
			t.Fatalf("cut=%d: want ErrShort, got %v", cut, err)
		}
	}
}

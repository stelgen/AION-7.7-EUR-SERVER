package wire

import (
	"encoding/binary"
	"errors"
	"testing"
)

// buildFrame — симуляция клиента: собирает кадр канона (plaintext — крипты НЕТ,
// R2 вердикт 21.10: DummyCrypt no-op). csum = u16 LE от (packetSize + 2) = wireLen.
func buildFrame(t *testing.T, seed uint32, opcode uint16, payload []byte) []byte {
	t.Helper()
	body := append([]byte{byte(opcode)}, payload...)
	ps := len(body)                                          // opcode + payload
	csum := uint32(ps + ChecksumSize + 2)                    // = wireLen
	body = append(body, byte(csum&0xFF), byte((csum>>8)&0xFF))

	out := make([]byte, HeaderSize+len(body))
	binary.LittleEndian.PutUint16(out, uint16(HeaderSize+len(body)))
	copy(out[HeaderSize:], body)

	// КРИПТЫ НЕТ (DummyCrypt = no-op, R2 вердикт) — тело идёт как есть.
	_ = seed // seed остаётся в API для совместимости (vestigial)
	return out
}

// TestRoundtrip — build → read → payload равен (plaintext: rolling key vestigial).
func TestRoundtrip(t *testing.T) {
	seed := uint32(0x5F3A7C21)
	payload := []byte("builder:Admin;char:TestChar;skill:105;")

	// несколько пакетов подряд — rolling key ведётся как vestigial канона
	frames := [][]byte{
		buildFrame(t, seed, 0x01, payload),
		buildFrame(t, seed, 0x02, []byte("second")),
	}

	dec := NewDecoder(seed)
	var consumed int
	var got []*Frame
	for _, f := range frames {
		n, fr, err := dec.Feed(f)
		if err != nil {
			t.Fatalf("Feed: %v", err)
		}
		consumed += n
		got = append(got, fr)
	}

	if got[0].Opcode != 0x01 || string(got[0].Payload) != string(payload) {
		t.Fatalf("roundtrip mismatch: op=%d payload=%q", got[0].Opcode, got[0].Payload)
	}
	if got[1].Opcode != 0x02 || string(got[1].Payload) != "second" {
		t.Fatalf("second frame mismatch: %+v", got[1])
	}
	if consumed != len(frames[0])+len(frames[1]) {
		t.Fatalf("consumed %d, want %d", consumed, len(frames[0])+len(frames[1]))
	}
}

// TestRoundtripEncoder — Encoder.Build → Decoder.Feed: payload должен сохраниться бит-в-бит.
func TestRoundtripEncoder(t *testing.T) {
	seed := uint32(0xDEADBEEF)
	payload := []byte("npc_id=1234;state=alive;")
	raw := Build(0x02, payload)

	// КРИПТЫ НЕТ — Encoder уже шлёт каноничный plaintext-кадр.
	_ = seed

	dec := NewDecoder(0)
	n, fr, err := dec.Feed(raw)
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	if n != len(raw) {
		t.Fatalf("consumed %d, want %d", n, len(raw))
	}
	if fr.Opcode != 0x02 || string(fr.Payload) != string(payload) {
		t.Fatalf("encoder roundtrip mismatch: op=%d payload=%q", fr.Opcode, fr.Payload)
	}
}

// TestBadChecksumRejected — невалидный csum → reject.
func TestBadChecksumRejected(t *testing.T) {
	raw := buildFrame(t, 42, 0x01, []byte("data"))

	// портим csum (последний байт кадра)
	raw[len(raw)-1] ^= 0xFF

	dec := NewDecoder(0)
	_, _, err := dec.Feed(raw)
	if !errors.Is(err, ErrBadChecksum) {
		t.Fatalf("want ErrBadChecksum, got %v", err)
	}
}

// TestBadSizeRejected — packetSize <= 0 или > MaxPacketSize → reject.
func TestBadSizeRejected(t *testing.T) {
	dec := NewDecoder(1)
	buf := []byte{0xFF, 0xFF, 0x01, 0x02} // len=0xFFFF → packetSize > MaxPacketSize
	_, _, err := dec.Feed(buf)
	if !errors.Is(err, ErrBadSize) {
		t.Fatalf("want ErrBadSize, got %v", err)
	}

	dec2 := NewDecoder(1)
	// len=2 → packetSize=0 (self-inclusive len указывает нулевое тело)
	_, _, err2 := dec2.Feed([]byte{0x02, 0x00})
	if !errors.Is(err2, ErrBadSize) {
		t.Fatalf("want ErrBadSize (packetSize=0), got %v", err2)
	}
}

// TestTruncated — пустой/обрезанный кадр → ErrShort.
func TestTruncated(t *testing.T) {
	dec := NewDecoder(1)
	buf := []byte{0x0A, 0x00} // len=10, тела нет
	_, _, err := dec.Feed(buf)
	if !errors.Is(err, ErrShort) {
		t.Fatalf("want ErrShort, got %v", err)
	}

	_, _, err = dec.Feed(nil)
	if !errors.Is(err, ErrShort) {
		t.Fatalf("want ErrShort (nil buf), got %v", err)
	}
}

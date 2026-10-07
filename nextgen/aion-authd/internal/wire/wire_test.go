package wire

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

// Golden-факты wire 2110 (live: дизasm AuthGateD + gate authdclient + probe 07.10).

func TestConnectFrameBytes(t *testing.T) {
	// гейт: Assemble("cdd", 0, sid=7, BE(192.168.0.253)=0xC0A800FD) → LE = FD 00 A8 C0.
	want := []byte{0x00, 0x07, 0, 0, 0, 0xFD, 0x00, 0xA8, 0xC0}
	got := ConnectFrame(7, [4]byte{192, 168, 0, 253})
	if !bytes.Equal(got, want) {
		t.Fatalf("ConnectFrame: got %x want %x", got, want)
	}
	f, err := ReadFrame(bytes.NewReader(got))
	if err != nil {
		t.Fatal(err)
	}
	if f.Type != FConnect || f.Sid != 7 || f.IP != [4]byte{192, 168, 0, 253} {
		t.Fatalf("roundtrip: %+v", f)
	}
}

func TestPacketFrameSelfInclusiveLen(t *testing.T) {
	blob := make([]byte, 191) // asm-форма login
	fr, err := PacketFrame(9, blob)
	if err != nil {
		t.Fatal(err)
	}
	// [02][sid=9][len=193 LE][blob]
	if !bytes.Equal(fr[0:5], []byte{0x02, 0x09, 0, 0, 0}) {
		t.Fatalf("hdr: %x", fr[0:5])
	}
	if l := binary.LittleEndian.Uint16(fr[5:7]); l != 193 {
		t.Fatalf("len: got %d want 193 (самоинклюзивный)", l)
	}
	if len(fr) != 7+191 {
		t.Fatalf("total: %d", len(fr))
	}
	f, err := ReadFrame(bytes.NewReader(fr))
	if err != nil || f.Sid != 9 || !bytes.Equal(f.Blob, blob) {
		t.Fatalf("roundtrip: %+v err=%v", f, err)
	}
}

func TestGreetingAndUnknown(t *testing.T) {
	if got, want := Greeting(0x0000c621), []byte{0x03, 0x21, 0xC6, 0x00, 0x00}; !bytes.Equal(got, want) {
		t.Fatalf("greeting: got %x want %x (live RAW AUTHD A>G [03] 21c60000)", got, want)
	}
	if got, want := UnknownSession(42), []byte{0x01, 42, 0, 0, 0}; !bytes.Equal(got, want) {
		t.Fatalf("unknown: %x", got)
	}
}

func TestReplyPktLenField(t *testing.T) {
	fr, err := ReplyPkt(5, 3, make([]byte, 52))
	if err != nil {
		t.Fatal(err)
	}
	// len = body+2 = (1+52)+2 = 55; total = 7+52
	if l := binary.LittleEndian.Uint16(fr[5:7]); l != 55 {
		t.Fatalf("len: got %d want 55", l)
	}
	if fr[7] != 3 {
		t.Fatalf("type byte: %x", fr[7])
	}
	f, err := ReadFrame(bytes.NewReader(fr))
	if err != nil || f.Sid != 5 || f.Blob[0] != 3 || len(f.Blob) != 53 {
		t.Fatalf("roundtrip: %+v err=%v", f, err)
	}
}

func TestTooBigBody(t *testing.T) {
	// body = 1+len(payload) ≤ MaxBody: payload MaxBody → body MaxBody+1 = ошибка
	if _, err := ReplyPkt(1, 3, make([]byte, MaxBody)); !errors.Is(err, ErrTooBig) {
		t.Fatalf("want ErrTooBig, got %v", err)
	}
	if _, err := ReplyPkt(1, 3, make([]byte, MaxBody-1)); err != nil {
		t.Fatalf("payload MaxBody-1 валиден, got %v", err)
	}
	// blob ≤ MaxBody валиден; MaxBody+1 = ошибка
	if _, err := PacketFrame(1, make([]byte, MaxBody)); err != nil {
		t.Fatalf("blob MaxBody валиден, got %v", err)
	}
	if _, err := PacketFrame(1, make([]byte, MaxBody+1)); !errors.Is(err, ErrTooBig) {
		t.Fatalf("want ErrTooBig, got %v", err)
	}
	// гейт сам рвёт при body > 0x1ffb — парсим так же
	bad := make([]byte, 7)
	bad[0] = 0x02
	binary.LittleEndian.PutUint16(bad[5:7], MaxBody+3) // body = MaxBody+1
	if _, err := ReadFrame(bytes.NewReader(bad)); !errors.Is(err, ErrTooBig) {
		t.Fatalf("read: want ErrTooBig, got %v", err)
	}
}

func TestBadFrameType(t *testing.T) {
	if _, err := ReadFrame(bytes.NewReader([]byte{0x09, 0, 0, 0, 0})); !errors.Is(err, ErrBadFrame) {
		t.Fatalf("want ErrBadFrame, got %v", err)
	}
}

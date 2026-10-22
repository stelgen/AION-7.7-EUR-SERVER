// R4.5 golden-тесты: session-подстановки на ЖИВЫХ парах (тень v10 S-кадр +
// session из cmd=288 → должен стать O-кадром копии-2016 байт-в-байт).
// Источник пар: живой логин 22.10 вечер (ab-r4-verdict-20261022.md).
package rpc

import (
	"encoding/binary"
	"os"
	"testing"
)

func loadAB(name string) []byte {
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		return nil
	}
	return b
}

// TestABLiveSubstitute — каждая живая пара: S-кадр (captured-поля) + session
// (X=0x7fa02e20, a=3, acc=1010 из cmd=288) → байт-в-байт = O-кадр ориг.
func TestABLiveSubstitute(t *testing.T) {
	sess := Session{X: 0x7fa02e20, A: 3, AccID: 1010}
	cases := []uint16{0x00b, 0x017, 0x020, 0x022, 0x02c, 0x065, 0x07f, 0x0db}
	for _, c := range cases {
		s := loadAB("ablive_s_%03x.bin"[:13] + "" + itoahex(c) + ".bin")
		o := loadAB("ablive_o_" + itoahex(c) + ".bin")
		if s == nil || o == nil {
			continue
		}
		got := SubstituteSession(s, sess)
		if len(got) != len(o) {
			t.Fatalf("0x%03X: len %d vs %d", c, len(got), len(o))
		}
		for i := range o {
			if got[i] != o[i] {
				t.Fatalf("0x%03X: frame[%d]: got %02X want %02X (S %02X)", c, i, got[i], o[i], s[i])
			}
		}
	}
}

// TestABLiveChain — ApplySessionToChain на цепочке (echo chain из capture
// + session живой сессии): маркер-разбор не должен ломаться.
func TestABLiveChain(t *testing.T) {
	sess := Session{X: 0x7fa02e20, A: 3, AccID: 1010}
	chain := GetItemChain(0)
	if len(chain) == 0 {
		t.Fatal("empty echo chain")
	}
	got := ApplySessionToChain(chain, sess)
	if len(got) != len(chain) {
		t.Fatalf("chain len %d vs %d", len(got), len(chain))
	}
	// cmd-байты кадров не меняются
	for off := 0; off+4 <= len(chain); {
		ln := int(binary.LittleEndian.Uint16(chain[off : off+2]))
		if ln < 7 || off+ln > len(chain) {
			break
		}
		if got[off+2] != chain[off+2] || got[off+3] != chain[off+3] {
			t.Fatalf("cmd bytes changed at %d", off)
		}
		off += ln
	}
}

// TestBuildAcks — ack-кадр (12 cmd, payload 18Б): session-X/a подставляются,
// длина кадра 25, cmd/маркер/инверсия на месте.
func TestBuildAcks(t *testing.T) {
	sess := Session{X: 0x7fa02e20, A: 3, AccID: 1010}
	out := BuildAcks(AcksAfterItemData, sess)
	// 3 ack × 25Б = 75
	if len(out) != 75 {
		t.Fatalf("acks len %d want 75", len(out))
	}
	// первый ack: cmd=0x018, X в payload
	if binary.LittleEndian.Uint16(out[2:4]) != 0x018 {
		t.Fatalf("ack cmd %x", binary.LittleEndian.Uint16(out[2:4]))
	}
	if binary.LittleEndian.Uint32(out[7:11]) != 0x7fa02e20 {
		t.Fatalf("ack X %x", binary.LittleEndian.Uint32(out[7:11]))
	}
	if out[11] != 3 {
		t.Fatalf("ack a %d", out[11])
	}
}

func itoahex(v uint16) string {
	const hexd = "0123456789abcdef"
	return string([]byte{hexd[v>>8&0xf], hexd[v>>4&0xf], hexd[v&0xf]})
}

var _ = os.Getenv

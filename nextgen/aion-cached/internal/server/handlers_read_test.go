// Golden-тесты R4 — кадры live capture 22.10 (тест data r4-live-20261022.txt,
// логин юзера через cdmirror-стенд). Проверяют: Decoder принимает 0xAE (Т26),
// хендлеры парсят раскладки read-пути без ошибок.
package server

import (
	"bufio"
	"encoding/binary"
	"os"
	"strings"
	"testing"

	"aion-cached/internal/wire"
)

// TestR4LiveFrames — прогон live-кадров через Decoder (маркер 0xAE, Т26):
// каждый кадр должен распарситься без ErrBadMarker/ErrBadSize.
func TestR4LiveFrames(t *testing.T) {
	f, err := os.Open("testdata/r4-live-20261022.txt")
	if err != nil {
		t.Skip("нет testdata r4-live-20261022.txt")
	}
	defer f.Close()
	dec := wire.NewDecoder()
	n := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		i := strings.Index(line, "wire=")
		if i < 0 {
			continue
		}
		wireHex := line[i+5:]
		// кадр → убрать хвост-2Б (Validate работает с buf[2:total-2] как в Feed)
		raw := mustHex(t, wireHex)
		total := int(binary.LittleEndian.Uint16(raw[0:2]))
		if total != len(raw) {
			t.Fatalf("len-поле %d != факт %d (%s)", total, len(raw), wireHex[:40])
		}
		if _, _, ferr := dec.Feed(raw); ferr != nil {
			t.Fatalf("live frame rejected: %v (%s)", ferr, wireHex[:40])
		}
		n++
	}
	if n == 0 {
		t.Fatal("ни одного кадра в testdata")
	}
	t.Logf("R4 live frames passed: %d", n)
}

// TestR4HandlersParse — поля раскладок read-пути (hex из capture).
func TestR4HandlersParse(t *testing.T) {
	// RQ_LOAD: [X=0x06f38f70][a=4][charId=1002][accId=1010]
	load := mustHex(t, "708ff30604000000ea030000f2030000")
	if r, ok := parseSession(load[:12]); !ok || r.X != 0x06f38f70 || r.A != 4 {
		t.Fatalf("session prefix = %+v", r)
	}
	if charID, accID := le32(load[8:12]), le32(load[12:16]); charID != 1002 || accID != 1010 {
		t.Fatalf("RQ_LOAD char/acc = %d/%d", charID, accID)
	}
	// RQ_LOAD_ITEM cat=02 page=00
	item := mustHex(t, "708ff30604000000ea03000002000000f203000000")
	if cat, page := le32(item[12:16]), item[20]; cat != 2 || page != 0 {
		t.Fatalf("LOAD_ITEM cat/page = %d/%d", cat, page)
	}
	item2 := mustHex(t, "708ff30604000000ea03000003000000f203000001")
	if cat, page := le32(item2[12:16]), item2[20]; cat != 3 || page != 1 {
		t.Fatalf("LOAD_ITEM#2 cat/page = %d/%d", cat, page)
	}
	// LEAVE_WORLD: acc=1010 char=1002
	leave := mustHex(t, "f2030000ea030000909e8e0642c7ae446cebbc44a23f0f4400000000")
	if acc, ch := le32(leave[0:4]), le32(leave[4:8]); acc != 1010 || ch != 1002 {
		t.Fatalf("LEAVE acc/char = %d/%d", acc, ch)
	}
}

// TestWireMarkersCached — Т26: маркер-пара AE/AF.
func TestWireMarkersCached(t *testing.T) {
	frame := mustHex(t, "17000b00aef4ff708ff30604000000ea030000f2030000")
	// Accept: AE допустим (cached live).
	if _, err := wire.Validate(frame[2:len(frame)-2], wire.MarkerCachedC2S); err != nil {
		t.Fatalf("AE frame rejected: %v", err)
	}
	// BuildS2CCached: маркер 0xAF.
	resp := wire.BuildS2CCached(1, []byte{1, 2, 3})
	if resp[4] != wire.MarkerCachedS2C {
		t.Fatalf("BuildS2CCached marker = %#x", resp[4])
	}
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b := make([]byte, len(s)/2)
	for i := 0; i < len(b); i++ {
		hi := hexVal(s[2*i])
		lo := hexVal(s[2*i+1])
		if hi < 0 || lo < 0 {
			t.Fatalf("bad hex %q", s)
		}
		b[i] = byte(hi<<4 | lo)
	}
	return b
}

func hexVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

package proto

import (
	"encoding/binary"
	"os"
	"strings"
	"testing"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name + ".hex")
	if err != nil {
		t.Fatalf("fixture %s: %v", name, err)
	}
	hex := strings.Fields(string(raw))
	b := make([]byte, len(hex))
	for i, h := range hex {
		if len(h) != 2 {
			t.Fatalf("fixture %s: bad token %q", name, h)
		}
		v := 0
		for _, ch := range h {
			v <<= 4
			switch {
			case ch >= '0' && ch <= '9':
				v |= int(ch - '0')
			case ch >= 'a' && ch <= 'f':
				v |= int(ch-'a') + 10
			default:
				t.Fatalf("fixture %s: bad hex %q", name, h)
			}
		}
		b[i] = byte(v)
	}
	return b
}

// fixture от реального прод-обмена 05.10.2026 (первый пакет каждого типа).
func TestFixtureLangCheck(t *testing.T) {
	f := loadFixture(t, "langcheck") // 08 00 65 00 01 00 6e 65
	if len(f) != 8 {
		t.Fatalf("len=%d", len(f))
	}
	if binary.LittleEndian.Uint16(f) != 8 {
		t.Fatalf("total=%d", binary.LittleEndian.Uint16(f))
	}
	if binary.LittleEndian.Uint16(f[2:]) != TypeLangCheck {
		t.Fatalf("type=%d", binary.LittleEndian.Uint16(f[2:]))
	}
	seq, lang, err := ParseLangCheck(f[4:])
	if err != nil {
		t.Fatal(err)
	}
	if seq != 1 {
		t.Fatalf("seq=%d", seq)
	}
	if lang != En("en") || UnEn(lang) != "en" {
		t.Fatalf("lang=0x%04x %q", lang, UnEn(lang))
	}
	want := BuildLangCheckReply(seq, lang)
	if binary.LittleEndian.Uint16(want[2:]) != TypeLangCheckReply {
		t.Fatalf("reply type=%d", binary.LittleEndian.Uint16(want[2:]))
	}
	rep := loadFixture(t, "langcheck_reply")
	for i := range rep {
		if want[i] != rep[i] {
			t.Fatalf("reply byte %d: got %02x want %02x", i, want[i], rep[i])
		}
	}
}

func TestFixtureRequest(t *testing.T) {
	f := loadFixture(t, "request") // 12 00 e9 03 01 00 00 00 00 00 00 00 b0 04 00 00 6e 65
	if len(f) != 18 {
		t.Fatalf("len=%d", len(f))
	}
	if binary.LittleEndian.Uint16(f[2:]) != TypeCaptchaRequest {
		t.Fatalf("type=%d", binary.LittleEndian.Uint16(f[2:]))
	}
	seq, codepage, lang, err := ParseRequest(f[4:])
	if err != nil {
		t.Fatal(err)
	}
	if seq != 1 || codepage != CodepageUTF16 || lang != En("en") {
		t.Fatalf("seq=%d codepage=%d lang=%04x", seq, codepage, lang)
	}
	built := BuildRequest(seq, lang, codepage)
	for i := range built {
		if built[i] != f[i] {
			t.Fatalf("build byte %d: got %02x want %02x", i, built[i], f[i])
		}
	}
}

func TestFixtureReply(t *testing.T) {
	f := loadFixture(t, "reply") // 2212 байт живого ответа
	if len(f) != 2212 {
		t.Fatalf("len=%d", len(f))
	}
	if binary.LittleEndian.Uint16(f[2:]) != TypeCaptchaReply {
		t.Fatalf("type=%d", binary.LittleEndian.Uint16(f[2:]))
	}
	rep, err := ParseReply(f[4:])
	if err != nil {
		t.Fatal(err)
	}
	if rep.Seq != 1 {
		t.Fatalf("seq=%d", rep.Seq)
	}
	if len(rep.DDS) != int(DDSImageSize) {
		t.Fatalf("dds=%d", len(rep.DDS))
	}
	if string(rep.DDS[0:4]) != "DDS " {
		t.Fatalf("dds magic=%q", rep.DDS[0:4])
	}
	if len(rep.TextUTF16) != 12 {
		t.Fatalf("text len=%d", len(rep.TextUTF16))
	}
	// 6 цифр
	for i := 0; i < 6; i++ {
		ch := binary.LittleEndian.Uint16(rep.TextUTF16[i*2:])
		if ch < 0x30 || ch > 0x39 {
			t.Fatalf("char %d = %04x не цифра", i, ch)
		}
	}
	// собрать такой же по структуре — поле-консты должны совпасть
	built := BuildReply(rep.Seq, rep.DDS, rep.TextUTF16)
	if len(built) != len(f) {
		t.Fatalf("built len=%d want %d", len(built), len(f))
	}
	for i := range f {
		if i >= 16 { // DDS/текст — наши данные не обязаны совпадать байт-в-байт
			break
		}
		if built[i] != f[i] {
			t.Fatalf("built byte %d: got %02x want %02x", i, built[i], f[i])
		}
	}
}

func TestReplyTotalOf(t *testing.T) {
	if ReplyTotalOf(2176) != 2212 {
		t.Fatalf("total=%d", ReplyTotalOf(2176))
	}
}

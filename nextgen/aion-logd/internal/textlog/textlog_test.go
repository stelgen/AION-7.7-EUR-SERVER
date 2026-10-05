package textlog

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"aion-logd/internal/proto"
)

// loadFixture — testdata/textlog_records.txt: "time RAWHEX" (реальные пакеты).
func loadFixture(t *testing.T) [][]byte {
	t.Helper()
	f, err := os.Open("testdata/textlog_records.txt")
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	defer f.Close()
	var out [][]byte
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' {
			continue
		}
		hexPart := line[strings.IndexByte(line, ' ')+1:]
		b, err := hex.DecodeString(strings.ReplaceAll(hexPart, " ", ""))
		if err != nil {
			t.Fatalf("fixture hex: %v", err)
		}
		out = append(out, b)
	}
	if len(out) < 3 {
		t.Fatalf("fixture пуст: %d", len(out))
	}
	return out
}

// Тест на ЖИВЫХ сэмплах: боевая capture 05.10 + mirror-эталон оригинала.
func TestParseRealSamples(t *testing.T) {
	for i, raw := range loadFixture(t) {
		p, _, err := proto.Parse(raw)
		if err != nil {
			t.Fatalf("pkt %d: %v", i, err)
		}
		if p.Type != proto.TypeTextLog {
			t.Fatalf("pkt %d: type=%d", i, p.Type)
		}
		rec, err := Parse(p.Body)
		if err != nil {
			t.Fatalf("pkt %d parse: %v", i, err)
		}
		if rec.ID != 928 {
			t.Errorf("pkt %d: id=%d want 928", i, rec.ID)
		}
		if len(rec.Entries) < 2 {
			t.Errorf("pkt %d: entries=%d, ожидались сессия (char+acc)", i, len(rec.Entries))
		}
		// сессия юзера: (1002,"SteLGeN")(1010,"Stelgen")
		if rec.Entries[0].Key != 1002 || rec.Entries[0].Name != "SteLGeN" ||
			rec.Entries[1].Key != 1010 || rec.Entries[1].Name != "Stelgen" {
			t.Errorf("pkt %d entries: %+v", i, rec.Entries)
		}
		if rec.Stamp == "" || !strings.HasPrefix(rec.Stamp, "2026-10-05") {
			t.Errorf("pkt %d stamp=%q", i, rec.Stamp)
		}
		t.Logf("pkt %d: %s", i, rec.String())
	}
}

// 251b-вариант = 2 сессии (4 записи) — greedy-парсер НЕ должен заглатывать хвост.
func TestParseTwoSessions(t *testing.T) {
	var raw251 []byte
	for _, raw := range loadFixture(t) {
		if len(raw) == 251 {
			raw251 = raw
			break
		}
	}
	if raw251 == nil {
		t.Fatal("нет 251b в fixture")
	}
	p, _, _ := proto.Parse(raw251)
	rec, err := Parse(p.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.Entries) != 4 {
		t.Fatalf("entries=%d want 4 (2 сессии): %+v", len(rec.Entries), rec.Entries)
	}
	if rec.World != 210040000 {
		t.Logf("note: world не найден в хвосте (ok) — %s", rec.TailHex[:min(60, len(rec.TailHex))])
	}
}

// Синтетика: мусорный хвост не превращается в «имя».
func TestTailNotEaten(t *testing.T) {
	body := []byte{0xa0, 0x03, 0x00, 0x00}          // id=928
	body = append(body, 0xea, 0x03, 0x00, 0x00)     // key=1002
	for _, c := range "SteLGeN" {                   // имя UTF-16LE
		body = append(body, byte(c), 0)
	}
	body = append(body, 0, 0)                       // NUL
	body = append(body, []byte{
		0xff, 0xff, 0x80, 0x00, 0xff, 0xff, 0x00, 0x00, // блоки-флаги хвоста
		0xea, 0x07, 0x0a, 0x00, 0x01, 0x00, 0x05, 0x00, // 2026-10-05 ...
		0x0b, 0x00, 0x3a, 0x00, 0x39, 0x00, 0x25, 0x00,
	}...)
	rec, err := Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.Entries) != 1 {
		t.Fatalf("entries=%d want 1: %+v", len(rec.Entries), rec.Entries)
	}
	if rec.Entries[0].Name != "SteLGeN" {
		t.Fatalf("name=%q", rec.Entries[0].Name)
	}
	if rec.Stamp != "2026-10-05 11:58:57.037" {
		t.Fatalf("stamp=%q", rec.Stamp)
	}
	if rec.World != 0 {
		t.Fatalf("world=%d", rec.World)
	}
}

func TestStringLine(t *testing.T) {
	r := &Record{ID: 928, Entries: []Entry{{1002, "SteLGeN"}, {1010, "Stelgen"}},
		World: 210040000, Stamp: "2026-10-05 11:59:00.350"}
	got := r.String()
	for _, want := range []string{"id=928", "n=2", `[1002:"SteLGeN"]`, `[1010:"Stelgen"]`, "world=210040000", "time=2026-10-05 11:59:00.350"} {
		if !strings.Contains(got, want) {
			t.Errorf("нет %q в %q", want, got)
		}
	}
	if !bytes.Contains([]byte(got), []byte(`[1002:"SteLGeN"]`)) {
		t.Error("формат entries")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
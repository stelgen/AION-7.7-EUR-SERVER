package records

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"
)

// loadFixture — testdata/status_records.txt: "svc seq hex" (317 живых записей capture 05.10).
func loadFixture(t *testing.T) []struct {
	Svc, Seq int
	Body     []byte
} {
	t.Helper()
	f, err := os.Open("../../testdata/status_records.txt")
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	defer f.Close()
	var rows []struct {
		Svc, Seq int
		Body     []byte
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		p := strings.Fields(sc.Text())
		if len(p) != 3 {
			continue
		}
		svc, _ := strconv.Atoi(p[0])
		seq, _ := strconv.Atoi(p[1])
		b, err := parseHexFast(p[2])
		if err != nil {
			t.Fatalf("hex: %v", err)
		}
		rows = append(rows, struct {
			Svc, Seq int
			Body     []byte
		}{svc, seq, b})
	}
	if len(rows) < 300 {
		t.Fatalf("fixture мал: %d", len(rows))
	}
	return rows
}

func parseHexFast(s string) ([]byte, error) {
	b := make([]byte, len(s)/2)
	for i := range b {
		hi := hexVal(s[2*i])
		lo := hexVal(s[2*i+1])
		b[i] = hi<<4 | lo
	}
	return b, nil
}

func hexVal(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

func TestParseAllFixture(t *testing.T) {
	rows := loadFixture(t)
	for i, r := range rows {
		rec, err := ParseStatus(r.Body)
		if err != nil {
			t.Fatalf("строка %d: %v", i, err)
		}
		if uint32(r.Svc) != rec.SvcType {
			t.Fatalf("строка %d: svc fixture=%d != parsed=%d", i, r.Svc, rec.SvcType)
		}
		if rec.SysTime[0] != 2026 {
			t.Fatalf("строка %d: год %d", i, rec.SysTime[0])
		}
	}
}

func TestFixtureInvariants(t *testing.T) {
	rows := loadFixture(t)
	prevTick := map[uint32]uint32{}
	prevSeq := map[uint32]int{}
	for i, r := range rows {
		rec, err := ParseStatus(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		// движковый ms (@188) монотонен в рамках клиента; u64@8 (TickRaw) НЕ mono — не проверяем
		if p, ok := prevTick[rec.SvcType]; ok && rec.EngineMs < p {
			t.Fatalf("строка %d: engineMs уменьшился у svc %d", i, rec.SvcType)
		}
		prevTick[rec.SvcType] = rec.EngineMs
		// seq (fixture col 2) монотонен в рамках клиента
		if p, ok := prevSeq[rec.SvcType]; ok && r.Seq < p {
			t.Fatalf("строка %d: seq уменьшился у svc %d", i, rec.SvcType)
		}
		prevSeq[rec.SvcType] = r.Seq
		// год корректен, час в сутках
		if rec.SysTime[0] != 2026 || rec.SysTime[4] > 23 {
			t.Fatalf("строка %d: systime %+v", i, rec.SysTime)
		}
	}
}

func TestWorldFlagAndString(t *testing.T) {
	rows := loadFixture(t)
	seenWorld := false
	for _, r := range rows {
		rec, _ := ParseStatus(r.Body)
		if w := rec.World(); w == 210010000 {
			seenWorld = true
		}
		if s := rec.String(); len(s) < 20 {
			t.Fatalf("string: %q", s)
		}
	}
	if !seenWorld {
		t.Fatal("мир 210010000 не встретился в capture")
	}
}

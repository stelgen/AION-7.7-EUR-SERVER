package records

import (
	"encoding/hex"
	"strings"
	"testing"
)

// Реальный сэмпл из прод- capture 05.10 (badstatus.raw → svc701, ключ форта).
func TestParseVarRealSample(t *testing.T) {
	raw := "BD 02 00 00 63 1B 00 00 4C 00 44 00 46 00 35 00 5F 00 46 00 6F 00 72 00 74 00 72 00 65 00 73 00 73 00 5F 00 37 00 30 00 31 00 31 00 00 00"
	body, err := hex.DecodeString(replaceSpaces(raw))
	if err != nil {
		t.Fatal(err)
	}
	v, err := ParseVar(body)
	if err != nil {
		t.Fatal(err)
	}
	if v.Svc != 701 || v.ID != 7011 {
		t.Fatalf("svc=%d id=%d", v.Svc, v.ID)
	}
	if !strings.Contains(v.Text, "LDF5_Fortress_7011") {
		t.Fatalf("text=%q", v.Text)
	}
	t.Logf("%s", v.String())
}

// Слишком короткое тело = ошибка.
func TestParseVarTooShort(t *testing.T) {
	if _, err := ParseVar([]byte{1, 2, 3}); err == nil {
		t.Fatal("ожидали ошибку на коротком теле")
	}
}

func replaceSpaces(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' {
			out = append(out, s[i])
		}
	}
	return string(out)
}
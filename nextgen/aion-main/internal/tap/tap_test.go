package tap

import (
	"os"
	"path/filepath"
	"testing"

	"aion-main/internal/ops"
)

// TestRunRealCapture — РЕАЛЬНАЯ копия трафика юзера (capture 0810d, tshark-follow hex).
// VERDICT: почти всё в реестре; unknown = только известные исключения.
func TestRunRealCapture(t *testing.T) {
	reg, err := ops.Load("../../ops.yaml")
	if err != nil {
		t.Skipf("ops.yaml недоступен: %v", err)
	}
	dir := os.Getenv("CAP7777")
	if dir == "" {
		t.Skip("CAP7777 env не задан (песочница-локальные capture-хексы)")
	}
	s2c := readHex(t, filepath.Join(dir, "d_s2c.hex"))
	c2s := readHex(t, filepath.Join(dir, "d_c2s.hex"))
	if s2c == nil || c2s == nil {
		t.Skip("capture-хекс не найден")
	}
	st := Run(s2c, c2s, reg, func(string, ...any) {})
	if st.S2CFrames < 2000 {
		t.Fatalf("S2C кадров мало: %d", st.S2CFrames)
	}
	if st.Invalid != 0 {
		t.Fatalf("invalid: %d", st.Invalid)
	}
	// 0x0182 SM_STRONGHOLDS теперь В реестре → unknown уменьшился против R1
	for _, u := range dedup(st.UnknownS2C) {
		if u == "0x0182" {
			t.Fatalf("0x0182 должен быть в реестре (SM_STRONGHOLDS)")
		}
	}
}

func readHex(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	out := make([]byte, 0, len(data)/2)
	var hi byte
	for i, c := range data {
		var v byte
		switch {
		case c >= '0' && c <= '9':
			v = c - '0'
		case c >= 'a' && c <= 'f':
			v = c - 'a' + 10
		case c >= 'A' && c <= 'F':
			v = c - 'A' + 10
		default:
			continue
		}
		if i%2 == 0 {
			hi = v << 4
		} else {
			out = append(out, hi|v)
		}
	}
	return out
}

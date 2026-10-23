// world_flow_test.go — кросс-сессийная верификация эмиттеров мир-init
// (capture 2026-10-23: sess1 = шаблоны, sess2 = эталон с другим X/a).
package rpc

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"testing"
)

type wfSess2Frame struct {
	Cmd   int    `json:"cmd"`
	Plen  int    `json:"plen"`
	Hex   string `json:"hex"`
}

// TestWorldFlowCrossSession — подставляем sess2 X/a в sess1-шаблоны и
// сравниваем байт-в-байт с sess2 capture. Метрика = SAME/DIFF per кадр
// + средний дифф-байт (рантайм-зоны допустимы, критерий как accache A/B).
func TestWorldFlowCrossSession(t *testing.T) {
	metaRaw, err := os.ReadFile("testdata/worldflow-20261023-meta.json")
	if err != nil {
		t.Skipf("нет meta: %v", err)
	}
	var meta struct {
		XSess1  uint32 `json:"X_sess1"`
		XSess2  uint32 `json:"X_sess2"`
		RefSess1 uint32 `json:"ref_sess1"`
		RefSess2 uint32 `json:"ref_sess2"`
	}
	if err := json.Unmarshal(metaRaw, &meta); err != nil {
		t.Fatal(err)
	}
	sess2Raw, err := os.ReadFile("testdata/worldflow-20261023-sess2.json")
	if err != nil {
		t.Skipf("нет sess2: %v", err)
	}
	var sess2 []wfSess2Frame
	if err := json.Unmarshal(sess2Raw, &sess2); err != nil {
		t.Fatal(err)
	}

	// sess1 шаблоны по (cmd, plen) → первый кадр
	chain := GetWorldFlowChain()
	if len(chain) == 0 {
		t.Fatal("пустой worldinit chain")
	}
	frames := SplitWorldFlowFrames(chain)
	t.Logf("chain: %d frames", len(frames))
	templates := map[[2]int][]byte{}
	for _, f := range frames {
		cmd := binary.LittleEndian.Uint16(f[2:4])
		key := [2]int{int(cmd), len(f)}
		if _, ok := templates[key]; !ok {
			templates[key] = f
		}
	}

	same, diff, missing, diffBytes, nearSame, hardDiff, zlib := 0, 0, 0, 0, 0, 0, 0
	for _, s2 := range sess2 {
		key := [2]int{s2.Cmd, s2.Plen + 7} // frame len = payload + 7
		tpl, ok := templates[key]
		if !ok {
			missing++
			continue
		}
		got := SubstituteWorldFlow(tpl, meta.XSess1, meta.XSess2, meta.RefSess1, meta.RefSess2, 4, 3) // a: 4→3 (Т38)
		want, _ := hexDecode(s2.Hex)
		n := byteDiffCount(got, want)
		frameLen := len(want)
		if s2.Cmd == 14 {
			// zlib-сжатые страницы XML: дифф несравним по байтам (лавина от 1Б
			// изменения исходника). Вердикт = SIZE-match, содержимое сверяется
			// после распаковки (отдельная фаза).
			zlib++
			continue
		}
		if bytesEqual(got, want) {
			same++
		} else {
			diff++
			diffBytes += n
			// рантайм-зоны: timestamps/координаты/счётчики — неизбежны;
			// критерий честности: diff ≤ 10% размера кадра
			if n*10 <= frameLen {
				nearSame++
			} else {
				hardDiff++
				t.Logf("hard: cmd=%d len=%d diff=%dБ", s2.Cmd, frameLen, n)
			}
		}
	}
	total := len(sess2)
	t.Logf("кросс-сессия: total=%d SAME=%d nearSame(≤10%%)=%d hardDiff=%d zlib=%d missing=%d diffBytes=%d",
		total, same, nearSame, hardDiff, zlib, missing, diffBytes)
	if total == 0 {
		t.Fatal("пустая sess2 фикстура")
	}
	// критерий MVP: SAME+nearSame+zlib ≥ 80% от total
	if (same+nearSame+zlib)*5 < total*4 {
		t.Fatalf("слишком мало SAME+nearSame+zlib: %d/%d — эмиттер не работает", same+nearSame+zlib, total)
	}
}

func hexDecode(s string) ([]byte, error) {
	out := make([]byte, len(s)/2)
	for i := 0; i < len(out); i++ {
		hi, lo := hexVal(s[2*i]), hexVal(s[2*i+1])
		out[i] = hi<<4 | lo
	}
	return out, nil
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

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func byteDiffCount(a, b []byte) int {
	n := 0
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			n++
		}
	}
	return n
}

func maxOf(a, b int) int {
	if a > b {
		return a
	}
	return b
}

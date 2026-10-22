// ChainReplay тесты: полные response chains из capture.
package rpc

import (
	"encoding/binary"
	"testing"
)

func TestLoadChainExists(t *testing.T) {
	data := LoadChain("chain_rq_load_0.bin")
	if len(data) == 0 {
		t.Fatal("chain_rq_load_0.bin empty or missing")
	}
	if len(data) < 100 {
		t.Fatalf("chain too short: %d", len(data))
	}
	// первый кадр = RP cmd=1
	cmd := binary.LittleEndian.Uint16(data[2:4])
	if cmd != 1 {
		t.Fatalf("first frame cmd = %d, want 1", cmd)
	}
	t.Logf("chain_rq_load_0: %d bytes, first cmd=%d ✓", len(data), cmd)
}

func TestItemChains(t *testing.T) {
	echo := LoadChain("chain_rq_load_item_0.bin")
	data := LoadChain("chain_rq_load_item_1.bin")
	if len(echo) == 0 || len(data) == 0 {
		t.Fatal("item chains missing")
	}
	t.Logf("echo chain: %d bytes, data chain: %d bytes", len(echo), len(data))
	if len(data) <= len(echo) {
		t.Fatalf("data chain (%d) should be larger than echo (%d)", len(data), len(echo))
	}
	// data chain должен содержать cmd=109 (item blocks)
	// парсим кадры в конкатенированном буфере
	found109 := false
	off := 0
	for off+7 <= len(data) {
		total := int(binary.LittleEndian.Uint16(data[off : off+2]))
		if total < 7 || off+total > len(data) {
			break
		}
		cmd := binary.LittleEndian.Uint16(data[off+2 : off+4])
		if cmd == 109 {
			found109 = true
		}
		off += total
	}
	if !found109 {
		t.Fatal("data chain doesn't contain cmd=109 (item blocks)")
	}
	t.Log("data chain contains cmd=109 ✓")
}

func TestGetItemChainPage(t *testing.T) {
	echo := GetItemChain(0)
	data := GetItemChain(1)
	if len(echo) == 0 || len(data) == 0 {
		t.Fatal("GetItemChain returned empty")
	}
	if len(data) <= len(echo) {
		t.Fatalf("page=1 data (%d) should be > page=0 echo (%d)", len(data), len(echo))
	}
	t.Logf("page=0: %dБ (echo), page=1: %dБ (data) ✓", len(echo), len(data))
}

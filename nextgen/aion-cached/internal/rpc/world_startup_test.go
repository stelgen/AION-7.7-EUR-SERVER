package rpc

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestWorldStartupIdentitySess1 — эмиссия с sess1-константами = байт-в-байт capture W5.
func TestWorldStartupIdentitySess1(t *testing.T) {
	got := EmitWorldStartup(WorldFlowSess1X, WorldFlowSess1CharRef, WorldFlowSess1A)
	want := GetWorldStartupChain()
	if want == nil {
		t.Fatal("chain_worldstartup_20261023.bin не найден (embed)")
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("identity FAIL: got=%dB want=%dB", len(got), len(want))
	}
}

// TestWorldStartupFrameCount — 120 кадров, канон W5 (первый O=196, всего 0x3D=88).
func TestWorldStartupFrameCount(t *testing.T) {
	frames := SplitWorldFlowFrames(GetWorldStartupChain())
	if len(frames) != 120 {
		t.Fatalf("кадров = %d, ожидалось 120", len(frames))
	}
	if cmd := binary.LittleEndian.Uint16(frames[0][2:4]); cmd != 196 {
		t.Fatalf("первый O cmd=%d, ожидался 196 (RP_HIGHEST_LEVEL_CHARACTER_INFO_ACK)", cmd)
	}
	n3d := 0
	for _, f := range frames {
		if binary.LittleEndian.Uint16(f[2:4]) == 0x3D {
			n3d++
		}
	}
	if n3d != 44 {
		t.Fatalf("0x3D в цепочке = %d, ожидалось 44 (88 было ошибкой: считал S+O вместе)", n3d)
	}
	n7a, nbf, nbb := 0, 0, 0
	for _, f := range frames {
		switch binary.LittleEndian.Uint16(f[2:4]) {
		case 0x7A:
			n7a++
		case 0xBF:
			nbf++
		case 0xBB:
			nbb++
		}
	}
	if n7a != 2 || nbf != 1 || nbb != 1 {
		t.Fatalf("канон W5 нарушен: 0x7A=%d 0xBF=%d 0xBB=%d (ожидается 2/1/1)", n7a, nbf, nbb)
	}
}

// TestWorldStartupResubstitution — смена X: структура сохраняется, X-зона заменяется.
func TestWorldStartupResubstitution(t *testing.T) {
	newX := uint32(0x00ABCDEF)
	got := EmitWorldStartup(newX, WorldFlowSess1CharRef, WorldFlowSess1A)
	frames := SplitWorldFlowFrames(got)
	if len(frames) != 120 {
		t.Fatalf("кадров после подстановки = %d, ожидалось 120", len(frames))
	}
	leX := make([]byte, 4)
	binary.LittleEndian.PutUint32(leX, newX)
	// хотя бы в одном кадре шапки должен появиться новый X (frame[5:25])
	found := false
	for _, f := range frames {
		end := 25
		if len(f) < end {
			end = len(f)
		}
		if bytes.Contains(f[:end], leX) {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("новый X не найден в шапках после подстановки")
	}
	// структура (len-поле) неизменна: первые 2 байта каждого кадра совпадают с эталоном
	ref := SplitWorldFlowFrames(GetWorldStartupChain())
	for i := range frames {
		if frames[i][0] != ref[i][0] || frames[i][1] != ref[i][1] {
			t.Fatalf("кадр %d: len-поле изменилось (%x vs %x)", i, frames[i][:2], ref[i][:2])
		}
	}
}

// TestStartupTracker — новый X = новый цикл, тот же X = продолжение.
func TestStartupTracker(t *testing.T) {
	var tr StartupTracker
	if !tr.OnSession(0x111) {
		t.Fatal("первый 288 должен дать new-cycle=true")
	}
	if tr.OnSession(0x111) {
		t.Fatal("тот же X — не новый цикл")
	}
	if !tr.OnSession(0x222) {
		t.Fatal("новый X (рестарт пары) — новый цикл")
	}
	tr.MarkEmitted()
	if !tr.Done() {
		t.Fatal("после MarkEmitted Done()=true")
	}
	if !tr.OnSession(0x333) {
		t.Fatal("новый X после эмиссии — снова новый цикл")
	}
	if tr.Done() {
		t.Fatal("новый цикл должен сбросить startedOK")
	}
}

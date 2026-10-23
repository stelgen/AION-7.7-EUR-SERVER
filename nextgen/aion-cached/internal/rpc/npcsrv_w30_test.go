package rpc

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestNpcW30FullReplay — полный replay W30: все C по cseq → конкатенация = bin (228 кадров).
func TestNpcW30FullReplay(t *testing.T) {
	cseq := NpcW30CSeq()
	if len(cseq) == 0 {
		t.Fatal("cseq пуст — manifest не прочитан")
	}
	r := NewNpcSrvReplay()
	var emitted bytes.Buffer
	for _, cmd := range cseq {
		if b := r.EmitNpcSrv(cmd); len(b) > 0 {
			emitted.Write(b)
		}
	}
	all := SplitWorldFlowFrames(GetNpcW30Blob())
	if len(all) != 228 {
		t.Fatalf("кадров в bin = %d, ожидалось 228", len(all))
	}
	got := SplitWorldFlowFrames(emitted.Bytes())
	if len(got) != 228 {
		t.Fatalf("emitted кадров = %d, ожидалось 228", len(got))
	}
	for i := range got {
		if !bytes.Equal(got[i], all[i]) {
			t.Fatalf("кадр %d отличается от capture", i)
		}
	}
}

// TestNpcW301to1 — C277→O187 ×150 (1:1 по occurrence).
func TestNpcW301to1(t *testing.T) {
	if n := NpcW30BatchCount(277); n != 150 {
		t.Fatalf("277 occurrence = %d, ожидалось 150", n)
	}
}

// TestNpcW30FireForget — 179/224 = тишина (канон W30).
func TestNpcW30FireForget(t *testing.T) {
	for _, cmd := range []uint16{179, 224} {
		r := NewNpcSrvReplay()
		for i := 0; i < 5; i++ {
			if b := r.EmitNpcSrv(cmd); len(b) != 0 {
				t.Fatalf("cmd=%d call#%d: тишина ожидалась, получено %dБ", cmd, i+1, len(b))
			}
		}
	}
}

// TestNpcW30FrameSanity — все кадры bin: валидный wire (len-поле = факт).
func TestNpcW30FrameSanity(t *testing.T) {
	all := SplitWorldFlowFrames(GetNpcW30Blob())
	if len(all) != 228 {
		t.Fatalf("кадров = %d", len(all))
	}
	for i, f := range all {
		total := int(binary.LittleEndian.Uint16(f[0:2]))
		if total != len(f) {
			t.Fatalf("кадр %d: len-поле %d ≠ факт %d", i, total, len(f))
		}
	}
}

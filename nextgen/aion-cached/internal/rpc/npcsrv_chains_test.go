// NPCSvr-флоу тесты (R4.7.1): цепочки из живого capture подъёма мира 22.10.
package rpc

import "testing"

func TestNpcSrvReplayChains(t *testing.T) {
	r := NewNpcSrvReplay()
	// 201 → 1035Б (0x01C SAVE-протокол)
	out := r.EmitNpcSrv(201)
	if len(out) != 1035 {
		t.Fatalf("201: len %d want 1035", len(out))
	}
	// 161 → 9Б
	if out := r.EmitNpcSrv(161); len(out) != 9 {
		t.Fatalf("161: len %d want 9", len(out))
	}
	// 211: 1-й вызов = тишина (канон capture)
	if out := r.EmitNpcSrv(211); len(out) != 0 {
		t.Fatalf("211 call#1: len %d want 0 (silent)", len(out))
	}
	// 211: 2-й вызов = 754Б (113-397+53-140+113-217)
	if out := r.EmitNpcSrv(211); len(out) != 754 {
		t.Fatalf("211 call#2: len %d want 754", len(out))
	}
	// 215 (VERSION) → 6390Б (16+123+204)
	if out := r.EmitNpcSrv(215); len(out) != 6390 {
		t.Fatalf("215: len %d want 6390", len(out))
	}
	// 248 → 22Б; 311 → 25Б
	if out := r.EmitNpcSrv(248); len(out) != 22 {
		t.Fatalf("248: len %d", len(out))
	}
	if out := r.EmitNpcSrv(311); len(out) != 25 {
		t.Fatalf("311: len %d", len(out))
	}
	// 325 → 14458Б (221×2 + 93-7949 + 93-6479)
	if out := r.EmitNpcSrv(325); len(out) != 14458 {
		t.Fatalf("325: len %d want 14458", len(out))
	}
	// 326 → 146Б
	if out := r.EmitNpcSrv(326); len(out) != 146 {
		t.Fatalf("326: len %d", len(out))
	}
	// 178 → тишина (нет capture)
	if out := r.EmitNpcSrv(178); len(out) != 0 {
		t.Fatalf("178: len %d want 0", len(out))
	}
	// неизвестный cmd → тишина
	if out := r.EmitNpcSrv(999); len(out) != 0 {
		t.Fatalf("999: len %d want 0", len(out))
	}
}

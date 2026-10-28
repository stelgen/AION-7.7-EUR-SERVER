// NPCSvr-флоу тесты (R4.6.2): канон = W30-бёрст capture 23.10 (S=0 → W30 = арбитр);
package rpc

import "testing"

func TestNpcSrvReplayChains(t *testing.T) {
	r := NewNpcSrvReplay()
	// 201 → 1035Б (0x01C SAVE-протокол)
	out := r.EmitNpcSrv(201, nil)
	if len(out) != 1035 {
		t.Fatalf("201: len %d want 1035", len(out))
	}
	// 161 → 9Б
	if out := r.EmitNpcSrv(161, nil); len(out) != 9 {
		t.Fatalf("161: len %d want 9", len(out))
	}
	// 211: 1-й вызов = тишина (канон capture)
	if out := r.EmitNpcSrv(211, nil); len(out) != 0 { // 211 гейтится W30-манифестом (22.10/23.10 канон); 24.10-вариант 537/397 = ⏳ контекст-зависим
		t.Fatalf("211 call#1: len %d want 0 (silent)", len(out))
	}
	// 211: 2-й вызов = 754Б (113-397+53-140+113-217)
	if out := r.EmitNpcSrv(211, nil); len(out) != 754 {
		t.Fatalf("211 call#2: len %d want 754", len(out))
	}
	// 215 (VERSION) → 6348Б (16+123; W30 23.10: O204 переехал в 325-батч)
	if out := r.EmitNpcSrv(215, nil); len(out) != 6348 {
		t.Fatalf("215: len %d want 6390", len(out))
	}
	// 248 → 22Б; 311 → 25Б
	if out := r.EmitNpcSrv(248, nil); len(out) != 22 {
		t.Fatalf("248: len %d", len(out))
	}
	if out := r.EmitNpcSrv(311, nil); len(out) != 25 {
		t.Fatalf("311: len %d", len(out))
	}
	// 325 → 14500Б (93×2 + 204 + 221×2; W30 23.10)
	if out := r.EmitNpcSrv(325, nil); len(out) != 14500 {
		t.Fatalf("325: len %d want 14458", len(out))
	}
	// 326 → 146Б
	if out := r.EmitNpcSrv(326, nil); len(out) != 146 {
		t.Fatalf("326: len %d", len(out))
	}
	// 178 → тишина (нет capture)
	if out := r.EmitNpcSrv(178, nil); len(out) != 0 {
		t.Fatalf("178: len %d want 0", len(out))
	}
	// неизвестный cmd → тишина
	if out := r.EmitNpcSrv(999, nil); len(out) != 0 {
		t.Fatalf("999: len %d want 0", len(out))
	}
}

// TestNpcSrvGPCrossCapture — R5 #7 КРОСС-CAPTURE сверка GP-батча:
// 23.10 (W30 блоб, мир-init УСПЕШЕН): C204/C110/C346/C253 = канон-тишина (батчи [[]]),
// C326 (один вызов) → пачка 7 кадров [218..224] = {147×3, 36, 37, 238, 222}.
// 24.10 (cdmirror-20261024 02:44:37): 5 запросов → те же 7 ответов (группировка иная,
// порядок 147-интерлива в 23.10 vs группировка в 24.10; набор идентичен по cmd/len).
// ВЫВОД: «GP-батч-блокер» снят — тишина на 204/110/346/253 канон-валид (23.10 proof),
// W30-пачка 326 уже покрывает батч-ответы.
func TestNpcSrvGPCrossCapture(t *testing.T) {
	r := NewNpcSrvReplay()
	for _, cmd := range []uint16{204, 110, 346, 253} {
		if b := r.EmitNpcSrv(cmd, nil); b != nil {
			t.Fatalf("cmd=%d: ожидалась канон-тишина (23.10), получено %dБ", cmd, len(b))
		}
	}
	// 326 → W30-пачка 7 кадров (occurrence-пачка [218..224])
	b326 := r.EmitNpcSrv(326, nil)
	if b326 == nil {
		t.Fatal("326: пачка не эмитирована")
	}
	frames := SplitWorldFlowFrames(b326)
	if len(frames) != 7 {
		t.Fatalf("326: кадров в пачке = %d, want 7", len(frames))
	}
	wantCmd := []uint16{147, 36, 37, 238, 147, 222, 147}
	for i, fr := range frames {
		cmd := uint16(fr[2]) | uint16(fr[3])<<8
		if cmd != wantCmd[i] {
			t.Fatalf("326 пачка[%d]: cmd=%d, want %d", i, cmd, wantCmd[i])
		}
	}
	// эталон 24.10 (те же cmd/len, группировка по запросам): сверка набора
	wantLens := map[uint16]int{147: 12, 36: 83, 37: 9, 238: 9, 222: 9}
	for i, fr := range frames {
		cmd := uint16(fr[2]) | uint16(fr[3])<<8
		if wantLens[cmd] != len(fr) {
			t.Fatalf("326 пачка[%d]: len=%d, want %d для cmd=%d", i, len(fr), wantLens[cmd], cmd)
		}
	}
}

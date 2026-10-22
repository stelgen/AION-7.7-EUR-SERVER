package payload

import (
	"testing"

	"aion-accache/internal/proto"
)

// Фикстуры = ПОЛНЫЕ кадры из живого capture (accountcache-ref/capture-20261007/frames.txt,
// логины accId=1010 (stelgen) и accId=7 (аккаунт "1")).

func mustFrame(t *testing.T, hex string) *proto.Frame {
	t.Helper()
	w, err := proto.ParseFrame(mustHex(t, hex))
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	if len(s)%2 != 0 {
		t.Fatal("odd hex")
	}
	b := make([]byte, len(s)/2)
	for i := range b {
		hi, lo := hexVal(s[i*2]), hexVal(s[i*2+1])
		if hi < 0 || lo < 0 {
			t.Fatal("bad hex")
		}
		b[i] = byte(hi<<4 | lo)
	}
	return b
}

func hexVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	}
	return -1
}

func hexStr(b []byte) string {
	const d = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = d[v>>4]
		out[i*2+1] = d[v&0xF]
	}
	return string(out)
}

// АКСИОМА №1: первый dword payload = accountId (1010 у stelgen, 7 у аккаунта "1").
func TestAccountIDAxiom(t *testing.T) {
	if acc, err := AccountID(mustHex(t, "f2030000")); err != nil || acc != 1010 {
		t.Fatalf("acc = %d, %v; want 1010", acc, err)
	}
	if acc, err := AccountID(mustHex(t, "07000000")); err != nil || acc != 7 {
		t.Fatalf("acc = %d, %v; want 7 (capture: аккаунт \"1\")", acc, err)
	}
}

func TestVersionFrames(t *testing.T) {
	req := mustFrame(t, "0f000100ebfeff0300000001000000")
	vr, err := ParseVersionReq(req.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if vr.A != 3 || vr.B != 1 {
		t.Fatalf("VersionReq = %+v", vr)
	}
	// Build golden: S2C маркер 0xEC, байт-в-байт с capture.
	w, err := proto.BuildS2C(1, BuildVersionResp())
	if err != nil {
		t.Fatal(err)
	}
	if hexStr(w) != "0b000100ecfeff03000000" {
		t.Fatalf("version resp wire = %s", hexStr(w))
	}
}

func TestFirstLoadReqRespGolden(t *testing.T) {
	// C2S accId=1010: 0b000400ebfbfff2030000
	req := mustFrame(t, "0b000400ebfbfff2030000")
	r, err := ParseFirstLoadReq(req.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if r.AccountID != 1010 {
		t.Fatalf("acc = %d", r.AccountID)
	}
	// C2S accId=7: 0b000400ebfbff07000000
	req2 := mustFrame(t, "0b000400ebfbff07000000")
	r2, _ := ParseFirstLoadReq(req2.Payload)
	if r2.AccountID != 7 {
		t.Fatalf("acc2 = %d", r2.AccountID)
	}
	// S2C capture (accId=1010): [accId][1][0][00][0x6AC61FCA][0000]
	w, err := proto.BuildS2C(4, BuildFirstLoadResp(1010, 1, 0, 0, 0x6AC61FCA, 0))
	if err != nil {
		t.Fatal(err)
	}
	if hexStr(w) != "1a000400ecfbfff2030000010000000000000000ca1fc66a0000" {
		t.Fatalf("firstload resp wire = %s", hexStr(w))
	}
	// S2C capture (accId=7): все нули кроме A=1.
	w2, _ := proto.BuildS2C(4, BuildFirstLoadResp(7, 1, 0, 0, 0, 0))
	if hexStr(w2) != "1a000400ecfbff07000000010000000000000000000000000000" {
		t.Fatalf("firstload resp wire(7) = %s", hexStr(w2))
	}
}

func TestBMPackRespGolden(t *testing.T) {
	w, err := proto.BuildS2C(5, BuildBMPackResp(1010, 0))
	if err != nil {
		t.Fatal(err)
	}
	if hexStr(w) != "0d000500ecfafff20300000000" {
		t.Fatalf("bmpack resp wire = %s", hexStr(w))
	}
}

func TestCharLoginReq(t *testing.T) {
	f := mustFrame(t, "53001000ebefff0100000000000000ea030002f203000032003000320036002d00310030002d00300037005400310033003a00330035003a00350035002e003800360030000000010000000200000000000000")
	if f.Cmd != 16 {
		t.Fatalf("cmd=%d", f.Cmd)
	}
	r, err := ParseCharLoginReq(f.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if r.AccountID != 1010 || r.F1 != 1 || r.F2 != 0 || r.CharRef != 0x020003EA {
		t.Fatalf("CharLoginReq = %+v", r)
	}
	if r.Stamp != "2026-10-07T13:35:55.860" {
		t.Fatalf("stamp = %q", r.Stamp)
	}
	if hexStr(r.Tail) != "010000000200000000000000" {
		t.Fatalf("tail = %s", hexStr(r.Tail))
	}
	// S2C golden: [accId][8×FF][12×0]
	w, _ := proto.BuildS2C(16, BuildCharLoginResp(1010))
	if hexStr(w) != "1f001000eceffff2030000ffffffffffffffff000000000000000000000000" {
		t.Fatalf("charlogin resp wire = %s", hexStr(w))
	}
}

func TestCharLogoutReq(t *testing.T) {
	f := mustFrame(t, "4b001100ebeeff01000000ea03000232003000320036002d00310030002d00300037005400310033003a00330036003a00310039002e003400390033000000010000000200000000000000")
	r, err := ParseCharLogoutReq(f.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if r.AccountID != 0 || r.CharRef != 0x020003EA || r.F1 != 1 {
		t.Fatalf("CharLogoutReq = %+v", r)
	}
	if r.Stamp != "2026-10-07T13:36:19.493" {
		t.Fatalf("stamp = %q", r.Stamp)
	}
}

func TestFatigueReqGolden(t *testing.T) {
	f := mustFrame(t, "1f001900ebe6fff203000000000000000000009320c66ae0dfc56a00000000")
	r, err := ParseFatigueReq(f.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if r.AccountID != 1010 || r.Point != 0 || r.NpcKill != 0 ||
		r.UpdateTime != 0x6AC62093 || r.LimitReset != 0x6AC5DFE0 || r.LimitPlay != 0 {
		t.Fatalf("FatigueReq = %+v", r)
	}
}

func TestUpdateTrialReq(t *testing.T) {
	f := mustFrame(t, "2f000800ebf7fff2030000a320c66a0000000095030000000000004600000000000000000000000000000000000000")
	r, err := ParseUpdateTrialReq(f.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if r.AccountID != 1010 || r.P1 != 0x6AC620A3 || r.P3 != 0x395 || r.P5 != 0x46 {
		t.Fatalf("UpdateTrialReq = %+v", r)
	}
}

func TestJumpingReq(t *testing.T) {
	f := mustFrame(t, "0e001f00ebe0ff01f20300000001")
	r, err := ParseJumpingReq(f.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if r.Flag != 1 || r.AccountID != 1010 || r.Value != 0x0100 { // LE-чтение '00 01' = 0x0100
		t.Fatalf("JumpingReq = %+v", r)
	}
}

func TestTransformReq(t *testing.T) {
	f := mustFrame(t, "10002500ebdafff2000000f203000000")
	r, err := ParseTransformReq(f.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if r.SubType != 0xF2 || r.AccountID != 1010 || r.Flag != 0 {
		t.Fatalf("TransformReq = %+v", r)
	}
}

func TestLunaRespGoldens(t *testing.T) {
	// cmd 27 S2C: [F2 00 01 00][accId][u16 0]
	w, err := proto.BuildS2C(27, BuildLunaUpdateResp(1010))
	if err != nil {
		t.Fatal(err)
	}
	if hexStr(w) != "11001b00ece4fff2000100f20300000000" {
		t.Fatalf("lunaupdate resp wire = %s", hexStr(w))
	}
	// cmd 26 S2C: [accId][ts][u32 0] (capture N=19)
	w2, _ := proto.BuildS2C(26, BuildLunaLoadResp(1010, 0x6AC5DFE0))
	if hexStr(w2) != "13001a00ece5fff2030000e0dfc56a00000000" {
		t.Fatalf("lunaload resp wire = %s", hexStr(w2))
	}
}

// TestR4PushGoldens — пуши ориг R4-ФИНАЛ (22.10, capture-20261022 + capture-20261007):
// полные S2C-кадры байт-в-байт с golden. Константы (0x640831AE, 0x000A07EA) и
// lastLogin=0x6AC5E7B9/maxLevel=80 — прод-факты (см. journal теорий ROADMAP Т7/Т8).
func TestR4PushGoldens(t *testing.T) {
	// ACP_13 (0x0D, после VERSION): [ts] — константа в обоих corpus.
	w, err := proto.BuildS2C(0x0d, BuildCharCustomSavedPush(CharCustomSavedTsDefault))
	if err != nil {
		t.Fatal(err)
	}
	if hexStr(w) != "0b000d00ecf2ffae310864" {
		t.Fatalf("ACP_13 wire = %s", hexStr(w))
	}
	// ACP_15 (0x0F, при CHAR_LOGIN): [charRef][0x000A07EA][3].
	w15, _ := proto.BuildS2C(0x0f, BuildCharCreatedPush(0x020003EA))
	if hexStr(w15) != "13000f00ecf0ffea030002ea070a0003000000" {
		t.Fatalf("ACP_15 wire = %s", hexStr(w15))
	}
	// ACP_20 (0x14, при CHAR_LOGIN): [charRef][16×0][lastLogin][12×0][1][8×0][flag][maxLevel].
	w20, _ := proto.BuildS2C(0x14, BuildCharInfoRefreshPush(0x020003EA, 0x6AC5E7B9, 0, 80))
	want20 := "3f001400ecebff" +
		"ea030002" + "00000000000000000000000000000000" + "b9e7c56a" +
		"000000000000000000000000" + "01000000" + "000000000000000000000000" + "50000000"
	if hexStr(w20) != want20 {
		t.Fatalf("ACP_20 wire = %s", hexStr(w20))
	}
	// CONFIRM_LUNA_REWARD (0x1C, при CHAR_LOGIN): [accId][charId][u16 0].
	w28, _ := proto.BuildS2C(0x1c, BuildConfirmLunaRewardPush(1010, 1002))
	if hexStr(w28) != "11001c00ece3fff2030000ea0300000000" {
		t.Fatalf("ACP_28 wire = %s", hexStr(w28))
	}
	// Кросс-чар: charB (0x020003EB → charId 1003) из capture-20261022 @423/@486.
	w15b, _ := proto.BuildS2C(0x0f, BuildCharCreatedPush(0x020003EB))
	if hexStr(w15b) != "13000f00ecf0ffeb030002ea070a0003000000" {
		t.Fatalf("ACP_15 charB wire = %s", hexStr(w15b))
	}
	w28b, _ := proto.BuildS2C(0x1c, BuildConfirmLunaRewardPush(1010, 1003))
	if hexStr(w28b) != "11001c00ece3fff2030000eb0300000000" {
		t.Fatalf("ACP_28 charB wire = %s", hexStr(w28b))
	}
}

// TestLunaUpdateAccLowByte — R4-live 22.10: префикс cmd 27 = [acc&0xFF][00][01][00]
// (1010→F2, 1023→FF; «константа F2» была артефактом одного аккаунта).
func TestLunaUpdateAccLowByte(t *testing.T) {
	w, _ := proto.BuildS2C(27, BuildLunaUpdateResp(1023))
	if hexStr(w) != "11001b00ece4ffff000100ff0300000000" {
		t.Fatalf("lunaupdate 1023 wire = %s", hexStr(w))
	}
}

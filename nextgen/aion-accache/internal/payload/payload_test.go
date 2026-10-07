package payload

import (
	"testing"

	"aion-accache/internal/proto"
)

// Фикстуры = ПОЛНЫЕ кадры из живого capture (frames.txt, логины 13:35/13:48).

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

func TestChannelPrefix(t *testing.T) {
	rest, ok := SplitChannel(mustHex(t, "f2030000ea030002"))
	if !ok {
		t.Fatal("want channel prefix")
	}
	if hexStr(rest) != "ea030002" {
		t.Fatalf("rest = %s", hexStr(rest))
	}
	if _, ok := SplitChannel(mustHex(t, "01000000")); ok {
		t.Fatal("want no channel prefix for 01000000")
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
	resp := mustFrame(t, "0b000100ecfeff03000000")
	vp, err := ParseVersionResp(resp.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if vp.A != 3 {
		t.Fatalf("VersionResp = %+v", vp)
	}
}

func TestFirstLoadResp(t *testing.T) {
	f := mustFrame(t, "1a000400ecfbfff2030000010000000000000000ca1fc66a0000")
	if f.Direction != proto.DirS2C {
		t.Fatal("want S2C")
	}
	r, err := ParseFirstLoadResp(f.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if r.Channel != ChannelID || r.A != 1 || r.B != 0 || r.Flag != 0 || r.Ts != 0x6AC61FCA {
		t.Fatalf("FirstLoadResp = %+v", r)
	}
}

func TestBMPackResp(t *testing.T) {
	f := mustFrame(t, "0d000500ecfafff20300000000")
	r, err := ParseBMPackResp(f.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if r.Channel != ChannelID || r.Status != 0 {
		t.Fatalf("BMPackResp = %+v", r)
	}
}

func TestLunaResp(t *testing.T) {
	f := mustFrame(t, "13001a00ece5fff2030000e0dfc56a00000000") // LOAD_LUNA S2C
	r, err := ParseLunaResp(f.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if r.Ts != 0x6AC5DFE0 {
		t.Fatalf("LunaResp = %+v", r)
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
	if r.F1 != 1 || r.F2 != 0 || r.CharRef != 0x020003EA || r.Channel != ChannelID {
		t.Fatalf("CharLoginReq = %+v", r)
	}
	if r.Stamp != "2026-10-07T13:35:55.860" {
		t.Fatalf("stamp = %q", r.Stamp)
	}
	if hexStr(r.Tail) != "010000000200000000000000" {
		t.Fatalf("tail = %s", hexStr(r.Tail))
	}
}

func TestCharLogoutReq(t *testing.T) {
	f := mustFrame(t, "4b001100ebeeff01000000ea03000232003000320036002d00310030002d00300037005400310033003a00330036003a00310039002e003400390033000000010000000200000000000000")
	r, err := ParseCharLogoutReq(f.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if r.CharRef != 0x020003EA || r.Channel != 0 {
		t.Fatalf("CharLogoutReq = %+v", r)
	}
	if r.Stamp != "2026-10-07T13:36:19.493" {
		t.Fatalf("stamp = %q", r.Stamp)
	}
}

func TestFatigueReq(t *testing.T) {
	f := mustFrame(t, "1f001900ebe6fff203000000000000000000009320c66ae0dfc56a00000000")
	r, err := ParseFatigueReq(f.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if r.Channel != ChannelID || r.Ts1 != 0x6AC62093 || r.Ts2 != 0x6AC5DFE0 {
		t.Fatalf("FatigueReq = %+v", r)
	}
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

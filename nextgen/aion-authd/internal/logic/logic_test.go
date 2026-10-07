package logic

import (
	"encoding/binary"
	"testing"
	"time"

	"aion-authd/internal/config"
	"aion-authd/internal/store"
)

func testDeps(t *testing.T) (*Deps, *store.MapStore) {
	t.Helper()
	cfg := &config.Config{}
	cfg.FillDefaults()
	cfg.OnlineTTLSec = 300
	st := store.NewMap()
	return New(cfg, st), st
}

// blob 191Б asm-формы: [00][decbuf34][dword][tail152] — как собирает гейт.
func asmBlob(user, pwd string, otp, dword uint32) []byte {
	b := []byte{0x00}
	dec := make([]byte, 34)
	copy(dec[0:14], user)
	copy(dec[14:30], pwd)
	binary.LittleEndian.PutUint32(dec[30:34], otp)
	b = append(b, dec...)
	var d [4]byte
	binary.LittleEndian.PutUint32(d[:], dword)
	b = append(b, d[:]...)
	return append(b, make([]byte, 152)...)
}

func TestLoginOkAutoCreate(t *testing.T) {
	d, _ := testDeps(t)
	ses := &GateSession{Sid: 1}
	rep := d.Login(ses, asmBlob("StelGeN", "x", 0, 0x12345678), "192.168.0.253")
	if rep == nil || rep.Typ != 3 {
		t.Fatalf("want type=3, got %+v", rep)
	}
	// К-5: normalize = trim+lower; автосоздание
	if ses.AccID != 1 || ses.User != "stelgen" {
		t.Fatalf("ses: acc=%d user=%q", ses.AccID, ses.User)
	}
	if len(rep.Payload) != 52 {
		t.Fatalf("payload: %d want 52 (probe-live)", len(rep.Payload))
	}
	// [accId][token][16×0][2000][unk1][20×0]
	if uid := binary.LittleEndian.Uint32(rep.Payload[0:4]); uid != 1 {
		t.Fatalf("uid: %d", uid)
	}
	if tok := binary.LittleEndian.Uint32(rep.Payload[4:8]); tok != ses.Token || tok == 0 {
		t.Fatalf("token: %d", tok)
	}
	// канон ориг 07.10 05:28: [accId][token][8×0][2000][unk1 Rnd][28×0]
	if max := binary.LittleEndian.Uint32(rep.Payload[16:20]); max != 2000 {
		t.Fatalf("maxUsers: %d", max)
	}
	for _, b := range rep.Payload[20:24] {
		_ = b // unk1 = random dword (ориг-канон)
	}
	if e, on := d.IsOnline("stelgen"); !on || e.UID != 1 {
		t.Fatalf("online-флаг не поставлен: %+v %v", e, on)
	}
}

func TestLoginNonASCIIFail(t *testing.T) {
	d, _ := testDeps(t)
	ses := &GateSession{Sid: 1}
	rep := d.Login(ses, asmBlob("рус", "", 0, 0), "127.0.0.1")
	if rep == nil || rep.Typ != 1 {
		t.Fatalf("want fail, got %+v", rep)
	}
	if len(rep.Payload) != 1 || rep.Payload[0] != 2 { // [code u8] — R5-дифф 07.10 (ориг = 1Б)
		t.Fatalf("payload: %x want [02]", rep.Payload)
	}
	if !rep.Close {
		t.Fatal("фейл должен помечать Close ([01][sid] как ориг)")
	}
}

func TestLoginBadBlobSilence(t *testing.T) {
	d, _ := testDeps(t)
	ses := &GateSession{Sid: 1}
	// 86Б blob (tail47) → ТИШИНА (probe-live 07.10: authd молчит на не-asm-размере)
	if rep := d.Login(ses, append([]byte{0x00}, make([]byte, 85)...), "127.0.0.1"); rep != nil {
		t.Fatalf("86Б: want silence, got %+v", rep)
	}
}

func TestLoginReloginOnlineSilence(t *testing.T) {
	d, _ := testDeps(t)
	ses1 := &GateSession{Sid: 1}
	if rep := d.Login(ses1, asmBlob("1", "1", 0, 0), "127.0.0.1"); rep == nil || rep.Typ != 3 {
		t.Fatalf("первый логин: %+v", rep)
	}
	// повторный логин онлайн-акка = ТИШИНА (live-паритет: ориг молчит, флаг TTL)
	ses2 := &GateSession{Sid: 2}
	if rep := d.Login(ses2, asmBlob("1", "2", 0, 0), "127.0.0.2"); rep != nil {
		t.Fatalf("relogin: want silence, got %+v", rep)
	}
	// и никаких новых аккаунтов
	if _, _, err := d.Store.GetOrCreate("1", false); err != nil {
		t.Fatalf("акк «1» должен существовать")
	}
}

func TestLoginReloginFail7Option(t *testing.T) {
	d, _ := testDeps(t)
	d.Cfg.ReloginPolicy = "fail7"
	ses1 := &GateSession{Sid: 1}
	d.Login(ses1, asmBlob("1", "1", 0, 0), "127.0.0.1")
	ses2 := &GateSession{Sid: 2}
	rep := d.Login(ses2, asmBlob("1", "1", 0, 0), "127.0.0.1")
	if rep == nil || rep.Typ != 1 || len(rep.Payload) != 1 || rep.Payload[0] != 7 {
		t.Fatalf("fail7: %+v", rep)
	}
}

func TestLoginBlockedFlag(t *testing.T) {
	d, st := testDeps(t)
	_, _, _ = st.GetOrCreate("bad", true) // создать, потом флагнуть
	_ = st.SetFlag("bad", func(a *store.Account) { a.BlockFlag = 1 })
	rep := d.Login(&GateSession{Sid: 1}, asmBlob("bad", "", 0, 0), "127.0.0.1")
	if rep == nil || rep.Typ != 1 || len(rep.Payload) != 1 || rep.Payload[0] != 22 {
		t.Fatalf("blocked: %+v", rep)
	}
}

func TestLoginBlockMsg(t *testing.T) {
	d, st := testDeps(t)
	acc, _, _ := st.GetOrCreate("bmsg", true)
	st.AddBlock(acc.UID, 5, "причина")
	rep := d.Login(&GateSession{Sid: 1}, asmBlob("bmsg", "", 0, 0), "127.0.0.1")
	if rep == nil || rep.Typ != 1 || len(rep.Payload) != 1 || rep.Payload[0] != 22 {
		t.Fatalf("block_msg: %+v", rep)
	}
}

func TestLoginNoAutoCreateFail(t *testing.T) {
	d, _ := testDeps(t)
	d.Cfg.AutoCreate = false
	rep := d.Login(&GateSession{Sid: 1}, asmBlob("ghost", "", 0, 0), "127.0.0.1")
	if rep == nil || rep.Typ != 1 {
		t.Fatalf("no-autocreate: %+v", rep)
	}
}

func TestSweepTTL(t *testing.T) {
	d, _ := testDeps(t)
	d.Cfg.OnlineTTLSec = 1
	d.markOnline("u", 1, 1)
	if d.SweepOnline() != 0 {
		t.Fatal("не время — снимать нельзя")
	}
	d.mu.Lock()
	d.online["u"].Since = d.online["u"].Since.Add(-2 * time.Second)
	d.mu.Unlock()
	if n := d.SweepOnline(); n != 1 {
		t.Fatalf("sweep: %d", n)
	}
}

func TestBuildType4(t *testing.T) {
	cfg := &config.Config{}
	cfg.FillDefaults()
	p := BuildType4([]byte{192, 168, 0, 125}, 7777)
	if len(p) != 31 {
		t.Fatalf("len: %d want 31 (pt 32 = [04]+payload)", len(p))
	}
	if p[0] != 1 || p[1] != 1 || p[2] != 1 || p[3] != 192 || p[4] != 168 || p[5] != 0 || p[6] != 125 {
		t.Fatalf("ip-зона: %x", p[:8])
	}
	if port := binary.LittleEndian.Uint16(p[7:9]); port != 7777 { // live 61 1e
		t.Fatalf("port: %d", port)
	}
}

func TestBuildType7(t *testing.T) {
	p := BuildType7(1)
	if len(p) != 15 {
		t.Fatalf("len: %d want 15 (pt 16 = [07]+payload)", len(p))
	}
	if p[8] != 1 {
		t.Fatalf("serverID: %d", p[8])
	}
}

func TestParseLoginBlobFields(t *testing.T) {
	b := asmBlob("stelgen", "secret", 0xFFFFFFFF, 0xDEADBEEF)
	lb, ok := ParseLoginBlob(b, true)
	if !ok || lb.User != "stelgen" || lb.Pwd != "secret" || lb.Otp != 0xFFFFFFFF || lb.Dword != 0xDEADBEEF {
		t.Fatalf("parse: %+v ok=%v", lb, ok)
	}
}

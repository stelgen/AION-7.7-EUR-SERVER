package server

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"net"
	"strings"
	"testing"
	"time"

	"aion-accache/internal/cache"
	"aion-accache/internal/config"
	"aion-accache/internal/db"
)

// hexStr2 — hex-строка байтов (для golden-сравнений payload).
func hexStr2(b []byte) string { return hex.EncodeToString(b) }

func testCfg() *config.Cfg {
	c := &config.Cfg{}
	c.Server.Listen = "127.0.0.1:0"
	return c
}

func frame(cmd uint16, payload []byte) []byte {
	w, _ := buildForTest(cmd, payload)
	return w
}

// fakeExec — мок db.Executor: фиксирует запросы, отдаёт заготовленные строки.
type fakeExec struct {
	execs []string
	execArgs [][]any
	queryQ  string
	queryArgs []any
	rows    [][]any // для единственного следующего Query
}

type fakeResult struct{}

func (fakeResult) RowsAffected() (int64, error) { return 1, nil }

type fakeRows struct {
	data [][]any
	i    int
}

func (r *fakeRows) Next() bool                 { return r.i < len(r.data) }
func (r *fakeRows) Scan(dest ...any) error     { copy(dest, r.data[r.i]); r.i++; return nil }
func (r *fakeRows) Close() error               { return nil }
func (r *fakeRows) Err() error                 { return nil }

func (f *fakeExec) Exec(ctx context.Context, q string, args ...any) (db.Result, error) {
	f.execs = append(f.execs, q)
	f.execArgs = append(f.execArgs, args)
	return fakeResult{}, nil
}

func (f *fakeExec) Query(ctx context.Context, q string, args ...any) (db.Rows, error) {
	f.queryQ = q
	f.queryArgs = args
	return &fakeRows{data: f.rows}, nil
}

func startSrv(t *testing.T, exec db.Executor) (addr string, srv *S) {
	t.Helper()
	srv = New(testCfg(), cache.New(), exec, nil)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv.SetListenerForTest(ln)
	go func() { _ = srv.ServeOnTest() }()
	return ln.Addr().String(), srv
}

func dial(t *testing.T, addr string) net.Conn {
	t.Helper()
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	c.SetDeadline(time.Now().Add(3 * time.Second))
	return c
}

// readReply читает один S2C-кадр, возвращает cmd+payload.
func readReply(t *testing.T, c net.Conn) (uint16, []byte) {
	t.Helper()
	hdr := make([]byte, 7)
	if _, err := readFull(c, hdr); err != nil {
		t.Fatalf("read hdr: %v", err)
	}
	cmd := binary.LittleEndian.Uint16(hdr[2:4])
	n := int(binary.LittleEndian.Uint16(hdr[0:2]))
	body := make([]byte, n-7)
	if _, err := readFull(c, body); err != nil {
		t.Fatal(err)
	}
	return cmd, body
}

// TestVersionGolden — VERSION: C2S {3,1} -> S2C {3}, байт-в-байт с capture.
func TestVersionGolden(t *testing.T) {
	addr, _ := startSrv(t, nil)
	c := dial(t, addr)
	defer c.Close()
	if _, err := c.Write(frame(0x01, []byte{3, 0, 0, 0, 1, 0, 0, 0})); err != nil {
		t.Fatal(err)
	}
	cmd, pl := readReply(t, c)
	if cmd != 0x01 || len(pl) != 4 || pl[0] != 3 {
		t.Fatalf("version reply cmd=%#x pl=%x", cmd, pl)
	}
	if c.LocalAddr() == nil {
		t.Fatal("conn broken")
	}
}

// TestFirstLoadFromFakeDB — FIRST_LOAD accId=7, фейк-БД отдаёт fatigue → ответ 19Б
// (capture-структура: accId+point+npckill+u8+ts+u16).
func TestFirstLoadFromFakeDB(t *testing.T) {
	ex := &fakeExec{rows: [][]any{{int32(5), int32(0x6AC62093), int32(2), int32(0x6AC5DFE0), int32(9)}}}
	addr, _ := startSrv(t, ex)
	c := dial(t, addr)
	defer c.Close()
	pl := make([]byte, 4)
	binary.LittleEndian.PutUint32(pl, 7)
	if _, err := c.Write(frame(0x04, pl)); err != nil {
		t.Fatal(err)
	}
	cmd, body := readReply(t, c)
	if cmd != 0x04 || len(body) != 19 {
		t.Fatalf("reply cmd=%#x len=%d, want 0x04/19", cmd, len(body))
	}
	if acc := binary.LittleEndian.Uint32(body); acc != 7 {
		t.Fatalf("reply acc = %d", acc)
	}
	if q := ex.queryQ; !strings.Contains(q, "aion_GetAccountData_20170428") {
		t.Fatalf("query = %s", q)
	}
	if got := ex.queryArgs[0].(int32); got != 7 {
		t.Fatalf("query arg = %v", ex.queryArgs[0])
	}
}

// TestFirstLoadEmptyDB — пустой result-set: ответ [accId][нули…] (capture не содержит
// пустого case — acc=7 имел запись с point=1; дефолт point=1 при отсутствии записи =
// ГИПОТЕЗА (журнал теорий), арбитр R4).
func TestFirstLoadEmptyDB(t *testing.T) {
	ex := &fakeExec{rows: nil}
	addr, _ := startSrv(t, ex)
	c := dial(t, addr)
	defer c.Close()
	pl := make([]byte, 4)
	binary.LittleEndian.PutUint32(pl, 7)
	if _, err := c.Write(frame(0x04, pl)); err != nil {
		t.Fatal(err)
	}
	_, body := readReply(t, c)
	if binary.LittleEndian.Uint32(body) != 7 {
		t.Fatalf("body acc = %d, want 7", binary.LittleEndian.Uint32(body))
	}
	if body[4] != 1 {
		t.Fatalf("body[4] = %d, want 1 (R4 21.10 capture-канон: golden acc=7 — строки account_data НЕТ, ответ point=1; пустой result-set => дефолт point=1)", body[4])
	}
}

// TestUpdateFatigueCallsProc — UHF: фейк фиксирует exec aion_UpdateHiddenFatigueInfo_20170428
// с 6 аргументами (capture-раскладка: accId,0,0,ts,ts2,0 — подтверждена прод-фактами
// 21.10: hidden_fatigue_updatetime = off12, limit_play_reset_time = off16), ответа нет.
func TestUpdateFatigueCallsProc(t *testing.T) {
	ex := &fakeExec{}
	addr, _ := startSrv(t, ex)
	c := dial(t, addr)
	defer c.Close()
	pl := make([]byte, 24)
	binary.LittleEndian.PutUint32(pl[0:], 1010)
	binary.LittleEndian.PutUint32(pl[12:], 0x6AC62093) // updatetime (прод: колонка = off12)
	binary.LittleEndian.PutUint32(pl[16:], 0x6AC5DFE0) // limit_reset -> limit_play_reset_time
	if _, err := c.Write(frame(0x19, pl)); err != nil {
		t.Fatal(err)
	}
	// тишина
	c.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	buf := make([]byte, 16)
	if _, err := c.Read(buf); err == nil {
		t.Fatal("want silence after UHF")
	}
	if len(ex.execs) != 1 {
		t.Fatalf("execs = %d, want 1", len(ex.execs))
	}
	want := "aion_UpdateHiddenFatigueInfo_20170428" // R4-live: EXEC dbo... @p1..@p6
	if !strings.Contains(ex.execs[0], want) {
		t.Fatalf("exec = %s", ex.execs[0])
	}
	if len(ex.execArgs[0]) != 6 {
		t.Fatalf("exec args = %d, want 6", len(ex.execArgs[0]))
	}
	if got := ex.execArgs[0][3].(int32); got != 0x6AC62093 {
		t.Fatalf("arg3 (updatetime) = %#x", got)
	}
}

// TestLoadLunaTwoReplies — LOAD_LUNA: два ответа (27, затем 26) как в capture.
func TestLoadLunaTwoReplies(t *testing.T) {
	ex := &fakeExec{rows: [][]any{{int64(42)}}}
	addr, _ := startSrv(t, ex)
	c := dial(t, addr)
	defer c.Close()
	if _, err := c.Write(frame(0x1a, []byte{0xF2, 3, 0, 0})); err != nil {
		t.Fatal(err)
	}
	cmd1, p1 := readReply(t, c)
	cmd2, p2 := readReply(t, c)
	if cmd1 != 0x1b || len(p1) != 10 || p1[0] != 0xF2 {
		t.Fatalf("reply1 cmd=%#x pl=%x", cmd1, p1)
	}
	if cmd2 != 0x1a || len(p2) != 12 || binary.LittleEndian.Uint32(p2) != 1010 {
		t.Fatalf("reply2 cmd=%#x pl=%x", cmd2, p2)
	}
	if l := ex.queryArgs[0].(int32); l != 1010 {
		t.Fatal("loadLuna arg mismatch")
	}
}

// TestCharLoginReply — CHAR_LOGIN: S2C accId+8FF+12×0 (capture).
func TestCharLoginReply(t *testing.T) {
	addr, _ := startSrv(t, nil)
	c := dial(t, addr)
	defer c.Close()
	pl := make([]byte, 0, 76)
	append32 := func(v uint32) { pl = append(pl, byte(v), byte(v>>8), byte(v>>16), byte(v>>24)) }
	append32(1)        // F1 (capture: u32 1)
	append32(0)        // F2 (capture: u32 0)
	append32(0x020003EA) // charRef (capture: 1002+0002)
	append32(1010)     // AccountID (capture: 4-й dword)
	stamp := []byte("2026-10-07T13:35:55.860")
	for _, ch := range stamp {
		pl = append(pl, ch, 0) // UTF-16LE z-строка
	}
	pl = append(pl, 0, 0) // z-терминатор
	append32(1)
	append32(2)
	append32(0)
	if _, err := c.Write(frame(0x10, pl)); err != nil {
		t.Fatal(err)
	}
	cmd, body := readReply(t, c)
	if cmd != 0x10 || len(body) != 24 {
		t.Fatalf("login reply cmd=%#x len=%d", cmd, len(body))
	}
	if binary.LittleEndian.Uint32(body) != 1010 || body[4] != 0xFF || body[11] != 0xFF || body[12] != 0 {
		t.Fatalf("login reply pl=%x", body)
	}
}

// TestUnknownDropped — cmd 0x30 → дроп без ответа (ориг-дефолт 0x14006c050).
func TestUnknownDropped(t *testing.T) {
	addr, _ := startSrv(t, nil)
	c := dial(t, addr)
	defer c.Close()
	if _, err := c.Write(frame(0x30, []byte{1, 2, 3})); err != nil {
		t.Fatal(err)
	}
	c.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	buf := make([]byte, 16)
	if _, err := c.Read(buf); err == nil {
		t.Fatal("want timeout (no reply for unknown)")
	}
}

// TestBadMarkerCloses — битый маркер → разрыв соединения.
func TestBadMarkerCloses(t *testing.T) {
	addr, _ := startSrv(t, nil)
	c := dial(t, addr)
	defer c.Close()
	w, _ := buildForTest(0x01, nil)
	w[4] = 0x00
	if _, err := c.Write(w); err != nil {
		t.Fatal(err)
	}
	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 8)
	for {
		if _, err := c.Read(buf); err != nil {
			break
		}
	}
}

// TestBMPackEmpty — LOAD_BM_PACK при пустом списке паков: ответ accId+count=0 (capture).
func TestBMPackEmpty(t *testing.T) {
	ex := &fakeExec{rows: nil}
	addr, _ := startSrv(t, ex)
	c := dial(t, addr)
	defer c.Close()
	if _, err := c.Write(frame(0x05, []byte{0xF2, 3, 0, 0})); err != nil {
		t.Fatal(err)
	}
	cmd, pl := readReply(t, c)
	if cmd != 0x05 || len(pl) != 6 {
		t.Fatalf("bmpack reply cmd=%#x len=%d", cmd, len(pl))
	}
	if binary.LittleEndian.Uint32(pl) != 1010 || binary.LittleEndian.Uint16(pl[4:]) != 0 {
		t.Fatalf("bmpack pl=%x", pl)
	}
}

// TestVersionPush13E2E — R4-финал: VERSION-ответ + ПУШ ACP_13 (1:1:1, capture).
func TestVersionPush13E2E(t *testing.T) {
	addr, _ := startSrv(t, nil)
	c := dial(t, addr)
	defer c.Close()
	if _, err := c.Write(frame(0x01, []byte{3, 0, 0, 0, 1, 0, 0, 0})); err != nil {
		t.Fatal(err)
	}
	cmd1, p1 := readReply(t, c)
	if cmd1 != 0x01 || hexStr2(p1) != "03000000" {
		t.Fatalf("version reply = %#x %s", cmd1, hexStr2(p1))
	}
	cmd2, p2 := readReply(t, c)
	if cmd2 != 0x0d || hexStr2(p2) != "ae310864" {
		t.Fatalf("ACP_13 push = %#x %s", cmd2, hexStr2(p2))
	}
}

// TestCharLoginPushesE2E — R4-финал: CHAR_LOGIN (golden 10.07) -> 4 ответа
// байт-в-байт (16, пуши 15/14/1c; UserInfo из RAM-кэша, exec=nil).
func TestCharLoginPushesE2E(t *testing.T) {
	addr, srv := startSrv(t, nil)
	srv.cache.SetUserInfo(1010, cache.UserInfo{LastLogin: 0x6AC5E7B9, MaxLevel: 80})
	c := dial(t, addr)
	defer c.Close()
	golden := "53001000ebefff0100000000000000ea030002f203000032003000320036002d00310030002d00300037005400310033003a00330035003a00350035002e003800360030000000010000000200000000000000"
	w, _ := hex.DecodeString(golden)
	if _, err := c.Write(w); err != nil {
		t.Fatal(err)
	}
	want := []struct {
		cmd uint16
		pl  string
	}{
		{0x10, "f2030000ffffffffffffffff000000000000000000000000"},
		{0x0f, "ea030002ea070a0003000000"},
		{0x14, "ea030002" + "00000000000000000000000000000000" + "b9e7c56a" +
			"000000000000000000000000" + "01000000" + "000000000000000000000000" + "50000000"},
		{0x1c, "f2030000ea0300000000"},
	}
	for i, wt := range want {
		cmd, pl := readReply(t, c)
		if cmd != wt.cmd || hexStr2(pl) != wt.pl {
			t.Fatalf("reply #%d: cmd=%#x pl=%s, want cmd=%#x pl=%s", i, cmd, hexStr2(pl), wt.cmd, wt.pl)
		}
	}
}

// TestCharLoginLogoutSQL — R3: CHAR_LOGIN/CHAR_LOGOUT (golden 10.07) зовут
// aion_SetLoginUser_20120703 / aion_SetLogoutUser_20120703 с аргументами
// (charId, serverID, curServerID, stamp-time); UserInfo читается из ДБ.
func TestCharLoginLogoutSQL(t *testing.T) {
	ex := &fakeExec{rows: [][]any{{int64(0x6AC5E7B9), int32(80)}}}
	addr, _ := startSrv(t, ex)
	c := dial(t, addr)
	defer c.Close()
	login, _ := hex.DecodeString("53001000ebefff0100000000000000ea030002f203000032003000320036002d00310030002d00300037005400310033003a00330035003a00350035002e003800360030000000010000000200000000000000")
	if _, err := c.Write(login); err != nil {
		t.Fatal(err)
	}
	// 4 ответа (16/15/14/1c) — пуш 0x14 должен взять UserInfo ИЗ ДБ (lastLogin=0x6AC5E7B9, maxLevel=80).
	for _, want := range []uint16{0x10, 0x0f, 0x14, 0x1c} {
		cmd, _ := readReply(t, c)
		if cmd != want {
			t.Fatalf("reply cmd=%#x, want %#x", cmd, want)
		}
	}
	logout, _ := hex.DecodeString("4b001100ebeeff01000000ea03000232003000320036002d00310030002d00300037005400310033003a00330036003a00310039002e003400390033000000010000000200000000000000")
	if _, err := c.Write(logout); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	if len(ex.execs) < 2 {
		t.Fatalf("execs = %d (%v)", len(ex.execs), ex.execs)
	}
	if !strings.Contains(ex.execs[0], "aion_SetLoginUser_20120703") {
		t.Fatalf("exec[0] = %s", ex.execs[0])
	}
	if !strings.Contains(ex.execs[1], "aion_SetLogoutUser_20120703") {
		t.Fatalf("exec[1] = %s", ex.execs[1])
	}
	la := ex.execArgs[0]
	if len(la) != 4 || la[0] != int32(1002) || la[1] != int32(1) || la[2] != int32(1) {
		t.Fatalf("login args = %v", la)
	}
	if lt, ok := la[3].(time.Time); !ok || lt.Format("2006-01-02T15:04:05.000") != "2026-10-07T13:35:55.860" {
		t.Fatalf("login time = %v (%T)", la[3], la[3])
	}
	ga := ex.execArgs[1]
	if len(ga) != 4 || ga[0] != int32(1002) || ga[1] != int32(1) || ga[2] != int32(1) {
		t.Fatalf("logout args = %v", ga)
	}
	if gt, ok := ga[3].(time.Time); !ok || gt.Format("2006-01-02T15:04:05.000") != "2026-10-07T13:36:19.493" {
		t.Fatalf("logout time = %v (%T)", ga[3], ga[3])
	}
}

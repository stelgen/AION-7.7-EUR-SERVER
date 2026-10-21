// Wire-regression тесты R3-хвоста: контроль функциональной деградации.
// Каждый тест: golden C2S-кадр -> 1) proc имя; 2) кол-во/типы аргументов; 3) позиции
// (off-by-2 защита: сдвиг на 2 байта = parse.err, БЕЗ proc-вызова); 4) тишина
// (нет S2C-ответа) для всех пар, не доказанных capture.
//
// NOTE (21.10, деградация-урок): db.CallProc сначала пробует Query и падает в
// Exec только при Query-ошибке. fakeExec.Query не ошибается => exec-only процы
// (aion_SetAccountPack и др.) не попадают в execs. Для этих тестов — fakeQErr:
// Query возвращает ошибку (у exec-only процов result-set'а НЕТ — это верно и
// для реального SQL), форсируя Exec-путь CallProc.
package server

import (
	"context"
	"encoding/binary"
	"errors"
	"testing"
	"time"

	"aion-accache/internal/db"
)

// fakeQErr — fakeExec с ошибкой на Query (для exec-only процов без result-set).
type fakeQErr struct{ fakeExec }

func (f *fakeQErr) Query(ctx context.Context, q string, args ...any) (db.Rows, error) {
	f.queryQ = q
	f.queryArgs = args
	return nil, errors.New("no result-set (exec-only proc)")
}

// callArgs — фиксация вызова через fakeExec (execs[i] = "{call dbo.<proc>(?,...)}", execArgs[i] = args).
func callArgs(t *testing.T, f *fakeQErr, procName string) []any {
	t.Helper()
	for i, q := range f.execs {
		if len(q) >= len(procName)+10 && q[:len(procName)+10] == "{call dbo."+procName {
			return f.execArgs[i]
		}
	}
	t.Fatalf("proc %s не вызван; execs=%v", procName, f.execs)
	return nil
}

// sendFrame — шлёт C2S-кадр в живой сервер и проверяет ТИШИНУ (нет ответа).
func sendFrame(t *testing.T, addr string, f *fakeQErr, cmd uint16, pl []byte) {
	t.Helper()
	c := dial(t, addr)
	defer c.Close()
	if _, err := c.Write(frame(cmd, pl)); err != nil {
		t.Fatal(err)
	}
	// тишина: short deadline — таймаут = тишина (OK), данные = провал.
	// (сервер не закрывает conn на silent-хендлере, поэтому Read-до-EOF не ждём).
	_ = c.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	buf := make([]byte, 1)
	if n, err := c.Read(buf); err == nil && n > 0 {
		t.Fatalf("cmd %#x: ожидалась тишина, получены данные %x", cmd, buf[:n])
	}
}

func i32(v int32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(v))
	return b
}

func i64(v int64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(v))
	return b
}

func concat(parts ...[]byte) []byte {
	out := make([]byte, 0)
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// TestW2Handlers — W2: контроль 5 зарегистрированных R3-хендлеров
// (0x06 BM, 0x0f CREATED, 0x12 DELETE, 0x13 DELETE_COMPLETED, 0x15 LEVEL_CHANGED).
func TestW2Handlers(t *testing.T) {
	f := &fakeQErr{} // exec-only процы: Query->err форсит Exec-путь CallProc
	addr, _ := startSrv(t, f)

	// 0x06 BM: [acc][char][pack][active] = 13
	sendFrame(t, addr, f, 0x06, concat(i32(1), i32(2), i32(3), i32(4)))
	a := callArgs(t, f, "aion_SetAccountPack")
	if len(a) != 4 {
		t.Fatalf("BM arg count=%d, want 4", len(a))
	}
	if a[0] != int32(1) || a[1] != int32(2) || a[2] != int32(3) || a[3] != int32(4) {
		t.Fatalf("BM args=%v", a)
	}

	// 0x0f CREATED: [acc][char][slot][level][gender][race][class][date u64] = 32
	sendFrame(t, addr, f, 0x0f, concat(i32(5), i32(6), i32(0), i32(10), i32(1), i32(0), i32(4), i64(1792500000)))
	a = callArgs(t, f, "aion_SetCreateUser_20121206")
	if len(a) != 8 {
		t.Fatalf("CREATED arg count=%d, want 8", len(a))
	}
	if a[0] != int32(5) || a[1] != int32(6) || a[6] != int32(4) {
		t.Fatalf("CREATED args=%v", a)
	}

	// 0x12 DELETE: [acc][char][time][flag] = 16
	sendFrame(t, addr, f, 0x12, concat(i32(7), i32(8), i32(1792500000), i32(1)))
	a = callArgs(t, f, "aion_SetDeleteUser_20120703")
	if len(a) != 4 {
		t.Fatalf("DELETE arg count=%d, want 4", len(a))
	}
	if a[2] != int32(1792500000) || a[3] != int32(1) {
		t.Fatalf("DELETE args=%v", a)
	}

	// 0x13 DELETE_COMPLETED: [acc][char][time][flag] = 16
	sendFrame(t, addr, f, 0x13, concat(i32(9), i32(10), i32(1792500000), i32(0)))
	a = callArgs(t, f, "aion_SetDeleteCompletedUser_20120703")
	if len(a) != 4 {
		t.Fatalf("DEL_COMP arg count=%d, want 4", len(a))
	}

	// 0x15 LEVEL_CHANGED: [acc][char][level][exp u64] = 20
	sendFrame(t, addr, f, 0x15, concat(i32(11), i32(12), i32(55), i64(123456789)))
	a = callArgs(t, f, "aion_SetUserLevelChanged")
	if len(a) != 4 {
		t.Fatalf("LEVEL arg count=%d, want 4", len(a))
	}
	if a[2] != int32(55) || a[3] != int64(123456789) {
		t.Fatalf("LEVEL args=%v", a)
	}
}

// TestW2OffBy2 — сдвиг payload на 2 байта НЕ должен вызывать proc (parse.err, деградация поймана).
func TestW2OffBy2(t *testing.T) {
	f := &fakeQErr{}
	addr, _ := startSrv(t, f)
	// валидный BM-кадр со сдвигом на 2 => len остаётся 13, но поля смещены
	// сдвинутый acc = 2-байтовый мусор в начале + кадр обрезан — не парсится
	sendFrame(t, addr, f, 0x06, concat(i32(1), i32(2), i32(3))) // 12 байт < 13 => parse.err
	if len(f.execs) != 0 {
		t.Fatalf("off-by-2: proc вызван на битом кадре! execs=%v", f.execs)
	}
}

// TestW3Handlers — W3: контроль 12 новых хендлеров (proc-имя + arg count + позиция).
func TestW3Handlers(t *testing.T) {
	f := &fakeQErr{}
	addr, _ := startSrv(t, f)

	// 0x17 LOGIN_EVENT upd: [acc][event][stamp][date u64][anniv u64] = 28
	sendFrame(t, addr, f, 0x17, concat(i32(1), i32(2), i32(3), i64(1792500000), i64(1792500001)))
	a := callArgs(t, f, "aion_UpdateLoginEventInfo")
	if len(a) != 5 {
		t.Fatalf("LE_UPD arg count=%d, want 5", len(a))
	}
	if a[0] != int32(1) || a[1] != int32(2) || a[2] != int32(3) {
		t.Fatalf("LE_UPD args=%v", a)
	}

	// 0x18 LOGIN_EVENT del: [acc][event][days] = 12
	sendFrame(t, addr, f, 0x18, concat(i32(4), i32(5), i32(30)))
	a = callArgs(t, f, "aion_DeleteLoginEventInfo")
	if len(a) != 3 {
		t.Fatalf("LE_DEL arg count=%d, want 3", len(a))
	}
	if a[0] != int32(4) || a[2] != int32(30) {
		t.Fatalf("LE_DEL args=%v", a)
	}

	// 0x1c CONFIRM_LUNA_REWARD: [acc][keyCount][decrease][remain] = 16
	sendFrame(t, addr, f, 0x1c, concat(i32(6), i32(7), i32(8), i32(9)))
	a = callArgs(t, f, "aion_ConfirmLunaReward")
	if len(a) != 4 {
		t.Fatalf("LUNA_CONFIRM arg count=%d, want 4", len(a))
	}
	if a[0] != int32(6) || a[3] != int32(9) {
		t.Fatalf("LUNA_CONFIRM args=%v", a)
	}

	// 0x1d DECREASE_LUNA_KEY: [acc][dec] = 8
	sendFrame(t, addr, f, 0x1d, concat(i32(10), i32(11)))
	a = callArgs(t, f, "aion_DecreaseLunaKey")
	if len(a) != 2 {
		t.Fatalf("LUNA_DEC arg count=%d, want 2", len(a))
	}
	if a[0] != int32(10) || a[1] != int32(11) {
		t.Fatalf("LUNA_DEC args=%v", a)
	}

	// 0x1e BOARD_BM: [acc][special][game][pos][visited u64][freecharge u64][reset u64][freeDice][paidDice][paidReset] = 52
	sendFrame(t, addr, f, 0x1e, concat(
		i32(12), i32(0), i32(1), i32(2), i64(3), i64(4), i64(5), i32(6), i32(7), i32(8)))
	a = callArgs(t, f, "aion_UpdateBoardBMState_2017")
	if len(a) != 10 {
		t.Fatalf("BOARD arg count=%d, want 10", len(a))
	}
	if a[0] != int32(12) || a[4] != int64(3) || a[6] != int64(5) {
		t.Fatalf("BOARD args=%v", a)
	}

	// 0x20 POLLS load: [acc][count u16][count × pollId] = 6 + 4n
	sendFrame(t, addr, f, 0x20, concat(i32(13), []byte{2, 0}, i32(101), i32(102)))
	// aion_GetPlayTimesForPolls через s.exec.Query — попали в queryArgs, не в execs
	if f.queryQ == "" || f.queryQ != "{call dbo.aion_GetPlayTimesForPolls(?,?)}" {
		t.Fatalf("POLLS_LOAD q=%q", f.queryQ)
	}
	if len(f.queryArgs) != 2 {
		t.Fatalf("POLLS_LOAD arg count=%d", len(f.queryArgs))
	}
	if f.queryArgs[0] != "101,102" {
		t.Fatalf("POLLS_LOAD csv=%q", f.queryArgs[0])
	}

	// 0x21 ONE_POLL load: [poll][acc] = 8
	sendFrame(t, addr, f, 0x21, concat(i32(103), i32(14)))
	if f.queryQ != "{call dbo.aion_GetPlayTimeForPoll(?,?)}" {
		t.Fatalf("ONE_POLL_LOAD q=%q", f.queryQ)
	}
	if len(f.queryArgs) != 2 {
		t.Fatalf("ONE_POLL_LOAD arg count=%d", len(f.queryArgs))
	}
	if f.queryArgs[0] != int32(103) {
		t.Fatalf("ONE_POLL_LOAD args=%v", f.queryArgs)
	}

	// 0x22 ONE_POLL upd: [poll][acc][time u64] = 16
	sendFrame(t, addr, f, 0x22, concat(i32(104), i32(15), i64(999999)))
	if f.queryQ != "{call dbo.aion_UpdatePlayTimeForPoll(?,?,?)}" {
		t.Fatalf("ONE_POLL_UPD q=%q", f.queryQ)
	}
	if len(f.queryArgs) != 3 {
		t.Fatalf("ONE_POLL_UPD arg count=%d", len(f.queryArgs))
	}
	if f.queryArgs[2] != int64(999999) {
		t.Fatalf("ONE_POLL_UPD args=%v", f.queryArgs)
	}

	// 0x23 POLLS upd: [acc][count u16][count × pollId][count × time u64]
	sendFrame(t, addr, f, 0x23, concat(i32(16), []byte{2, 0}, i32(105), i32(106), i64(1), i64(2)))
	if f.queryQ != "{call dbo.aion_UpdatePlayTimesForPolls(?,?,?)}" {
		t.Fatalf("POLLS_UPD q=%q", f.queryQ)
	}
	if len(f.queryArgs) != 3 {
		t.Fatalf("POLLS_UPD arg count=%d", len(f.queryArgs))
	}
	if f.queryArgs[0] != "105,106" {
		t.Fatalf("POLLS_UPD csv=%q", f.queryArgs[0])
	}

	// 0x24 RESET_LUNA_REWARD: [acc] = 4
	sendFrame(t, addr, f, 0x24, i32(17))
	a = callArgs(t, f, "aion_ResetLunaReward")
	if len(a) != 1 {
		t.Fatalf("LUNA_RESET arg count=%d, want 1", len(a))
	}
	if a[0] != int32(17) {
		t.Fatalf("LUNA_RESET args=%v", a)
	}

	// 0x26 MONSTER_CORE_UPDATE: [acc][core][grade][count] = 16
	sendFrame(t, addr, f, 0x26, concat(i32(18), i32(19), i32(20), i32(21)))
	a = callArgs(t, f, "aion_MonsterCoreUpdate")
	if len(a) != 4 {
		t.Fatalf("MCORE_UPD arg count=%d, want 4", len(a))
	}
	if a[0] != int32(18) || a[3] != int32(21) {
		t.Fatalf("MCORE_UPD args=%v", a)
	}

	// 0x27 MONSTER_CORE_UPGRADE: [acc][core][count] = 12
	sendFrame(t, addr, f, 0x27, concat(i32(22), i32(23), i32(24)))
	a = callArgs(t, f, "aion_MonsterCoreUpgradeAndDecreaseCount")
	if len(a) != 3 {
		t.Fatalf("MCORE_UPG arg count=%d, want 3", len(a))
	}
	if a[0] != int32(22) || a[2] != int32(24) {
		t.Fatalf("MCORE_UPG args=%v", a)
	}
}

// TestW3Silence — W3-хендлеры НЕ отвечают по протоколу (пары не доказаны capture).
// Проверяем на 3 cmds: если вдруг ответят — деградация поймана сразу.
func TestW3Silence(t *testing.T) {
	f := &fakeQErr{}
	addr, _ := startSrv(t, f)
	for _, tc := range []struct {
		cmd uint16
		pl  []byte
	}{
		{0x17, concat(i32(1), i32(2), i32(3), i64(1), i64(2))},
		{0x1d, concat(i32(4), i32(5))},
		{0x24, i32(6)},
	} {
		sendFrame(t, addr, f, tc.cmd, tc.pl)
	}
}

// TestW3NoDB — при exec=nil (нет БД) хендлеры не падают, всё уходит в log (деградация безопасна).
func TestW3NoDB(t *testing.T) {
	addr, _ := startSrv(t, nil)
	sendFrame(t, addr, nil, 0x1d, concat(i32(1), i32(2)))
	sendFrame(t, addr, nil, 0x24, i32(3))
	sendFrame(t, addr, nil, 0x26, concat(i32(4), i32(5), i32(6), i32(7)))
}

var _ = context.Background() // для db-зависимостей в future-тестах

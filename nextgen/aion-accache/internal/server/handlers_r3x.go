// R3.6 — дозаполнение T1-хендлеров по proc-телам REF58 (luna/board/polls/monstercore/login-event).
// Политика (README): ответ ТОЛЬКО где пара доказана capture; здесь все cmds НЕнаблюдены
// в capture => тишина (как ориг при дефолт-хендлере), но SQL выполняется по proc-телам.
// Раскладки payload = гипотезы (журнал теорий ROADMAP): арбитр = будущий capture/R4.
package server

import (
	"context"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"

	"aion-accache/internal/db"
)

// u32/u64 — little-endian чтение с проверкой длины.
func u32s(pl []byte, off int) (int32, bool) {
	if off+4 > len(pl) {
		return 0, false
	}
	return int32(binary.LittleEndian.Uint32(pl[off:])), true
}

func u64s(pl []byte, off int) (int64, bool) {
	if off+8 > len(pl) {
		return 0, false
	}
	return int64(binary.LittleEndian.Uint64(pl[off:])), true
}

// 0x17 UPDATE_LOGIN_EVENT_RECORD -> aion_UpdateLoginEventInfo
// (account_id INT, event_id INT, stamp_count INT, recent_count_update_date DATETIME,
//  recent_anniversity_reward_time BIGINT). Гипотеза C2S: [acc][event][stamp][date u64][anniv u64] = 28.
func hUpdateLoginEventRecord(s *S, c *Conn, cmd uint16, pl []byte) {
	acc, ok1 := u32s(pl, 0)
	event, ok2 := u32s(pl, 4)
	stamp, ok3 := u32s(pl, 8)
	datet, ok4 := u64s(pl, 12)
	anniv, ok5 := u64s(pl, 20)
	if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 {
		s.logf("loginevent upd parse.err len=%d", len(pl))
		return
	}
	if s.exec != nil {
		_, _, err := db.CallProc(context.Background(), s.exec, "aion_UpdateLoginEventInfo",
			acc, event, stamp, time.Unix(datet, 0), anniv)
		if err != nil {
			s.logf("loginevent upd db.err acc=%d err=%v", acc, err)
			s.send("db.err", "accache", err.Error())
		}
	}
	s.logf("loginevent upd acc=%d event=%d stamp=%d", acc, event, stamp)
}

// 0x18 DELETE_LOGIN_EVENT_RECORD -> aion_DeleteLoginEventInfo(account_id, event_id, days). C2S 12.
func hDeleteLoginEventRecord(s *S, c *Conn, cmd uint16, pl []byte) {
	acc, ok1 := u32s(pl, 0)
	event, ok2 := u32s(pl, 4)
	days, ok3 := u32s(pl, 8)
	if !ok1 || !ok2 || !ok3 {
		s.logf("loginevent del parse.err len=%d", len(pl))
		return
	}
	if s.exec != nil {
		_, _, err := db.CallProc(context.Background(), s.exec, "aion_DeleteLoginEventInfo", acc, event, days)
		if err != nil {
			s.logf("loginevent del db.err acc=%d err=%v", acc, err)
			s.send("db.err", "accache", err.Error())
		}
	}
	s.logf("loginevent del acc=%d event=%d days=%d", acc, event, days)
}

// 0x1c CONFIRM_LUNA_REWARD -> aion_ConfirmLunaReward(accountId, keyCount, decrease, remain). C2S 16.
// Proc возвращает код (0/-1/-2/-3) — в лог; ответа клиенту нет (пара не доказана).
func hConfirmLunaReward(s *S, c *Conn, cmd uint16, pl []byte) {
	acc, ok1 := u32s(pl, 0)
	keyCount, ok2 := u32s(pl, 4)
	decrease, ok3 := u32s(pl, 8)
	remain, ok4 := u32s(pl, 12)
	if !ok1 || !ok2 || !ok3 || !ok4 {
		s.logf("lunaconfirm parse.err len=%d", len(pl))
		return
	}
	if s.exec != nil {
		rows, res, err := db.CallProc(context.Background(), s.exec, "aion_ConfirmLunaReward", acc, keyCount, decrease, remain)
		if err != nil {
			s.logf("lunaconfirm db.err acc=%d err=%v", acc, err)
			s.send("db.err", "accache", err.Error())
		}
		if rows != nil {
			rows.Close()
		}
		if res != nil {
			ra, _ := res.RowsAffected()
			s.logf("lunaconfirm acc=%d key=%d dec=%d remain=%d rows=%d", acc, keyCount, decrease, remain, ra)
		}
	} else {
		s.logf("lunaconfirm acc=%d key=%d dec=%d remain=%d (no-db)", acc, keyCount, decrease, remain)
	}
}

// 0x1d DECREASE_LUNA_KEY -> aion_DecreaseLunaKey(accountId, decreaseKey). C2S 8.
func hDecreaseLunaKey(s *S, c *Conn, cmd uint16, pl []byte) {
	acc, ok1 := u32s(pl, 0)
	dec, ok2 := u32s(pl, 4)
	if !ok1 || !ok2 {
		s.logf("lunadec parse.err len=%d", len(pl))
		return
	}
	if s.exec != nil {
		_, _, err := db.CallProc(context.Background(), s.exec, "aion_DecreaseLunaKey", acc, dec)
		if err != nil {
			s.logf("lunadec db.err acc=%d err=%v", acc, err)
			s.send("db.err", "accache", err.Error())
		}
	}
	s.logf("lunadec acc=%d dec=%d", acc, dec)
}

// 0x1e UPDATE_BOARD_BM_STATE -> aion_UpdateBoardBMState_2017(accountId, isSpecial, gameId, currentPos,
// visitedPos BIGINT, lastFreechargeTime BIGINT, lastResetTime BIGINT, freeDice, paidDice, paidReset). C2S 52.
func hUpdateBoardBMState(s *S, c *Conn, cmd uint16, pl []byte) {
	if len(pl) < 52 {
		s.logf("boardbm parse.err len=%d", len(pl))
		return
	}
	acc := must32(pl, 0)
	special := must32(pl, 4)
	game := must32(pl, 8)
	pos := must32(pl, 12)
	visited := must64(pl, 16)
	freecharge := must64(pl, 24)
	reset := must64(pl, 32)
	freeDice := must32(pl, 40)
	paidDice := must32(pl, 44)
	paidReset := must32(pl, 48)
	if s.exec != nil {
		_, _, err := db.CallProc(context.Background(), s.exec, "aion_UpdateBoardBMState_2017",
			acc, special, game, pos, visited, freecharge, reset, freeDice, paidDice, paidReset)
		if err != nil {
			s.logf("boardbm db.err acc=%d err=%v", acc, err)
			s.send("db.err", "accache", err.Error())
		}
	}
	s.logf("boardbm acc=%d special=%d game=%d pos=%d dice=%d/%d", acc, special, game, pos, freeDice, paidDice)
}

// 0x20 LOAD_PREVIOUS_PLAYTIMES_FOR_POLLS -> aion_GetPlayTimesForPolls(pollIdsCsv NVARCHAR, accountId).
// Гипотеза C2S: [accId u32][count u16][count × pollId u32]. Ответ S2C НЕ доказан capture => тишина.
func hLoadPlayTimesForPolls(s *S, c *Conn, cmd uint16, pl []byte) {
	acc, ok1 := u32s(pl, 0)
	if !ok1 || len(pl) < 6 {
		s.logf("polls load parse.err len=%d", len(pl))
		return
	}
	count := int(binary.LittleEndian.Uint16(pl[4:6]))
	if len(pl) < 6+4*count {
		s.logf("polls load parse.err len=%d count=%d", len(pl), count)
		return
	}
	ids := make([]string, count)
	for i := 0; i < count; i++ {
		ids[i] = strconv.Itoa(int(binary.LittleEndian.Uint32(pl[6+i*4:])))
	}
	csv := strings.Join(ids, ",")
	if s.exec != nil {
		rows, err := s.exec.Query(context.Background(),
			"{call dbo.aion_GetPlayTimesForPolls(?,?)}", csv, acc)
		if err != nil {
			s.logf("polls load db.err acc=%d err=%v", acc, err)
			s.send("db.err", "accache", err.Error())
			return
		}
		if rows != nil {
			rows.Close()
		}
	}
	s.logf("polls load acc=%d ids=%s", acc, csv)
}

// 0x21 LOAD_PREVIOUS_PLAYTIME_FOR_ONE_POLL -> aion_GetPlayTimeForPoll(pollId, accountId). C2S 8.
func hLoadPlayTimeForOnePoll(s *S, c *Conn, cmd uint16, pl []byte) {
	poll, ok1 := u32s(pl, 0)
	acc, ok2 := u32s(pl, 4)
	if !ok1 || !ok2 {
		s.logf("poll load parse.err len=%d", len(pl))
		return
	}
	if s.exec != nil {
		rows, err := s.exec.Query(context.Background(),
			"{call dbo.aion_GetPlayTimeForPoll(?,?)}", poll, acc)
		if err != nil {
			s.logf("poll load db.err acc=%d poll=%d err=%v", acc, poll, err)
			s.send("db.err", "accache", err.Error())
			return
		}
		if rows != nil {
			rows.Close()
		}
	}
	s.logf("poll load acc=%d poll=%d", acc, poll)
}

// 0x22 UPDATE_PREVIOUS_PLAYTIME_FOR_ONE_POLL -> aion_UpdatePlayTimeForPoll(pollId, accountId, playTime BIGINT). C2S 16.
func hUpdatePlayTimeForOnePoll(s *S, c *Conn, cmd uint16, pl []byte) {
	poll, ok1 := u32s(pl, 0)
	acc, ok2 := u32s(pl, 4)
	t, ok3 := u64s(pl, 8)
	if !ok1 || !ok2 || !ok3 {
		s.logf("poll upd parse.err len=%d", len(pl))
		return
	}
	if s.exec != nil {
		rows, err := s.exec.Query(context.Background(),
			"{call dbo.aion_UpdatePlayTimeForPoll(?,?,?)}", poll, acc, t)
		if err != nil {
			s.logf("poll upd db.err acc=%d poll=%d err=%v", acc, poll, err)
			s.send("db.err", "accache", err.Error())
			return
		}
		if rows != nil {
			rows.Close()
		}
	}
	s.logf("poll upd acc=%d poll=%d time=%d", acc, poll, t)
}

// 0x23 UPDATE_PREVIOUS_PLAYTIMES_FOR_POLLS -> aion_UpdatePlayTimesForPolls(pollIdsCsv, playTimesCsv, accountId).
// Гипотеза C2S: [accId u32][count u16][count × pollId u32][count × playTime u64]. Тишина.
func hUpdatePlayTimesForPolls(s *S, c *Conn, cmd uint16, pl []byte) {
	acc, ok1 := u32s(pl, 0)
	if !ok1 || len(pl) < 6 {
		s.logf("polls upd parse.err len=%d", len(pl))
		return
	}
	count := int(binary.LittleEndian.Uint16(pl[4:6]))
	if len(pl) < 6+count*(4+8) {
		s.logf("polls upd parse.err len=%d count=%d", len(pl), count)
		return
	}
	ids := make([]string, count)
	times := make([]string, count)
	for i := 0; i < count; i++ {
		ids[i] = strconv.Itoa(int(binary.LittleEndian.Uint32(pl[6+i*4:])))
		times[i] = strconv.FormatInt(int64(binary.LittleEndian.Uint64(pl[6+count*4+i*8:])), 10)
	}
	idCsv, timeCsv := strings.Join(ids, ","), strings.Join(times, ",")
	if s.exec != nil {
		rows, err := s.exec.Query(context.Background(),
			"{call dbo.aion_UpdatePlayTimesForPolls(?,?,?)}", idCsv, timeCsv, acc)
		if err != nil {
			s.logf("polls upd db.err acc=%d err=%v", acc, err)
			s.send("db.err", "accache", err.Error())
			return
		}
		if rows != nil {
			rows.Close()
		}
	}
	s.logf("polls upd acc=%d ids=%s", acc, idCsv)
}

// 0x24 RESET_LUNA_REWARD -> aion_ResetLunaReward(accountId). C2S 4.
func hResetLunaReward(s *S, c *Conn, cmd uint16, pl []byte) {
	acc, ok := u32s(pl, 0)
	if !ok {
		s.logf("lunareset parse.err len=%d", len(pl))
		return
	}
	if s.exec != nil {
		if _, _, err := db.CallProc(context.Background(), s.exec, "aion_ResetLunaReward", acc); err != nil {
			s.logf("lunareset db.err acc=%d err=%v", acc, err)
			s.send("db.err", "accache", err.Error())
		}
	}
	s.logf("lunareset acc=%d", acc)
}

// 0x26 MONSTER_CORE_UPDATE_VALUE -> aion_MonsterCoreUpdate(accountId, coreId, coreGrade, coreCount). C2S 16.
func hMonsterCoreUpdate(s *S, c *Conn, cmd uint16, pl []byte) {
	acc, ok1 := u32s(pl, 0)
	core, ok2 := u32s(pl, 4)
	grade, ok3 := u32s(pl, 8)
	cnt, ok4 := u32s(pl, 12)
	if !ok1 || !ok2 || !ok3 || !ok4 {
		s.logf("mcore upd parse.err len=%d", len(pl))
		return
	}
	if s.exec != nil {
		if _, _, err := db.CallProc(context.Background(), s.exec, "aion_MonsterCoreUpdate", acc, core, grade, cnt); err != nil {
			s.logf("mcore upd db.err acc=%d err=%v", acc, err)
			s.send("db.err", "accache", err.Error())
		}
	}
	s.logf("mcore upd acc=%d core=%d grade=%d cnt=%d", acc, core, grade, cnt)
}

// 0x27 MONSTER_CORE_UPGRADE -> aion_MonsterCoreUpgradeAndDecreaseCount(accountId, coreId, countForUpgrade). C2S 12.
func hMonsterCoreUpgrade(s *S, c *Conn, cmd uint16, pl []byte) {
	acc, ok1 := u32s(pl, 0)
	core, ok2 := u32s(pl, 4)
	cnt, ok3 := u32s(pl, 8)
	if !ok1 || !ok2 || !ok3 {
		s.logf("mcore upg parse.err len=%d", len(pl))
		return
	}
	if s.exec != nil {
		if _, _, err := db.CallProc(context.Background(), s.exec, "aion_MonsterCoreUpgradeAndDecreaseCount", acc, core, cnt); err != nil {
			s.logf("mcore upg db.err acc=%d err=%v", acc, err)
			s.send("db.err", "accache", err.Error())
		}
	}
	s.logf("mcore upg acc=%d core=%d cnt=%d", acc, core, cnt)
}

// must32/must64 — для фиксированных раскладок (длина уже проверена).
func must32(pl []byte, off int) int32 {
	return int32(binary.LittleEndian.Uint32(pl[off:]))
}

func must64(pl []byte, off int) int64 {
	return int64(binary.LittleEndian.Uint64(pl[off:]))
}

// shipLog — ship-событие прокорма деградаций (используется тестами).
func (s *S) shipLog(ev, msg string) {
	s.send(ev, "accache", fmt.Sprint(msg))
}

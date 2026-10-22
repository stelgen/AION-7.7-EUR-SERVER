// Package db: DB-слой ACS (R3). Тела процедур сняты с REF58_AionAccountCacheD
// (accountcache-ref/db-procs-77-ref58.rpt); вызовы = {call dbo.<proc>(?, ...)} позиционно.
// SQLStore — go-mssqldb; Executor — абстракция для фейка в тестах.
// connStr = СЕКРЕТ: только на VM/env (S6), в гите плейсхолдеры.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/microsoft/go-mssqldb"
)

// Rows/Result — собственные интерфейсы (моки в тестах без sqlmock).
type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Close() error
	Err() error
}

type Result interface {
	RowsAffected() (int64, error)
}

// Executor — абстракция БД (фейк в тестах / SQLStore в проде).
type Executor interface {
	Exec(ctx context.Context, query string, args ...any) (Result, error)
	Query(ctx context.Context, query string, args ...any) (Rows, error)
}

// SQLStore — реальная БД (go-mssqldb).
type SQLStore struct{ db *sql.DB }

type sqlResult struct{ r sql.Result }

func (s sqlResult) RowsAffected() (int64, error) { return s.r.RowsAffected() }

type sqlRows struct{ r *sql.Rows }

func (s sqlRows) Next() bool                { return s.r.Next() }
func (s sqlRows) Scan(dest ...any) error    { return s.r.Scan(dest...) }
func (s sqlRows) Close() error              { return s.r.Close() }
func (s sqlRows) Err() error                { return s.r.Err() }

func NewSQLStore(conn string) (*SQLStore, error) {
	d, err := sql.Open("sqlserver", conn)
	if err != nil {
		return nil, fmt.Errorf("accache db: %w", err)
	}
	d.SetMaxOpenConns(10) // common.xml numberOfDBThreads=10
	if err := d.Ping(); err != nil {
		d.Close()
		return nil, fmt.Errorf("accache db ping: %w", err)
	}
	return &SQLStore{db: d}, nil
}

func (s *SQLStore) Exec(ctx context.Context, q string, args ...any) (Result, error) {
	r, err := s.db.ExecContext(ctx, q, args...)
	return sqlResult{r}, err
}

func (s *SQLStore) Query(ctx context.Context, q string, args ...any) (Rows, error) {
	r, err := s.db.QueryContext(ctx, q, args...)
	return sqlRows{r}, err
}

// readOnly — обёртка Executor для тени A/B (FORK-SPEC): Query проходит (чтение —
// включая UserInfo для байт-паритета пушей), Exec (пись в БД) ГЛУШИТСЯ — прод-БД
// не получает непроверенные раскладки. Включается конфигом db.read_only.
type readOnly struct{ ex Executor }

type nopResult struct{}

func (nopResult) RowsAffected() (int64, error) { return 0, nil }

func (r readOnly) Exec(ctx context.Context, q string, args ...any) (Result, error) {
	return nopResult{}, nil
}

func (r readOnly) Query(ctx context.Context, q string, args ...any) (Rows, error) {
	return r.ex.Query(ctx, q, args...)
}

// NewReadOnly — глушит записи, чтения — как есть.
func NewReadOnly(ex Executor) Executor { return readOnly{ex} }

// CallProc — универсальный вызов процедуры: CallProc(ex, "aion_loadLuna", accId).
// Если proc без result-set — возвращается Result (rows=nil).
func CallProc(ctx context.Context, ex Executor, name string, args ...any) (Rows, Result, error) {
	// R4-live 22.10: go-mssqldb НЕ поддерживает ODBC-эскейп {call ...} (mssql: Incorrect
	// syntax near '{') — эталон синтаксиса = aion-authd internal/store/mssql.go (EXEC).
	// Позиционные @p1..@pN — драйвер биндит по порядку.
	q := "EXEC dbo." + name
	if len(args) > 0 {
		parts := make([]string, len(args))
		for i := range args {
			parts[i] = fmt.Sprintf("@p%d", i+1)
		}
		q += " " + strings.Join(parts, ", ")
	}
	rows, err := ex.Query(ctx, q, args...)
	if err != nil {
		if res, e2 := ex.Exec(ctx, q, args...); e2 == nil {
			return nil, res, nil
		}
		return nil, nil, fmt.Errorf("accache db %s: %w", name, err)
	}
	return rows, nil, nil
}

// Fatigue — 5 колонок aion_GetAccountData_20170428.
type Fatigue struct {
	Point, UpdateTime, NpcKill, LimitReset, LimitAccum int32
}

// GetAccountData — proc aion_GetAccountData_20170428(accountId):
// SELECT hidden_fatigue_point, hidden_fatigue_updatetime, isnull(hidden_fatigue_npckill,0),
//        limit_play_reset_time, limit_play_accum_time FROM account_data WHERE account_id=?
// Пустой result-set = аккаунта нет (nil, nil).
func GetAccountData(ctx context.Context, ex Executor, accountID int32) (*Fatigue, error) {
	rows, err := ex.Query(ctx, "EXEC dbo.aion_GetAccountData_20170428 @accountId=@p1", accountID)
	if err != nil {
		return nil, fmt.Errorf("accache db GetAccountData: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	var f Fatigue
	if err := rows.Scan(&f.Point, &f.UpdateTime, &f.NpcKill, &f.LimitReset, &f.LimitAccum); err != nil {
		return nil, err
	}
	return &f, rows.Err()
}

// UpdateHiddenFatigueInfo — proc aion_UpdateHiddenFatigueInfo_20170428:
// (accountId, hidden_fatigue_point, hidden_fatigue_npcKill, hidden_fatigue_updatetime,
//  limit_reset_time, limit_play_time).
func UpdateHiddenFatigueInfo(ctx context.Context, ex Executor, accID, point, npckill, updatetime, limitReset, limitPlay int32) error {
	_, err := ex.Exec(ctx,
		"EXEC dbo.aion_UpdateHiddenFatigueInfo_20170428 @p1, @p2, @p3, @p4, @p5, @p6",
		accID, point, npckill, updatetime, limitReset, limitPlay)
	if err != nil {
		return fmt.Errorf("accache db UpdateHiddenFatigueInfo: %w", err)
	}
	return nil
}

// PackRow — строка aion_GetAccountPackList(accountId): pack_type, expire_date.
type PackRow struct {
	PackType   int32
	ExpireDate sql.NullString
}

func GetAccountPackList(ctx context.Context, ex Executor, accountID int32) ([]PackRow, error) {
	rows, err := ex.Query(ctx, "EXEC dbo.aion_GetAccountPackList @accountId=@p1", accountID)
	if err != nil {
		return nil, fmt.Errorf("accache db GetAccountPackList: %w", err)
	}
	defer rows.Close()
	var out []PackRow
	for rows.Next() {
		var p PackRow
		if err := rows.Scan(&p.PackType, &p.ExpireDate); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetLuna — proc aion_loadLuna(accid): SELECT luna FROM account_luna WHERE accountId=?.
// Нет строки = 0 (луна по умолчанию).
func GetLuna(ctx context.Context, ex Executor, accountID int32) (int64, error) {
	rows, err := ex.Query(ctx, "EXEC dbo.aion_loadLuna @p1", accountID)
	if err != nil {
		return 0, fmt.Errorf("accache db GetLuna: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return 0, rows.Err()
	}
	var l int64
	if err := rows.Scan(&l); err != nil {
		return 0, err
	}
	return l, rows.Err()
}

// TrialAccountData — колонки aion_GetTrialAccountData (7.7 = 6 колонок,
// прод-факт 21.10: reset_time, sell_gold_sum, trade_gold_sum, decompose_sum,
// gather_sum, extract_gather_sum — UpdateTime-колонки в proc НЕТ).
type TrialAccountData struct {
	ResetTime, SellGold, TradeGold             int64
	DecomposeSum, GatherSum, ExtractGatherSum  int32
}

func GetTrialAccountData(ctx context.Context, ex Executor, accountID int32) (*TrialAccountData, error) {
	rows, err := ex.Query(ctx, "{call dbo.aion_GetTrialAccountData(?)}", accountID)
	if err != nil {
		return nil, fmt.Errorf("accache db GetTrialAccountData: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	var t TrialAccountData
	if err := rows.Scan(&t.ResetTime, &t.SellGold, &t.TradeGold,
		&t.DecomposeSum, &t.GatherSum, &t.ExtractGatherSum); err != nil {
		return nil, err
	}
	return &t, rows.Err()
}

// SetTrialAccountData — proc aion_SetTrialAccountData(accountId, updateTime, resetTime,
// sellGold bigint, tradeGold bigint, decomposeSum, gatherSum, extractGatherSum).
// Раскладка C2S cmd 8 = гипотеза (см. payload.UpdateTrialReq), арбитр R4.
func SetTrialAccountData(ctx context.Context, ex Executor, accID int32, p *UpdateTrialArgs) error {
	_, err := ex.Exec(ctx,
		"EXEC dbo.aion_SetTrialAccountData @p1, @p2, @p3, @p4, @p5, @p6, @p7, @p8",
		accID, p.UpdateTime, p.ResetTime, p.SellGold, p.TradeGold, p.DecomposeSum, p.GatherSum, p.ExtractGatherSum)
	if err != nil {
		return fmt.Errorf("accache db SetTrialAccountData: %w", err)
	}
	return nil
}

// UpdateTrialArgs — аргументы aion_SetTrialAccountData (гипотеза ролей полей cmd 8).
type UpdateTrialArgs struct {
	UpdateTime, ResetTime       int32
	SellGold, TradeGold         int64
	DecomposeSum, GatherSum, ExtractGatherSum int32
}

// --- R3: global_user_data (UserInfo для пуша CHAR_INFO_REFRESH + login/logout) ---

// UserInfo — срез global_user_data (R3/R4-финал 22.10).
//   LastLogin = last_login_time строки чара (unix sec; Т10 ⏳: RAM-race ориг);
//   MaxLevel  = MAX(user_level) по НОРМ-серверам аккаунта (Т11 ✅: паттерн
//   maxLevAmongNormalServers из proc user_board_bm; у чара 1003 level=1, в пушах 80).
type UserInfo struct {
	LastLogin int64
	MaxLevel  int32
}

// GetUserInfo — ПРЯМОЙ SELECT (read-proc для этого среза в списке 101 procs REF58 НЕТ —
// проверено по db-procs-77-ref58.rpt; writers = aion_SetLoginUser*/aion_SetUserInfo*).
// last_login_time возвращается как DATEDIFF(s,'1970-01-01',...) — unix-секунды, скан int64.
// Пустой result-set = ok=false (строки нет — эмиттер берёт RAM-кэш/нули; кэш НЕ перезаписываем).
func GetUserInfo(ctx context.Context, ex Executor, accID, charID int32) (UserInfo, bool, error) {
	// R4-live: именованные @p1..@p3 (go-mssqldb; '?' без перезаписи = syntax err).
	q := "SELECT DATEDIFF(s,'1970-01-01',g.last_login_time)," +
		" (SELECT ISNULL(MAX(user_level),0) FROM global_user_data WITH (NOLOCK) WHERE account_id = @p1 AND is_special_server = 0 AND delete_date = 0)" +
		" FROM global_user_data g WITH (NOLOCK)" +
		" WHERE g.account_id = @p2 AND g.char_id = @p3 AND g.delete_date = 0"
	rows, err := ex.Query(ctx, q, accID, accID, charID)
	if err != nil {
		return UserInfo{}, false, fmt.Errorf("accache db GetUserInfo: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return UserInfo{}, false, rows.Err()
	}
	var ui UserInfo
	if err := rows.Scan(&ui.LastLogin, &ui.MaxLevel); err != nil {
		return UserInfo{}, false, err
	}
	return ui, true, rows.Err()
}

// SetLoginUser — proc aion_SetLoginUser_20120703(char_id, server_id, cur_server_id,
// login_time datetime): UPDATE global_user_data SET last_login_time=@login_time,
// cur_server_id=@cur, last_logout_time=@login_time WHERE char_id AND server_id.
// Варианты 20121120/20121206 (глобальный global_char_id/class_type/race_type) = Т15 ⏳
// (источник аргументов не доказан; 20120703 не портит колонки, арбитр = live A/B SQL-дифф).
func SetLoginUser(ctx context.Context, ex Executor, charID, serverID, curServerID int32, loginTime time.Time) error {
	_, err := ex.Exec(ctx, "EXEC dbo.aion_SetLoginUser_20120703 @char_id=@p1, @server_id=@p2, @cur_server_id=@p3, @login_time=@p4",
		charID, serverID, curServerID, loginTime)
	if err != nil {
		return fmt.Errorf("accache db SetLoginUser: %w", err)
	}
	return nil
}

// SetLogoutUser — proc aion_SetLogoutUser_20120703(char_id, server_id, cur_server_id,
// logout_time datetime): UPDATE last_logout_time=@logout_time, cur_server_id=@cur
// WHERE char_id AND server_id. Вариант 20121206 (+class/race) = Т15 ⏳.
func SetLogoutUser(ctx context.Context, ex Executor, charID, serverID, curServerID int32, logoutTime time.Time) error {
	_, err := ex.Exec(ctx, "EXEC dbo.aion_SetLogoutUser_20120703 @char_id=@p1, @server_id=@p2, @cur_server_id=@p3, @logout_time=@p4",
		charID, serverID, curServerID, logoutTime)
	if err != nil {
		return fmt.Errorf("accache db SetLogoutUser: %w", err)
	}
	return nil
}

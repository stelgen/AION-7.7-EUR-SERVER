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

// CallProc — универсальный вызов процедуры: CallProc(ex, "aion_loadLuna", accId).
// Если proc без result-set — возвращается Result (rows=nil).
func CallProc(ctx context.Context, ex Executor, name string, args ...any) (Rows, Result, error) {
	q := "{call dbo." + name
	if len(args) > 0 {
		q += "(" + strings.Repeat("?,", len(args))
		q = q[:len(q)-1] + ")}"
	} else {
		q += ")}"
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
	rows, err := ex.Query(ctx, "{call dbo.aion_GetAccountData_20170428(?)}", accountID)
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
		"{call dbo.aion_UpdateHiddenFatigueInfo_20170428(?,?,?,?,?,?)}",
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
	rows, err := ex.Query(ctx, "{call dbo.aion_GetAccountPackList(?)}", accountID)
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
	rows, err := ex.Query(ctx, "{call dbo.aion_loadLuna(?)}", accountID)
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

// TrialAccountData — колонки aion_GetTrialAccountData.
type TrialAccountData struct {
	UpdateTime, ResetTime                       int32
	SellGold, TradeGold                         int64
	DecomposeSum, GatherSum, ExtractGatherSum   int32
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
	if err := rows.Scan(&t.UpdateTime, &t.ResetTime, &t.SellGold, &t.TradeGold,
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
		"{call dbo.aion_SetTrialAccountData(?,?,?,?,?,?,?,?)}",
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

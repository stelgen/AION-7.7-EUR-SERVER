// Package db: DB-слой ACS. Тела процедур сняты с REF58_AionAccountCacheD
// (accountcache-ref/db-procs-77-ref58.rpt). SQL = {call ...} позиционно.
// SQLStore — R3-полировка; здесь каркас с ключевыми proc-вызовами.
package db

import (
	"context"
	"database/sql"
	"fmt"
)

// Executor — абстракция БД (фейк в тестах / sql.DB в проде).
type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// Fatigue — результат aion_GetAccountData_20170428.
type Fatigue struct {
	Point, UpdateTime, NpcKill, LimitReset, LimitAccum int32
}

// SQL — имена/вызовы процедур (позиционные, как зовёт бинарь).
var (
	SQLGetAccountData    = "{call dbo.aion_GetAccountData_20170428(?)}"
	SQLSetAccountData    = "{call dbo.aion_SetAccountData(?,?,?,?,?)}"
	SQLGetAccountPackList = "{call dbo.aion_GetAccountPackList(?)}"
	SQLSetAccountPack    = "{call dbo.aion_SetAccountPack(?,?,?)}"
	SQLSetLoginUser      = "{call dbo.aion_SetLoginUser_20121206(?,?)}"
	SQLSetLogoutUser     = "{call dbo.aion_SetLogoutUser_20121206(?,?)}"
	SQLSetCreateUser     = "{call dbo.aion_SetCreateUser_20160303(?)}"
)

// GetAccountData — proc aion_GetAccountData_20170428(accountId):
// SELECT hidden_fatigue_point, hidden_fatigue_updatetime, isnull(hidden_fatigue_npckill,0),
//        limit_play_reset_time, limit_play_accum_time FROM account_data WHERE account_id=?
func GetAccountData(ctx context.Context, ex Executor, accountID int) (*Fatigue, error) {
	rows, err := ex.QueryContext(ctx, SQLGetAccountData, accountID)
	if err != nil {
		return nil, fmt.Errorf("accache db: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil // нет записи = нет аккаунта (как ориг: пустой result-set)
	}
	var f Fatigue
	if err := rows.Scan(&f.Point, &f.UpdateTime, &f.NpcKill, &f.LimitReset, &f.LimitAccum); err != nil {
		return nil, err
	}
	return &f, rows.Err()
}

// GetAccountPackList — proc aion_GetAccountPackList(accountId): pack_type, expire_date.
type PackRow struct {
	PackType   int
	ExpireDate sql.NullString
}

func GetAccountPackList(ctx context.Context, ex Executor, accountID int) ([]PackRow, error) {
	rows, err := ex.QueryContext(ctx, SQLGetAccountPackList, accountID)
	if err != nil {
		return nil, err
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

// SetLoginUser / SetLogoutUser / SetCreateUser — каркасные обёртки (сигнатуры сверить в R3).
func SetLoginUser(ctx context.Context, ex Executor, args ...any) error {
	_, err := ex.ExecContext(ctx, SQLSetLoginUser, args...)
	return err
}
func SetLogoutUser(ctx context.Context, ex Executor, args ...any) error {
	_, err := ex.ExecContext(ctx, SQLSetLogoutUser, args...)
	return err
}
func SetCreateUser(ctx context.Context, ex Executor, args ...any) error {
	_, err := ex.ExecContext(ctx, SQLSetCreateUser, args...)
	return err
}

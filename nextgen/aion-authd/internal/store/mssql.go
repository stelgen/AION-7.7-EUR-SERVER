//go:build !nomssql

package store

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	_ "github.com/microsoft/go-mssqldb" // driver "sqlserver"
)

// SQLStore — Store поверх AionAccounts (database/sql + go-mssqldb).
//
// Дефолтные запросы = тела C1-проц (ap_GStat/ap_SLog/MakeBlockInfo) без OUTPUT-
// параметров — прямые SELECT/UPDATE по схеме ReleaseAuthDBSchema.sql. Реальные
// имена таблиц/проц нашего AionAccounts сверяются на R0 (sp_helptext) и
// переопределяются в конфиге (qAccount/qInsert/qBlocks/qLogLogin).
//
// ConnStr — СЕКРЕТ: только config на VM / env AUTHD_CONNSTR.
type SQLStore struct {
	db *sql.DB

	qAccount  string // @account → uid, pay_stat, login_flag, warn_flag, block_flag, block_flag2, last_world
	qInsert   string // @account → uid (SCOPE_IDENTITY)
	qBlocks   string // @uid → reason, msg
	qLogLogin string // @uid, @now, @ip

	mu     sync.Mutex
	closed bool
}

const (
	defQAccount = `SELECT uid, pay_stat, login_flag, warn_flag, block_flag, block_flag2, ISNULL(last_world, 0)
FROM user_account WITH (NOLOCK) WHERE account = @account`
	defQInsert = `INSERT INTO user_account (account, pay_stat, login_flag, warn_flag, block_flag, block_flag2)
VALUES (@account, 0, 0, 0, 0, 0);
SELECT CAST(SCOPE_IDENTITY() AS int);`
	defQBlocks   = `SELECT reason, msg FROM block_msg WITH (NOLOCK) WHERE uid = @uid`
	defQLogLogin = `UPDATE user_account SET last_login = @now, last_ip = @ip WHERE uid = @uid`
)

// ErrNotFound — акка нет в БД (и autoCreate выключен).
var ErrNotFound = errors.New("store: аккаунт не найден")

// NewSQL — открыть пул. connStr секретный (не логировать!).
func NewSQL(driver, connStr string) (*SQLStore, error) {
	if connStr == "" {
		return nil, errors.New("store: пустой connStr (config db.connStr / env AUTHD_CONNSTR)")
	}
	d := driver
	if d == "" || d == "mssql" {
		d = "sqlserver" // go-mssqldb
	}
	db, err := sql.Open(d, connStr)
	if err != nil {
		return nil, fmt.Errorf("store: %w", err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Minute)
	return &SQLStore{
		db:        db,
		qAccount:  defQAccount,
		qInsert:   defQInsert,
		qBlocks:   defQBlocks,
		qLogLogin: defQLogLogin,
	}, nil
}

// SetQueries — переопределение SQL из конфига (пустые не трогаем).
func (s *SQLStore) SetQueries(qAccount, qInsert, qBlocks, qLogLogin string) {
	if qAccount != "" {
		s.qAccount = qAccount
	}
	if qInsert != "" {
		s.qInsert = qInsert
	}
	if qBlocks != "" {
		s.qBlocks = qBlocks
	}
	if qLogLogin != "" {
		s.qLogLogin = qLogLogin
	}
}

// GetOrCreate — SELECT по account; нет строк + autoCreate → INSERT + SCOPE_IDENTITY.
func (s *SQLStore) GetOrCreate(name string, autoCreate bool) (Account, bool, error) {
	var uid, pay, lf, wf, bf, bf2 uint32
	var lw uint8
	err := s.db.QueryRow(s.qAccount, sql.Named("account", name)).
		Scan(&uid, &pay, &lf, &wf, &bf, &bf2, &lw)
	switch {
	case err == nil:
		return Account{UID: uid, Name: name, PayStat: pay, LoginFlag: lf, WarnFlag: wf,
			BlockFlag: bf, BlockFlag2: bf2, LastWorld: lw}, false, nil
	case errors.Is(err, sql.ErrNoRows):
		if !autoCreate {
			return Account{}, false, ErrNotFound
		}
		return s.create(name)
	default:
		return Account{}, false, fmt.Errorf("store: qAccount: %w", err)
	}
}

func (s *SQLStore) create(name string) (Account, bool, error) {
	var uid int32
	if err := s.db.QueryRow(s.qInsert, sql.Named("account", name)).Scan(&uid); err != nil {
		return Account{}, false, fmt.Errorf("store: qInsert: %w", err)
	}
	return Account{UID: uint32(uid), Name: name}, true, nil
}

// Blocks — block_msg по uid (C1: MakeBlockInfo — reason+msg).
func (s *SQLStore) Blocks(uid uint32) ([]BlockReason, error) {
	rows, err := s.db.Query(s.qBlocks, sql.Named("uid", int32(uid)))
	if err != nil {
		return nil, fmt.Errorf("store: qBlocks: %w", err)
	}
	defer rows.Close()
	var out []BlockReason
	for rows.Next() {
		var r BlockReason
		if err := rows.Scan(&r.Code, &r.Msg); err != nil {
			return nil, fmt.Errorf("store: qBlocks scan: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// LogLogin — ap_SLog-аналог (last_login/last_ip).
func (s *SQLStore) LogLogin(a Account, ip string) error {
	_, err := s.db.Exec(s.qLogLogin,
		sql.Named("uid", int32(a.UID)),
		sql.Named("now", time.Now().UTC()),
		sql.Named("ip", ip))
	if err != nil {
		return fmt.Errorf("store: qLogLogin: %w", err)
	}
	return nil
}

// Close — закрыть пул.
func (s *SQLStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	return s.db.Close()
}

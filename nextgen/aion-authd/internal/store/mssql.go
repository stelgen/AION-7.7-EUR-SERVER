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
// Дефолты (09.10, R0-procs: тела реальных procs сняты → authd-ref/procs-aionaccounts-77.rpt):
//   - qAccount = ap_GPwdWithFlag (автосоздание ВНУТРИ SQL: нет акка + ASCII → ap_AutoReg,
//     flag=3, pwd=0x00×16; не-ASCII = NULL-выход) + ap_GStat (uid/flags + last_login fix);
//   - qInsert  = ap_AutoReg;
//   - qBlocks  = ap_GetRestriction (block_msg: reason, msg);
//   - qLogLogin = прямой UPDATE last_login/last_ip (эффект «maddaemon fix 08» внутри ap_GStat);
//     ap_SLog (полный last_login/logout/world/game/ip) — проца логаута, ключ qLogout (T2).
// Переопределяются в конфиге (qAccount/qInsert/qBlocks/qLogLogin).
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
	// Комбинированный вызов реальных проц логина (R0-контракт 09.10).
	// Строка ВСЕГДА одна; pwd=NULL → акка нет и автосоздание не сработало (не-ASCII).
	defQAccount = `DECLARE @pwd binary(16), @flag tinyint, @otp tinyint;
EXEC dbo.ap_GPwdWithFlag @account=@account, @pwd=@pwd OUTPUT, @flag=@flag OUTPUT, @otpflag=@otp OUTPUT;
DECLARE @uid int, @pay int, @lf int, @wf int, @bf int, @bf2 int, @sub int, @lw tinyint, @bed datetime, @fs binary(16), @v12 binary(16);
EXEC dbo.ap_GStat @account=@account, @uid=@uid OUTPUT, @payStat=@pay OUTPUT, @loginFlag=@lf OUTPUT,
@warnFlag=@wf OUTPUT, @blockFlag=@bf OUTPUT, @blockFlag2=@bf2 OUTPUT, @subFlag=@sub OUTPUT,
@lastworld=@lw OUTPUT, @block_end_date=@bed OUTPUT, @forbidden_servers=@fs OUTPUT, @VAR12=@v12 OUTPUT;
SELECT CAST(@uid AS int), CAST(@pay AS int), CAST(@lf AS int), CAST(@wf AS int), CAST(@bf AS int),
CAST(@bf2 AS int), ISNULL(@lw, 0), CAST(@flag AS int), @pwd;`
	defQInsert   = `EXEC dbo.ap_AutoReg @account=@account;`
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

// GetOrCreate — реальный путь ориг: ap_GPwdWithFlag (автосоздание внутри SQL) + ap_GStat.
// created = свежий AutoReg (flag=3, pwd нулевой); pwd=NULL (не-ASCII, не создан) → ErrNotFound.
func (s *SQLStore) GetOrCreate(name string, autoCreate bool) (Account, bool, error) {
	var uid, pay, lf, wf, bf, bf2, flag sql.NullInt32
	var lw sql.NullInt32
	var pwd []byte
	err := s.db.QueryRow(s.qAccount, sql.Named("account", name)).
		Scan(&uid, &pay, &lf, &wf, &bf, &bf2, &lw, &flag, &pwd)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) && !autoCreate {
			return Account{}, false, ErrNotFound
		}
		return Account{}, false, fmt.Errorf("store: qAccount: %w", err)
	}
	if !uid.Valid || pwd == nil { // не существует и автосоздание не сработало (не-ASCII логин)
		return Account{}, false, ErrNotFound
	}
	created := flag.Valid && flag.Int32 == 3 && allZero(pwd) // свежий ap_AutoReg (live: pwd=0x00×16, flag=3)
	return Account{
		UID: uint32(uid.Int32), Name: name,
		PayStat: u32(pay), LoginFlag: u32(lf), WarnFlag: u32(wf),
		BlockFlag: u32(bf), BlockFlag2: u32(bf2), LastWorld: uint8(clampByte(lw)),
	}, created, nil
}

func (s *SQLStore) create(name string) (Account, bool, error) {
	if _, err := s.db.Exec(s.qInsert, sql.Named("account", name)); err != nil {
		return Account{}, false, fmt.Errorf("store: qInsert: %w", err)
	}
	acc, _, err := s.GetOrCreate(name, false) // после ap_AutoReg акк существует
	return acc, true, err
}

func allZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return len(b) > 0
}

func u32(n sql.NullInt32) uint32 {
	if !n.Valid {
		return 0
	}
	return uint32(n.Int32)
}

func clampByte(n sql.NullInt32) int32 {
	if !n.Valid {
		return 0
	}
	return n.Int32 & 0xff
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

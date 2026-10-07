// Package store — слой БД authd (AionAccounts).
//
// Эталон семантики: C1 L2Auth (CAccount/CAuthSocket + ReleaseAuthDBSchema.sql):
// user_account (uid IDENTITY, account varchar(14), pay_stat, login_flag,
// block_flag, block_flag2, block_end_date, last_login/last_logout/last_ip),
// block_msg (uid, reason, msg), user_auth (password binary(16)).
//
// Live-факты нашего Aion L2Authd (06.10): пароль НЕ проверяется вообще,
// любой ASCII-логин автосоздаёт акк; фейл только на не-ASCII логин.
package store

// Account — строка акка (проекция user_account).
type Account struct {
	UID        uint32
	Name       string
	PayStat    uint32
	LoginFlag  uint32
	WarnFlag   uint32
	BlockFlag  uint32 // block_flag  (custom)
	BlockFlag2 uint32 // block_flag2 (standard)
	LastWorld  uint8
}

// Blocked — flags-эвристика C1 (blockFlag_custom/blockFlag_standard).
func (a Account) Blocked() bool { return a.BlockFlag != 0 || a.BlockFlag2 != 0 }

// BlockReason — строка block_msg.
type BlockReason struct {
	Code int32
	Msg  string
}

// Store — интерфейс БД authd. Все методы потокобезопасны.
type Store interface {
	// GetOrCreate — ап ГСтат+автосоздание: акк по имени; created=true если INSERT;
	// ErrNotFound если акка нет и autoCreate=false.
	GetOrCreate(name string, autoCreate bool) (Account, bool, error)
	// Blocks — block_msg по uid (C1: MakeBlockInfo).
	Blocks(uid uint32) ([]BlockReason, error)
	// LogLogin — ap_SLog-аналог (last_login/last_ip), best-effort.
	LogLogin(a Account, ip string) error
	Close() error
}

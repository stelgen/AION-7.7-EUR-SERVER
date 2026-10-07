// Package logic — логика authd (порт C1 CAccount на live-фактах Aion L2Authd).
//
// Live-доказано 06-07.10 (логины юзера + probe):
//   - пароль НЕ проверяется вообще; любой ASCII-логин автосоздаёт акк;
//   - фейл ТОЛЬКО на не-ASCII/пустой логин → LoginFail (18Б wire);
//   - login-ok = type=3 (serverlist: [accId][token][пад][maxUsers][unk1]…, 52Б);
//   - [05] (CM_SERVER_LIST) → type=4 (server-info 42Б: [04][010101][IP][port]…);
//   - [02] (CM_PLAY) → type=7 (play-ok 26Б: [07][pk1][pk2][serverID][6×0]);
//   - relogin онлайн-акка → ТИШИНА (гейт сам отдаст LoginFail по таймауту);
//   - online-флаг живёт TTL 2-6 мин, [01]-дисконнект гейта флаг НЕ снимает.
package logic

import (
	crand "crypto/rand"
	"encoding/binary"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"aion-authd/internal/config"
	"aion-authd/internal/store"
)

// Ответ authd на пакет гейта (nil = тишина — live-паритет).
type Reply struct {
	Typ     byte // тип [02]-пакета (клиентский опкод = тип по relay-контракту гейта)
	Payload []byte
	Close   bool // после ответа слать [01][sid] (ориг так закрывает сессию после фейла — R5-дифф)
}

// GateSession — сессия на коннекте гейта ([00] CltConnect).
type GateSession struct {
	Sid uint32
	IP  [4]byte

	// заполняется на успешном логине:
	AccID uint32
	User  string
	Token uint32
}

// OnlineEntry — онлайн-флаг (аналог AccountDB/OneTimeLogOut).
type OnlineEntry struct {
	UID   uint32
	User  string
	Sid   uint32
	Since time.Time
}

// Deps — зависимости логики.
type Deps struct {
	Cfg   *config.Config
	Store store.Store

	mu     sync.Mutex
	online map[string]*OnlineEntry // ключ = username (нормализованный)
}

// New — создать логику.
func New(cfg *config.Config, st store.Store) *Deps {
	return &Deps{Cfg: cfg, Store: st, online: map[string]*OnlineEntry{}}
}

// ---------- online-флаг ----------

// IsOnline — акк онлайн (флаг не протух)?
func (d *Deps) IsOnline(user string) (*OnlineEntry, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	e, ok := d.online[user]
	if !ok {
		return nil, false
	}
	if time.Since(e.Since) > time.Duration(d.Cfg.OnlineTTLSec)*time.Second {
		delete(d.online, user) // ленивая чистка при чтении
		return nil, false
	}
	return e, true
}

// markOnline — пометить акк онлайн (на login-ok type=3 — probe-факт: флаг ставится
// именно на type=3, гейт использует это для своего online-кэша).
func (d *Deps) markOnline(user string, uid, sid uint32) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.online[user] = &OnlineEntry{UID: uid, User: user, Sid: sid, Since: time.Now()}
}

// ClearOnline — снять флаг ([01]-дисконнект гейта — ТОЛЬКО если cfg.ClearOnlineOnDisconnect;
// live-паритет: ориг не снимает, флаг живёт до TTL/GS-logout).
func (d *Deps) ClearOnline(user string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.online, user)
}

// SweepOnline — чистка протухших флагов (возвращает сколько снято; ev=sweep).
func (d *Deps) SweepOnline() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := 0
	ttl := time.Duration(d.Cfg.OnlineTTLSec) * time.Second
	for u, e := range d.online {
		if time.Since(e.Since) > ttl {
			delete(d.online, u)
			n++
		}
	}
	return n
}

// OnlineCount — текущее число онлайн-флагов.
func (d *Deps) OnlineCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.online)
}

// ---------- разбор логин-blob ----------

// LoginBlob — разобранный "cbdb"-blob ([00][decbuf34][dword][tail]).
type LoginBlob struct {
	User  string
	Pwd   string
	Otp   uint32
	Dword uint32
}

// ParseLoginBlob — decbuf = user14+pwd16+otp4 (BuildLoginDecbuf гейта; asm arg3=0x22=34).
// strict=true: blob строго 191Б (live: 86Б → тишина оригинала).
func ParseLoginBlob(blob []byte, strict bool) (*LoginBlob, bool) {
	if strict && len(blob) != 191 {
		return nil, false // asm-размер нарушен (live: authd молчит)
	}
	if len(blob) < 1+34+4 {
		return nil, false
	}
	if blob[0] != 0x00 {
		return nil, false // не логин
	}
	dec := blob[1 : 1+34]
	out := &LoginBlob{
		User:  readCStr(dec[0:14]),
		Pwd:   readCStr(dec[14:30]),
		Otp:   binary.LittleEndian.Uint32(dec[30:34]),
		Dword: binary.LittleEndian.Uint32(blob[35:39]),
	}
	return out, true
}

// readCStr — C-строка до NUL из фикс-поля.
func readCStr(b []byte) string {
	if i := strings.IndexByte(string(b), 0); i >= 0 {
		b = b[:i]
	}
	return string(b)
}

// isPrintableASCII — live-валидатор: непустой, все байты 0x21..0x7e
// (не-ASCII логин = единственный live-фейл оригинала).
func isPrintableASCII(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 0x21 || s[i] > 0x7e {
			return false
		}
	}
	return true
}

// Normalize — TrimSpace+ToLower (К-5, эталон .trim().toLowerCase(); гейт делает то же
// перед decbuf — тут идемпотентная страховка: «StelGeN» vs «stelgen» = разные акка).
func Normalize(user string) string {
	return strings.ToLower(strings.TrimSpace(user))
}

// ---------- процедуры ----------

// Login — обработка login-blob (blob[0]=0x00). Возвращает Reply или nil (тишина).
// Reasons тишины: кривой размер blob (strict), relogin онлайн-акка (live).
func (d *Deps) Login(s *GateSession, blob []byte, ip string) *Reply {
	b, ok := ParseLoginBlob(blob, d.Cfg.StrictAsmBlob)
	if !ok {
		log.Printf("login: sid=%d blob len=%d — asm-форма нарушена (strict=%v) → ТИШИНА (live-паритет 86Б)",
			s.Sid, len(blob), d.Cfg.StrictAsmBlob)
		return nil
	}
	user := Normalize(b.User)
	if !isPrintableASCII(user) {
		log.Printf("login: sid=%d user=%q — не-ASCII/пустой логин → LOGIN_FAIL(%d) (live-фейл)",
			s.Sid, b.User, d.Cfg.FailCodeBadUser)
		return d.fail(s, d.Cfg.FailCodeBadUser, "bad-user")
	}
	acc, created, err := d.Store.GetOrCreate(user, d.Cfg.AutoCreate)
	if err != nil {
		log.Printf("login: sid=%d user=%q DB ERR: %v → LOGIN_FAIL(%d)", s.Sid, user, err, d.Cfg.FailCodeDB)
		return d.fail(s, d.Cfg.FailCodeDB, "db")
	}
	_ = d.Store.LogLogin(acc, ip) // ap_SLog-аналог, best-effort

	if acc.Blocked() {
		log.Printf("login: sid=%d user=%q uid=%d BLOCKED (flags %d/%d) → LOGIN_FAIL(%d)",
			s.Sid, user, acc.UID, acc.BlockFlag, acc.BlockFlag2, d.Cfg.FailCodeBlocked)
		return d.fail(s, d.Cfg.FailCodeBlocked, "blocked")
	}
	if rs, err := d.Store.Blocks(acc.UID); err == nil && len(rs) > 0 {
		log.Printf("login: sid=%d user=%q uid=%d block_msg: %d строк → LOGIN_FAIL(%d)",
			s.Sid, user, acc.UID, len(rs), d.Cfg.FailCodeBlocked)
		return d.fail(s, d.Cfg.FailCodeBlocked, "blocked")
	}

	// relogin онлайн-акка: live-паритет = ТИШИНА (флаг TTL; гейт отдаст LoginFail(1) по
	// своему таймауту). Опция fail7 — немедленный фейл (kick-семантика эталона Mobius).
	if _, on := d.IsOnline(user); on {
		if d.Cfg.ReloginPolicy == "fail7" {
			log.Printf("login: sid=%d user=%q relogin ОНЛАЙН-акка → LOGIN_FAIL(7) (fail7)", s.Sid, user)
			return d.fail(s, 7, "relogin")
		}
		log.Printf("login: sid=%d user=%q relogin ОНЛАЙН-акка → ТИШИНА (live-паритет, TTL=%ds)",
			s.Sid, user, d.Cfg.OnlineTTLSec)
		return nil
	}

	token := rand32()
	d.markOnline(user, acc.UID, s.Sid)
	s.AccID, s.User, s.Token = acc.UID, user, token
	log.Printf("login OK: sid=%d user=%q uid=%d created=%v payStat=%d token=%08x → type=3",
		s.Sid, user, acc.UID, created, acc.PayStat, token)
	return &Reply{Typ: 3, Payload: BuildType3(acc.UID, token, d.Cfg)}
}

// ServerList — [05] CM_SERVER_LIST → type=4 (server-info 42Б: IP/порт мира).
func (d *Deps) ServerList(s *GateSession) *Reply {
	ip := net.ParseIP(d.Cfg.WorldIP).To4()
	if ip == nil {
		log.Printf("serverlist: BAD worldIP %q — фейл(%d)", d.Cfg.WorldIP, d.Cfg.FailCodeDB)
		return d.fail(s, d.Cfg.FailCodeDB, "world-ip")
	}
	log.Printf("serverlist: sid=%d → type=4 (%s:%d)", s.Sid, d.Cfg.WorldIP, d.Cfg.WorldPort)
	return &Reply{Typ: 4, Payload: BuildType4(ip, d.Cfg.WorldPort)}
}

// Play — [02] CM_PLAY → type=7 (play-ok 26Б: pk1/pk2 Rnd + serverID).
func (d *Deps) Play(s *GateSession) *Reply {
	log.Printf("play: sid=%d acc=%d → type=7 (pk1/pk2 Rnd, serverID=%d)", s.Sid, s.AccID, d.Cfg.ServerID)
	return &Reply{Typ: 7, Payload: BuildType7(d.Cfg.ServerID)}
}

// fail — LOGIN_FAIL: type=1, payload = [code u8] (1 БАЙТ — R5-дифф 07.10: ориг шлёт
// [02][sid][len=4][01][0x14]; после EncryptSecondary pt = [01][code,0,0,0,0...] —
// клиент читает [01][D mid] одинаково для 1Б и 4Б форм, но байт-паритет = 1Б).
// Признак Close: после фейла ориг шлёт [01][sid] (закрытие сессии — live-дифф 07.10).
func (d *Deps) fail(s *GateSession, code uint32, why string) *Reply {
	return &Reply{Typ: 1, Payload: []byte{byte(code)}, Close: true}
}

// ---------- сборка payload'ов (арбитр = R5 fork-дифф O-vs-N) ----------

// BuildType3 — payload login-ok/serverlist, live 52Б.
// КАНОН от ориг (probe 07.10 05:28, PA жив): [accId][token][8×0][2000][unk1 Rnd][28×0]
// (первый живой ориг-payload: f7030000 ed58c651 00000000 00000000 d0070000 08cafb0b ...).
// unk1 у ориг динамический (ранее наблюдали 0xa0c69f0b — считаем random dword).
func BuildType3(accID, token uint32, cfg *config.Config) []byte {
	p := make([]byte, 0, 52)
	p = binary.LittleEndian.AppendUint32(p, accID)
	p = binary.LittleEndian.AppendUint32(p, token)
	p = append(p, make([]byte, 8)...)
	p = binary.LittleEndian.AppendUint32(p, cfg.MaxUsers)
	p = binary.LittleEndian.AppendUint32(p, rand32())
	p = append(p, make([]byte, 28)...)
	return p
}

// BuildType4 — payload server-info (live 31Б; клиентский pt 32 = [04]+payload):
// [01 01 01][IP][port u16 LE][00000000][f4 01 01 01][00000002][01 00 01][7×0].
func BuildType4(worldIP []byte, worldPort uint16) []byte {
	p := []byte{0x01, 0x01, 0x01}
	p = append(p, worldIP...)
	p = binary.LittleEndian.AppendUint16(p, worldPort)
	p = append(p, 0x00, 0x00, 0x00, 0x00)
	p = append(p, 0xf4, 0x01, 0x01, 0x01)
	p = append(p, 0x00, 0x00, 0x00, 0x02)
	p = append(p, 0x01, 0x00, 0x01)
	p = append(p, make([]byte, 7)...)
	return p
}

// BuildType7 — payload play-ok (live 15Б; клиентский pt 16 = [07]+payload):
// [pk1 u32 Rnd][pk2 u32 Rnd][serverID байт][6×0] (эталон SessionKey: playOk=Rnd).
func BuildType7(serverID byte) []byte {
	p := make([]byte, 15)
	_, _ = crand.Read(p[0:8])
	p[8] = serverID
	return p
}

func rand32() uint32 {
	var b [4]byte
	_, _ = crand.Read(b[:])
	return binary.LittleEndian.Uint32(b[:])
}

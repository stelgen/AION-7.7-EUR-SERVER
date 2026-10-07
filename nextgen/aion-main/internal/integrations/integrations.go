// Package integrations — клиентские интеграции к соседям стека (канон nextgen/README §2-3).
// R3 статус: каркасы-стабы + канон; реальные коннекты по фазам (см. ROADMAP aion-main).
//
//	2006 CacheD64  — мир-канал RQ382/RP255 (cached-ref). БЛОКЕР: wire 7.7 не снят
//	                 (aion-cache R1 = pktmon 2006). Подключение вслепую = риск прод-кешу. OFFLINE.
//	2220 ACS       — AccountCacheServer (accountcache-ref, dispatch-77). Коннект Server64->ACS
//	                 короткоживущий RPC; сделаем после R3.5 (не блокирует MVP).
//	2104 L2Authd   — Server64-канал authd: R6-блокер aion-authd (роль 2104 = дизasm). OFFLINE.
//	2002 NPCSvr    — Server64 тут СЕРВЕР (8 коннектов NPCSvr = критерий мира) — включается
//	                 только на R4 fork-стенде (S4), не в тени.
package integrations

import (
	"fmt"
	"net"
	"time"
)

// Status — состояние интеграции для self-статуса (S2).
type Status struct {
	Name    string
	Addr    string
	Online  bool
	Note    string
	LastErr string
	Since   time.Time
}

// CacheDClient — клиент мир-канала 2006 (MVP: заглушка-интерфейс, REAL после aion-cache R1).
type CacheDClient struct {
	Addr string // 127.0.0.1:2006
	conn net.Conn
}

// NewCacheDClient — фабрика; НЕ подключается автоматически (R1 capture first).
func NewCacheDClient(addr string) *CacheDClient { return &CacheDClient{Addr: addr} }

// Ping — заглушка: проверяет только TCP-доступность порта (dial+close, без RPC — не сбивать сессию кеша).
func (c *CacheDClient) Ping(timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", c.Addr, timeout)
	if err != nil {
		return fmt.Errorf("cached ping %s: %w", c.Addr, err)
	}
	_ = conn.Close() // руками RPC не шлём: wire 7.7 не снят (aion-cache R1)
	return nil
}

// Statuses — снапшот всех интеграций для /status (S2).
func Statuses() []Status {
	var out []Status
	cd := NewCacheDClient("127.0.0.1:2006")
	err := cd.Ping(300 * time.Millisecond)
	out = append(out, Status{Name: "cached2006", Addr: "127.0.0.1:2006", Online: err == nil,
		Note: "ping-only (no RPC until aion-cache R1)", LastErr: fmt.Sprint(err), Since: time.Now()})
	return out
}
// Package ship — высылка телеметрии logd во внешний стак (стандарт TELEMETRY-SPEC).
//
// Транспорт (любая комбинация, всё опционально):
//   - syslog RFC5424: UDP (fire-and-forget) или TCP (octet-counted framing + reconnect);
//     приёмник на Linux = rsyslog (стандарт де-факто) / Vector / Fluent Bit → Loki/Grafana.
//   - HTTP: POST application/x-ndjson (готов к Vector HTTP source / будущему Loki push).
//   - файл ndjson: ТОЛЬКО по явному конфигу (требование «не срать файлами»).
//
// Правила бест-практис, реализованные здесь:
//   - ship НИКОГДА не блокирует горячий путь: очередь chan + drop со счётчиками;
//   - недоступный приёмник = ошибка со счётчиком (свой backoff через попытку в self-цикле),
//     паники глотаются (recover) — коннектор «не настроен/упал» не валит logd;
//   - self-статус (ev=self) каждые self_sec: uptime, sent/dropped, ошибки по синкам;
//   - raw-хекс в событиях капится (max_raw), полные дампы остаются в локальных .raw;
//   - при ship.enabled=false и file.enabled=false Send() — no-op (прод без приёмника).
package ship

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Имена событий (стабильный контракт приёмника — см. nextgen/TELEMETRY-SPEC.md).
const (
	EvStart         = "start"
	EvStop          = "stop"
	EvConnUp        = "conn.up"
	EvConnDown      = "conn.down"
	EvVersion       = "version"
	EvServerStarted = "server.started"
	EvStatus        = "status"
	EvText          = "text"
	EvParseErr      = "parse.err"
	EvDB            = "db"
	EvDBErr         = "db.err"
	EvSelf          = "self"
	EvSweep         = "sweep"
)

// Event — одно событие телеметрии. Все поля опциональны кроме Ev.
type Event struct {
	Ev     string         `json:"ev"`
	Svc    string         `json:"svc,omitempty"`
	Remote string         `json:"remote,omitempty"`
	Msg    string         `json:"msg,omitempty"`
	Err    string         `json:"err,omitempty"`
	Raw    string         `json:"raw,omitempty"`
	Data   map[string]any `json:"data,omitempty"`
}

type SyslogCfg struct {
	Net      string `yaml:"net"`      // udp|tcp (пусто = выкл)
	Host     string `yaml:"host"`     // host:port
	Facility int    `yaml:"facility"` // 1..23 (default 1 = user)
}

type HTTPCfg struct {
	URL     string `yaml:"url"`     // http://host[:port]/path — POST ndjson
	Timeout int    `yaml:"timeout"` // сек (default 3)
}

type FileCfg struct {
	Enabled bool   `yaml:"enabled"`
	Dir     string `yaml:"dir"` // сюда пишется YYYY-MM-DD.ndjson
}

type Cfg struct {
	Enabled bool      `yaml:"enabled"` // мастер сетевых синков (syslog+http)
	Syslog  SyslogCfg `yaml:"syslog"`
	HTTP    HTTPCfg   `yaml:"http"`
	File    FileCfg   `yaml:"file"`
	Queue   int       `yaml:"queue"`    // default 4096
	SelfSec int       `yaml:"self_sec"` // self-статус, default 300
	MaxRaw  int       `yaml:"max_raw"`  // cap raw hex, default 512
}

// S — шиппер. nil-safe: все методы проверяют nil.
type S struct {
	cfg  Cfg
	ch   chan Event
	host string
	wg   sync.WaitGroup

	sent, dropped atomic.Uint64
	errSyslog     atomic.Uint64
	errHTTP       atomic.Uint64
	errFile       atomic.Uint64
	syslogOK      atomic.Uint64
	httpOK        atomic.Uint64

	smu   sync.Mutex
	tcp   net.Conn
	fw    *os.File
	fname string

	hc    *http.Client
	start time.Time
}

// New — создать шиппер (Run() запускается отдельно).
func New(cfg Cfg) *S {
	if cfg.Queue <= 0 {
		cfg.Queue = 4096
	}
	if cfg.SelfSec <= 0 {
		cfg.SelfSec = 300
	}
	if cfg.MaxRaw <= 0 {
		cfg.MaxRaw = 512
	}
	if cfg.Syslog.Facility <= 0 {
		cfg.Syslog.Facility = 1
	}
	if cfg.HTTP.Timeout <= 0 {
		cfg.HTTP.Timeout = 3
	}
	host, _ := os.Hostname()
	return &S{cfg: cfg, ch: make(chan Event, cfg.Queue), host: host,
		hc: &http.Client{Timeout: time.Duration(cfg.HTTP.Timeout) * time.Second}, start: time.Now()}
}

// Enabled — есть ли хоть один активный синк.
func (s *S) Enabled() bool { return s != nil && (s.cfg.Enabled || s.cfg.File.Enabled) }

// Send — НЕблокирующая постановка в очередь. Переполнение = drop со счётчиком.
func (s *S) Send(ev Event) {
	if !s.Enabled() {
		return
	}
	if ev.Raw != "" && len(ev.Raw) > s.cfg.MaxRaw {
		ev.Raw = ev.Raw[:s.cfg.MaxRaw] + "..."
	}
	if len(ev.Msg) > 500 {
		ev.Msg = ev.Msg[:500] + "..."
	}
	if len(ev.Err) > 500 {
		ev.Err = ev.Err[:500] + "..."
	}
	select {
	case s.ch <- ev:
	default:
		s.dropped.Add(1)
	}
}

// Run — потребитель очереди + self-тикер. Блокирует до ctx.Done().
func (s *S) Run(ctx context.Context) {
	self := time.NewTicker(time.Duration(s.cfg.SelfSec) * time.Second)
	defer self.Stop()
	defer s.closeSinks()
	for {
		select {
		case <-ctx.Done():
			s.drain(300 * time.Millisecond)
			return
		case ev := <-s.ch:
			s.dispatch(ev)
		case <-self.C:
			s.dispatch(s.selfEvent())
		}
	}
}

func (s *S) dispatch(ev Event) {
	defer func() { _ = recover() }() // ship не должен никогда валить logd
	if ev.Ev == "" {
		ev.Ev = "unknown"
	}
	line, err := s.marshal(ev)
	if err != nil {
		s.dropped.Add(1)
		return
	}
	ts := time.Now()
	if s.cfg.Enabled {
		if s.cfg.Syslog.Host != "" && (s.cfg.Syslog.Net == "udp" || s.cfg.Syslog.Net == "tcp") {
			if s.syslogSend(ev, line, ts) {
				s.sent.Add(1)
				s.syslogOK.Add(1)
			} else {
				s.errSyslog.Add(1)
			}
		}
		if s.cfg.HTTP.URL != "" {
			if s.httpSend(line) {
				s.sent.Add(1)
				s.httpOK.Add(1)
			} else {
				s.errHTTP.Add(1)
			}
		}
	}
	if s.cfg.File.Enabled && s.cfg.File.Dir != "" {
		if s.fileSend(line, ts) {
			s.sent.Add(1)
		} else {
			s.errFile.Add(1)
		}
	}
}

func (s *S) marshal(ev Event) ([]byte, error) {
	b, err := json.Marshal(ev)
	if err != nil {
		b, err = json.Marshal(map[string]any{"ev": ev.Ev, "_marshal_err": fmt.Sprint(err)})
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

// ---------- syslog RFC5424 ----------

func (s *S) syslogSend(ev Event, payload []byte, ts time.Time) bool {
	defer func() { _ = recover() }()
	f := s.cfg.Syslog.Facility
	pri := f<<3 | 6 // info
	switch ev.Ev {
	case EvParseErr, EvDBErr:
		pri = f<<3 | 3 // error
	case EvSelf, EvStop:
		pri = f<<3 | 4 // notice
	}
	hdr := fmt.Sprintf("<%d>1 %s %s aion-logd 2051 %s - - ",
		pri, ts.Format("2006-01-02T15:04:05.000000Z07:00"), s.host, ev.Ev)
	full := append([]byte(hdr), payload...)
	if s.cfg.Syslog.Net == "udp" {
		if len(full) > 1400 { // одна датаграмма: трюк — режем payload, шапка цела
			keep := 1400 - len(hdr)
			if keep < 0 {
				keep = 0
			}
			full = append([]byte(hdr), payload[:keep]...)
		}
		c, err := net.DialTimeout("udp", s.cfg.Syslog.Host, 2*time.Second)
		if err != nil {
			return false
		}
		defer c.Close()
		_ = c.SetWriteDeadline(time.Now().Add(2 * time.Second))
		_, err = c.Write(full)
		return err == nil
	}
	// TCP: octet-counted framing, персистентный коннект
	s.smu.Lock()
	defer s.smu.Unlock()
	if s.tcp == nil {
		c, err := net.DialTimeout("tcp", s.cfg.Syslog.Host, 2*time.Second)
		if err != nil {
			return false
		}
		s.tcp = c
	}
	frame := fmt.Appendf(nil, "%d ", len(full))
	frame = append(frame, full...)
	_ = s.tcp.SetWriteDeadline(time.Now().Add(3 * time.Second))
	if _, err := s.tcp.Write(frame); err != nil {
		_ = s.tcp.Close()
		s.tcp = nil
		return false
	}
	return true
}

// ---------- HTTP ndjson ----------

func (s *S) httpSend(line []byte) bool {
	defer func() { _ = recover() }()
	resp, err := s.hc.Post(s.cfg.HTTP.URL, "application/x-ndjson",
		bytes.NewReader(append(append([]byte{}, line...), '\n')))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500 // 4xx = наш косяк формата, но доставлено
}

// ---------- file ndjson ----------

func (s *S) fileSend(line []byte, ts time.Time) bool {
	defer func() { _ = recover() }()
	s.smu.Lock()
	defer s.smu.Unlock()
	fn := ts.Format("2006-01-02") + ".ndjson"
	if s.fw == nil || s.fname != fn {
		if s.fw != nil {
			_ = s.fw.Close()
		}
		if err := os.MkdirAll(s.cfg.File.Dir, 0o755); err != nil {
			return false
		}
		f, err := os.OpenFile(s.cfg.File.Dir+"/"+fn, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return false
		}
		s.fw, s.fname = f, fn
	}
	_, err := s.fw.Write(append(append([]byte{}, line...), '\n'))
	return err == nil
}

func (s *S) closeSinks() {
	s.smu.Lock()
	defer s.smu.Unlock()
	if s.tcp != nil {
		_ = s.tcp.Close()
		s.tcp = nil
	}
	if s.fw != nil {
		_ = s.fw.Close()
		s.fw = nil
	}
}

func (s *S) drain(wait time.Duration) {
	t := time.After(wait)
	for {
		select {
		case ev := <-s.ch:
			s.dispatch(ev)
		case <-t:
			return
		}
	}
}

func (s *S) selfEvent() Event {
	return Event{
		Ev:  EvSelf,
		Msg: "ship self-status",
		Data: map[string]any{
			"sent":       s.sent.Load(),
			"dropped":    s.dropped.Load(),
			"err_syslog": s.errSyslog.Load(),
			"err_http":   s.errHTTP.Load(),
			"err_file":   s.errFile.Load(),
			"ok_syslog":  s.syslogOK.Load(),
			"ok_http":    s.httpOK.Load(),
			"queue":      len(s.ch),
			"queue_cap":  s.cfg.Queue,
			"uptime_s":   int(time.Since(s.start).Seconds()),
		},
	}
}

// Stats — счётчики (для лога при остановке).
func (s *S) Stats() (sent, dropped uint64) {
	return s.sent.Load(), s.dropped.Load()
}
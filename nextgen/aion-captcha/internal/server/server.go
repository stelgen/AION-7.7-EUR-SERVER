// Package server — TCP-сервер капчи (замена CAPTCHAImageServer.exe).
// Протокол: handshake 101→102, затем 1001→1002 (рендер на каждый запрос).
// Телеметрия по TELEMETRY-SPEC (ship из logd как есть): start/stop/conn.up/down/
// captcha (rate-limited 1/100)/parse.err/self; ship — НЕ критичный путь.
package server

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"aion-captcha/internal/proto"
	"aion-captcha/internal/render"
	"aion-captcha/internal/ship"
)

// Config — конфиг сервера (config.yaml).
type Config struct {
	Listen       string        `yaml:"listen"`        // default 0.0.0.0:22206
	SessionCount int           `yaml:"session_count"` // max conns (default 1024)
	Verbose      bool          `yaml:"verbose"`       // stdout-лог каждого события
	Render       render.Config `yaml:"render"`
}

// FillDefaults — дефолты.
func (c *Config) FillDefaults() {
	if c.Listen == "" {
		c.Listen = "0.0.0.0:22206"
	}
	if c.SessionCount <= 0 {
		c.SessionCount = 1024
	}
	c.Render.FillDefaults()
}

// Server — инстанс.
type Server struct {
	cfg Config
	sh  *ship.S
	rnd *rand.Rand
	rndMu sync.Mutex

	conns   int64
	reqs    uint64
	replies uint64
	perr    uint64
	start   time.Time
}

// New — создать.
func New(cfg Config, sh *ship.S) *Server {
	cfg.FillDefaults()
	return &Server{cfg: cfg, sh: sh, rnd: rand.New(rand.NewSource(time.Now().UnixNano())), start: time.Now()}
}

// Run — слушает до закрытия ctx.
func (s *Server) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.cfg.Listen)
	if err != nil {
		s.send(ship.Event{Ev: ship.EvDBErr, Msg: fmt.Sprintf("listen %s: %v", s.cfg.Listen, err)})
		return err
	}
	defer ln.Close()
	s.send(ship.Event{Ev: ship.EvStart, Msg: "aion-captcha started", Data: map[string]any{
		"listen": s.cfg.Listen, "sessions": s.cfg.SessionCount,
		"render": map[string]any{"w": s.cfg.Render.Width, "h": s.cfg.Render.Height,
			"len": s.cfg.Render.Length, "charset": s.cfg.Render.Charset},
	}})
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		c, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				s.send(ship.Event{Ev: ship.EvStop, Msg: "aion-captcha stopped", Data: map[string]any{
					"uptime_sec": int(time.Since(s.start).Seconds()), "requests": atomic.LoadUint64(&s.reqs),
					"replies": atomic.LoadUint64(&s.replies), "parse_err": atomic.LoadUint64(&s.perr),
				}})
				return nil
			default:
			}
			continue
		}
		if atomic.AddInt64(&s.conns, 1) > int64(s.cfg.SessionCount) {
			atomic.AddInt64(&s.conns, -1)
			c.Close()
			continue
		}
		go s.handleConn(ctx, c)
	}
}

func (s *Server) rand() *rand.Rand {
	s.rndMu.Lock()
	defer s.rndMu.Unlock()
	return s.rnd
}

func (s *Server) handleConn(ctx context.Context, c net.Conn) {
	remote := c.RemoteAddr().String()
	defer func() {
		atomic.AddInt64(&s.conns, -1)
		c.Close()
		s.send(ship.Event{Ev: ship.EvConnDown, Remote: remote, Data: map[string]any{
			"requests": atomic.LoadUint64(&s.reqs),
		}})
	}()
	s.send(ship.Event{Ev: ship.EvConnUp, Remote: remote})

	// рукопожатие: ждём 101, отвечаем 102 (эхо seq+lang)
	var rbuf []byte = make([]byte, 0x2000)
	fr, _, err := proto.ReadFrame(c, rbuf)
	if err != nil {
		if err != io.EOF {
			s.parseErr(remote, err, nil)
		}
		return
	}
	if fr.Type != proto.TypeLangCheck {
		s.parseErr(remote, fmt.Errorf("handshake: got type=%d want 101", fr.Type), fr.Body)
		return
	}
	seq, lang, err := proto.ParseLangCheck(fr.Body)
	if err != nil {
		s.parseErr(remote, err, fr.Body)
		return
	}
	if _, err := c.Write(proto.BuildLangCheckReply(seq, lang)); err != nil {
		return
	}

	// цикл запросов: 1001 → 1002
	for {
		fr, _, err := proto.ReadFrame(c, rbuf)
		if err != nil {
			if err != io.EOF && ctx.Err() == nil {
				s.parseErr(remote, err, nil)
			}
			return
		}
		switch fr.Type {
		case proto.TypeCaptchaRequest:
			rseq, codepage, lang, err := proto.ParseRequest(fr.Body)
			if err != nil {
				s.parseErr(remote, err, fr.Body)
				return
			}
			s.replyCaptcha(c, remote, rseq, codepage, lang)
		case proto.TypeLangCheck:
			// повторный lang-check (допустим) — снова эхо
			lseq, lang, err := proto.ParseLangCheck(fr.Body)
			if err == nil {
				_, _ = c.Write(proto.BuildLangCheckReply(lseq, lang))
			}
		default:
			s.parseErr(remote, fmt.Errorf("type=%d не поддержан", fr.Type), fr.Body)
			return
		}
	}
}

func (s *Server) replyCaptcha(c net.Conn, remote string, seq uint16, codepage uint32, lang uint16) {
	n := atomic.AddUint64(&s.reqs, 1)
	r := s.rand()
	cfg := s.cfg.Render
	cfg.FillDefaults()
	text := render.TextGen(r, cfg)

	t0 := time.Now()
	img := render.Image(r, cfg, text)
	data, err := render.EncodeDXT1(img)
	if err != nil {
		s.parseErr(remote, err, nil)
		return
	}
	dds, err := render.DDSBlobOf(data)
	if err != nil {
		s.parseErr(remote, err, nil)
		return
	}
	tu16, err := render.UTF16LE(text)
	if err != nil {
		s.parseErr(remote, err, nil)
		return
	}
	reply := proto.BuildReply(seq, dds, tu16)
	if _, err := c.Write(reply); err != nil {
		return
	}
	atomic.AddUint64(&s.replies, 1)

	// телеметрия: каждый 100-й запрос + медленные (>50мс)
	ms := time.Since(t0).Milliseconds()
	if n%100 == 1 || ms > 50 {
		s.send(ship.Event{Ev: "captcha", Svc: "captcha", Remote: remote, Msg: fmt.Sprintf("seq=%d text=%s", seq, text), Data: map[string]any{
			"seq": seq, "text": text, "ms": ms, "codepage": codepage,
			"lang": proto.UnEn(lang), "req_n": n, "reply_total": len(reply),
		}})
	}
}

func (s *Server) parseErr(remote string, err error, raw []byte) {
	atomic.AddUint64(&s.perr, 1)
	ev := ship.Event{Ev: ship.EvParseErr, Remote: remote, Err: err.Error()}
	if len(raw) > 0 {
		ev.Raw = hexOf(raw, 64)
	}
	s.send(ev)
}

func hexOf(b []byte, cap int) string {
	n := len(b)
	if n > cap {
		n = cap
	}
	out := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			out += " "
		}
		out += fmt.Sprintf("%02X", b[i])
	}
	if len(b) > cap {
		out += " …"
	}
	return out
}

func (s *Server) send(ev ship.Event) {
	if s.sh != nil {
		s.sh.Send(ev)
	}
	if s.cfg.Verbose {
		fmt.Fprintf(os.Stdout, "%s ev=%s remote=%s msg=%s err=%s\n", time.Now().Format("15:04:05.000"), ev.Ev, ev.Remote, ev.Msg, ev.Err)
	}
}

// Package server — TCP-приёмник logd: accept, фрейминг, handshake, dispatch.
// State machine как в LogServerSocket::OnRead: [len u16][type][BB][~type][payload],
// длина ≤ 0x2000; Version до любых данных; неизвестные payload — в .raw (см. proto).
// Л1–Л4 (05.10.2026): type-9 парсер, ship-телеметрия, InitializeCount на svc==3,
// retention-sweep. Ship nil-safe: без приёмника поведение прежнее.
package server

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"aion-logd/internal/logdb"
	"aion-logd/internal/proto"
	"aion-logd/internal/records"
	"aion-logd/internal/ship"
	"aion-logd/internal/textlog"
	"aion-logd/internal/writer"
)

type Config struct {
	Listen        string            `yaml:"listen"`   // :2051
	Builder       uint32            `yaml:"builder"`  // наш builderNumber (≥10003)
	BaseDir       string            `yaml:"base_dir"` // корень логов (в проде — D:\SAION\aion-logd\logs)
	Dirs          map[string]string `yaml:"dirs"`     // svcType → каталог (переопределения)
	MaxConn       int               `yaml:"max_conn"`
	CaptureAll    bool              `yaml:"capture_all"`  // дампить ВЕСЬ трафик в .raw (capture-режим)
	InitSvc       int               `yaml:"init_svc"`     // Л3: InitializeCount на ServerStarted с этим svc (3 = MAIN); 0 = первый как раньше
	RetentionDays int               `yaml:"retention_days"` // Л4: чистка base_dir старше N дней; 0 = выкл
}

type Server struct {
	cfg      Config
	wr       *writer.W
	sh       *ship.S
	wg       sync.WaitGroup
	db       *logdb.DB
	initDone uint32
}

func New(cfg Config, wr *writer.W) *Server {
	if cfg.Builder < proto.MinBld {
		cfg.Builder = proto.MinBld + 1
	}
	if cfg.MaxConn <= 0 {
		cfg.MaxConn = 64
	}
	return &Server{cfg: cfg, wr: wr}
}

// SetDB — опциональный DB-слой (таймеры + InitializeCount при старте мира).
func (s *Server) SetDB(d *logdb.DB) { s.db = d }

// SetShip — опциональная телеметрия (TELEMETRY-SPEC). nil = выключено.
func (s *Server) SetShip(sh *ship.S) { s.sh = sh }

func (s *Server) send(ev ship.Event) {
	if s.sh != nil {
		s.sh.Send(ev)
	}
}

func (s *Server) Run(ctx context.Context, ln net.Listener) error {
	go func() { <-ctx.Done(); _ = ln.Close() }()
	// Л4: retention-sweep при старте и каждые 30 мин
	if s.cfg.RetentionDays > 0 {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.sweep("start")
			t := time.NewTicker(30 * time.Minute)
			defer t.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-t.C:
					s.sweep("tick")
				}
			}
		}()
	}
	sem := make(chan struct{}, s.cfg.MaxConn)
	for {
		c, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				s.wg.Wait()
				return nil
			default:
			}
			return fmt.Errorf("accept: %w", err)
		}
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			_ = c.Close()
			s.wg.Wait()
			return nil
		}
		s.wg.Add(1)
		go func(c net.Conn) {
			defer func() { <-sem; s.wg.Done(); _ = c.Close() }()
			s.serve(ctx, c)
		}(c)
	}
}

func (s *Server) sweep(when string) {
	n, err := s.wr.Sweep(s.cfg.RetentionDays)
	if err != nil {
		log.Printf("[sweep] %v", err)
	}
	if n > 0 {
		log.Printf("[sweep] retention=%dd удалено файлов: %d", s.cfg.RetentionDays, n)
		s.send(ship.Event{Ev: ship.EvSweep, Msg: when, Data: map[string]any{"removed": n, "retention_days": s.cfg.RetentionDays}})
	}
}

// serve — коннект: handshake Version → поток пакетов (накопление по len).
func (s *Server) serve(ctx context.Context, c net.Conn) {
	remote := c.RemoteAddr().String()
	r := bufio.NewReaderSize(c, 16*1024)
	statusN, textN, badN := 0, 0, 0
	var acc []byte
	connSvc := "" // svc из ServerStarted этого коннекта (для .err-каталога текст-логов)
	started := time.Now()

	verTx := proto.Build(proto.TypeVersion, proto.VersionBody(s.cfg.Builder, nil))
	vtTx := proto.VersionAndTime()
	if s.cfg.CaptureAll {
		_ = s.wr.WriteIO("tx-ver", verTx, time.Now())
		_ = s.wr.WriteIO("tx-vt", vtTx, time.Now())
	}
	if _, err := c.Write(verTx); err != nil {
		log.Printf("[%s] ver write: %v", remote, err)
		return
	}
	if _, err := c.Write(vtTx); err != nil {
		log.Printf("[%s] vt write: %v", remote, err)
		return
	}
	log.Printf("[%s] connected, Version(builder=%d)+VT отправлены первыми", remote, s.cfg.Builder)
	s.send(ship.Event{Ev: ship.EvConnUp, Remote: remote, Msg: "handshake sent",
		Data: map[string]any{"listen": s.cfg.Listen, "builder": s.cfg.Builder}})

	defer func() { // conn.down — сколько чего принято за сессию
		s.send(ship.Event{Ev: ship.EvConnDown, Remote: remote, Svc: connSvc,
			Data: map[string]any{"dur_s": int(time.Since(started).Seconds()),
				"statuses": statusN, "texts": textN, "parse_err": badN}})
	}()

	for {
		_ = c.SetReadDeadline(time.Now().Add(5 * time.Minute)) // keepalive-фолбэк
		chunk := make([]byte, 4096)
		n, err := r.Read(chunk)
		if n > 0 {
			if s.cfg.CaptureAll { // дамп ДО парсинга: ресинк ничего не съест молча
				_ = s.wr.WriteIO("rx", chunk[:n], time.Now())
			}
			acc = append(acc, chunk[:n]...)
			for len(acc) >= 5 {
				pkt, used, perr := proto.Parse(acc)
				if perr == proto.ErrShort {
					break // ждём ещё
				}
				if perr != nil {
					log.Printf("[%s] bad packet: %v — ресинк на 1 байт", remote, perr)
					badN++
					acc = acc[1:]
					continue
				}
				acc = acc[used:]
				if s.cfg.CaptureAll {
					_ = s.wr.WriteRaw("capture", pkt.Type, pkt.Raw, time.Now())
				}

				switch pkt.Type {
				case proto.TypeVersion:
					b, minB := proto.ParseVersion(pkt.Body)
					log.Printf("[%s] Version: builder=%d min=%d", remote, b, minB)
					s.send(ship.Event{Ev: ship.EvVersion, Remote: remote,
						Data: map[string]any{"builder": b, "min": minB}})
					if _, werr := c.Write(proto.Build(proto.TypeVersion,
						proto.VersionBody(s.cfg.Builder, nil))); werr != nil {
						return
					}
					if _, werr := c.Write(proto.VersionAndTime()); werr != nil {
						return
					}

				case proto.TypeVerAndTimeReq:
					if _, werr := c.Write(proto.VersionAndTime()); werr != nil {
						return
					}

				case proto.TypeAlive:
					// ФИНАЛЬНАЯ РАЗГАДКА ФЛАПА (mirror-корреляция 05.10): клиент пингует type-11
					// каждые 90с, СЕРВЕР ОТВЕЧАЕТ type-4 (8 нулей) — ProcessAliveResponse.
					if _, werr := c.Write(proto.Build(proto.TypeControl, make([]byte, 8))); werr != nil {
						return
					}

				case proto.TypeServerStarted:
					if len(pkt.Body) == 12 {
						a := binary.LittleEndian.Uint32(pkt.Body)
						b2 := binary.LittleEndian.Uint32(pkt.Body[4:8])
						c2 := int32(binary.LittleEndian.Uint32(pkt.Body[8:12]))
						log.Printf("[%s] ServerStarted: id=%d svc=%d x=%d", remote, a, b2, c2)
						connSvc = fmt.Sprintf("svc%d", b2)
						s.send(ship.Event{Ev: ship.EvServerStarted, Remote: remote, Svc: connSvc,
							Data: map[string]any{"id": a, "svc": b2, "x": c2}})
						// Л3: оригинал инициализирует счётчики при СТАРТЕ МИРА —
						// мир = MAIN, его ServerStarted = svc 3 (mirror: data-коннект (1,3,0)).
						if s.db != nil && (s.cfg.InitSvc == 0 || int(b2) == s.cfg.InitSvc) &&
							atomic.CompareAndSwapUint32(&s.initDone, 0, 1) {
							if err := s.db.InitializeCount(ctx); err != nil {
								atomic.StoreUint32(&s.initDone, 0)
								log.Printf("[logdb] InitializeCount: %v (ретрай по следующему ServerStarted)", err)
								s.send(ship.Event{Ev: ship.EvDBErr, Svc: connSvc, Msg: "InitializeCount", Err: err.Error()})
							} else {
								log.Printf("[logdb] InitializeCount(world=%d) выполнен (svc=%d)", s.db.WorldID, b2)
								s.send(ship.Event{Ev: ship.EvDB, Svc: connSvc, Msg: "InitializeCount",
									Data: map[string]any{"world": s.db.WorldID, "svc": b2}})
							}
						}
					}

				case proto.TypeStatus:
					if rec, rerr := records.ParseStatus(pkt.Body); rerr == nil {
						_ = s.wr.WriteStatus(fmt.Sprintf("svc%d", rec.SvcType), rec.String(), time.Now())
						statusN++
						if statusN%100 == 1 {
							log.Printf("[%s] status: svc=%d world=%d pos=(%.1f,%.1f,%.1f) %s (n=%d)",
								remote, rec.SvcType, rec.World(), rec.X, rec.Y, rec.Z,
								rec.SysTimeString(), statusN)
						}
						s.send(ship.Event{Ev: ship.EvStatus, Remote: remote, Svc: fmt.Sprintf("svc%d", rec.SvcType),
							Data: map[string]any{
								"world": rec.World(), "x": rec.X, "y": rec.Y, "z": rec.Z,
								"metric1": rec.Metric1, "metric2": rec.Metric2, "metric3": rec.Metric3,
								"metric4": rec.Metric4, "engine_ms": rec.EngineMs, "n": statusN,
							}})
					} else {
						log.Printf("[%s] status parse: %v — в .raw", remote, rerr)
						_ = s.wr.WriteRaw("badstatus", pkt.Type, pkt.Raw, time.Now())
						badN++
						s.send(ship.Event{Ev: ship.EvParseErr, Remote: remote, Svc: "badstatus",
							Err: rerr.Error(), Raw: hexHead(pkt.Raw)})
					}

				case proto.TypeTextLog:
					// Л1: текст-логи/онлайн-таблица (см. internal/textlog)
					rec, terr := textlog.Parse(pkt.Body)
					if terr != nil {
						_ = s.wr.WriteRaw("textlog.bad", pkt.Type, pkt.Raw, time.Now())
						badN++
						log.Printf("[%s] textlog parse: %v — в .raw", remote, terr)
						s.send(ship.Event{Ev: ship.EvParseErr, Remote: remote, Svc: dirOr(connSvc, "textlog"),
							Err: terr.Error(), Raw: hexHead(pkt.Raw)})
						break
					}
					dir := dirOr(connSvc, "textlog")
					line := fmt.Sprintf("%s %s", time.Now().Format("15:04:05"), rec.String())
					if werr := s.wr.WriteLine(dir, line, time.Now()); werr != nil {
						log.Printf("[%s] textlog write: %v", remote, werr)
					}
					textN++
					if textN%10 == 1 {
						log.Printf("[%s] %s (n=%d)", remote, rec.String(), textN)
					}
					s.send(ship.Event{Ev: ship.EvText, Remote: remote, Svc: dir, Msg: rec.String(),
						Data: map[string]any{"id": rec.ID, "n": len(rec.Entries), "world": rec.World,
							"stamp": rec.Stamp, "body_len": rec.BodyLen}})

				default: // неизвестный payload — TBD-layout, в .raw
					_ = s.wr.WriteRaw("payload", pkt.Type, pkt.Raw, time.Now())
					badN++
					log.Printf("[%s] data type=%d len=%d → .raw (layout TBD, живой capture)", remote, pkt.Type, len(pkt.Body))
					s.send(ship.Event{Ev: ship.EvParseErr, Remote: remote, Svc: "payload",
						Msg: fmt.Sprintf("type=%d", pkt.Type), Raw: hexHead(pkt.Raw)})
				}
			}
		}
		if err != nil {
			if err == io.EOF || isTimeout(err) {
				return
			}
			select {
			case <-ctx.Done():
				return
			default:
				log.Printf("[%s] read: %v", remote, err)
				return
			}
		}
	}
}

func dirOr(svc, def string) string {
	if svc != "" {
		return svc
	}
	return def
}

func hexHead(b []byte) string {
	if len(b) > 128 {
		b = b[:128]
	}
	return fmt.Sprintf("% X", b)
}

func isTimeout(err error) bool {
	ne, ok := err.(net.Error)
	return ok && ne.Timeout()
}
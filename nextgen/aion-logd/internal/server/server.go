// Package server — TCP-приёмник logd: accept, фрейминг, handshake, dispatch.
// State machine как в LogServerSocket::OnRead: [len u16][type][BB][~type][payload],
// длина ≤ 0x2000; Version до любых данных; неизвестные payload — в .raw (см. proto).
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
	"time"

	"aion-logd/internal/proto"
	"aion-logd/internal/writer"
)

type Config struct {
	Listen     string            `yaml:"listen"`   // :2051
	Builder    uint32            `yaml:"builder"`  // наш builderNumber (≥10003)
	BaseDir    string            `yaml:"base_dir"` // корень логов (в проде — D:\AION_LIVE_SERVER\...\log)
	Dirs       map[string]string `yaml:"dirs"`     // svcType → каталог (переопределения)
	MaxConn    int               `yaml:"max_conn"`
	CaptureAll bool              `yaml:"capture_all"` // дампить ВЕСЬ трафик в .raw (capture-режим)
}

type Server struct {
	cfg Config
	wr  *writer.W
	wg  sync.WaitGroup
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

func (s *Server) Run(ctx context.Context, ln net.Listener) error {
	go func() { <-ctx.Done(); _ = ln.Close() }()
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

// serve — коннект: handshake Version → поток пакетов (накопление по len).
func (s *Server) serve(ctx context.Context, c net.Conn) {
	remote := c.RemoteAddr().String()
	r := bufio.NewReaderSize(c, 16*1024)
	var acc []byte

	// СЕРВЕР говорит первым (мимик LogServerSocket::OnCreate: SendVersion+VT при accept):
	// итерация 05.10 доказала — клиенты подключаются и ЖДУТ инициативы сервера.
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
					// mimic LogServerSocket::OnCreate: свой Version + VersionAndTime
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

				case proto.TypeServerStarted:
					if len(pkt.Body) == 12 {
						a := binary.LittleEndian.Uint32(pkt.Body)
						b2 := binary.LittleEndian.Uint32(pkt.Body[4:8])
						c2 := int32(binary.LittleEndian.Uint32(pkt.Body[8:12]))
						log.Printf("[%s] ServerStarted: id=%d svc=%d x=%d", remote, a, b2, c2)
					}

				case proto.TypeStatus:
					// статус-поток (body 194 const): счётчики/floats + SYSTEMTIME-хвост — парсить по мере надобности
					_ = pkt.Body

				default: // TypeData и всё, что не разобрано — TBD-layout, в .raw
					_ = s.wr.WriteRaw("payload", pkt.Type, pkt.Raw, time.Now())
					log.Printf("[%s] data type=%d len=%d → .raw (layout TBD, живой capture)", remote, pkt.Type, len(pkt.Body))
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
		_ = binary.LittleEndian // (импорт сохранён для Len-расчётов в будущем payload-парсере)
	}
}

func isTimeout(err error) bool {
	ne, ok := err.(net.Error)
	return ok && ne.Timeout()
}

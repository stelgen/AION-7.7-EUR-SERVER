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
	Listen  string            `yaml:"listen"`   // :2051
	Builder uint32            `yaml:"builder"`  // наш builderNumber (≥10003)
	BaseDir string            `yaml:"base_dir"` // корень логов (в проде — D:\AION_LIVE_SERVER\...\log)
	Dirs    map[string]string `yaml:"dirs"`     // svcType → каталог (переопределения)
	MaxConn int               `yaml:"max_conn"`
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
	handshaked := false
	r := bufio.NewReaderSize(c, 16*1024)
	var acc []byte

	for {
		_ = c.SetReadDeadline(time.Now().Add(5 * time.Minute)) // keepalive-фолбэк
		chunk := make([]byte, 4096)
		n, err := r.Read(chunk)
		if n > 0 {
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
					handshaked = true

				case proto.TypeVerAndTimeReq:
					if _, werr := c.Write(proto.VersionAndTime()); werr != nil {
						return
					}

				case proto.TypeUnknown3, proto.TypeVerAndTime:
					// в оригинале: заглушка/ответ; просто фиксируем живость
					_ = handshaked

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

// Package server — TCP-листенеры каналов CacheD64: 2006 (server), 2007 (admin), 2009 (npcDb).
// Канон CServerSocket.cpp: per-conn Reader с rolling key, dispatch по opcode.
package server

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"

	"aion-cached/internal/wire"
)

// Config — конфиг (порт + seed). Минимум для R1.
type Config struct {
	ServerPort   int    `json:"server_port"`
	AdminPort    int    `json:"admin_port"`
	NpcDbPort    int    `json:"npcdb_port"`
	CryptSeedHex string `json:"crypt_seed_hex"`
}

// DefaultConfig — канон: 2006/2007/2009, дефолт-seed.
func DefaultConfig() Config {
	return Config{ServerPort: 2006, AdminPort: 2007, NpcDbPort: 2009, CryptSeedHex: "5F3A7C21"}
}

// Seed возвращает ключ DummyCrypt из конфига.
func (c Config) Seed() uint32 {
	var v uint32
	fmt.Sscanf(c.CryptSeedHex, "%x", &v)
	return v
}

// Load загружает key=value конфиг (простейший формат R1).
func (c *Config) Load(path string) error {
	lines, err := readFileLines(path)
	if err != nil {
		return err
	}
	for _, ln := range lines {
		var k, v string
		if _, err := fmt.Sscanf(ln, "%s %s", &k, &v); err != nil {
			continue
		}
		switch k {
		case "server_port":
			fmt.Sscanf(v, "%d", &c.ServerPort)
		case "admin_port":
			fmt.Sscanf(v, "%d", &c.AdminPort)
		case "npcdb_port":
			fmt.Sscanf(v, "%d", &c.NpcDbPort)
		case "crypt_seed_hex":
			c.CryptSeedHex = v
		}
	}
	return nil
}

// Channel — тип канала (opcode-таблица).
type Channel int

const (
	// ChanServer — 2006, L2 serverHandlers ↔ наш RQ/RP (ServerToDb/DbToServer).
	ChanServer Channel = iota
	// ChanAdmin — 2007, GQ/GP interactive (AdminToCache/CacheToAdmin).
	ChanAdmin
	// ChanNpcDb — 2009, npcDbHandlers (NPCSvr64).
	ChanNpcDb
)

// Server — набор листенеров.
type Server struct {
	cfg Config
}

// New — конструктор.
func New(cfg Config) *Server { return &Server{cfg: cfg} }

// Run — поднимает все три канала, блокируется.
func (s *Server) Run() error {
	ports := []struct {
		ch   Channel
		port int
		name string
	}{
		{ChanServer, s.cfg.ServerPort, "server"},
		{ChanAdmin, s.cfg.AdminPort, "admin"},
		{ChanNpcDb, s.cfg.NpcDbPort, "npcDb"},
	}
	for _, p := range ports {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", p.port))
		if err != nil {
			return fmt.Errorf("listen %s :%d: %w", p.name, p.port, err)
		}
		log.Printf("[cached] %s channel listening on :%d", p.name, p.port)
		go s.acceptLoop(ln, p.ch, p.name)
	}
	select {} // блок навсегда
}

func (s *Server) acceptLoop(ln net.Listener, ch Channel, name string) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[cached] %s accept error: %v", name, err)
			return
		}
		go s.handleConn(conn, ch, name)
	}
}

// handleConn — конвейер: приём кадра → wire.Feed → Route → ответ.
// Канон L2 C1: reject при bad size / bad csum → socket Close.
func (s *Server) handleConn(conn net.Conn, ch Channel, name string) {
	defer conn.Close()
	dec := wire.NewDecoder(s.cfg.Seed())
	buf := make([]byte, wire.MaxPacketSize+wire.HeaderSize)
	var pending []byte

	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		pending = append(pending, buf[:n]...)

		for len(pending) >= wire.HeaderSize {
			consumed, frame, ferr := dec.Feed(pending)
			if ferr != nil {
				if isErrShort(ferr) {
					break // ждём ещё байт
				}
				// канон CServerSocket: bad size / bad csum → Close
				log.Printf("[cached] %s reject frame: %v — closing conn (canon L2 behavior)", name, ferr)
				return
			}
			pending = pending[consumed:]

			resp := Route(ch, frame)
			if len(resp) > 0 {
				if _, werr := conn.Write(resp); werr != nil {
					log.Printf("[cached] %s write error: %v", name, werr)
					return
				}
			}
		}
		if len(pending) > wire.MaxPacketSize+wire.HeaderSize {
			log.Printf("[cached] %s buffer overflow, closing", name)
			return
		}
	}
}

// isErrShort — проверка ошибки «нужно ещё байт» (фолбек, если err-тип не раскрыт).
func isErrShort(err error) bool {
	type unwrapper interface{ Unwrap() error }
	e := err
	for e != nil {
		if e == wire.ErrShort {
			return true
		}
		u, ok := e.(unwrapper)
		if !ok {
			return false
		}
		e = u.Unwrap()
	}
	return false
}

// readFileLines — построчное чтение key=value конфига с фильтром пустых и комментариев.
func readFileLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, ln := range bytes.Split(data, []byte("\n")) {
		ln = bytes.TrimSpace(ln)
		if len(ln) > 0 && ln[0] != '#' {
			lines = append(lines, string(ln))
		}
	}
	return lines, nil
}

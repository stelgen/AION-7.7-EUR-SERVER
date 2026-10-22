// Package server — TCP-листенеры каналов CacheD64: 2006 (server), 2007 (admin), 2009 (npcDb).
// Канон CServerSocket.cpp: per-conn Reader с rolling key, dispatch по opcode.
package server

import (
	"bufio"
	"time"
	"bytes"
	"fmt"
	"log"
	"net"
	"os"

	"encoding/binary"

	"aion-cached/internal/rpc"
	"aion-cached/internal/wire"
)

// Config — конфиг (порт + seed). Минимум для R1.
type Config struct {
	ServerPort   int    `json:"server_port"`
	AdminPort    int    `json:"admin_port"`
	NpcDbPort    int    `json:"npcdb_port"`
	CryptSeedHex string `json:"crypt_seed_hex"`
	DBConn       string `json:"db_conn"`
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
		case "db_conn":
			c.DBConn = v
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
	cfg    Config
	testLn map[Channel]net.Listener
	SDB    *SettingsDB // DB-персист настроек (nil = RAM-only)
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

// SetListenerForTest — тестовый вход: подмена listener канала.
func (s *Server) SetListenerForTest(ln net.Listener, ch Channel) {
	if s.testLn == nil {
		s.testLn = map[Channel]net.Listener{}
	}
	s.testLn[ch] = ln
}

// ServeOnTest — тестовый прогон accept-loop одного канала (блокирует).
func (s *Server) ServeOnTest(ch Channel) error {
	name := map[Channel]string{ChanServer: "server", ChanAdmin: "admin", ChanNpcDb: "npcDb"}[ch]
	ln, ok := s.testLn[ch]
	if !ok {
		ln2, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return err
		}
		ln = ln2
		s.testLn[ch] = ln
	}
	s.acceptLoop(ln, ch, name)
	return nil
}

// handleConn — конвейер: приём кадра → wire.Feed → Route → ответ.
// Канон L2 C1: reject при bad size / bad csum → socket Close.
func (s *Server) handleConn(conn net.Conn, ch Channel, name string) {
	defer conn.Close()
	if ch == ChanAdmin {
		// 2007 interactive = ТЕКСТОВЫЙ протокол (L2 C1 CAdminSocket.cpp канон:
		// строки до '\n', packetId первым полем) — НЕ бинарный wire.
		r := bufio.NewReader(conn)
		_ = conn.SetReadDeadline(time.Time{})
		lineFn := func() (string, error) {
			return r.ReadString('\n')
		}
		handleAdminText(ch, name, lineFn, func(out string) {
			if out != "" {
				_, _ = conn.Write([]byte(out + "\n"))
			}
		})
		return
	}
	dec := wire.NewDecoder()
	buf := make([]byte, wire.MaxPayload+wire.HeaderWire)
	var pending []byte
	mode := modeServer // Server64 по умолчанию; NPRelay64 переключает handshake'ом
	var sessX uint32   // R4: сессионный хендл из cmd=288 (0 если ещё не установлен)

	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		pending = append(pending, buf[:n]...)

		for len(pending) >= wire.HeaderWire {
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

			// R3 GQ/GP: NPRelay64 handshake переключает conn в relay-режим
			if ch == ChanServer && maybeRelayHandshake(&mode, frame.Cmd) {
				name := rpc.Name(6, frame.Cmd) // GQ-имя из .profile (idx 6)
				_ = relayGQDispatch(&mode, ch, frame.Cmd, frame.Payload, name)
				continue // pending уже съеден выше
			}

			// R4: сессионный хендл из cmd=288 (первый кадр)
			if frame.Cmd == 288 && len(frame.Payload) >= 12 {
				sessX = binary.LittleEndian.Uint32(frame.Payload[0:4])
			}

			resp := Route(ch, frame)
			// R4: read-path emission (replay) после обычного dispatch
			if ch == ChanServer && sessX != 0 {
				switch frame.Cmd {
				case 11: // RQ_LOAD → ChainReplay (RP cmd=1 char struct)
					loadChain := rpc.GetLoadChain()
					if len(loadChain) > 0 {
						resp = append(resp, loadChain...)
					}
					log.Printf("[cached] %s RQ_LOAD → chain replay %d bytes", name, len(loadChain))
				case 14: // RQ_LOAD_ITEM → ChainReplay (page=00 echo, page=01 data)
					page := byte(0)
					if len(frame.Payload) > 20 {
						page = frame.Payload[20]
					}
					chain := rpc.GetItemChain(page)
					if len(chain) > 0 {
						resp = append(resp, chain...)
						log.Printf("[cached] %s RQ_LOAD_ITEM page=%d → chain replay %d bytes", name, page, len(chain))
					}
				}
			}
			if len(resp) > 0 {
				if _, werr := conn.Write(resp); werr != nil {
					log.Printf("[cached] %s write error: %v", name, werr)
					return
				}
			}
		}
		if len(pending) > wire.MaxPayload+wire.HeaderWire {
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

// charIdFromFrame — извлекает charId из RQ_LOAD(11)/RQ_LOAD_ITEM(14)/U_SAVE(16) payload.
// Раскладка: [X][a][charId][accId] — charId на offset 8.
func charIdFromFrame(frame *wire.Frame) uint32 {
	if len(frame.Payload) >= 12 {
		return binary.LittleEndian.Uint32(frame.Payload[8:12])
	}
	return 0
}

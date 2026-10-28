// Package server — TCP-листенеры каналов CacheD64: 2006 (server), 2007 (admin), 2009 (npcDb).
// Канон CServerSocket.cpp: per-conn Reader с rolling key, dispatch по opcode.
package server

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"encoding/binary"
	"encoding/hex"

	"aion-cached/internal/db"
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
	WorldConn    string `json:"world_conn"`
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
		case "world_conn":
			c.WorldConn = v
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
	cfg     Config
	testLn  map[Channel]net.Listener
	SDB     *SettingsDB // DB-персист настроек (nil = RAM-only)
	WorldEx db.Executor // мир-БД (_AionWorldNew114_rc) для DB-генератора (nil = chain-only)
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
	mode := modeServer   // Server64 по умолчанию; NPRelay64 переключает handshake'ом
	var sess rpc.Session // R4.5: сессионные поля из cmd=288 (X/a/accId)
	sessSet := false
	var npcReplay *rpc.NpcSrvReplay // R4.7.1: NPCSvr-коннект (переключается по cmd=201)
	emittedWorldInit := false       // R5: мир-init отработан в этом коннекте
	var pushedVersion bool          // R5 #8 v20: RP_VERSION пуш выдан (после первого C282)
	var startup rpc.StartupTracker  // R4.6: детектор циклов мир-init (новый X = рестарт пары)

	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		pending = append(pending, buf[:n]...)

		for {
			// R3 GQ/GP (дизasm-канон 23.10): relay-кадр = [u16 total][type u8≤5][0xBE]...
			// — короче HeaderWire и без 0xEB-маркера; Feed reject'нул бы его
			// (bad size/marker) и закрыл conn ⇒ детектим на СЫРОМ буфере до Feed.
			if ch == ChanServer && len(pending) >= 4 {
				if typ, total, ok := relayFrame(pending); ok {
					if len(pending) < total {
						break // ждём ещё байт
					}
					log.Printf("[cached] %s RAWR> type=%d total=%d hex=%s", name, typ, total, rawHead(pending[:total]))
					payload := append([]byte(nil), pending[4:total]...)
					pending = pending[total:]
					_ = relayWireDispatch(&mode, typ, payload)
					continue
				}
			}
			if len(pending) < wire.HeaderWire {
				break
			}
			consumed, frame, ferr := dec.Feed(pending)
			if ferr != nil {
				if isErrShort(ferr) {
					break // ждём ещё байт
				}
				// канон CServerSocket: bad size / bad csum → Close
				log.Printf("[cached] %s reject frame: %v — closing conn (canon L2 behavior)", name, ferr)
				return
			}
			// R5 #6 (v19): RAW-лог КАЖДОГО кадра (hex) — вердикт зависших цепочек вслепую невозможен.
			log.Printf("[cached] %s RAW> cmd=%d total=%d hex=%s", name, frame.Cmd, consumed, rawHead(pending[:consumed]))
			pending = pending[consumed:]

			// R4.5: сессионные поля из cmd=288 (первый кадр): [X u32][a u32][accId u16]
			if frame.Cmd == 288 && len(frame.Payload) >= 12 {
				sess = rpc.Session{
					X:     binary.LittleEndian.Uint32(frame.Payload[0:4]),
					A:     frame.Payload[4], // Т38: варьируется (3/4) по сессии
					AccID: binary.LittleEndian.Uint16(frame.Payload[8:10]),
				}
				sessSet = true
				if startup.OnSession(sess.X) {
					// R4.6: новый X = Server64 перезапустил мир (рестарт пары) —
					// сброс флага мир-init для повторной эмиссии (корень крэш-лупа R5 #3/#4:
					// флаг переживал рестарт, тень молчала на новый бёрст).
					emittedWorldInit = false
					log.Printf("[cached] %s новый цикл мир-init (X=%#x) — emittedWorldInit сброшен", name, sess.X)
				}
				log.Printf("[cached] session captured X=%#x a=%d acc=%d", sess.X, sess.A, sess.AccID)
			}

			// R4.6.2: NPCSvr-коннект БЕЗ session-288 — детект ВНЕ sessSet-гейта
			// (старый код: NPCSvr не шлёт 288 → sessSet=false → NPCSvr-путь мёртв;
			// W30-бёрст S=0 — прямое live-доказательство молчания тени).
			if ch == ChanServer && frame.Cmd == 201 && npcReplay == nil {
				npcReplay = rpc.NewNpcSrvReplay()
				log.Printf("[cached] %s NPCSvr-коннект (cmd=201, без session-288) — NPCSvr-режим ВКЛ", name)
			}
			var npcResp []byte
			if ch == ChanServer && npcReplay != nil {
				npcResp = npcReplay.EmitNpcSrv(frame.Cmd, frame.Payload)
			}
			resp := Route(ch, frame)
			// R5 #8 v20: Server64 висит на «Waiting for Cache Version» и не шлёт init
			// (C201/C215) — корень замка #6. В эталоне версия приходит по 2006-каналу
			// (C215→O16, 27Б). Выдаём RP_VERSION превентивно после первого C282-полла.
			if ch == ChanServer && frame.Cmd == 282 && !pushedVersion && npcReplay == nil {
				pushedVersion = true
				resp = append(resp, rpc.O16VersionFrame()...)
				log.Printf("[cached] %s v20: пуш RP_VERSION (O16 27Б) после первого C282", name)
			}
			// R4/R4.5: read-path emission (chain + session-подстановки + acks Т39)
			// R4.6.2: в NPCSvr-режиме login-path выключен (нельзя смешивать каналы)
			if ch == ChanServer && sessSet && npcReplay == nil {
				switch frame.Cmd {
				case 11: // R5: RQ_LOAD → ПОЛНЫЙ мир-init load-сегмент (118 кадров capture-20261023)
					if sessSet && !emittedWorldInit {
						var charRef uint32 = rpc.WorldFlowSess1CharRef
						var charId uint32
						if len(frame.Payload) >= 10 {
							charId = uint32(binary.LittleEndian.Uint16(frame.Payload[8:10]))
							charRef = 0x02000000 | charId
						}
						if s.WorldEx != nil {
							// Этап 5: DB-генератор — данные чара из мир-БД
							ctx := context.Background()
							wd, err := db.GetCharWorldData(ctx, s.WorldEx, charId)
							items, ierr := db.GetUserItems(ctx, s.WorldEx, charId)
							if err == nil && ierr == nil {
								// R4.6: startup-цепочка W5 (120 кадров = world-flow 118 + 0x7A/0xBF хвост)
								chainFrames := rpc.SplitWorldFlowFrames(rpc.GetWorldStartupChain())
								wd2 := rpc.CharWorldData{Lev: wd.Lev, LastNormalX: wd.LastNormalX, LastNormalY: wd.LastNormalY, LastNormalZ: wd.LastNormalZ, ResurrectX: wd.ResX, ResurrectY: wd.ResY, ResurrectZ: wd.ResZ, CharName: "SteLGeN-잉기스온"}
								var itemRows []rpc.DBItemRow
								for _, it := range items {
									itemRows = append(itemRows, rpc.DBItemRow{NameID: it.NameID, Amount: it.Amount, Slot: it.Slot, Warehouse: it.Warehouse})
								}
								dbFrames := rpc.EmitWorldFlowDB(chainFrames, sess.X, charRef, wd2, itemRows)
								for _, df := range dbFrames {
									resp = append(resp, df...)
								}
								emittedWorldInit = true
								log.Printf("[cached] %s RQ_LOAD → world-flow DB %d frames (char=%d lev=%d items=%d)", name, len(dbFrames), charId, wd.Lev, len(items))
							} else {
								log.Printf("[cached] %s RQ_LOAD DB-запрос fail (err=%v ierr=%v) — fallback chain", name, err, ierr)
								loadChain := rpc.EmitWorldStartupLoad(sess.X, charRef, sess.A)
								if len(loadChain) > 0 {
									resp = append(resp, loadChain...)
									emittedWorldInit = true
								}
							}
						} else {
							// chain-replay fallback (WorldEx nil)
							loadChain := rpc.EmitWorldStartupLoad(sess.X, charRef, sess.A)
							if len(loadChain) > 0 {
								resp = append(resp, loadChain...)
								emittedWorldInit = true
							}
							log.Printf("[cached] %s RQ_LOAD → world-flow load %d bytes (session-applied, R5)", name, len(loadChain))
						}
					} else {
						loadChain := rpc.ApplySessionToChain(rpc.GetLoadChain(), sess)
						if len(loadChain) > 0 {
							resp = append(resp, loadChain...)
						}
						log.Printf("[cached] %s RQ_LOAD → chain replay %d bytes (session-applied, fallback)", name, len(loadChain))
					}
				case 14: // RQ_LOAD_ITEM → ChainReplay (page=00 echo, page=01 data) + acks (Т39)
					if emittedWorldInit {
						log.Printf("[cached] %s RQ_LOAD_ITEM skipped (world-init already emitted)", name)
					} else {
						page := byte(0)
						if len(frame.Payload) > 20 {
							page = frame.Payload[20]
						}
						chain := rpc.ApplySessionToChain(rpc.GetItemChain(page), sess)
						if len(chain) > 0 {
							resp = append(resp, chain...)
						}
						if page == 1 { // Т39: RP-подтверждения после data-цепочки (живой порядок)
							resp = append(resp, rpc.BuildAcks(rpc.AcksAfterItemData, sess)...)
							log.Printf("[cached] %s RQ_LOAD_ITEM page=1 → chain %d bytes + %d acks", name, len(chain), len(rpc.AcksAfterItemData))
						} else {
							log.Printf("[cached] %s RQ_LOAD_ITEM page=%d → chain replay %d bytes", name, page, len(chain))
						}
					}
				case 278: // C> 0x116 (world-ready) → вторая пачка RP-подтверждений (Т39)
					resp = append(resp, rpc.BuildAcks(rpc.AcksAfterWorldReady, sess)...)
					log.Printf("[cached] %s world-ready → %d acks", name, len(rpc.AcksAfterWorldReady))
				}
				// R5: мир-init replay — для cmds с captured-цепочками, не покрытых login-path/NPCSvr
				if resp == nil || len(resp) == 0 {
				}
			}
			// R4.6.2: NPCSvr-ответы аппендим вне sessSet-гейта (NPCSvr без 288)
			if len(npcResp) > 0 {
				resp = append(resp, npcResp...)
			}
			if len(resp) > 0 {
				log.Printf("[cached] %s RAW< total=%d frames=%d hex=%s", name, len(resp), countFrames(resp), rawHead(resp))
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

// rawHead — hex-превью кадра для RAW-лога (R5 #6, v19): кадры ≤ 256Б полностью,
// крупные — head 96Б + tail 16Б (endpoints достаточны для сверки цепочек).
func rawHead(b []byte) string {
	const full = 256
	if len(b) <= full {
		return hex.EncodeToString(b)
	}
	return hex.EncodeToString(b[:96]) + "..." + hex.EncodeToString(b[len(b)-16:])
}

// countFrames — число wire-кадров в конкатенированном блобе (walk по u16 total).
func countFrames(b []byte) int {
	n := 0
	for i := 0; i+wire.HeaderWire <= len(b); n++ {
		total := int(binary.LittleEndian.Uint16(b[i : i+2]))
		if total < wire.HeaderWire || i+total > len(b) {
			break
		}
		i += total
	}
	return n
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

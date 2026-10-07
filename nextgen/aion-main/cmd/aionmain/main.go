// aionmain — тень (fork-mode, S4) игрового ядра Server64: слушает копию трафика :7778,
// дешифрует по канону 7.x, диспетчерит по ops.yaml, raw-first лог (S3).
// R2-каркас: хендлеры = заглушки (raw-лог); R3 = MVP движения/чата/инвентаря.
package main

import (
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"aion-main/internal/crypt"
	"aion-main/internal/ops"
	"aion-main/internal/wire"
)

func main() {
	cfg := flag.String("config", "ops.yaml", "YAML-реестр опкодов (S7)")
	listen := flag.String("listen", ":7778", "адрес тени (fork, S4)")
	flag.Parse()

	reg, err := ops.Load(*cfg)
	if err != nil {
		log.Fatalf("FATAL config: %v", err)
	}
	// Канарейка-баннер (S7): конфиг прочитан, ключевой пакет на месте.
	_, hasSMKey := reg.Lookup(0x48, "SM")
	log.Printf("[CANARY] cfg loaded: packets=%d, SM_KEY(0x48)=%v, mode=shadow %s", reg.Count(), hasSMKey, *listen)
	if !hasSMKey {
		log.Fatalf("FATAL: SM_KEY отсутствует в реестре — конфиг битый, откат")
	}

	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatalf("FATAL listen %s: %v", *listen, err)
	}
	log.Printf("[START] aion-main shadow on %s (R2 каркас)", *listen)
	for {
		c, err := ln.Accept()
		if err != nil {
			log.Printf("accept err: %v", err)
			continue
		}
		go handle(c, reg)
	}
}

func handle(conn net.Conn, reg *ops.Registry) {
	defer conn.Close()
	start := time.Now()
	peer := conn.RemoteAddr().String()
	log.Printf("[CONN] %s", peer)

	// --- SM_KEY первым пакетом, ОТКРЫТЫМ (канон 7.x, live-подтверждено) ---
	base := uint32(time.Now().UnixNano()>>17) & 0xFFFFFFFF
	fk := crypt.FalseKey(base)
	body := make([]byte, 9)
	binary.LittleEndian.PutUint16(body[0:2], crypt.EncodeOpcode(0x48))
	body[2] = crypt.ServerPacketCode
	binary.LittleEndian.PutUint16(body[3:5], ^crypt.EncodeOpcode(0x48))
	binary.LittleEndian.PutUint32(body[5:9], fk)
	if _, err := conn.Write(wire.Frame(body)); err != nil {
		log.Printf("[ERR] write SM_KEY: %v", err)
		return
	}
	key := crypt.NewKeyPair(base)

	// --- чтение потока ---
	buf := make([]byte, 1<<16)
	stream := make([]byte, 0, 1<<16)
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			// S3 raw-first: сырые байты в лог до любого парсинга при ошибке кадра
			stream = append(stream, buf[:n]...)
			bodies, tail := wire.Split(stream)
			stream = stream[len(stream)-tail:]
			for _, body := range bodies {
				dec, ok := key.Decrypt(body, crypt.C2S)
				if !ok {
					log.Printf("[RAW-FAIL] %s body(%d): %s", peer, len(body), hex.EncodeToString(body[:min(24, len(body))]))
					continue
				}
				op, payload, ok := wire.ParseClientBody(dec)
				if !ok {
					log.Printf("[PARSE-FAIL] %s: %s", peer, hex.EncodeToString(dec[:min(24, len(dec))]))
					continue
				}
				name := fmt.Sprintf("UNK_0x%04X", op)
				if p, ok := reg.Lookup(op, "CM"); ok {
					name = p.Name
				}
				log.Printf("[C2S] %s %s len=%d payload=%s t=%.1fs", peer, name, len(payload),
					hex.EncodeToString(payload[:min(32, len(payload))]), time.Since(start).Seconds())
			}
		}
		if err == io.EOF {
			log.Printf("[DISC] %s t=%.1fs", peer, time.Since(start).Seconds())
			return
		}
		if err != nil {
			log.Printf("[ERR] %s: %v", peer, err)
			return
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
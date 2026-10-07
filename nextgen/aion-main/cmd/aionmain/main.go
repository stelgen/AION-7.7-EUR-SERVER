// aionmain — тень (fork-mode, S4) игрового ядра Server64: слушает копию трафика :7778,
// дешифрует по канону 7.x, диспетчерит по ops.yaml, raw-first лог (S3), мир-стейт по раскладкам (R3.5),
// динамические подмены раскладок (R3.6), tap-режим разбора КОПИИ трафика (R4, FORK-SPEC).
package main

import (
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"aion-main/internal/crypt"
	"aion-main/internal/handlers"
	"aion-main/internal/integrations"
	"aion-main/internal/ops"
	"aion-main/internal/tap"
	"aion-main/internal/wire"
	"aion-main/internal/world"
)

var hs handlers.Registry

func main() {
	mode := flag.String("mode", "shadow", "shadow|tap (FORK-SPEC: tap = разбор КОПИИ трафика)")
	cfg := flag.String("config", "ops.yaml", "YAML-реестр опкодов (S7)")
	layouts := flag.String("layouts", "internal/world/testdata/layouts", "раскладки мира (capture S4-паритет)")
	listen := flag.String("listen", ":7778", "адрес тени (fork, S4)")
	s2cHex := flag.String("s2c", "", "tap: hex-дамп S2C (tshark follow)")
	c2sHex := flag.String("c2s", "", "tap: hex-дамп C2S")
	patchYaml := flag.String("patches", "internal/world/testdata/layouts.yaml", "динамические подмены раскладок (R3.6)")
	flag.Parse()

	reg, err := ops.Load(*cfg)
	if err != nil {
		log.Fatalf("FATAL config: %v", err)
	}

	// --- R4 tap-режим: пассивный разбор копии трафика, VERDICT против реестра ---
	if *mode == "tap" {
		s2c := readHexFile(*s2cHex)
		c2s := readHexFile(*c2sHex)
		tap.Run(s2c, c2s, reg, func(f string, a ...any) { log.Printf(f, a...) })
		return
	}

	// --- R3.5 мир-раскладки + R3.6 подмены ---
	wls, werr := world.New(*layouts)
	if werr != nil {
		log.Printf("[WARN] layouts: %v (мир без раскладок)", werr)
	}
	patches, perr := world.LoadPatches(*patchYaml)
	if perr != nil {
		log.Printf("[WARN] patches: %v", perr)
	}

	// --- R3 хендлеры; initFn = мир-последовательность при CM_LEVEL_READY (с R3.6-патчами) ---
	hs = handlers.Build(func(s *handlers.Session, send handlers.Sender) {
		if wls == nil {
			return
		}
		vals := map[string]uint32{"hp": s.HP, "mp": s.MP}
		for _, e := range wls.InitSequence() {
			raw := []byte(e[1])
			if p, ok := patches[e[0]]; ok {
				raw = world.Apply(raw, p, vals)
			}
			send(e[0], raw)
		}
		log.Printf("[WORLD] init sequence sent (%d раскладок, %d Б)", wls.Count(), wls.Bytes())
	})

	// --- Канарейка-баннер (S7): конфиг прочитан, ключевой пакет на месте ---
	_, hasSMKey := reg.Lookup(0x48, "SM")
	log.Printf("[CANARY] cfg loaded: packets=%d, SM_KEY(0x48)=%v, layouts=%d(%dB), mode=shadow %s",
		reg.Count(), hasSMKey, wls.Count(), wls.Bytes(), *listen)
	if !hasSMKey {
		log.Fatalf("FATAL: SM_KEY отсутствует в реестре — конфиг битый, откат")
	}

	// --- S2: self-статус HTTP ---
	go func() {
		http.HandleFunc("/status", func(w http.ResponseWriter, _ *http.Request) {
			cd := integrations.Statuses()
			fmt.Fprintf(w, "{\"svc\":\"aion-main\",\"mode\":\"shadow\",\"packets\":%d,\"cached\":%v,\"ts\":\"%s\"}",
				reg.Count(), cd[0].Online, time.Now().Format(time.RFC3339))
		})
		if err := http.ListenAndServe("127.0.0.1:10221", nil); err != nil {
			log.Printf("[S2] status http: %v", err)
		}
	}()

	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatalf("FATAL listen %s: %v", *listen, err)
	}
	log.Printf("[START] aion-main shadow on %s (R3.5+R4)", *listen)
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
	body := make([]byte, 9)
	binary.LittleEndian.PutUint16(body[0:2], crypt.EncodeOpcode(0x48))
	body[2] = crypt.ServerPacketCode
	binary.LittleEndian.PutUint16(body[3:5], ^crypt.EncodeOpcode(0x48))
	binary.LittleEndian.PutUint32(body[5:9], crypt.FalseKey(base))
	if _, err := conn.Write(wire.Frame(body)); err != nil {
		log.Printf("[ERR] write SM_KEY: %v", err)
		return
	}
	// канон: ДВА независимых ключа (EncryptionKeyPair keys[SERVER]/keys[CLIENT])
	srvKey := crypt.NewKeyPair(base) // SM-отправка
	clKey := crypt.NewKeyPair(base)  // C2S-приём
	sess := &handlers.Session{Peer: peer, HP: 12345, MP: 678}

	// --- чтение потока ---
	buf := make([]byte, 1<<16)
	stream := make([]byte, 0, 1<<16)
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			// S3 raw-first: сырые байты до любого парсинга при ошибке кадра
			stream = append(stream, buf[:n]...)
			bodies, tail := wire.Split(stream)
			stream = stream[len(stream)-tail:]
			for _, body := range bodies {
				dec, ok := clKey.Decrypt(body, crypt.C2S)
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
				send := senderFor(conn, srvKey, reg)
				if h, ok := hs[name]; ok {
					h(sess, payload, send, log.Default())
				} else {
					log.Printf("[UNHANDLED] %s (R3.5+ backlog)", name)
				}
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

// senderFor — отправка SM-пакета: имя -> op (ops.yaml) -> BuildServerFrame -> serverKey encrypt -> write.
func senderFor(conn net.Conn, key *crypt.KeyPair, reg *ops.Registry) handlers.Sender {
	return func(name string, payload []byte) {
		p, ok := reg.Lookup2(name)
		if !ok {
			log.Printf("[SEND-SKIP] %s: нет в ops.yaml (реестр)", name)
			return
		}
		frame := wire.BuildServerFrame(p.Op, payload)
		key.Encrypt(frame[2:]) // шифруем тело (size не шифруется — канон AionServerPacket.write)
		if _, err := conn.Write(frame); err != nil {
			log.Printf("[ERR] write %s: %v", name, err)
		}
	}
}

func readHexFile(path string) []byte {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("FATAL read %s: %v", path, err)
	}
	out := make([]byte, 0, len(data)/2)
	var hi byte
	hiSet := false
	for _, c := range data {
		var v byte
		switch {
		case c >= '0' && c <= '9':
			v = c - '0'
		case c >= 'a' && c <= 'f':
			v = c - 'a' + 10
		case c >= 'A' && c <= 'F':
			v = c - 'A' + 10
		default:
			continue
		}
		if !hiSet {
			hi = v << 4
			hiSet = true
		} else {
			out = append(out, hi|v)
			hiSet = false
		}
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

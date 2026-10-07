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
	"net/http"
	"time"

	"aion-main/internal/crypt"
	"aion-main/internal/handlers"
	"aion-main/internal/integrations"
	"aion-main/internal/ops"
	"aion-main/internal/wire"
	"aion-main/internal/world"
)

var hs handlers.Registry

func main() {
	cfg := flag.String("config", "ops.yaml", "YAML-реестр опкодов (S7)")
	layouts := flag.String("layouts", "internal/world/testdata/layouts", "раскладки мира (capture S4-паритет)")
	listen := flag.String("listen", ":7778", "адрес тени (fork, S4)")
	flag.Parse()

	reg, err := ops.Load(*cfg)
	if err != nil {
		log.Fatalf("FATAL config: %v", err)
	}
	wls, werr := world.New(*layouts)
	if werr != nil {
		log.Printf("[WARN] layouts: %v (мир без раскладок)", werr)
	}
	hs = handlers.Build(func(send handlers.Sender) {
		if wls == nil {
			return
		}
		for _, e := range wls.InitSequence() {
			send(e[0], []byte(e[1]))
		}
		log.Printf("[WORLD] init sequence sent (%d раскладок, %d Б)", wls.Count(), wls.Bytes())
	})
	// Канарейка-баннер (S7): конфиг прочитан, ключевой пакет на месте.
	_, hasSMKey := reg.Lookup(0x48, "SM")
	log.Printf("[CANARY] cfg loaded: packets=%d, SM_KEY(0x48)=%v, layouts=%d(%dB), mode=shadow %s", reg.Count(), hasSMKey, wls.Count(), wls.Bytes(), *listen)
	if !hasSMKey {
		log.Fatalf("FATAL: SM_KEY отсутствует в реестре — конфиг битый, откат")
	}

	go func() {
		http.HandleFunc("/status", func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprintf(w, "{\"svc\":\"aion-main\",\"mode\":\"shadow\",\"packets\":%d,\"cached\":%v,\"ts\":\"%s\"}",
				reg.Count(), integrations.Statuses()[0].Online, time.Now().Format(time.RFC3339))
		})
		if err := http.ListenAndServe("127.0.0.1:10221", nil); err != nil {
			log.Printf("[S2] status http: %v", err)
		}
	}()

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
	// канон: ДВА независимых ключа (EncryptionKeyPair keys[SERVER]/keys[CLIENT])
	srvKey := crypt.NewKeyPair(base) // SM-отправка
	clKey := crypt.NewKeyPair(base)  // C2S-приём
	sess := &handlers.Session{Peer: peer}

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
				// R3: диспетчеризация MVP-хендлеров; Sender = шифрованный SM-фрейм (serverKey)
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Package main — forkprobe: синтетический клиент для fork-режима aion-gate.
// Прогон living-обмена «клиент → наш гейт(fork) → оригинальный AuthGateD»:
//   welcome оригинала (passthrough) → CM_AUTH_GG → ответ орига (passthrough) →
//   26b-пинги 0x05/0x02 (best-effort).
// Локальные вердикты «raw байт-в-байт» печатаются здесь; параллельно наш гейт
// пишет свои FORK-сравнения (ориг vs наш shadow) в gate-prod.log.
package main

import (
	"encoding/binary"
	"encoding/hex"
	"flag"
	"log"
	"net"
	"time"

	"aion-gate/internal/proto"
)

func main() {
	addr := flag.String("addr", "192.168.0.125:2106", "fork-гейт addr:port")
	pings := flag.Bool("pings", true, "после AUTH_GG послать 26b-пинги 0x05/0x02 (best-effort)")
	wait := flag.Int("wait", 3, "секунд держать соединение после обменов")
	flag.Parse()

	key1 := proto.GenerateInitialKey(0x04bd)
	bf1, err := proto.NewBlowfish(key1[:])
	if err != nil {
		log.Fatalf("key1: %v", err)
	}
	conn, err := net.DialTimeout("tcp", *addr, 10*time.Second)
	if err != nil {
		log.Fatalf("dial %s: %v", *addr, err)
	}
	defer conn.Close()
	log.Printf("forkprobe: подключён к %s", *addr)

	// welcome (passthrough от оригинала)
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	wraw, err := proto.ReadFrame(conn)
	if err != nil {
		log.Fatalf("welcome: %v", err)
	}
	pt, err := proto.DecryptPrimary(bf1, wraw)
	if err != nil || len(pt) < 173 {
		log.Fatalf("welcome не расшифрован key1: len=%d err=%v", len(wraw), err)
	}
	sid := binary.LittleEndian.Uint32(pt[1:5])
	v := binary.LittleEndian.Uint32(pt[5:9])
	var key2 [16]byte
	copy(key2[:], pt[153:169])
	log.Printf("welcome: wire=%d ECB=%d sid=%d(0x%08x) V=%d(0x%08x) key2=%s",
		len(wraw)+2, len(wraw), sid, sid, v, v, hex.EncodeToString(key2[:]))

	bf2, err := proto.NewBlowfish(key2[:])
	if err != nil {
		log.Fatalf("key2: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	// CM_AUTH_GG: [07][sid][19×0] (24Б pt) — форма live-клиента 7.7
	gg := make([]byte, 24)
	gg[0] = 0x07
	binary.LittleEndian.PutUint32(gg[1:5], sid)
	if _, err := conn.Write(proto.WriteFrame(proto.EncryptSecondary(bf2, gg))); err != nil {
		log.Fatalf("authgg send: %v", err)
	}
	rep, err := proto.ReadFrame(conn)
	if err != nil {
		log.Fatalf("authgg reply: %v", err)
	}
	drep, err := proto.DecryptSecondary(bf2, rep)
	if err != nil {
		log.Fatalf("authgg reply не расшифрован key2: %v", err)
	}
	log.Printf("O>us AUTH_GG reply: wire=%d pt(%d)=%s", len(rep)+2, len(drep), hex.EncodeToString(drep))
	want := make([]byte, 32) // live-форма: [0b][sid][27×0]
	want[0] = 0x0b
	binary.LittleEndian.PutUint32(want[1:5], sid)
	if string(drep) == string(want) {
		log.Printf("VERDICT AUTH_GG: SAME — raw pt байт-в-байт == live-форма [0b][sid][27×0]")
	} else {
		log.Printf("VERDICT AUTH_GG: DIFF pt=%s want=%s", hex.EncodeToString(drep), hex.EncodeToString(want))
	}

	if *pings {
		// 26b-пинги: до логина ориг может молчать — best-effort, короткий дедлайн
		for _, op := range []byte{0x05, 0x02} {
			_ = conn.SetReadDeadline(time.Now().Add(6 * time.Second))
			p := make([]byte, 24)
			p[0] = op
			if _, err := conn.Write(proto.WriteFrame(proto.EncryptSecondary(bf2, p))); err != nil {
				log.Printf("op=%02x send: %v (дальше пинги пропущены)", op, err)
				break
			}
			rep, err := proto.ReadFrame(conn)
			if err != nil {
				log.Printf("op=%02x: ответа нет до дедлайна (%v) — ориг молчит до логина (норма)", op, err)
				break // после таймаута поток может рассинхрониться
			}
			drep, derr := proto.DecryptSecondary(bf2, rep)
			if derr != nil {
				log.Printf("op=%02x: reply wire=%d не расшифрован: %v", op, len(rep)+2, derr)
				break
			}
			log.Printf("O>us op=%02x reply: wire=%d pt(%d)=%s", op, len(rep)+2, len(drep), hex.EncodeToString(drep))
		}
	}
	time.Sleep(time.Duration(*wait) * time.Second)
	log.Printf("forkprobe: готово")
}
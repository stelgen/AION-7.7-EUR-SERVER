// forkauthd — R5 fork-прокси authd-wire (2110):
//
//	наш aion-gate ──front(:2116)──▶ forkauthd ──живой путь──▶ ориг L2Authd (:2110)
//	                                     └──копия фреймов──▶ наш aion-authd (shadow :2117)
//
// Живой путь юзера 1-в-1 (ответы ТОЛЬКО от ориг); shadow-ответы — в лог (N>) с
// SAME/DIFF против ориг-ответа (O>) по ключу (frame, sid, type). Ориг НЕ трогаем
// лишним трафиком: копируются ТОЛЬКО фреймы, которые сам шлёт наш гейт.
//
// Лог: C> (гейт→fork) / O> (ориг→гейт) / N> (shadow→fork) полные hex → дифф O-vs-N.
// Токены динамичны (type=3/7 содержат Rnd) — байтовый DIFF там ожидаем; арбитр =
// структура (длины/зоны нулей), а не побайтовое равенство.
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"aion-authd/internal/wire"
)

// tag — ключ сопоставления ответов O> и N>.
type tag struct {
	ft  byte   // тип фрейма ([02]/[03]/[01])
	sid uint32 // sid сессии
	sub byte   // для [02]: первый байт тела (type-байт ответа)
}

func keyOf(f wire.Frame) tag {
	if f.Type == wire.FPacket && len(f.Blob) > 0 {
		return tag{f.Type, f.Sid, f.Blob[0]}
	}
	return tag{f.Type, f.Sid, 0}
}

func dump(tagS, dir string, raw []byte, f wire.Frame) string {
	n := len(raw)
	if n > 400 {
		n = 400
	}
	head := hex.EncodeToString(raw[:n])
	if len(raw) > n {
		head += "..."
	}
	if f.Type == wire.FPacket {
		op := byte(0)
		if len(f.Blob) > 0 {
			op = f.Blob[0]
		}
		return fmt.Sprintf("%s %s [%02x] sid=%d type=%02x len=%d hex=%s", tagS, dir, f.Type, f.Sid, op, len(raw), head)
	}
	return fmt.Sprintf("%s %s [%02x] sid=%d len=%d hex=%s", tagS, dir, f.Type, f.Sid, len(raw), head)
}

type fwd struct {
	gate, orig, shadow net.Conn

	wmu sync.Mutex // запись в gate (только O>-горутина пишет)

	mu    sync.Mutex
	lastO map[tag][]byte
	log   *log.Logger
}

// pump — читает фреймы из src целиком (сырые), зовёт onFrame.
func (fw *fwd) pump(name string, src net.Conn, onFrame func([]byte, wire.Frame)) {
	for {
		raw, f, err := wire.ReadFrameRaw(src)
		if err != nil {
			fw.log.Printf("%s: EOF/err: %v", name, err)
			return
		}
		onFrame(raw, f)
	}
}

// run — одна сессия fork'а (один коннект гейта).
func run(gate net.Conn, origAddr, shadowAddr string, logger *log.Logger) {
	defer gate.Close()
	fw := &fwd{gate: gate, lastO: map[tag][]byte{}, log: logger}
	logger.Printf("=== gate connected %s ===", gate.RemoteAddr())

	orig, err := net.DialTimeout("tcp", origAddr, 5*time.Second)
	if err != nil {
		logger.Printf("ORIG dial FAIL: %v — сессия закрыта (живой путь недоступен)", err)
		return
	}
	fw.orig = orig
	defer orig.Close()
	logger.Printf("orig connected %s (живой путь)", orig.RemoteAddr())

	shadow, serr := net.DialTimeout("tcp", shadowAddr, 3*time.Second)
	if serr != nil {
		logger.Printf("shadow dial FAIL (копия отключена, живой путь работает): %v", serr)
	} else {
		fw.shadow = shadow
		defer shadow.Close()
		logger.Printf("shadow connected %s (наш authd)", shadow.RemoteAddr())
	}

	done := make(chan struct{}, 2)

	// Гейт → ориг (+копия в shadow)
	go fw.pump("gate", gate, func(raw []byte, f wire.Frame) {
		logger.Printf("%s", dump("C>", "G>F", raw, f))
		if _, err := orig.Write(raw); err != nil {
			logger.Printf("orig write: %v", err)
			done <- struct{}{}
			return
		}
		if fw.shadow != nil {
			if _, err := fw.shadow.Write(raw); err != nil {
				logger.Printf("shadow write (отключаю копию): %v", err)
				fw.shadow = nil
			}
		}
	})

	// Ориг → гейт (живой путь) + запоминаем последний O> для диффа
	go fw.pump("orig", orig, func(raw []byte, f wire.Frame) {
		k := keyOf(f)
		fw.mu.Lock()
		fw.lastO[k] = raw
		fw.mu.Unlock()
		logger.Printf("%s", dump("O>", "O>G", raw, f))
		fw.wmu.Lock()
		_, werr := fw.gate.Write(raw)
		fw.wmu.Unlock()
		if werr != nil {
			logger.Printf("gate write: %v", werr)
			done <- struct{}{}
		}
	})

	// Shadow → только лог + diff (юзеру НЕ идёт)
	if fw.shadow != nil {
		go fw.pump("shadow", fw.shadow, func(raw []byte, f wire.Frame) {
			k := keyOf(f)
			fw.mu.Lock()
			last, ok := fw.lastO[k]
			fw.mu.Unlock()
			verdict := "N-ONLY (ориг не отвечал на этот ключ)"
			if ok {
				if string(last) == string(raw) {
					verdict = "SAME"
				} else {
					verdict = fmt.Sprintf("DIFF (len O=%d N=%d)", len(last), len(raw))
				}
			}
			logger.Printf("%s VERDICT=%s", dump("N>", "S>F", raw, f), verdict)
			if ok && verdict != "SAME" {
				logger.Printf("  O-full hex=%s", hex.EncodeToString(last))
				logger.Printf("  N-full hex=%s", hex.EncodeToString(raw))
			}
		})
	}

	<-done
	logger.Printf("=== gate session closed ===")
}

func main() {
	front := flag.String("front", ":2116", "listen (наш гейт ходит сюда)")
	origAddr := flag.String("orig", "127.0.0.1:2110", "ориг L2Authd (живой путь)")
	shadowAddr := flag.String("shadow", "127.0.0.1:2117", "наш aion-authd (копия)")
	logFile := flag.String("log", "fork-authd.log", "лог-файл (дублируется в stdout)")
	flag.Parse()

	f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		log.Fatalf("log: %v", err)
	}
	logger := log.New(io.MultiWriter(os.Stdout, f), "", log.LstdFlags)

	ln, err := net.Listen("tcp", *front)
	if err != nil {
		logger.Fatalf("listen: %v", err)
	}
	logger.Printf("forkauthd: front=%s orig=%s shadow=%s (R5: живой путь через ориг, ответы shadow — только в лог)", *front, *origAddr, *shadowAddr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			logger.Fatalf("accept: %v", err)
		}
		go run(conn, *origAddr, *shadowAddr, logger)
	}
}
// forkauthd — R5 fork-прокси authd-wire (2110):
//
//	наш aion-gate ──front(:2116)──▶ forkauthd ──живой путь──▶ ориг L2Authd (:2110)
//	                                     └──копия фреймов──▶ наш aion-authd (shadow :2117)
//
// Живой путь юзера 1-в-1 (ответы ТОЛЬКО от ориг); shadow-ответы — в лог (N>) с
// SAME/DIFF против ориг-ответа (O>) по ключу (frame, sid, type). Ориг НЕ трогаем
// лишним трафиком: копируются ТОЛЬКО фреймы, которые сам шлёт наш гейт.
//
// Матчинг O-vs-N (арбитраж 09.10): FIFO-очередь пар per-key (sid,type) — каждая C>-заявка
// ждёт ДВА ответа (ориг+shadow) в порядке прихода; вердикт только при полной паре.
// Старая схема «lastO/lastCT» давала ложные N-ONLY, когда shadow отвечал быстрее ориг
// (гонка) — теперь N, пришедший раньше O, ждёт его и вердикт вычисляется на втором ответе.
// Незапрошенные фреймы (authd сам пушит type=4) = SINGLE, пары не портят.
package main

import (
	"bytes"
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

// pair — одна заявка C>, ждущая двух ответов (FIFO per-key).
type pair struct {
	cAt  time.Time
	oRaw []byte // ответ ориг (nil = ещё не пришёл)
	nRaw []byte // ответ shadow (nil = ещё не пришёл)
}

type fwd struct {
	gate, orig, shadow net.Conn

	wmu sync.Mutex // запись в gate (только O>-горутина пишет)

	mu   sync.Mutex
	keys map[tag][]*pair // FIFO очередей in-flight пар по ключу
	log  *log.Logger
}

// pairTimeout — сколько ждать вторую сторону пары (мир/ориг молчит >60с = дроп).
const pairTimeout = 60 * time.Second

// onSide — фиксирует ответ стороны ("o"/"n") в FIFO-очередь пар по ключу ОТВЕТА
// (заявка C> и ответ живут в разных ключах: {00,sid} vs {02,sid,type}); пара
// создаётся первым пришедшим и закрывается вторым — гонка N-быстрее-O рулится.
func (fw *fwd) onSide(k tag, side string, raw []byte) (verdict string, done bool, oRaw, nRaw []byte) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	q := fw.keys[k]
	for i, p := range q {
		if (side == "o" && p.oRaw == nil) || (side == "n" && p.nRaw == nil) {
			if side == "o" {
				p.oRaw = raw
			} else {
				p.nRaw = raw
			}
			if p.oRaw != nil && p.nRaw != nil { // пара собрана → снять с очереди
				q = append(q[:i], q[i+1:]...)
				if len(q) == 0 {
					delete(fw.keys, k)
				} else {
					fw.keys[k] = q
				}
				if bytes.Equal(p.oRaw, p.nRaw) {
					return "SAME", true, p.oRaw, p.nRaw
				}
				return fmt.Sprintf("DIFF (len O=%d N=%d)", len(p.oRaw), len(p.nRaw)), true, p.oRaw, p.nRaw
			}
			fw.keys[k] = q
			return "WAIT (пара ждёт вторую сторону)", false, nil, nil
		}
	}
	np := &pair{cAt: time.Now()} // нет слота — первый пришедший открывает пару
	if side == "o" {
		np.oRaw = raw
	} else {
		np.nRaw = raw
	}
	fw.keys[k] = append(q, np)
	return "WAIT (пара ждёт вторую сторону)", false, nil, nil
}

// sweepQueue — дроп зависших пар (>60с без второго ответа) — защита от роста карты.
func (fw *fwd) sweepQueue() {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	now := time.Now()
	for k, q := range fw.keys {
		kept := q[:0]
		for _, p := range q {
			if now.Sub(p.cAt) > pairTimeout {
				fw.log.Printf("ARBITRATION-DROP key=[%02x/%d/%02x] o=%v n=%v (возраст >%s)",
					k.ft, k.sid, k.sub, p.oRaw != nil, p.nRaw != nil, pairTimeout)
				continue
			}
			kept = append(kept, p)
		}
		if len(kept) == 0 {
			delete(fw.keys, k)
		} else {
			fw.keys[k] = kept
		}
	}
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
	fw := &fwd{gate: gate, keys: map[tag][]*pair{}, log: logger} // gate ОБЯЗАТЕЛЕН: O>-pump пишет в него (nil → panic, поймано 07.10)
	logger.Printf("=== gate connected %s ===", gate.RemoteAddr())
	go func() { // свип зависших пар
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for range t.C {
			fw.sweepQueue()
		}
	}()

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

	// Ориг → гейт (живой путь) + фиксация в арбитраже
	go fw.pump("orig", orig, func(raw []byte, f wire.Frame) {
		k := keyOf(f)
		verdict, pairDone, oRaw, nRaw := fw.onSide(k, "o", raw)
		if pairDone && verdict != "SAME" {
			logger.Printf("  O-full hex=%s", hex.EncodeToString(oRaw))
			logger.Printf("  N-full hex=%s", hex.EncodeToString(nRaw))
		}
		logger.Printf("%s VERDICT=%s", dump("O>", "O>G", raw, f), verdictLine(verdict, pairDone))
		fw.wmu.Lock()
		_, werr := fw.gate.Write(raw)
		fw.wmu.Unlock()
		if werr != nil {
			logger.Printf("gate write: %v", werr)
			done <- struct{}{}
		}
	})

	// Shadow → только лог + вердикт при полной паре (юзеру НЕ идёт)
	if fw.shadow != nil {
		go fw.pump("shadow", fw.shadow, func(raw []byte, f wire.Frame) {
			k := keyOf(f)
			verdict, pairDone, oRaw, nRaw := fw.onSide(k, "n", raw)
			logger.Printf("%s VERDICT=%s", dump("N>", "S>F", raw, f), verdictLine(verdict, pairDone))
			if pairDone && verdict != "SAME" {
				logger.Printf("  O-full hex=%s", hex.EncodeToString(oRaw))
				logger.Printf("  N-full hex=%s", hex.EncodeToString(nRaw))
			}
		})
	}

	<-done
	logger.Printf("=== gate session closed ===")
}

// verdictLine — человекочитаемая строка вердикта: для неполной пары вердикт не финален.
func verdictLine(v string, done bool) string {
	if done {
		return v
	}
	return v // WAIT/SINGLE уже помечены текстом
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

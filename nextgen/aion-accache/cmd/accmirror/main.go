// accmirror v1.3 — R1 capture-прокси aion-accache (паттерн FORK-SPEC §2 "Mirror-копия")
// + SHADOW-TAP с АВТО-РЕКОННЕКТОМ (R4 live A/B): -shadow addr — фоновый дайлер держит
// коннект к тени (aion-accache :2222); рестарт тени НЕ рвёт тап навсегда (урок v1.2:
// после рестарта тени тап умирал до конца клиентской сессии — SHADOW0 WRITE FAIL лес).
// Копия каждого C> кадра уходит в тень, ответы тени логируются тегом S> (один лог с C>/O>).
// front :2220 (настоящий порт ориг-ACS) -> back :2221 (копия ACS с байтовой правкой порта).
// Лог raw-first (LOGGING-SPEC): каждый ФРЕЙМ [u16 len][u16 cmd][m][u16 ~cmd][payload]
// (len = ПОЛНАЯ длина кадра, вкл. len-поле; m: C2S=0xEB, S2C=0xEC) целиком hex'ом с
// направлением C> / O> / S> + разбор заголовка. v1.0 имел офф-бай-2 (total=len+2) — кадры
// неправильно резались (байты в логе при этом НЕ терялись). Если фрейминг не сходится —
// пишем чанк RAW> целиком и продолжаем (никаких молчаливых съеданий).
package main

import (
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const banner = "accmirror v1.3 (aion-accache R1, len=total, shadow-tap auto-reconnect)"

var (
	lg    *log.Logger
	mu    sync.Mutex
	stats struct {
		framesC, framesO, framesS, rawChunks uint64
	}
)

// shadowTap — держатель тень-коннекта с авто-реконнектом (v1.3).
// run() = фоновый дайлер + ридер (ответы тени -> S>); send() = отправка C-кадра;
// нет коннекта в момент кадра = кадр в тень НЕ доходит (SHADOW SKIP), дайлер пересобирает.
type shadowTap struct {
	mu   sync.Mutex
	addr string
	c    net.Conn
}

func newShadowTap(addr string) *shadowTap {
	t := &shadowTap{addr: addr}
	go t.run()
	return t
}

func (t *shadowTap) run() {
	for {
		c, err := net.DialTimeout("tcp", t.addr, 3*time.Second)
		if err != nil {
			lg.Printf("SHADOW DIAL FAIL: %v (retry 2s)", err)
			time.Sleep(2 * time.Second)
			continue
		}
		t.mu.Lock()
		t.c = c
		t.mu.Unlock()
		lg.Printf("SHADOW (RE)UP %s", t.addr)
		t.readLoop(c)
		t.mu.Lock()
		if t.c == c {
			t.c = nil
		}
		t.mu.Unlock()
		_ = c.Close()
		lg.Printf("SHADOW DOWN (redial)")
	}
}

// readLoop — ответы тени -> S> (feed с forward=false, dst/tap не нужны).
func (t *shadowTap) readLoop(c net.Conn) {
	var buf []byte
	chunk := make([]byte, 16384)
	for {
		n, err := c.Read(chunk)
		if n > 0 {
			buf = append(buf, chunk[:n]...)
			buf = feed(nil, nil, buf, "S", false)
		}
		if err != nil {
			lg.Printf("S> closed: %v", err)
			return
		}
	}
}

func (t *shadowTap) send(fr []byte) {
	t.mu.Lock()
	c := t.c
	t.mu.Unlock()
	if c == nil {
		mu.Lock()
		lg.Printf("SHADOW SKIP (no conn) len=%d", len(fr))
		mu.Unlock()
		return
	}
	_ = c.SetWriteDeadline(time.Now().Add(30 * time.Second))
	if _, err := c.Write(fr); err != nil {
		lg.Printf("SHADOW WRITE FAIL: %v — redial идёт", err)
		t.mu.Lock()
		if t.c == c {
			t.c = nil
		}
		t.mu.Unlock()
		_ = c.Close()
	}
}

// feed разбирает поток и логирует полные фреймы; возвращает нераспарсенный хвост.
// tap — shadow-получатель каждого распарсенного кадра (v1.3); forward=false = dst не трогаем.
func feed(dst net.Conn, tap *shadowTap, buf []byte, tag string, forward bool) []byte {
	for {
		if len(buf) < 2 {
			return buf
		}
		total := int(binary.LittleEndian.Uint16(buf[:2])) // len = полная длина кадра (вкл. len-поле)
		if total < 7 || total > 0x2002 {                  // тело = total-2, лимит тела 0x2000
			mu.Lock()
			stats.rawChunks++
			lg.Printf("%s> RAW chunk len=%d hex=%s (framing out of range: total=%d)", tag, len(buf), hex.EncodeToString(buf), total)
			mu.Unlock()
			if forward {
				_ = dst.SetWriteDeadline(time.Now().Add(30 * time.Second))
				if _, err := dst.Write(buf); err != nil {
					lg.Printf("%s WRITE FAIL: %v", tag, err)
				}
			}
			return nil
		}
		if len(buf) < total {
			return buf // ждём остаток кадра
		}
		fr := buf[:total]
		line := headLine(fr)
		mu.Lock()
		if tag == "C" {
			stats.framesC++
		} else if tag == "O" {
			stats.framesO++
		} else {
			stats.framesS++
		}
		lg.Printf("%s> frame len=%d %s hex=%s", tag, total, line, hex.EncodeToString(fr))
		mu.Unlock()
		if forward {
			_ = dst.SetWriteDeadline(time.Now().Add(30 * time.Second))
			if _, err := dst.Write(fr); err != nil {
				lg.Printf("%s WRITE FAIL: %v", tag, err)
			}
		}
		if tap != nil {
			tap.send(fr)
		}
		buf = buf[total:]
	}
}

// headLine разбирает [u16 len-2][u16 cmd][0xEB][u16 ~cmd] — канон dispatch-77.md.
func headLine(fr []byte) string {
	if len(fr) < 6 {
		return fmt.Sprintf("hdr=incomplete(%d)", len(fr))
	}
	cmd := binary.LittleEndian.Uint16(fr[2:4])
	marker := fr[4]
	xcmd := binary.LittleEndian.Uint16(fr[5:7])
	ok := (marker == 0xEB || marker == 0xEC) && uint16(^cmd&0xFFFF) == xcmd
	return fmt.Sprintf("cmd=0x%02X(~0x%04X marker=0x%02X ok=%v) payload=%d", cmd, xcmd, marker, ok, len(fr)-7)
}

func pump(dst net.Conn, src net.Conn, tag string, forward bool, wg *sync.WaitGroup, tap *shadowTap) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			lg.Printf("%s PANIC: %v", tag, r)
		}
	}()
	var buf []byte
	chunk := make([]byte, 16384)
	for {
		n, err := src.Read(chunk)
		if n > 0 {
			buf = append(buf, chunk[:n]...)
			if len(buf) > 1<<20 { // защита от мусорного потока без фреймов
				mu.Lock()
				stats.rawChunks++
				lg.Printf("%s> RAW overflow len=%d hex(first512)=%s", tag, len(buf), hex.EncodeToString(buf[:512]))
				mu.Unlock()
				if forward {
					_ = dst.SetWriteDeadline(time.Now().Add(30 * time.Second))
					_, _ = dst.Write(buf)
				}
				buf = nil
			} else {
				buf = feed(dst, tap, buf, tag, forward)
			}
		}
		if err != nil {
			if len(buf) > 0 {
				mu.Lock()
				lg.Printf("%s> TAIL len=%d hex=%s", tag, len(buf), hex.EncodeToString(buf))
				mu.Unlock()
				if forward {
					_, _ = dst.Write(buf)
				}
			}
			lg.Printf("%s> closed: %v", tag, err)
			return
		}
	}
}

func handle(cl net.Conn, backAddr string, tap *shadowTap) {
	defer cl.Close()
	peer := cl.RemoteAddr().String()
	lg.Printf("=== CLIENT %s", peer)
	back, err := net.Dial("tcp", backAddr)
	if err != nil {
		lg.Printf("BACK DIAL FAIL: %v", err)
		return
	}
	defer back.Close()
	lg.Printf("BACK UP %s", backAddr)
	var wg sync.WaitGroup
	wg.Add(2)
	go pump(back, cl, "C", true, &wg, tap)
	go pump(cl, back, "O", true, &wg, nil)
	wg.Wait()
	lg.Printf("=== SESSION END %s", peer)
}

func main() {
	listen := flag.String("listen", "0.0.0.0:2220", "")
	backAddr := flag.String("back", "127.0.0.1:2221", "")
	shadowAddr := flag.String("shadow", "", "shadow-tap (A/B тень aion-accache), пусто = выкл")
	logPath := flag.String("log", "accmirror.log", "")
	flag.Parse()
	_ = os.MkdirAll(filepath.Dir(*logPath), 0755)
	f, err := os.OpenFile(*logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	lg = log.New(io.MultiWriter(f, os.Stderr), "", log.LstdFlags)
	lg.Printf("BANNER: %s listen=%s back=%s shadow=%s log=%s", banner, *listen, *backAddr, *shadowAddr, *logPath)
	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		lg.Printf("LISTEN FAIL: %v", err)
		log.Fatal(err)
	}
	lg.Printf("UP")
	go func() { // сводка раз в 60с
		for range time.Tick(60 * time.Second) {
			mu.Lock()
			lg.Printf("STATS framesC=%d framesO=%d framesS=%d rawChunks=%d", stats.framesC, stats.framesO, stats.framesS, stats.rawChunks)
			mu.Unlock()
		}
	}()
	var tap *shadowTap
	if *shadowAddr != "" {
		tap = newShadowTap(*shadowAddr)
	}
	for {
		cl, err := ln.Accept()
		if err != nil {
			lg.Printf("ACCEPT ERR: %v", err)
			continue
		}
		go handle(cl, *backAddr, tap)
	}
}

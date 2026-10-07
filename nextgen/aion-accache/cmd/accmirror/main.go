// accmirror — R1 capture-прокси aion-accache (паттерн FORK-SPEC §2 "Mirror-копия").
// front :2220 (настоящий порт ориг-ACS) -> back :2221 (копия ACS с байтовой правкой порта).
// Лог raw-first (LOGGING-SPEC): каждый ФРЕЙМ [u16 lenMinus2][u16 cmd][0xEB][u16 ~cmd][payload]
// целиком hex'ом с направлением C> (клиент->back) / O> (back->клиент) + разбор заголовка.
// Если фрейминг не сходится — пишем чанк RAW> целиком и продолжаем (никаких молчаливых съеданий).
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

const banner = "accmirror v1.0 (aion-accache R1, 5707423+)"

var (
	lg    *log.Logger
	mu    sync.Mutex
	stats struct {
		framesC, framesO, rawChunks uint64
	}
)

// feed разбирает поток и логирует полные фреймы; возвращает нераспарсенный хвост.
func feed(dst net.Conn, buf []byte, tag string, forward bool) []byte {
	for {
		if len(buf) < 2 {
			return buf
		}
		total := int(binary.LittleEndian.Uint16(buf[:2])) + 2
		if total < 6 || total > 0x2002 { // лимит протокола 0x2000 + допуск на кадр длины
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
		} else {
			stats.framesO++
		}
		lg.Printf("%s> frame len=%d %s hex=%s", tag, total, line, hex.EncodeToString(fr))
		mu.Unlock()
		if forward {
			_ = dst.SetWriteDeadline(time.Now().Add(30 * time.Second))
			if _, err := dst.Write(fr); err != nil {
				lg.Printf("%s WRITE FAIL: %v", tag, err)
			}
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
	ok := marker == 0xEB && uint16(^cmd&0xFFFF) == xcmd
	return fmt.Sprintf("cmd=0x%02X(~0x%04X marker=0x%02X ok=%v) payload=%d", cmd, xcmd, marker, ok, len(fr)-7)
}

func pump(dst net.Conn, src net.Conn, tag string, forward bool, wg *sync.WaitGroup) {
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
				buf = feed(dst, buf, tag, forward)
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

func handle(cl net.Conn, backAddr string) {
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
	go pump(back, cl, "C", true, &wg)
	go pump(cl, back, "O", true, &wg)
	wg.Wait()
	lg.Printf("=== SESSION END %s", peer)
}

func main() {
	listen := flag.String("listen", "0.0.0.0:2220", "")
	backAddr := flag.String("back", "127.0.0.1:2221", "")
	logPath := flag.String("log", "accmirror.log", "")
	flag.Parse()
	_ = os.MkdirAll(filepath.Dir(*logPath), 0755)
	f, err := os.OpenFile(*logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	lg = log.New(io.MultiWriter(f, os.Stderr), "", log.LstdFlags)
	lg.Printf("BANNER: %s listen=%s back=%s log=%s", banner, *listen, *backAddr, *logPath)
	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		lg.Printf("LISTEN FAIL: %v", err)
		log.Fatal(err)
	}
	lg.Printf("UP")
	go func() { // сводка раз в 60с
		for range time.Tick(60 * time.Second) {
			mu.Lock()
			lg.Printf("STATS framesC=%d framesO=%d rawChunks=%d", stats.framesC, stats.framesO, stats.rawChunks)
			mu.Unlock()
		}
	}()
	for {
		cl, err := ln.Accept()
		if err != nil {
			lg.Printf("ACCEPT ERR: %v", err)
			continue
		}
		go handle(cl, *backAddr)
	}
}

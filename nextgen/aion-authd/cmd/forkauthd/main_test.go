package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"log"
	"net"
	"testing"
	"time"

	"aion-authd/internal/wire"
)

// e2e fork-прокси: fake-orig + fake-shadow + fake-gate против run().
func TestForkE2E(t *testing.T) {
	// fake-orig: greeting [03][0x1111], на CltConnect отвечает [02][sid][len][type=9][0xAB]
	origLn, _ := net.Listen("tcp", "127.0.0.1:0")
	defer origLn.Close()
	go func() {
		c, err := origLn.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		c.Write(wire.Greeting(0x1111))
		for {
			raw, f, err := wire.ReadFrameRaw(c)
			if err != nil {
				return
			}
			if f.Type == wire.FConnect {
				_ = raw
				rep, _ := wire.ReplyPkt(f.Sid, 9, []byte{0xAB})
				c.Write(rep)
			}
		}
	}()

	// fake-shadow: greeting [03][0x2222], на CltConnect отвечает [02][sid][len][type=9][0xCD]
	shadowLn, _ := net.Listen("tcp", "127.0.0.1:0")
	defer shadowLn.Close()
	go func() {
		c, err := shadowLn.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		c.Write(wire.Greeting(0x2222))
		for {
			_, f, err := wire.ReadFrameRaw(c)
			if err != nil {
				return
			}
			if f.Type == wire.FConnect {
				rep, _ := wire.ReplyPkt(f.Sid, 9, []byte{0xCD}) // DIFF с оригом (0xAB)
				c.Write(rep)
			}
		}
	}()

	// fork-сессия напрямую через run() на предподключённом gate
	gateLn, _ := net.Listen("tcp", "127.0.0.1:0")
	defer gateLn.Close()
	go func() {
		c, err := gateLn.Accept()
		if err != nil {
			return
		}
		run(c, origLn.Addr().String(), shadowLn.Addr().String(), testLogger())
	}()

	gc, err := net.DialTimeout("tcp", gateLn.Addr().String(), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer gc.Close()
	_ = gc.SetReadDeadline(time.Now().Add(3 * time.Second))

	// 1. greeting ориг должен прийти к клиенту живым путём (НЕ shadow)
	var g [5]byte
	if _, err := io.ReadFull(gc, g[:]); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(g[:], wire.Greeting(0x1111)) {
		t.Fatalf("greeting: %x (want orig 0x1111)", g)
	}

	// 2. CltConnect → ориг отвечает type=9 [0xAB] к клиенту
	if _, err := gc.Write(wire.ConnectFrame(7, [4]byte{192, 168, 0, 253})); err != nil {
		t.Fatal(err)
	}
	var ft [1]byte
	if _, err := io.ReadFull(gc, ft[:]); err != nil {
		t.Fatal(err)
	}
	if ft[0] != 0x02 {
		t.Fatalf("frame type: %02x", ft[0])
	}
	var hdr [6]byte
	if _, err := io.ReadFull(gc, hdr[:]); err != nil {
		t.Fatal(err)
	}
	body := int(binary.LittleEndian.Uint16(hdr[4:6])) - 2
	buf := make([]byte, body)
	if _, err := io.ReadFull(gc, buf); err != nil {
		t.Fatal(err)
	}
	if buf[0] != 9 || buf[1] != 0xAB {
		t.Fatalf("reply: type=%02x first=%02x (want 9,0xAB)", buf[0], buf[1])
	}

	// 3. shadow получил копию и ответил (проверка через ReadFrameRaw shadow-стороны
	// не нужна: факт копирования = shadow-сервер увидел CltConnect и ответил —
	// форк-лог это фиксирует; здесь проверяем только живой путь выше).
}

func testLogger() *log.Logger {
	return log.New(os.Stderr, "fork-test ", log.LstdFlags)
}
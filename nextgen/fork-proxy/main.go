// fork-proxy: клиент <-> ОРИГИНАЛ (реальный обмен), копия клиентского трафика -> НАШ гейт.
// Лог: C> клиент(в оба), O> ответ оригинала(в клиент), N> ответ нашего(только лог).
package main

import (
	"encoding/hex"
	"flag"
	"log"
	"io"
	"net"
	"os"
	"time"
)

var lg *log.Logger

func pump(dst net.Conn, src net.Conn, tag string, toDst bool) {
	buf := make([]byte, 16384)
	hxs := make([]byte, 32768)
	for {
		n, rerr := src.Read(buf)
		if n > 0 {
			ch := buf[:n]
			hex.Encode(hxs, ch)
			if toDst {
				_ = dst.SetWriteDeadline(time.Now().Add(30 * time.Second))
				_, werr := dst.Write(ch)
				if werr != nil {
					lg.Printf("%s CLOSED (write fail): %v", tag, werr)
					return
				}
				lg.Printf("%s> len=%d hex=%s", tag, n, hxs[:n*2])
			} else {
				lg.Printf("%s> len=%d hex=%s", tag, n, hxs[:n*2])
			}
		}
		if rerr != nil {
			lg.Printf("%s CLOSED: %v", tag, rerr)
			return
		}
	}
}

func handle(cl net.Conn, origAddr, ourAddr string) {
	defer cl.Close()
	ip := cl.RemoteAddr().String()
	lg.Printf("=== CLIENT %s", ip)
	orig, oerr := net.Dial("tcp", origAddr)
	if oerr != nil {
		lg.Printf("ORIG DIAL FAIL: %v", oerr)
		return
	}
	defer orig.Close()
	lg.Printf("ORIG UP %s", origAddr)
	our, nerr := net.Dial("tcp", ourAddr)
	if nerr != nil {
		lg.Printf("OUR DIAL FAIL: %v", nerr)
	} else {
		defer our.Close()
		lg.Printf("OUR UP %s", ourAddr)
	}
	done := make(chan struct{}, 3)
	go func() { pump(orig, cl, "C", true); pump(our, cl, "C", our != nil); done <- struct{}{} }()
	go func() { pump(cl, orig, "O", true); done <- struct{}{} }()
	if our != nil {
		go func() { pump(nil, our, "N", false); done <- struct{}{} }()
	}
	<-done
	lg.Printf("=== SESSION END %s", ip)
}

func main() {
	listen := flag.String("listen", "0.0.0.0:2106", "")
	origAddr := flag.String("orig", "127.0.0.1:2109", "")
	ourAddr := flag.String("our", "127.0.0.1:2116", "")
	flag.Parse()
	f, err := os.OpenFile("fork.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	lg = log.New(io.MultiWriter(f, os.Stderr), "", log.LstdFlags)
	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		lg.Printf("LISTEN FAIL: %v", err)
		log.Fatal(err)
	}
	lg.Printf("fork-proxy up: %s -> orig=%s our=%s", *listen, *origAddr, *ourAddr)
	for {
		cl, err := ln.Accept()
		if err != nil {
			lg.Printf("ACCEPT ERR: %v", err)
			continue
		}
		go handle(cl, *origAddr, *ourAddr)
	}
}
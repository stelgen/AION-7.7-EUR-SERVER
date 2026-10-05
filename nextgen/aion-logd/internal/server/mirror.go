// Mirror — tcp-прокси 2051→upstream с hex-логом ОБАХ направлений (capture оригинала).
package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

func Mirror(ctx context.Context, listen, upstream string) error {
	ln, err := net.Listen("tcp", listen)
	if err != nil {
		return err
	}
	go func() { <-ctx.Done(); _ = ln.Close() }()
	for {
		c, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
			}
			return fmt.Errorf("accept: %w", err)
		}
		go mirrorConn(c, upstream)
	}
}

func mirrorConn(cli net.Conn, upstream string) {
	srv, err := net.Dial("tcp", upstream)
	if err != nil {
		log.Printf("mirror: upstream dial: %v", err)
		_ = cli.Close()
		return
	}
	tag := cli.RemoteAddr().String()
	log.Printf("mirror: [%s] established", tag)
	done := make(chan struct{}, 2)
	go func() { pump(srv, cli, "s2c:"+tag); done <- struct{}{} }()
	go func() { pump(cli, srv, "c2s:"+tag); done <- struct{}{} }()
	<-done
	_ = srv.Close()
	_ = cli.Close()
	<-done
}

func pump(src, dst net.Conn, tag string) {
	buf := make([]byte, 8192)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			f, oerr := os.OpenFile("C:/logd-capture/mirror-io.hex", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
			if oerr == nil {
				fmt.Fprintf(f, "%s %s %d % X\n", time.Now().Format("15:04:05.000"), tag, n, buf[:n])
				f.Close()
			}
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return
			}
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("mirror %s: %v", tag, err)
			}
			return
		}
	}
}

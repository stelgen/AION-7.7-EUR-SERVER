// probe — диагностический клиент authd (2110): шлёт login-blob как гейт и
// печатает ответы. Использование: probe.exe [addr] [user]
// Вердикт: authd ответил type=3 → healthy; тишина → authd/state-проблема.
package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"time"

	"aion-gate/internal/proto"
)

func readFull(c net.Conn, p []byte) error {
	_, err := io.ReadFull(c, p)
	return err
}

func main() {
	addr := "127.0.0.1:2110"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}
	user := "probeacc"
	if len(os.Args) > 2 {
		user = os.Args[2]
	}
	ip := "192.168.0.253" // LAN-IP клиента (как у ориг-сессий)
	if len(os.Args) > 3 {
		ip = os.Args[3]
	}
	sid := uint32(time.Now().UnixNano() & 0xffffff) // случайный sid: stale-sid эксперимент
	if len(os.Args) > 4 {
		if v, err := strconv.ParseUint(os.Args[4], 0, 32); err == nil {
			sid = uint32(v)
		}
	}
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		fmt.Println("DIAL FAIL:", err)
		return
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var h [5]byte
	if err := readFull(conn, h[:]); err == nil && h[0] == 3 {
		fmt.Printf("A>G [03] sid=%d (authd жив, приветствие есть)\n", binary.LittleEndian.Uint32(h[1:5]))
	}
	// blob = ровно как в handleLogin: [00][decbuf34][dw][tail]
	decbuf := proto.BuildLoginDecbuf(user, "", 0, 34)
	dw := uint32(0x12345678)
	tail := make([]byte, 152)
	copy(tail[40:48], []byte{0x68, 0xff, 0xda, 0xb3, 0xe2, 0xfd, 0xa8, 0x92})
	copy(tail[48:56], []byte{0x2d, 0x9c, 0xc7, 0xba, 0xa8, 0x7e, 0x0d, 0x49})
	// [00] CltConnect — БЕЗ него authd отвечает [01][sid] (unknown session)
	ipb := net.ParseIP(ip).To4()
	if ipb == nil {
		fmt.Println("BAD IP:", ip)
		return
	}
	var iparr [4]byte
	copy(iparr[:], ipb)
	cframe := proto.Assemble("cdd", byte(0), sid, binary.BigEndian.Uint32(iparr[:]))
	if _, err := conn.Write(cframe); err != nil {
		fmt.Println("WRITE FAIL:", err)
		return
	}
	fmt.Printf("G>A [00] sid=%d ip=%s\n", sid, ip)
	blob := proto.Assemble("cbdb", byte(0), decbuf, dw, tail)
	frame := proto.Assemble("cdh", byte(2), sid, uint16(len(blob)+2))
	if _, err := conn.Write(append(frame, blob...)); err != nil {
		fmt.Println("WRITE FAIL:", err)
		return
	}
	fmt.Printf("G>A [02] sid=777 login-blob user=%q blobLen=%d\n", user, len(blob))
	_ = conn.SetReadDeadline(time.Now().Add(6 * time.Second))
	for {
		var ft [1]byte
		if err := readFull(conn, ft[:]); err != nil {
			fmt.Println("CLOSED:", err)
			fmt.Println("ВЕРДИКТ: authd НЕ ответил на login (state/клин)")
			return
		}
		switch ft[0] {
		case 2:
			var hdr [6]byte
			if err := readFull(conn, hdr[:]); err != nil {
				return
			}
			id := binary.LittleEndian.Uint32(hdr[0:4])
			body := int(binary.LittleEndian.Uint16(hdr[4:6])) - 2
			buf := make([]byte, body)
			if err := readFull(conn, buf); err != nil {
				return
			}
			fmt.Printf("A>G [02] id=%d type=%d payload(%d)=%s\n", id, buf[0], len(buf)-1, hex.EncodeToString(buf[1:]))
			if buf[0] == 3 {
				fmt.Println("ВЕРДИКТ: authd ОТВЕТИЛ type=3 (healthy, аккаунт принят)")
				disc(conn, sid)
				return
			}
			if buf[0] == 1 {
				fmt.Println("ВЕРДИКТ: authd ответил LOGIN_FAIL type=1 code=", buf[1])
				disc(conn, sid)
				return
			}
		case 3:
			var r [4]byte
			_ = readFull(conn, r[:])
			fmt.Printf("A>G [03] sid=%d\n", binary.LittleEndian.Uint32(r[:]))
		case 1:
			var r [4]byte
			_ = readFull(conn, r[:])
			fmt.Printf("A>G [01] id=%d\n", binary.LittleEndian.Uint32(r[:]))
		default:
			fmt.Printf("A>G [%02x]\n", ft[0])
		}
	}
}

// disc — аккуратный [01]-дисконнект (иначе сессия утекает в authd и портит state).
func disc(c net.Conn, sid uint32) {
	_ = c.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, _ = c.Write(proto.Assemble("cd", byte(1), sid))
}
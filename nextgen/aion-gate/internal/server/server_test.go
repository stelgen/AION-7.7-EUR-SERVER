package server

import (
	"encoding/binary"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"aion-gate/internal/authdclient"
	"aion-gate/internal/config"
	"aion-gate/internal/proto"
)

// fake-authd reader: парсит gate→authd фреймы из as, отвечает [03] на [00].
type fakeAuthd struct {
	mu       sync.Mutex
	conn     net.Conn
	connects [][4]byte
	packets  [][]byte // blob'и [02]-фреймов
	pktIDs   []uint32
	disc     int
}

func runFakeAuthd(t *testing.T, as net.Conn) *fakeAuthd {
	t.Helper()
	f := &fakeAuthd{conn: as}
	go func() {
		hdr := make([]byte, 1)
		for {
			if _, err := asRead(as, hdr); err != nil {
				return
			}
			switch hdr[0] {
			case 0x00: // [00][4B sid][4B IP]
				rest := make([]byte, 8)
				if _, err := asRead(as, rest); err != nil {
					return
				}
				var ip [4]byte
				copy(ip[:], rest[4:8])
				f.mu.Lock()
				f.connects = append(f.connects, ip)
				f.mu.Unlock()
				sid := binary.LittleEndian.Uint32(rest[:4])
				_, _ = as.Write([]byte{0x03, byte(sid), byte(sid >> 8), byte(sid >> 16), byte(sid >> 24)})
			case 0x01: // [01][4B sid]
				rest := make([]byte, 4)
				if _, err := asRead(as, rest); err != nil {
					return
				}
				f.mu.Lock()
				f.disc++
				f.mu.Unlock()
			case 0x02: // [02][4B id][2B len][blob]
				rest := make([]byte, 6)
				if _, err := asRead(as, rest); err != nil {
					return
				}
				id := binary.LittleEndian.Uint32(rest[:4])
				body := int(binary.LittleEndian.Uint16(rest[4:6])) - 2
				blob := make([]byte, body)
				if _, err := asRead(as, blob); err != nil {
					return
				}
				f.mu.Lock()
				f.pktIDs = append(f.pktIDs, id)
				f.packets = append(f.packets, blob)
				f.mu.Unlock()
			default:
				return
			}
		}
	}()
	return f
}

func asRead(c net.Conn, p []byte) (int, error) {
	n := 0
	for n < len(p) {
		k, err := c.Read(p[n:])
		if err != nil {
			return n, err
		}
		n += k
	}
	return n, nil
}

func waitFor(t *testing.T, fn func() bool) {
	t.Helper()
	for i := 0; i < 500; i++ {
		if fn() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("таймаут")
}

func TestE2ESkeleton(t *testing.T) {
	srv, err := New(config.Gate{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ac, as := net.Pipe()
	f := runFakeAuthd(t, as)
	srv.SetAuthd(authdclient.DialConn(ac, srv.AuthdHandler()))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go srv.Serve(ln)

	// 1. клиент подключился → welcome 194
	cl, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer cl.Close()
	wl := make([]byte, 2)
	if _, err := asRead(cl, wl); err != nil {
		t.Fatal(err)
	}
	if wl[0] != 0xC2 || wl[1] != 0x00 {
		t.Fatalf("welcome len: %x %x", wl[0], wl[1])
	}
	ecb := make([]byte, 192)
	if _, err := asRead(cl, ecb); err != nil {
		t.Fatal(err)
	}
	// финальная раскладка: dword0 = fc = sid (rand32) ЦЕЛИКОМ (не скрамблится)
	dec := make([]byte, 8)
	srv.key1.Decrypt(dec, ecb[:8])
	sidFC := binary.LittleEndian.Uint32(dec[1:5]) // [0]=opcode 0x00

	// authd получил CltConnect
	waitFor(t, func() bool { f.mu.Lock(); defer f.mu.Unlock(); return len(f.connects) == 1 })

	sess := srv.SessionByIndex(0)
	if sess == nil {
		t.Fatal("нет сессии")
	}
	if sidFC != sess.ID {
		t.Fatalf("welcome dword0 (fc)=%x != sess.ID %x", sidFC, sess.ID)
	}

	// 2. 34b AUTH_GG → 42b SM_AUTH_GG: клиент шлёт EncryptSecondary([sid][20B]),
	// ответ = EncryptSecondary([sid][28×0]) (capture-структура [P][Q][Q][Q][P], ночь-4)
	gg := make([]byte, 24)
	binary.LittleEndian.PutUint32(gg, sess.ID) // [sid][20B]
	if _, err := cl.Write(proto.WriteFrame(proto.EncryptSecondary(sess.BF2, gg))); err != nil {
		t.Fatal(err)
	}
	rl := make([]byte, 2)
	if _, err := asRead(cl, rl); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint16(rl) != 42 {
		t.Fatalf("reply len: %d want 42", binary.LittleEndian.Uint16(rl))
	}
	body := make([]byte, 40)
	if _, err := asRead(cl, body); err != nil {
		t.Fatal(err)
	}
	decX := make([]byte, 40)
	for off := 0; off < 40; off += 8 {
		sess.BF2.Decrypt(decX[off:off+8], body[off:off+8])
	}
	if decX[0] != 0x0b || binary.LittleEndian.Uint32(decX[1:5]) != sess.ID || binary.LittleEndian.Uint32(decX[32:36]) != binary.LittleEndian.Uint32(decX[0:4]) {
		t.Fatalf("SM_AUTH_GG: dw0=%x dw8=%x", decX[0:4], decX[32:36])
	}
	for i := 5; i < 32; i++ {
		if decX[i] != 0 {
			t.Fatalf("SM_AUTH_GG zeros@%d: %02x", i, decX[i])
		}
	}

	// 3. логин 186b → relay "cbdb" в authd (decbuf = RSA-dec 128Б-блока с ведущими нулями)
	loginDec := make([]byte, 32)
	copy(loginDec, "LOGIN-DEC-BUF-32-BYTES")
	loginDec[31] = 0x21
	pub2 := &sess.RSA.Priv.PublicKey
	loginCT := new(big.Int).Exp(new(big.Int).SetBytes(loginDec), big.NewInt(int64(pub2.E)), pub2.N).FillBytes(make([]byte, 128))
	data := make([]byte, 184)
	copy(data[:128], loginCT) // полный RSA-1024 блок
	binary.LittleEndian.PutUint32(data[128:132], 0x7d5214) // sessionId
	binary.LittleEndian.PutUint32(data[148:152], 0x11223344)
	if _, err := cl.Write(proto.WriteFrame(data)); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { f.mu.Lock(); defer f.mu.Unlock(); return len(f.packets) == 1 })
	f.mu.Lock()
	blob := f.packets[0]
	pktID := f.pktIDs[0]
	f.mu.Unlock()
	wantBlob := proto.Assemble("cbdb", byte(0), loginDec, uint32(0x11223344), data[152:])
	if string(blob) != string(wantBlob) {
		t.Fatalf("relay blob: got %x want %x", blob, wantBlob)
	}
	if pktID != sess.ID {
		t.Fatalf("relay id: %d want %d", pktID, sess.ID)
	}

	// 4. push serverlist (type 4) от authd → клиент получает 26b-фрейм (как в capture!)
	sl := []byte("serverlist-data") // 15Б → n=24 → frame 26
	push := append([]byte{0x02, byte(sess.ID), byte(sess.ID >> 8), byte(sess.ID >> 16), byte(sess.ID >> 24), byte(len(sl)+3), 0x00, 0x04}, sl...)
	if _, err := as.Write(push); err != nil {
		t.Fatal(err)
	}
	pl := make([]byte, 2)
	if _, err := asRead(cl, pl); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint16(pl) != 26 {
		t.Fatalf("serverlist frame: %d want 26", binary.LittleEndian.Uint16(pl))
	}
	pbody := make([]byte, 24)
	if _, err := asRead(cl, pbody); err != nil {
		t.Fatal(err)
	}
	decSL := make([]byte, 24)
	for off := 0; off < 24; off += 8 {
		sess.BF2.Decrypt(decSL[off:off+8], pbody[off:off+8])
	}
	if string(decSL[:len(sl)]) != string(sl) {
		t.Fatalf("serverlist: %q", decSL[:len(sl)])
	}
}

// BlockIPs: IP из файла → мгновенный cc 22 и close.
func TestBlockedIP(t *testing.T) {
	dir := t.TempDir()
	listPath := filepath.Join(dir, "BlockIPs.txt")
	if err := os.WriteFile(listPath, []byte("127.0.0.1\n# comment\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv, err := New(config.Gate{BlockIPsFile: listPath}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !srv.ips.Blocked([4]byte{127, 0, 0, 1}) {
		t.Fatal("127.0.0.1 не в блок-листе")
	}
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	go srv.Serve(ln)
	cl, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer cl.Close()
	buf := make([]byte, 4)
	cl.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := asRead(cl, buf); err != nil {
		t.Fatal(err)
	}
	if buf[2] != 0x01 || buf[3] != 22 { // [len=4][01][16]
		t.Fatalf("cc: %x", buf)
	}
}

func TestBrute(t *testing.T) {
	b := NewBrute(3, 60, 120)
	if b.Fail("user") || b.Fail("user") {
		t.Fatal("заблокировал раньше времени")
	}
	if !b.Fail("user") {
		t.Fatal("не заблокировал на 3-й неудаче")
	}
	if !b.Blocked("user") {
		t.Fatal("Blocked=false после блока")
	}
	b.Reset("user")
	if b.Blocked("user") {
		t.Fatal("Reset не сбросил")
	}
}

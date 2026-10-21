// R4-реплей golden-корпуса (автономный, без юзерского трафика):
// 1) парсим C2S/S2C-кадры из nextgen/accountcache-ref/capture-20261007/frames.txt
// 2) поднимаем живой сервер (fakeDB/суррогат exec — как в server_test.go)
// 3) шлём ВСЕ C2S-кадры capture по порядку в ОДНО соединение
// 4) собираем все S2C-ответы байт-в-байт и сравниваем с последовательностью S2C
//    из capture (выравнивание по cmd: в capture есть push-кадры ACP_*/UPDATE_BM_PACK,
//    которые наш сервер ещё не эмитит — index-wise свёл бы все сравнения «крест-накрест»).
//    Точное совпадение = PASS; расхождение = XFAIL-отчёт (stdout: позиция, наш байт vs
//    capture-байт, cmd-имя). Тест НЕ фейлится на расхождении — покрытие фиксируется.
// 5) в конце assert: все C2S приняты без parse.err, сервер не упал.
package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"aion-accache/internal/db"
	"aion-accache/internal/dispatch"
)

// goldenFrame — одна строка capture-файла.
type goldenFrame struct {
	Dir  string // "C2S" | "S2C"
	Name string
	N    int
	Wire []byte
}

// parseGoldenFrames — читает golden-кадры из capture-20261007/frames.txt.
// Формат строки: "<DIR> <NAME> N= <n> wire=<hex>" (пробел после "N=").
func parseGoldenFrames(t *testing.T, path string) []goldenFrame {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open golden frames %s: %v", path, err)
	}
	defer f.Close()
	var out []goldenFrame
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 || (fields[0] != "C2S" && fields[0] != "S2C") {
			continue
		}
		g := goldenFrame{Dir: fields[0], Name: fields[1]}
		for i := 2; i < len(fields); i++ {
			fld := fields[i]
			// формат: "N= 15" (пробел после =) и "wire=<hex>" (слитно)
			if fld == "N=" && i+1 < len(fields) {
				if _, err := fmt.Sscanf(fields[i+1], "%d", &g.N); err != nil {
					t.Fatalf("bad N %q in %s", fields[i+1], line)
				}
				i++
				continue
			}
			kv := strings.SplitN(fld, "=", 2)
			if len(kv) != 2 {
				continue
			}
			if kv[0] == "wire" {
				b, err := hex.DecodeString(kv[1])
				if err != nil {
					t.Fatalf("bad wire hex %q in %s: %v", kv[1], line, err)
				}
				g.Wire = b
			}
		}
		if len(g.Wire) == 0 {
			t.Fatalf("no wire for %s %s", g.Dir, g.Name)
		}
		if int(binary.LittleEndian.Uint16(g.Wire[0:2])) != len(g.Wire) {
			t.Fatalf("%s %s: len-поле=%d, факт=%d", g.Dir, g.Name,
				binary.LittleEndian.Uint16(g.Wire[0:2]), len(g.Wire))
		}
		out = append(out, g)
	}
	return out
}

// replayRows — fakeRows с ЧЕСТНЫМ Scan (через указатели; fakeRows из server_test.go
// копирует any-значения в dest-слайс, не разыменовывая указатели — колонки остаются нулями).
type replayRows struct {
	data [][]any
	i    int
}

func (r *replayRows) Next() bool { return r.i < len(r.data) }
func (r *replayRows) Close() error { return nil }
func (r *replayRows) Err() error { return nil }

func (r *replayRows) Scan(dest ...any) error {
	row := r.data[r.i]
	for j, d := range dest {
		if j >= len(row) {
			break
		}
		switch dv := d.(type) {
		case *int32:
			switch x := row[j].(type) {
			case int32:
				*dv = x
			case int64:
				*dv = int32(x)
			case int:
				*dv = int32(x)
			}
		case *int64:
			switch x := row[j].(type) {
			case int64:
				*dv = x
			case int32:
				*dv = int64(x)
			case int:
				*dv = int64(x)
			}
		}
	}
	r.i++
	return nil
}

// replayExec — суррогат db.Executor: честный Scan + фиксация запросов +
// STATEFUL: account_data per-acc (updatetime мутится Exec'ом
// aion_UpdateHiddenFatigueInfo — как в ориг: после UPDATEHiddenFatigue
// первый FIRST_LOAD возвращает НОВЫЙ updatetime; capture-факт acc=1010:
// 0x6AC61FCA до апдейтов, 0x6AC620A3 после).
type replayExec struct {
	fakeExec
	mu    sync.Mutex
	accs  map[int32]*replayAcc
	trial map[int32]*replayTrial
}

type replayAcc struct {
	point, upd, npckill, limitReset, limitAccum int32
}

// replayTrial — прод-факты 21.10 (acc=1010): reset=0, sell=917, trade=70,
// decompose=0, gather=0, extract=0 (SQL AionAccountCacheD_rc.trial_account_data).
type replayTrial struct {
	reset, sell, trade, decompose, gather, extract int32
}

func newReplayExec() *replayExec {
	return &replayExec{
		accs: map[int32]*replayAcc{
			1010: {point: 1, upd: 0x6AC61FCA, npckill: 0, limitReset: 1791352800, limitAccum: 0}, // 1791352800 = 0x6AC5DFE0 (прод: limit_play_reset_time acc=1010)
			// acc=7 отсутствует в account_data — FIRST_LOAD#3 = пустая строка (golden ts=0)
		},
		trial: map[int32]*replayTrial{
			1010: {reset: 0, sell: 917, trade: 70, decompose: 0, gather: 0, extract: 0},
		},
	}
}

func (f *replayExec) Query(ctx context.Context, q string, args ...any) (db.Rows, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.queryQ = q
	f.queryArgs = args
	if strings.Contains(q, "aion_GetAccountData") && len(args) == 1 {
		acc := args[0].(int32)
		if a, ok := f.accs[acc]; ok {
			return &replayRows{data: [][]any{{a.point, a.upd, a.npckill, a.limitReset, a.limitAccum}}}, nil
		}
		return &replayRows{data: nil}, nil // пустой result-set как ориг
	}
	if strings.Contains(q, "aion_GetTrialAccountData") && len(args) == 1 {
		acc := args[0].(int32)
		if tr, ok := f.trial[acc]; ok {
			return &replayRows{data: [][]any{{tr.reset, tr.sell, tr.trade, tr.decompose, tr.gather, tr.extract}}}, nil
		}
		return &replayRows{data: nil}, nil
	}
	return &replayRows{data: f.rows}, nil
}

func (f *replayExec) Exec(ctx context.Context, q string, args ...any) (db.Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.execs = append(f.execs, q)
	f.execArgs = append(f.execArgs, args)
	// aion_UpdateHiddenFatigueInfo(acc, point, npckill, updatetime, limit_reset, limit_play):
	// прод-факт — updatetime-колонка = 4-й аргумент (wire-offset16). ТОЛЬКО updatetime:
	// golden FIRST_LOAD#2 point=1 несмотря на UPDATE point=0 (ACS не зеркалит point
	// с wire в account_data, либо роли point/npckill иные — журнал теорий).
	if strings.Contains(q, "aion_UpdateHiddenFatigueInfo") && len(args) >= 4 {
		acc := args[0].(int32)
		if a, ok := f.accs[acc]; ok {
			a.upd = args[3].(int32)
		}
	}
	return fakeResult{}, nil
}

// splitFrames — режет накопленный S2C-байтовый поток на кадры по [u16 len]
// (len = полная длина кадра, включая сам len).
func splitFrames(buf []byte) ([][]byte, error) {
	var out [][]byte
	i := 0
	for i+2 <= len(buf) {
		n := int(binary.LittleEndian.Uint16(buf[i : i+2]))
		if n < 7 || i+n > len(buf) {
			return out, fmt.Errorf("bad frame len=%d at offset %d (buf=%d)", n, i, len(buf))
		}
		out = append(out, buf[i:i+n])
		i += n
	}
	if i != len(buf) {
		return out, fmt.Errorf("trailing %d bytes after last frame", len(buf)-i)
	}
	return out, nil
}

func frameCmd(f []byte) uint16 { return binary.LittleEndian.Uint16(f[2:4]) }

// TestR4ReplayCapture — R4-реплей: golden C2S -> живой сервер -> байт-дифф с S2C capture.
// Расхождение = XFAIL-отчёт (stdout); тест НЕ падает на расхождении пар.
func TestR4ReplayCapture(t *testing.T) {
	// Ловим stdout/stderr сервера, чтобы поймать parse.err / unhandled.
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(os.Stderr)

	goldenPath := filepath.Join("..", "..", "..", "accountcache-ref", "capture-20261007", "frames.txt")
	frames := parseGoldenFrames(t, goldenPath)

	var c2s, s2c []goldenFrame
	for _, g := range frames {
		if g.Dir == "C2S" {
			c2s = append(c2s, g)
		} else {
			s2c = append(s2c, g)
		}
	}
	if len(c2s) == 0 || len(s2c) == 0 {
		t.Fatalf("golden corpus пуст: C2S=%d S2C=%d", len(c2s), len(s2c))
	}

	// 2) Живой сервер с fakeDB. account_data acc=1010 (прод-факт 21.10):
	//    point=1, updatetime=0x6AC61FCA (до апдейтов), limit_play_reset_time=
	//    1791352800=0x6AC5DFE0 (= ts в LOAD_LUNA!), acc=7 отсутствует.
	ex := newReplayExec()
	addr, srv := startSrv(t, ex)
	srv.cfg.Server.Verbose = true // чтобы parse.err/unhandled попадали в logBuf
	c := dial(t, addr)
	defer c.Close()

	// 3) Шлём ВСЕ C2S-кадры по порядку в ОДНО соединение.
	for _, g := range c2s {
		if _, err := c.Write(g.Wire); err != nil {
			t.Fatalf("write C2S %s: %v", g.Name, err)
		}
		time.Sleep(10 * time.Millisecond) // дать серверу обработать кадр в порядке
	}

	// 4) Читаем ВСЕ S2C-ответы байт-в-байт до тишины (deadline после последнего байта).
	c.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var gotRaw []byte
	buf := make([]byte, 4096)
	for {
		n, err := c.Read(buf)
		if n > 0 {
			gotRaw = append(gotRaw, buf[:n]...)
			c.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
		}
		if err != nil {
			if ne, ok := err.(net.Error); !ok || !ne.Timeout() {
				if err != io.EOF {
					t.Logf("read err: %v", err)
				}
			}
			break
		}
	}

	ourFrames, splitErr := splitFrames(gotRaw)
	if splitErr != nil {
		t.Logf("frame split: %v", splitErr)
	}

	// 4b) Выравнивание по cmd: для каждого golden S2C — ближайший следующий
	//     неиспользованный наш кадр с тем же cmd (порядок сохраняется).
	used := make([]bool, len(ourFrames))
	pair := make([]int, len(s2c)) // golden idx -> our idx (-1 = нет пары)
	for gi := range pair {
		pair[gi] = -1
	}
	for gi, want := range s2c {
		wantCmd := frameCmd(want.Wire)
		for oi := range ourFrames {
			if !used[oi] && frameCmd(ourFrames[oi]) == wantCmd {
				pair[gi] = oi
				used[oi] = true
				break
			}
		}
	}

	// 4c) Байт-дифф + XFAIL-отчёт в stdout.
	fmt.Printf("=== R4 replay: our=%d frames (%d bytes), golden S2C=%d ===\n",
		len(ourFrames), len(gotRaw), len(s2c))
	matched := 0
	total := len(s2c)
	for gi, want := range s2c {
		oi := pair[gi]
		if oi < 0 {
			fmt.Printf("XFAIL [%2d] %-24s : наш ответ ОТСУТСТВУЕТ (нет пары; our=%d кадров vs golden=%d)\n",
				gi, want.Name, len(ourFrames), total)
			continue
		}
		got := ourFrames[oi]
		if bytes.Equal(got, want.Wire) {
			matched++
			continue
		}
		max := len(got)
		if len(want.Wire) < max {
			max = len(want.Wire)
		}
		diffAt := -1
		for b := 0; b < max; b++ {
			if got[b] != want.Wire[b] {
				diffAt = b
				break
			}
		}
		if diffAt >= 0 {
			fmt.Printf("XFAIL [%2d] %-24s : diff pos=%d наш=%02x capture=%02x (len наш=%d capture=%d)\n",
				gi, want.Name, diffAt, got[diffAt], want.Wire[diffAt], len(got), len(want.Wire))
		} else {
			fmt.Printf("XFAIL [%2d] %-24s : байты совпали до min(len), но len наш=%d capture=%d\n",
				gi, want.Name, len(got), len(want.Wire))
		}
		fmt.Printf("       our:     %s\n", hex.EncodeToString(got))
		fmt.Printf("       capture: %s\n", hex.EncodeToString(want.Wire))
	}
	for oi := range ourFrames {
		if !used[oi] {
			cmd := frameCmd(ourFrames[oi])
			fmt.Printf("XFAIL [%2d] EXTRA                  : наш cmd=%#x (%s) — нет пары в golden S2C\n",
				oi, cmd, dispatch.Name(false, cmd))
		}
	}
	fmt.Printf("=== R4 replay result: matched %d / %d golden S2C (%.0f%%), our frames=%d ===\n",
		matched, total, float64(matched)*100/float64(total), len(ourFrames))

	// 5) Assert: все C2S приняты без parse.err, сервер не упал (listener жив).
	c2, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("сервер упал: re-dial: %v", err)
	}
	c2.Close()
	if strings.Contains(logBuf.String(), "parse.err") {
		t.Errorf("parse.err в логе сервера:\n%s", logBuf.String())
	}
	if strings.Contains(logBuf.String(), "unhandled") {
		t.Logf("unhandled (ожидаемо — пары не все реализованы):\n%s", logBuf.String())
	}
}

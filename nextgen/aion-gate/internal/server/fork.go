package server

import (
	"encoding/binary"
	"encoding/hex"
	"log"
	"math/big"
	"net"
	"strconv"
	"time"

	"aion-gate/internal/proto"
	"aion-gate/internal/ship"
)

// Режим mode: fork — форк-прокси (требование 06.10): клиент → НАШ гейт → ОРИГИНАЛЬНЫЙ
// AuthGateD (forkOrigAddr:forkOrigPort). Всё релеится байт-в-байт (клиент работает
// против оригинала — залогинится и скажет креды), параллельно SHADOW-анализ:
//  1. welcome оригинала расшифровывается static-ключом → sid/V/модуль/key2;
//  2. модуль ориг анскрамблится клиентским алгоритмом → N_orig (hex в лог) и
//     верифицируется серверный скрамбл гита: ServerScramble(N_orig) == wire?;
//  3. каждый фрейм обеих сторон расшифровывается key2 оригинала и логируется;
//  4. на AUTH_GG/26b строится НАШ shadow-ответ и сравнивается с ответом оригинала;
//  5. на LOGIN дампится ct (RSA-блоки) + N_orig → оффлайн-калибровка e (m^e mod N == ct).
// authd НЕ дialится (фантомных сессий нет — authd живёт у оригинала).

func (s *Server) forkAddr() string {
	return net.JoinHostPort(s.Cfg.ForkOrigAddr, strconv.Itoa(s.Cfg.ForkOrigPort))
}

func truncHex(b []byte, n int) string {
	if len(b) > n {
		return hex.EncodeToString(b[:n]) + "…"
	}
	return hex.EncodeToString(b)
}

// handleConnFork — прозрачный релей + shadow-сравнение.
func (s *Server) handleConnFork(conn net.Conn) {
	defer conn.Close()
	var ip [4]byte
	if ta, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		copy(ip[:], ta.IP.To4())
	}
	remote := net.IP(ip[:]).String()
	if s.ips.Blocked(ip) {
		sendCC(conn, 22)
		s.send(ship.Event{Ev: "cc", Svc: "fork", Remote: remote, Data: map[string]any{"code": 22}})
		return
	}
	up, err := net.DialTimeout("tcp", s.forkAddr(), 5*time.Second)
	if err != nil {
		log.Printf("fork: ориг %s недоступен: %v", s.forkAddr(), err)
		s.send(ship.Event{Ev: ship.EvParseErr, Svc: "fork", Remote: remote, Err: err.Error()})
		sendCC(conn, 45)
		return
	}
	defer up.Close()
	s.send(ship.Event{Ev: ship.EvConnUp, Svc: "fork", Remote: remote, Data: map[string]any{"up": s.forkAddr()}})
	defer s.send(ship.Event{Ev: ship.EvConnDown, Svc: "fork", Remote: remote})

	// welcome оригинала
	_ = up.SetReadDeadline(time.Now().Add(10 * time.Second))
	wraw, err := proto.ReadFrame(up)
	if err != nil {
		log.Printf("fork: нет welcome от ориг: %v", err)
		return
	}
	_ = up.SetReadDeadline(time.Now().Add(30 * time.Minute))
	_ = conn.SetReadDeadline(time.Now().Add(30 * time.Minute))
	var bf2 *proto.Blowfish
	var sid uint32
	if len(wraw) == proto.WelcomeLen-2 { // ReadFrame возвращает ECB-область (192Б без len-филда)
		sid, bf2 = s.forkParseWelcome(remote, wraw)
	} else {
		log.Printf("fork: welcome ориг unexpected len=%d (want %d) — passthrough без разбора", len(wraw), proto.WelcomeLen-2)
	}
	// клиенту — ПОЛНЫЙ фрейм (ReadFrame вернул ECB без len-филда — восстанавливаем,
	// байты идентичны оригинальным: total = len+2)
	if _, err := conn.Write(proto.WriteFrame(wraw)); err != nil {
		return
	}

	done := make(chan struct{})
	go func() { defer close(done); s.forkPipe("C>O", conn, up, bf2, sid, true) }()
	s.forkPipe("O>C", up, conn, bf2, sid, false)
	<-done
}

// forkParseWelcome — расшифровка welcome оригинала (static key1 + инверс EncryptPrimary),
// извлечение sid/V/модуль/key2, верификация скрамбл-модели гита на живом модуле.
// wraw — ECB-область welcome (192Б, payload от ReadFrame без len-филда).
func (s *Server) forkParseWelcome(remote string, wraw []byte) (uint32, *proto.Blowfish) {
	pt, err := proto.DecryptPrimary(s.key1, wraw)
	if err != nil || len(pt) < 173 {
		log.Printf("fork: welcome ориг не расшифровался key1/EncryptPrimary (len=%d, pt=%d): %v",
			len(wraw), len(pt), err)
		return 0, nil
	}
	sid := binary.LittleEndian.Uint32(pt[1:5])
	v := binary.LittleEndian.Uint32(pt[5:9])
	var modScr [128]byte
	copy(modScr[:], pt[9:137])
	var key2 [16]byte
	copy(key2[:], pt[153:169])

	// анскрамбл клиентским алгоритмом → N_orig
	nOrig := modScr
	proto.ScrambleModulus(&nOrig)
	nBig := new(big.Int).SetBytes(nOrig[:])

	log.Printf("FORK orig-welcome: sid=%d V=%d(0x%08x) key2=%s statics=%s",
		sid, v, v, hex.EncodeToString(key2[:]), hex.EncodeToString(pt[168:173]))
	log.Printf("FORK N_orig bits=%d top=0x%02x hex=%s", nBig.BitLen(), nOrig[0], hex.EncodeToString(nOrig[:]))
	log.Printf("FORK модуль ориг (scrambled wire) = %s", hex.EncodeToString(modScr[:]))

	// ГЛАВНАЯ ПРОВЕРКА: серверный скрамбл гита от N_orig должен дать ровно wire-модуль
	rescr := nOrig
	proto.ScrambleModulusServer(&rescr)
	if rescr == modScr {
		log.Printf("FORK ✅ скрамбл-модель гита ПОДТВЕРЖДЕНА на живом ориге: ServerScramble(N_orig) == wire")
	} else {
		log.Printf("FORK ⚠ скрамбл MISMATCH: ServerScramble(N_orig)=%s", hex.EncodeToString(rescr[:]))
	}

	// shadow: наш welcome (вариант 0 = серверный скрамбл) с теми же sid/V/key2
	wargs := &proto.WelcomeArgs{
		SessionID:    sid,
		AuthdSession: v,
		Modulus:      s.pool.Get().Modulus128(),
		Key2:         key2,
	}
	w := proto.BuildWelcomeVariant(wargs, s.key1, 0)
	{
		if opt, derr := proto.DecryptPrimary(s.key1, w[2:]); derr == nil && len(opt) >= 173 {
			log.Printf("FORK welcome-diff(наш вар.0 vs ориг): sid=EQ V=EQ key2=EQ | mod ours[0:16]=%s orig[0:16]=%s (N разные — ожидаемо)",
				hex.EncodeToString(opt[9:25]), hex.EncodeToString(pt[9:25]))
		}
	}

	bf2, berr := proto.NewBlowfish(key2[:])
	if berr != nil {
		log.Printf("fork: key2 ориг не гится: %v", berr)
		return sid, nil
	}
	s.send(ship.Event{Ev: "fork.welcome", Svc: "fork", Remote: remote, Data: map[string]any{
		"sid": sid, "v": v, "nbits": nBig.BitLen(),
	}})
	return sid, bf2
}

// forkPipe — релей одного направления + разбор фреймов.
func (s *Server) forkPipe(dir string, src, dst net.Conn, bf2 *proto.Blowfish, sid uint32, fromClient bool) {
	for {
		payload, err := proto.ReadFrame(src)
		if err != nil {
			log.Printf("fork: %s поток закрыт (%v)", dir, err)
			_ = src.Close()
			_ = dst.Close()
			return
		}
		if _, err := dst.Write(proto.WriteFrame(payload)); err != nil {
			log.Printf("fork: %s ошибка записи: %v", dir, err)
			_ = src.Close()
			_ = dst.Close()
			return
		}
		s.forkAnalyze(dir, payload, bf2, sid, fromClient)
	}
}

// forkAnalyze — расшифровка key2 оригинала + shadow-сравнение ответов.
func (s *Server) forkAnalyze(dir string, payload []byte, bf2 *proto.Blowfish, sid uint32, fromClient bool) {
	if bf2 == nil {
		log.Printf("FORK %s len=%d hex=%s (без расшифровки)", dir, len(payload), truncHex(payload, 64))
		return
	}
	pt, err := proto.DecryptSecondary(bf2, payload)
	if err != nil {
		log.Printf("FORK %s len=%d не расшифрован key2: %v hex=%s", dir, len(payload), err, truncHex(payload, 64))
		return
	}
	if len(pt) < 1 {
		return
	}
	op := pt[0]
	if !fromClient {
		// ориг → клиент: лог + сравнение с НАШИМ shadow-ответом (требование юзера:
		// «сравнивать наши ответы с оригом»)
		switch {
		case op == 0x0b: // SM_AUTH_GG
			ours := proto.BuildClassicAuthGG(sid, false)
			log.Printf("FORK O>C authgg orig pt=%s", hex.EncodeToString(pt))
			log.Printf("FORK O>C authgg ours=%s → %s", hex.EncodeToString(ours), diffVerdict(ours, pt))
		case op == 0x04 && len(pt) == 32: // server-info 42b — сравнение с нашей 26b-эмуляцией
			if ours := s.build26ReplyPt(0x05, nil); ours != nil {
				log.Printf("FORK O>C 0x04/42b orig=%s ours=%s → %s", hex.EncodeToString(pt), hex.EncodeToString(ours), diffVerdict(ours, pt))
			}
		case op == 0x07 && len(pt) == 16: // play-ok 26b (К-6: playOk1/2 = Rnd — сверяем ФОРМУ, не байты)
			if ours := s.build26ReplyPt(0x02, nil); ours != nil {
				log.Printf("FORK O>C 0x07/26b orig=%s ours=%s → %s (playOk Rnd — сверка формы [1:9])",
					hex.EncodeToString(pt), hex.EncodeToString(ours), diffVerdictMasked(ours, pt, 1, 9))
			}
		default:
			log.Printf("FORK O>C op=%02x len=%d pt=%s", op, len(pt), truncHex(pt, 96))
		}
		return
	}
	// клиент → ориг: лог запросов + наш shadow + дамп ct для калибровки e
	switch {
	case op == 0x07: // CM_AUTH_GG
		log.Printf("FORK C>O authgg pt=%s", hex.EncodeToString(pt))
		ours := proto.BuildClassicAuthGG(sid, false)
		log.Printf("FORK C>O authgg наш shadow-ответ=%s (сравнение придёт в O>C)", hex.EncodeToString(ours))
	case op == 0x00: // CM_LOGIN — ГЛАВНЫЙ дамп: ПОЛНЫЙ pt (хвост 7.7 вариативный, shape может не сойтись!) + N_orig
		log.Printf("FORK C>O LOGIN FULL pt(%d) = %s", len(pt), hex.EncodeToString(pt))
		opL, chunks, tail, ok := proto.SplitLogin(pt)
		log.Printf("FORK C>O LOGIN len=%d op=0x%02x (эталон 7.7 = 0x0B) shape_ok=%v chunks=%d", len(pt), opL, ok, len(chunks))
		for i, ct := range chunks {
			log.Printf("FORK C>O LOGIN ct[%d/%d] = %s", i+1, len(chunks), hex.EncodeToString(ct))
		}
		if ok {
			log.Printf("FORK C>O LOGIN tail(%d) = %s", len(tail), hex.EncodeToString(tail))
		}
		log.Printf("FORK C>O LOGIN: калибровка e — см. N_orig выше; m^17 / m^65537 mod N_orig == ct (креды от юзера)")
		s.send(ship.Event{Ev: "login", Svc: "fork", Data: map[string]any{"chunks": len(chunks), "len": len(pt)}})
	case op == 0x05 || op == 0x02: // 26b-пинги/запросы
		log.Printf("FORK C>O op=%02x len=%d pt=%s", op, len(pt), truncHex(pt, 48))
		if ours := s.build26ReplyPt(op, nil); ours != nil {
			log.Printf("FORK C>O наш shadow-ответ на %02x=%s (сравнение придёт в O>C)", op, hex.EncodeToString(ours))
		}
	default:
		log.Printf("FORK C>O op=%02x len=%d pt=%s", op, len(pt), truncHex(pt, 96))
	}
}

func diffVerdict(ours, orig []byte) string {
	if string(ours) == string(orig) {
		return "SAME"
	}
	return "DIFF"
}

// diffVerdictMasked — сравнение форм с маской: байты [from:to) исключаются
// (случайные поля playOk1/playOk2 эталонного SessionKey — К-6/P2-6).
func diffVerdictMasked(ours, orig []byte, from, to int) string {
	if len(ours) != len(orig) {
		return "DIFF-LEN"
	}
	for i := 0; i < len(ours); i++ {
		if i >= from && i < to {
			continue
		}
		if ours[i] != orig[i] {
			return "DIFF"
		}
	}
	return "SAME-ФОРМА"
}
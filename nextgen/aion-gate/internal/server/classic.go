package server

import (
	"encoding/binary"
	"log"
	"math/rand"
	"net"

	"aion-gate/internal/proto"
	"aion-gate/internal/ship"
)

// shipEv — телеметрия classic-события.
func shipEv(name string, sess *Session) ship.Event {
	return ship.Event{Ev: name, Remote: net.IP(sess.IP[:]).String(), Data: map[string]any{"sid": sess.ID}}
}

// Classic-режим (mode: classic, П4 байон-48): beyond-aion-совместимый флоу 4.8.
// State-машина: CONNECTED{0x07=AUTH_GG, 0x08=UPDATE_SESSION} →
// AUTHED_GG{0x00=LOGIN} → AUTHED_LOGIN{0x05=SERVER_LIST, 0x02=PLAY}.
// Прочее — лог (classic standalone, authd не задействован).

const (
	stConnected   uint8 = 0
	stAuthedGG    uint8 = 1
	stAuthedLogin uint8 = 2
)

func (s *Server) dispatchClassic(sess *Session, payload []byte) error {
	pt, err := proto.DecryptSecondary(sess.BF2, payload)
	if err != nil {
		log.Printf("classic: фрейм не расшифрован key2 (len=%d): %v — игнор", len(payload), err)
		return nil
	}
	if len(pt) < 1 {
		return nil
	}
	op := pt[0]
	switch sess.State {
	case stConnected:
		switch op {
		case 0x07: // CM_AUTH_GG
			return s.classicAuthGG(sess)
		case 0x08: // CM_UPDATE_SESSION_REQ — TODO релей
			log.Printf("classic: UPDATE_SESSION (op 0x08) — TODO")
			return nil
		default:
			log.Printf("classic: CONNECTED неожиданный op=%02x len=%d", op, len(pt))
			return nil
		}
	case stAuthedGG:
		if op == 0x00 { // CM_LOGIN
			return s.classicLogin(sess, pt)
		}
		log.Printf("classic: AUTHED_GG неожиданный op=%02x", op)
		return nil
	case stAuthedLogin:
		switch op {
		case 0x05: // CM_SERVER_LIST: [accountId][loginOk][C=7][6B][D][D]
			return s.classicServerList(sess, pt)
		case 0x02: // CM_PLAY: [accountId][loginOk][servId C][6B][Q random]
			return s.classicPlay(sess, pt)
		default:
			log.Printf("classic: AUTHED_LOGIN op=%02x len=%d — без ответа", op, len(pt))
			return nil
		}
	}
	return nil
}

// classicAuthGG — SM_AUTH_GG: живая 32Б форма или гит (ggXorTail=true).
func (s *Server) classicAuthGG(sess *Session) error {
	pt := proto.BuildClassicAuthGG(sess.ID, s.Cfg.GgXorTail)
	sess.State = stAuthedGG
	fr := proto.WriteFrame(proto.EncryptSecondary(sess.BF2, pt))
	dumpRaw("G>C classic authgg", fr)
	s.send(shipEv("classic.authgg", sess))
	return sess.write(fr)
}

// classicLogin — CM_LOGIN: RSA-чанки → креды (П3); SM_LOGIN_OK со случайными
// accountId/loginOk/playOk1/playOk2 (гит SessionKey); фейл → LOGIN_FAIL.
func (s *Server) classicLogin(sess *Session, pt []byte) error {
	chunks, tail, shapeOK := proto.SplitLogin(pt)
	if !shapeOK {
		log.Printf("classic login: форма НЕ по гиту (len=%d) → LOGIN_FAIL", len(pt))
		return s.classicLoginFail(sess)
	}
	ms := make([][]byte, 0, len(chunks))
	for i, ct := range chunks {
		m, err := sess.RSA.DecryptBlock(ct)
		if err != nil {
			log.Printf("classic login: RSA FAIL chunk %d/%d: %v", i+1, len(chunks), err)
			return s.classicLoginFail(sess)
		}
		ms = append(ms, m)
	}
	dec, ok := proto.DecodeLoginPlain(ms)
	if !ok {
		log.Printf("classic login decode FAIL (exp=%d, chunks=%d): user=%q pwd=%q otp=%08x",
			s.Cfg.RsaExponent, len(ms), dec.User, dec.Pwd, dec.Otp)
		return s.classicLoginFail(sess)
	}
	log.Printf("classic login OK: user=%q pwd=%q otp=%08x loginex=%v tail=%s",
		dec.User, dec.Pwd, dec.Otp, dec.Ex, truncHex(tail, 32))
	// Гит SessionKey: accountId = случайный, loginOk/playOk1/playOk2 = Rnd.
	// Реальной БД/аккаунтов у classic-гейта нет — авторизация формальная (TODO: GS-прокси).
	sess.AccID = rand.Uint32()
	sess.LoginOk = rand.Uint32()
	sess.PlayOk1 = rand.Uint32()
	sess.PlayOk2 = rand.Uint32()
	sess.State = stAuthedLogin
	ptOK := proto.BuildClassicLoginOK(sess.AccID, sess.LoginOk)
	fr := proto.WriteFrame(proto.EncryptSecondary(sess.BF2, ptOK))
	dumpRaw("G>C classic login-ok", fr)
	s.send(shipEv("classic.login", sess))
	return sess.write(fr)
}

func (s *Server) classicLoginFail(sess *Session) error {
	pt := proto.BuildClassicLoginFail(1) // responseId=1 (SYSTEM_ERROR; реестр AionAuthResponse — TODO)
	fr := proto.WriteFrame(proto.EncryptSecondary(sess.BF2, pt))
	dumpRaw("G>C classic login-fail", fr)
	return sess.write(fr)
}

// checkLogin — гит-валидация пары (accountId, loginOk) из CM_SERVER_LIST/CM_PLAY.
func (s *Session) checkLogin(accountID, loginOk uint32) bool {
	return accountID == s.AccID && loginOk == s.LoginOk
}

func (s *Server) classicServerList(sess *Session, pt []byte) error {
	if len(pt) < 9 {
		return s.classicLoginFail(sess)
	}
	accountID := binary.LittleEndian.Uint32(pt[1:5])
	loginOk := binary.LittleEndian.Uint32(pt[5:9])
	if !sess.checkLogin(accountID, loginOk) {
		log.Printf("classic serverlist: checkLogin FAIL acc=%d ok=%d", accountID, loginOk)
		return s.classicLoginFail(sess)
	}
	ip := net.ParseIP(s.Cfg.WorldIP).To4()
	if ip == nil {
		ip = net.IPv4(192, 168, 0, 125)
	}
	var ipb [4]byte
	copy(ipb[:], ip)
	slp := proto.BuildClassicServerList(ipb, uint16(s.Cfg.WorldPort), 1)
	fr := proto.WriteFrame(proto.EncryptSecondary(sess.BF2, slp))
	dumpRaw("G>C classic serverlist", fr)
	s.send(shipEv("classic.serverlist", sess))
	return sess.write(fr)
}

func (s *Server) classicPlay(sess *Session, pt []byte) error {
	if len(pt) < 9 {
		return s.classicLoginFail(sess)
	}
	accountID := binary.LittleEndian.Uint32(pt[1:5])
	loginOk := binary.LittleEndian.Uint32(pt[5:9])
	servID := pt[9]
	if !sess.checkLogin(accountID, loginOk) {
		log.Printf("classic play: checkLogin FAIL acc=%d ok=%d", accountID, loginOk)
		return s.classicLoginFail(sess)
	}
	pok := proto.BuildClassicPlayOK(sess.PlayOk1, sess.PlayOk2, servID)
	fr := proto.WriteFrame(proto.EncryptSecondary(sess.BF2, pok))
	dumpRaw("G>C classic play-ok", fr)
	// TODO(байон-48 §6f): playOk1/2 → GS (CM_ACCOUNT_RECONNECT_KEY); authd-контракта нет — WARN.
	log.Printf("classic play: servId=%d playOk1=%08x playOk2=%08x (в GS не прокинуты — WARN)", servID, sess.PlayOk1, sess.PlayOk2)
	s.send(shipEv("classic.play", sess))
	return sess.write(fr)
}
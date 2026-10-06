// Package server — T1 «фейлы как эталон» (релиз 08.10, источник = сорс эталона,
// НЕ догадки): reference/Mobius_AionEmu (Aion-Germany 7.7 → Aion-Lightning, клон
// STELGEN/projects/aion_server_2026-10-02/reference/Mobius_AionEmu, JDK-сорс).
//
// Сорс-факты, на которых построен этот файл:
//
//	SM_LOGIN_FAIL.java  = super(0x01) + writeD(response.getMessageId())  → pt 5Б;
//	SM_PLAY_FAIL.java   = super(0x06) + writeD(response.getMessageId())  → pt 5Б;
//	  (pt 5Б → EncryptSecondary roundup8+8 → wire 18 — ровно live-форма 18Б,
//	   пойманная 06.10 11:08 на не-ASCII логине);
//	AionAuthResponse.java = полный реестр messageIds (константы ниже);
//	CM_LOGIN.java: RSA-fail → sendPacket(SM_LOGIN_FAIL(SYSTEM_ERROR)) БЕЗ close;
//	  брутфорс-бан → close(SM_LOGIN_FAIL(BAN_IP), false); прочие фейлы → close(...,false);
//	CM_PLAY.java: GS offline → sendPacket(SM_PLAY_FAIL(SERVER_DOWN)) БЕЗ close;
//	  GS full → SM_PLAY_FAIL(SERVER_FULL); кривая SessionKey → close(SM_LOGIN_FAIL(SYSTEM_ERROR));
//	AccountController.login: релогин акка, сидящего на GS → kickAccountFromGameServer +
//	  вернуть ALREADY_LOGGED_IN(7) — СЛЕДУЮЩАЯ попытка проходит; акк уже на LS →
//	  closeNow старого соединения + ALREADY_LOGGED_IN(7).
//
// Триггеры T1 (наш план, закрытый сорсом):
//
//	(а) authd молчит после релея blob дольше loginTimeoutSec → SM_LOGIN_FAIL(loginFailCode,
//	    а если user в онлайн-кэше → loginFailOnline);
//	(б) relogin акка моложе onlineTtlSec после его login-ok → НЕМЕДЛЕННЫЙ
//	    SM_LOGIN_FAIL(loginFailOnline=7), blob ВСЁ РАВНО релеить (эталон: kick + 7,
//	    следующая попытка проходит; authd сам решает по своему флагу);
//	(в) authd молчит после релея [05]/[02] дольше playTimeoutSec → SM_PLAY_FAIL(playFailCode).
//
// После фейла соединение НЕ рвём (FailCloseSec=0 — план T1: «сообщение + экран логина жив»);
// эталон в дефолт-ветках делает close(packet,false) — включается FailCloseSec>0.
package server

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"

	"aion-gate/internal/proto"
	"aion-gate/internal/ship"
)

// opLoginFail/opPlayFail — опкоды SM_*_FAIL (Mobius serverpackets, super(...)).
const (
	opLoginFail = 0x01 // SM_LOGIN_FAIL
	opPlayFail  = 0x06 // SM_PLAY_FAIL
)

// AionAuthResponse messageId — ПОЛНЫЙ реестр из сорса эталона
// (reference/Mobius_AionEmu/.../network/aion/AionAuthResponse.java).
const (
	RespAuthed             uint32 = 0 // внутренний, клиенту не шлётся
	RespSystemError        uint32 = 1 // «System error»
	RespInvalidPassword    uint32 = 2 // ID/password mismatch
	RespInvalidPassword2   uint32 = 3
	RespFailedAccountInfo  uint32 = 4
	RespFailedSocialNumber uint32 = 5
	RespNoGSRegistered     uint32 = 6
	RespAlreadyLoggedIn    uint32 = 7 // «You are already logged in»
	RespServerDown         uint32 = 8 // «The selected server is down»
	RespInvalidPassword3   uint32 = 9
	RespNoSuchAccount      uint32 = 10
	RespDisconnected       uint32 = 11
	RespAgeLimit           uint32 = 12
	RespAlreadyLoggedIn2   uint32 = 13
	RespAlreadyLoggedIn3   uint32 = 14
	RespServerFull         uint32 = 15
	RespGMOnly             uint32 = 16 // «Server is being normalized»
	RespError17            uint32 = 17
	RespTimeExpired        uint32 = 18
	RespTimeExpired2       uint32 = 19
	RespSystemError2       uint32 = 20
	RespAlreadyUsedIP      uint32 = 21
	RespBanIP              uint32 = 22 // live: «аккаунт заблокирован» (cc22)
)

// authFailText — ЖИВЫЕ тексты клиента 7.7 EU для messageId (прогон юзера 08.10,
// полный цикл 1..22+45; юзер: «эти тексты — для логов/обработчика, чтобы не потерять»).
// Логируются на КАЖДУЮ выдачу фейла: код + имя + текст клиента.
var authFailText = map[uint32]string{
	1:  "Ошибка авторизации. Пожалуйста, попробуйте зайти в игру позже.",
	2:  "Неверный логин или пароль.",
	3:  "Неверный логин или пароль.",
	4:  "Невозможно найти информацию об аккаунте.",
	5:  "Отсутствуют паспортные данные",
	6:  "Ни один игровой сервер не был авторизован на сервере авторизации.",
	7:  "Вы уже залогинись.",
	8:  "Выбранный сервер временно недоступен. Подключение невозможно.",
	9:  "Введенная информация при входе в игру не соответсвует указанной ранее.",
	10: "Отсутсвует информация о входе в игру.",
	11: "Соединение было прервано обращением на главную страницу plaync.",
	12: "Ваш возраст не соответствует возрастному цензу игры.",
	13: "Под вашим аккаунтом зашли с другого компьютера.",
	14: "Вы уже в игре.",
	15: "Сервер переполнен.",
	16: "На сервере ведутся работы. Повторите попытку позже.",
	17: "Смените пароль и повторите попытку входа.",
	18: "Соединение невозможно. Время подписки закончилось, либо произошла временная ошибка соединения. Пожалуйста, обратитесь в службу поддержки.",
	19: "На аккаунте не осталось оплаченного времени.",
	20: "Системная ошибка. Пожалуйста, обратитесь в системную поддержку пользователей.",
	21: "Данный IP уже используется.",
	22: "Ваш аккаунт заблокирован.",
	45: "Вы сможете запустить AION только после авторизации на главной странице сайта.",
}

// respName — имя messageId по реестру AionAuthResponse (сорс Mobius 7.7).
func respName(id uint32) string {
	switch id {
	case RespAuthed:
		return "AUTHED"
	case RespSystemError:
		return "SYSTEM_ERROR"
	case RespInvalidPassword:
		return "INVALID_PASSWORD"
	case RespInvalidPassword2:
		return "INVALID_PASSWORD2"
	case RespFailedAccountInfo:
		return "FAILED_ACCOUNT_INFO"
	case RespFailedSocialNumber:
		return "FAILED_SOCIAL_NUMBER"
	case RespNoGSRegistered:
		return "NO_GS_REGISTERED"
	case RespAlreadyLoggedIn:
		return "ALREADY_LOGGED_IN"
	case RespServerDown:
		return "SERVER_DOWN"
	case RespInvalidPassword3:
		return "INVALID_PASSWORD3"
	case RespNoSuchAccount:
		return "NO_SUCH_ACCOUNT"
	case RespDisconnected:
		return "DISCONNECTED"
	case RespAgeLimit:
		return "AGE_LIMIT"
	case RespAlreadyLoggedIn2:
		return "ALREADY_LOGGED_IN2"
	case RespAlreadyLoggedIn3:
		return "ALREADY_LOGGED_IN3"
	case RespServerFull:
		return "SERVER_FULL"
	case RespGMOnly:
		return "GM_ONLY"
	case RespError17:
		return "ERROR_17"
	case RespTimeExpired:
		return "TIME_EXPIRED"
	case RespTimeExpired2:
		return "TIME_EXPIRED2"
	case RespSystemError2:
		return "SYSTEM_ERROR2"
	case RespAlreadyUsedIP:
		return "ALREADY_USED_IP"
	case RespBanIP:
		return "BAN_IP"
	}
	return "AUTHGATE_45"
}

// authFailTextOf — текст клиента для messageId ("" — не в реестре).
func authFailTextOf(id uint32) string { return authFailText[id] }

// pending-виды (таймер тишины authd).
const (
	pendLogin byte = 1 // ждём type=3 (login-ok)
	pendPlay  byte = 2 // ждём type=4/7 (serverlist/play-ok)
)

type pendingReq struct {
	kind  byte
	user  string // для pendLogin: нормализованный username (онлайн-кэш на fire)
	timer *time.Timer
}

// sendAuthFail — SM_LOGIN_FAIL/SM_PLAY_FAIL: pt [op][D messageId] → EncryptSecondary
// → wire 18 (форма живого LoginFail 06.10 11:08: [len=18][16Б ct]).
func (s *Server) sendAuthFail(sess *Session, op byte, messageId uint32, ev string) {
	pt := make([]byte, 5)
	pt[0] = op
	binary.LittleEndian.PutUint32(pt[1:5], messageId)
	fr := proto.WriteFrame(proto.EncryptSecondary(sess.BF2, pt))
	name := "SM_LOGIN_FAIL"
	if op == opPlayFail {
		name = "SM_PLAY_FAIL"
	}
	dumpRaw(fmt.Sprintf("G>C %s messageId=%d sid=%d", name, messageId, sess.ID), fr)
	// Лог на КАЖДУЮ выдачу ошибки: код + имя реестра + живой текст клиента (прогон 08.10)
	// — требование юзера: «видеть в логах на каждую обработку выдачу такой ошибки».
	log.Printf("%s -> КЛИЕНТ: messageId=%d (%s) текст=%q sid=%d ip=%s (failClose=%ds)",
		name, messageId, respName(messageId), authFailTextOf(messageId), sess.ID, net.IP(sess.IP[:]).String(), s.Cfg.FailCloseSec)
	if err := sess.write(fr); err == nil {
		s.send(ship.Event{Ev: ev, Data: map[string]any{"sid": sess.ID, "messageId": messageId, "code": respName(messageId), "text": authFailTextOf(messageId)}})
	}
	// План T1: соединение НЕ рвём (FailCloseSec=0); эталон в дефолт-ветках делает
	// close(packet,false) — при FailCloseSec>0 закрываем отложенно.
	if cs := s.Cfg.FailCloseSec; cs > 0 {
		time.AfterFunc(time.Duration(cs)*time.Second, sess.close)
	}
}

// armPending — таймер тишины authd (T1-а/в). kind: pendLogin/pendPlay.
// Повторный вызов заменяет прежний pending того же sid.
func (s *Server) armPending(sess *Session, kind byte, user string) {
	// ЕДИНЫЙ бизнес-таймаут тишины authd (юзер 08.10: «общий таймаут — бизнесовая
	// настройка, рвать сессии секунд за 15»; логин и play — один и тот же).
	ttlSec := s.Cfg.AuthdTimeoutSec
	s.cancelPending(sess.ID)
	if ttlSec <= 0 { // <=0 = таймеры выключены (конфиг)
		return
	}
	sid := sess.ID
	s.pendMu.Lock()
	s.pend[sid] = &pendingReq{kind: kind, user: user, timer: time.AfterFunc(time.Duration(ttlSec)*time.Second, func() {
		s.pendMu.Lock()
		p, ok := s.pend[sid]
		if ok && p.kind == kind {
			delete(s.pend, sid)
		} else {
			ok = false
		}
		s.pendMu.Unlock()
		if !ok {
			return
		}
		s.mu.Lock()
		cur := s.sess[sid]
		s.mu.Unlock()
		if cur == nil || cur != sess { // сессия умерла/пересоздана
			return
		}
		s.firePending(sess, kind, p.user)
	})}
	s.pendMu.Unlock()
}

// firePending — таймаут тишины authd сработал.
func (s *Server) firePending(sess *Session, kind byte, user string) {
	switch kind {
	case pendLogin:
		// T1-а: нет type=3 за loginTimeoutSec. Код: user в онлайн-кэше (флаг authd
		// ещё жив, вероятно probe-лок) → 7 ALREADY_LOGGED_IN; иначе loginFailCode.
		code := uint32(s.Cfg.LoginFailCode)
		if s.onlineRecent(user) {
			code = uint32(s.Cfg.LoginFailOnline)
		}
		log.Printf("login TIMEOUT sid=%d user=%q (authd молчал %ds) → SM_LOGIN_FAIL(%d) — сессия рвётся через %ds (failCloseSec)",
			sess.ID, user, s.Cfg.AuthdTimeoutSec, code, s.Cfg.FailCloseSec)
		s.sendAuthFail(sess, opLoginFail, code, "login.timeout")
	case pendPlay:
		log.Printf("play TIMEOUT sid=%d op=[05]/[02] (authd молчал %ds) → SM_PLAY_FAIL(%d) — сессия рвётся через %ds (failCloseSec)",
			sess.ID, s.Cfg.AuthdTimeoutSec, s.Cfg.PlayFailCode, s.Cfg.FailCloseSec)
		s.sendAuthFail(sess, opPlayFail, uint32(s.Cfg.PlayFailCode), "play.timeout")
	}
}

// testFailCode — код тест-фейла «матюгания ошибками»: cfgVal>0 = фиксированный код;
// cfgVal<0 (CYCLE) = seq-инкремент: 1..22 (messageId AionAuthResponse), затем 45
// (authgate-спец), по кругу. seq — адрес счётчика (&s.testFailSeqLogin / Play).
func testFailCode(seq *int64, cfgVal int) uint32 {
	if cfgVal > 0 {
		return uint32(cfgVal)
	}
	n := atomic.AddInt64(seq, 1)
	idx := (n - 1) % 23
	if idx < 22 {
		return uint32(idx + 1)
	}
	return 45
}

// cancelPending — authd ответил (ЛЮБОЙ [02]-пакет по id) или сессия умерла.
func (s *Server) cancelPending(sid uint32) {
	s.pendMu.Lock()
	if p, ok := s.pend[sid]; ok {
		delete(s.pend, sid)
		p.timer.Stop()
	}
	s.pendMu.Unlock()
}

// pendingUser — username активного login-pending (до отмены).
func (s *Server) pendingUser(sid uint32) string {
	s.pendMu.Lock()
	defer s.pendMu.Unlock()
	if p, ok := s.pend[sid]; ok && p.kind == pendLogin {
		return p.user
	}
	return ""
}

// markOnline — authd прислал type=3 (login-ok): акк помечен онлайн у authd
// (доказано probe-циклом 07.10: [01]-Disconnect флаг НЕ снимает, TTL ~2-6 мин).
func (s *Server) markOnline(user string) {
	if user == "" {
		return
	}
	s.onlineMu.Lock()
	s.online[user] = time.Now()
	s.onlineMu.Unlock()
}

// onlineRecent — relogin акка моложе onlineTtlSec после его login-ok.
func (s *Server) onlineRecent(user string) bool {
	if user == "" || s.Cfg.OnlineTtlSec <= 0 {
		return false
	}
	s.onlineMu.Lock()
	defer s.onlineMu.Unlock()
	last, ok := s.online[user]
	return ok && time.Since(last) < time.Duration(s.Cfg.OnlineTtlSec)*time.Second
}

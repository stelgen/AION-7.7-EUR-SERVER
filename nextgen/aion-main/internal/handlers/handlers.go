// Package handlers — MVP-хендлеры R3: движение/чат/пинг/версия/время.
// Диспетчер: имя (из ops.yaml) -> handler. Каждый handler получает payload C2S
// и пишет SM-ответы каноничным wire-фреймом (шифрование serverKey).
// MVP «соло-мир»: эхо/подтверждения; реальный мир-стейт — после интеграций (R3.5+).
package handlers

import (
	"log"
	"math"
	"time"
)

// Session — состояние одного клиентского соединения.
type Session struct {
	Peer     string
	HP, MP   uint32
	CharName string
	Account  string
	X, Y, Z  float32
	Heading  uint16
	WorldID  uint32
	Sent     int
}

// Sender — функция отправки SM-пакета (имя + payload); main.go оборачивает в шифрованный фрейм.
type Sender func(name string, payload []byte)

// Registry — map «имя CM-пакета» -> handler.
type Registry map[string]func(s *Session, payload []byte, send Sender, logger *log.Logger)

// Build — все MVP-хендлеры R3.
func Build(initFn func(*Session, Sender)) Registry {
	return Registry{
		// --- Версия/время/пинг (handshake-фаза) ---
		"CM_VERSION_CHECK": func(s *Session, p []byte, send Sender, l *log.Logger) {
			// канон: клиент прислал версию; отвечаем SM_VERSION_CHECK (вариант 0x00, len 132 в capture)
			send("SM_VERSION_CHECK", versionPayload())
		},
		"CM_TIME_CHECK": func(s *Session, p []byte, send Sender, l *log.Logger) {
			send("SM_TIME_CHECK", timePayload())
		},
		"CM_PING": func(s *Session, p []byte, send Sender, l *log.Logger) {
			// канон ServerPacketsOpcodes: SM_PONG = 0x8E (7.5 EU)
			send("SM_PONG", nil)
		},

		// --- Движение (MVP: обновляем стейт, эхо SM_MOVE как broadcast себе) ---
		"CM_MOVE": func(s *Session, p []byte, send Sender, l *log.Logger) {
			// CM_MOVE payload (7.x): сессия-зависима; MVP парсим по смещениям capture — ref: R2-дизасм
			if len(p) >= 16 {
				s.X = f32(p[0:4])
				s.Y = f32(p[4:8])
				s.Z = f32(p[8:12])
				s.Heading = u16(p[12:14])
				send("SM_MOVE", movePayload(s))
			}
		},

		// --- Чат (MVP: эхо публичного сообщения себе) ---
		"CM_CHAT_MESSAGE_PUBLIC": func(s *Session, p []byte, send Sender, l *log.Logger) {
			msg := chatText(p)
			log.Printf("[CHAT] %s: %q", s.Peer, msg)
			send("SM_MESSAGE", chatEchoPayload(s, msg))
		},

		// --- Заглушки-подтверждения (R3: не давать клиенту зависать) ---
		"CM_TARGET_SELECT": func(s *Session, p []byte, send Sender, l *log.Logger) {
			send("SM_LOOKATOBJECT", lookPayload(p))
		},
		"CM_LEVEL_READY": func(s *Session, p []byte, send Sender, l *log.Logger) {
			// клиент сообщил «мир загружен» — канон: SM_FLAG_INFO + последовательность мира (capture-раскладки)
			send("SM_FLAG_INFO", nil)
			if initFn != nil {
				initFn(s, send)
			}
		},
	}
}

// --- payload-конструкторы (MVP: минимальные каноничные структуры, длина сверяется по capture) ---

func versionPayload() []byte {
	// capture#3b: SM_VERSION_CHECK len=132 (тело 132-5=127) — MVP: хвост нулями, каноничный размер
	b := make([]byte, 127)
	putU32(b, 0, 7777) // версия-заглушка (уточнить дизасмом в R3.5)
	return b
}

func timePayload() []byte {
	// SM_TIME_CHECK: u32 время (сек) — MVP
	b := make([]byte, 4)
	putU32(b, 0, uint32(time.Now().Unix()))
	return b
}

func movePayload(s *Session) []byte {
	b := make([]byte, 16)
	putF32(b, 0, s.X)
	putF32(b, 4, s.Y)
	putF32(b, 8, s.Z)
	putU16(b, 12, s.Heading)
	return b
}

func lookPayload(req []byte) []byte {
	// SM_LOOKATOBJECT: u32 targetOID — эхо первых 4 байт запроса
	if len(req) >= 4 {
		return []byte{req[0], req[1], req[2], req[3]}
	}
	return nil
}

func chatEchoPayload(s *Session, msg string) []byte {
	// SM_MESSAGE (канон: u32 oid, u8 chatType, u32 senderOID, строка UTF-16LE...) — MVP упрощённо
	b := []byte("MVP:" + msg)
	return b
}

func chatText(p []byte) string {
	// MVP: байты payload как latin-строка (реальный UTF-16LE парс — R3.5)
	if len(p) > 8 {
		return string(p[8:])
	}
	return ""
}

func f32(b []byte) float32 {
	bits := uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
	return math.Float32frombits(bits)
}

func u16(b []byte) uint16 { return uint16(b[0]) | uint16(b[1])<<8 }

func putF32(b []byte, off int, v float32) { putU32(b, off, math.Float32bits(v)) }
func putU32(b []byte, off int, v uint32) {
	b[off] = byte(v)
	b[off+1] = byte(v >> 8)
	b[off+2] = byte(v >> 16)
	b[off+3] = byte(v >> 24)
}
func putU16(b []byte, off int, v uint16) {
	b[off] = byte(v)
	b[off+1] = byte(v >> 8)
}

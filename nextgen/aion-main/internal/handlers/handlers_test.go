package handlers

import (
	"log"
	"strings"
	"testing"
)

func TestPingHandler(t *testing.T) {
	hs := Build(nil)
	var sent []string
	s := &Session{Peer: "test:1"}
	hs["CM_PING"](s, nil, func(n string, p []byte) { sent = append(sent, n) }, log.Default())
	if len(sent) != 1 || sent[0] != "SM_PONG" {
		t.Fatalf("ping -> %v", sent)
	}
}

func TestMoveHandlerUpdatesState(t *testing.T) {
	hs := Build(nil)
	s := &Session{Peer: "test:2"}
	var last []byte
	hs["CM_MOVE"](s, makePayload(), func(n string, p []byte) { last = p; _ = n }, log.Default())
	if s.X != 100.5 || s.Y != 200.25 {
		t.Fatalf("pos: %v %v", s.X, s.Y)
	}
	if last == nil || len(last) < 12 {
		t.Fatalf("SM_MOVE payload: %v", last)
	}
}

func TestChatEcho(t *testing.T) {
	hs := Build(nil)
	s := &Session{Peer: "test:3", CharName: "Test"}
	var got string
	hs["CM_CHAT_MESSAGE_PUBLIC"](s, chatReq("hello"), func(n string, p []byte) {
		got = n + ":" + string(p)
	}, log.Default())
	if !strings.Contains(got, "SM_MESSAGE") || !strings.Contains(got, "hello") {
		t.Fatalf("chat echo: %q", got)
	}
}

// makePayload — CM_MOVE: X=100.5 Y=200.25 Z=1.0 heading=0
func makePayload() []byte {
	p := make([]byte, 16)
	putF32(p, 0, 100.5)
	putF32(p, 4, 200.25)
	putF32(p, 8, 1.0)
	return p
}

// chatReq — упрощённый CM_CHAT: 8 байт заголовка + текст
func chatReq(msg string) []byte {
	p := make([]byte, 8, 8+len(msg))
	return append(p, msg...)
}
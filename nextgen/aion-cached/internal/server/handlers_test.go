package server

import (
	"bytes"
	"os"
	"testing"

	"aion-cached/internal/wire"
)

// TestAdminHandlerInvoked — admin 0x01 (GQ_MAKEBUILDER) → хандлер вызван, ответ-кадр построен.
func TestAdminHandlerInvoked(t *testing.T) {
	called := false
	prev := adminHandlers[0x01]
	defer func() { adminHandlers[0x01] = prev }()
	adminHandlers[0x01] = func(ch Channel, opcode uint16, payload []byte) []byte {
		called = true
		if ch != ChanAdmin {
			t.Errorf("handler channel = %v, want ChanAdmin", ch)
		}
		if ch, _ := interface{}(nil).(interface{}); ch != nil { _ = ch }
		if opcode != 0x01 {
			t.Errorf("opcode = 0x%04X, want 0x0001", opcode)
		}
		return wire.BuildS2C(0x01, []byte{0x01}) // GP: ok
	}

	frame := &wire.Frame{Cmd: 0x01, Payload: []byte{0x05, 'b', 'u', 'i', 'l', 'd', 'r', 0x64}}
	resp := Route(ChanAdmin, frame)
	if !called {
		t.Fatalf("admin handler for opcode 0x01 was NOT invoked")
	}
	if resp == nil {
		t.Fatalf("handler returned nil response")
	}
	if !bytes.Contains(resp, []byte{0x01}) {
		t.Fatalf("response does not contain GP opcode 0x01: %v", resp)
	}
}

// TestServerStubHandlerInvoked — server 0x00 (CacheVersion) → хандлер вызван.
func TestServerStubHandlerInvoked(t *testing.T) {
	called := false
	prev := serverHandlers[0x00]
	defer func() { serverHandlers[0x00] = prev }()
	serverHandlers[0x00] = func(ch Channel, opcode uint16, payload []byte) []byte {
		called = true
		return wire.BuildS2C(0x00, []byte{0x01})
	}
	frame := &wire.Frame{Cmd: 0x00, Payload: nil}
	resp := Route(ChanServer, frame)
	if !called {
		t.Fatalf("server handler for opcode 0x00 was NOT invoked")
	}
	if resp == nil {
		t.Fatalf("handler returned nil response")
	}
}

// TestUnknownOpcodeRejected — неизвестный опкод → nil-ответ + ship unknown (проверка silent drop).
func TestUnknownOpcodeRejected(t *testing.T) {
	if resp := Route(ChanServer, &wire.Frame{Cmd: 0xEE}); resp != nil {
		t.Fatalf("unknown opcode should return nil, got %v", resp)
	}
}

// TestConfigLoad — проверка парсинга конфига.
func TestConfigLoad(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/test.conf"
	cfg := `# тестовый конфиг
server_port 2106
admin_port 2107
npcdb_port 2109
crypt_seed_hex ABCD1234
`
	if err := os.WriteFile(path, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	var c Config
	if err := c.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.ServerPort != 2106 || c.AdminPort != 2107 || c.NpcDbPort != 2109 {
		t.Fatalf("ports wrong: %+v", c)
	}
	if c.Seed() != 0xABCD1234 {
		t.Fatalf("seed = %x, want abcd1234", c.Seed())
	}
}

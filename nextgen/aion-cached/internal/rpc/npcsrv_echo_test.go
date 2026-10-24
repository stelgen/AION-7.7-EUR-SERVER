package rpc

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"testing"
)

// Тест echo-билдеров 24.10 (живой аргумент NPCSvr → эхо в ответе):
// C233[arg4|ts4] → O129[arg4][0000] · C46[arg4] → O162[arg4][010000] ·
// C40[arg4|01] → O157[arg4][01000000][010000]. Эталоны — capture 02:44 24.10.
func TestNpcEchoBuilders(t *testing.T) {
	// 233 → 129: C payload c40b0000 deabc46a → O129 payload c40b0000 0000 (6Б)
	out := NewNpcSrvReplay().EmitNpcSrv(233, mustHex("c40b0000deabc46a"))
	if len(out) != 13 { // 7 hdr + 6 payload
		t.Fatalf("233→129: len=%d, хочу 13", len(out))
	}
	if cmd := binary.LittleEndian.Uint16(out[2:4]); cmd != 129 {
		t.Fatalf("233→129: cmd=%d", cmd)
	}
	if !bytes.Equal(out[7:], mustHex("c40b00000000")) {
		t.Fatalf("233→129: payload=%x", out[7:])
	}
	// 46 → 162: C arg 02000000 → O162 [02000000][010000] (7Б payload, 14Б кадр)
	out = NewNpcSrvReplay().EmitNpcSrv(46, mustHex("02000000"))
	if len(out) != 14 || !bytes.Equal(out[7:], mustHex("02000000010000")) {
		t.Fatalf("46→162: %x", out)
	}
	// 40 → 157: C arg 01000000 → O157 [01000000][01000000][010000] (11Б payload, 18Б кадр)
	out = NewNpcSrvReplay().EmitNpcSrv(40, mustHex("0100000001000000"))
	if len(out) != 18 || !bytes.Equal(out[7:], mustHex("0100000001000000010000")) {
		t.Fatalf("40→157: %x", out)
	}
	// короткий payload → билдер молчит (нет аргумента)
	if out := NewNpcSrvReplay().EmitNpcSrv(233, nil); out != nil {
		t.Fatalf("233 без payload: хочу nil, got %x", out)
	}
}

func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

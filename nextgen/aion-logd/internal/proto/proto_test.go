package proto

import (
	"bytes"
	"testing"
)

func TestBuildParseRoundtrip(t *testing.T) {
	body := []byte{1, 2, 3, 4, 5}
	raw := Build(TypeData, body)
	p, used, err := Parse(raw)
	if err != nil || used != len(raw) {
		t.Fatalf("parse: %v used=%d", err, used)
	}
	if p.Type != TypeData || !bytes.Equal(p.Body, body) {
		t.Fatalf("pkt: %+v", p)
	}
	if p.Raw[3] != Marker || p.Raw[4] != ^byte(TypeData) {
		t.Fatalf("marker: % X", p.Raw[:5])
	}
}

func TestParseTwoPacketsAndSplit(t *testing.T) {
	a := Build(TypeServerStarted, []byte("x"))
	b := Build(TypeData, bytes.Repeat([]byte{7}, 20))
	buf := append(append([]byte{}, a...), b...)

	p1, used, err := Parse(buf)
	if err != nil || p1.Type != TypeServerStarted {
		t.Fatalf("first: %v %+v", err, p1)
	}
	p2, _, err := Parse(buf[used:])
	if err != nil || p2.Type != TypeData || len(p2.Body) != 20 {
		t.Fatalf("second: %v %+v", err, p2)
	}
}

func TestBadMarkers(t *testing.T) {
	raw := Build(TypeData, []byte("x"))
	raw[3] = 0xAA
	if _, _, err := Parse(raw); err == nil {
		t.Fatal("ломаный маркер должен дать ошибку")
	}
	// клиентский маркер 0xBA — валиден
	cv := ClientVersion(200604, MinBld)
	if _, _, err := Parse(cv); err != nil {
		t.Fatalf("CMarker должен быть валиден: %v", err)
	}
}

func TestVersionLayout(t *testing.T) {
	body := VersionBody(12345, nil)
	b, minB := ParseVersion(body)
	if b != 12345 || minB != MinBld {
		t.Fatalf("version: b=%d min=%d", b, minB)
	}
	raw := Build(TypeVersion, body)
	if raw[2] != TypeVersion || raw[3] != Marker || raw[4] != 0xFF {
		t.Fatalf("hdr: % X", raw[:5])
	}
	// SYSTEMTIME хвост: год 2026 = EA 07
	if raw[13] != 0xEA || raw[14] != 0x07 {
		t.Fatalf("systime year: % X", raw[13:15])
	}
}

func TestClientVersionLayout(t *testing.T) {
	// capture 05.10: клиент шлёт [0D 00][00][BA][FF][200604 u32][10003 u32]
	raw := ClientVersion(200604, MinBld)
	if len(raw) != 13 || raw[0] != 13 || raw[2] != TypeVersion || raw[3] != CMarker || raw[4] != 0xFF {
		t.Fatalf("cver: % X", raw)
	}
	p, _, err := Parse(raw)
	if err != nil || p.Type != TypeVersion || len(p.Body) != 8 {
		t.Fatalf("cver parse: %v", err)
	}
	b, minB := ParseVersion(p.Body)
	if b != 200604 || minB != MinBld {
		t.Fatalf("cver body: %d/%d", b, minB)
	}
	// SendServerStarted: body 12 = 3×u32; capture: (1,2,0)/(1,3,0)/(1,4,-22)
	ss := ServerStarted(1, 4, -22)
	p, _, err = Parse(ss)
	if err != nil || p.Type != TypeServerStarted || len(p.Body) != 12 {
		t.Fatalf("ss parse: %v", err)
	}
	if p.Body[8:12][0] != 0xEA { // -22 LE = EA FF FF FF
		t.Fatalf("ss tail: % X", p.Body[8:12])
	}
}

func TestVersionAndTime(t *testing.T) {
	raw := VersionAndTime()
	if len(raw) != 13 || raw[0] != 13 || raw[2] != 2 || raw[3] != Marker || raw[4] != 0xFD {
		t.Fatalf("vt: % X", raw)
	}
	q := ParseVersionAndTime(raw[5:])
	if q == 0 {
		t.Fatal("filetime пуст")
	}
	unix := int64(q/10000000 - 11644473600)
	if unix < 1600000000 {
		t.Fatalf("filetime weird: %d", unix)
	}
}

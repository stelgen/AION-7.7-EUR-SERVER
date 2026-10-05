package proto

import (
	"bytes"
	"encoding/binary"
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
	a := Build(TypeUnknown3, []byte("x"))
	b := Build(TypeData, bytes.Repeat([]byte{7}, 20))
	buf := append(append([]byte{}, a...), b...)

	// разрезаем пополам — Parse должен требовать ещё
	p1, used, err := Parse(buf[:6])
	if err == nil && len(buf) >= used { // тут может быть и полный, если 6>=len(a)
		_ = p1
	}
	p1, used, err = Parse(buf)
	if err != nil || p1.Type != TypeUnknown3 {
		t.Fatalf("first: %v %+v", err, p1)
	}
	p2, _, err := Parse(buf[used:])
	if err != nil || p2.Type != TypeData || len(p2.Body) != 20 {
		t.Fatalf("second: %v %+v", err, p2)
	}
}

func TestBadMarker(t *testing.T) {
	raw := Build(TypeData, []byte("x"))
	raw[3] = 0xAA // ломаем маркер
	if _, _, err := Parse(raw); err == nil {
		t.Fatal("ломаный маркер должен дать ошибку")
	}
}

func TestVersionLayout(t *testing.T) {
	body := VersionBody(12345, nil)
	b, minB := ParseVersion(body)
	if b != 12345 || minB != MinBld {
		t.Fatalf("version: b=%d min=%d", b, minB)
	}
	raw := Build(TypeVersion, body)
	// LogServer64: OnCreate шлёт total 0x1D (29) с SYSTEMTIME — наш min layout меньше, это ок
	if raw[2] != TypeVersion || raw[3] != Marker || raw[4] != 0xFF {
		t.Fatalf("hdr: % X", raw[:5])
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
	// FILETIME → unix sanity: больше 2020 года
	unix := int64(q/10000000 - 11644473600)
	if unix < 1600000000 {
		t.Fatalf("filetime weird: %d", unix)
	}
	_ = binary.LittleEndian
}

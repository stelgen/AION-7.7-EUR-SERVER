package proto

import (
	"encoding/binary"
	"fmt"
)

// Assemble @0x40e540 — сборка wire-поля по fmt-строке из .data AuthGateD
// (расшифровка jump-table 05.10: c=1B, h=u16 LE, d=u32 LE, b=blob(len,ptr),
// s=ASCII-строка, S=UTF-16LE строка + NUL). Используется и для welcome-plaintext,
// и для wire 2110 (fmt "cdd"/"cd"/"cdh"/"cbdb"/"cc").
//
// Типы аргументов: c — byte/int; h — uint16/int; d — uint32/int; b — []byte;
// s — string|[]byte; S — []byte (уже UTF-16LE).
func Assemble(f string, args ...any) []byte {
	buf := make([]byte, 0, 64)
	argi := 0
	next := func() any {
		if argi >= len(args) {
			panic(fmt.Sprintf("assemble: аргументы кончились на fmt[%d]=%q", argi, f[argi]))
		}
		a := args[argi]
		argi++
		return a
	}
	for i := 0; i < len(f); i++ {
		switch f[i] {
		case 'c':
			buf = append(buf, byteOf(next()))
		case 'h':
			var v [2]byte
			binary.LittleEndian.PutUint16(v[:], uint16Of(next()))
			buf = append(buf, v[:]...)
		case 'd':
			var v [4]byte
			binary.LittleEndian.PutUint32(v[:], uint32Of(next()))
			buf = append(buf, v[:]...)
		case 'b':
			blob, ok := next().([]byte)
			if !ok {
				panic("assemble: 'b' требует []byte")
			}
			buf = append(buf, blob...)
		case 's':
			switch v := next().(type) {
			case string:
				buf = append(buf, v...)
			case []byte:
				buf = append(buf, v...)
			default:
				panic("assemble: 's' требует string|[]byte")
			}
		case 'S':
			b, ok := next().([]byte)
			if !ok {
				panic("assemble: 'S' требует []byte (UTF-16LE)")
			}
			buf = append(buf, b...)
			buf = append(buf, 0, 0)
		default:
			panic("assemble: неизвестный fmt-символ " + string(f[i]))
		}
	}
	return buf
}

func num(v any) uint64 {
	switch x := v.(type) {
	case uint8:
		return uint64(x)
	case uint16:
		return uint64(x)
	case uint32:
		return uint64(x)
	case int:
		return uint64(x)
	case int64:
		return uint64(x)
	}
	panic(fmt.Sprintf("assemble: нечисловой аргумент %T", v))
}

func byteOf(v any) byte     { return byte(num(v)) }
func uint16Of(v any) uint16 { return uint16(num(v)) }
func uint32Of(v any) uint32 { return uint32(num(v)) }

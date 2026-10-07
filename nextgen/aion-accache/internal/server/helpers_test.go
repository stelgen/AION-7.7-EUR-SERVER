package server

import (
	"io"

	"aion-accache/internal/proto"
)

func buildForTest(cmd uint16, payload []byte) ([]byte, error) {
	return proto.Build(cmd, payload)
}

func readFull(c io.Reader, buf []byte) (int, error) {
	return io.ReadFull(c, buf)
}

package world

import (
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// FieldOp — одна подмена: смещение → директива ("u32:1001", "u16:120", "hp", "mp", "time").
type Patch struct {
	Name   string
	Fields map[int]string
}

// LoadPatches — читает layouts.yaml (S7: латиница-комменты).
func LoadPatches(path string) (map[string]Patch, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("world: read %s: %w", path, err)
	}
	var doc struct {
		Patches map[string]struct {
			Fields map[int]string `yaml:"fields"`
		} `yaml:"patches"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("world: yaml %s: %w", path, err)
	}
	out := map[string]Patch{}
	for name, p := range doc.Patches {
		out[name] = Patch{Name: name, Fields: p.Fields}
	}
	return out, nil
}

// Apply — применяет подмены к КОПИИ payload (раскладка не мутирует).
// Директивы: "u32:V"/"u16:V" — константа; "hp"/"mp" — значения сессии; "time" — unix now.
func Apply(payload []byte, p Patch, vals map[string]uint32) []byte {
	out := make([]byte, len(payload))
	copy(out, payload)
	for off, dir := range p.Fields {
		var v uint32
		var size int
		switch {
		case strings.HasPrefix(dir, "u32:"):
			vv, _ := strconv.ParseUint(dir[4:], 10, 32)
			v = uint32(vv)
			size = 4
		case strings.HasPrefix(dir, "u16:"):
			vv, _ := strconv.ParseUint(dir[4:], 10, 16)
			v = uint32(vv)
			size = 2
		case dir == "hp" || dir == "mp":
			v = vals[dir]
			size = 4
		case dir == "time":
			v = uint32(time.Now().Unix())
			size = 4
		default:
			continue
		}
		if off < 0 || off+size > len(out) {
			continue
		}
		for i := 0; i < size; i++ {
			out[off+i] = byte(v >> (8 * i))
		}
	}
	_ = binary.LittleEndian
	return out
}

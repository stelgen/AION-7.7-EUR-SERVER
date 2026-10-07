// Package ops — загрузка YAML-реестра опкодов (ops.yaml) и диспетчер «имя → handler».
// Реестр генерится из docs/opcodes-unified-0810.csv; правки арбитража — прямо в YAML (S7).
package ops

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Packet — запись реестра.
type Packet struct {
	Name string `yaml:"name"`
	Dir  string `yaml:"dir"`   // SM | CM
	Op   uint16 `yaml:"op"`
	Src  string `yaml:"src"` // статус согласия: FULL | DIFF | single-<источник>
}

// Registry — индексы реестра.
type Registry struct {
	Packets     []Packet
	byOpSM      map[uint16]Packet
	byOpCM      map[uint16]Packet
	byName      map[string]Packet
}

// Load читает ops.yaml (S7: латиница-комменты, байтово-осторожно).
func Load(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ops: read %s: %w", path, err)
	}
	var doc struct {
		Packets []Packet `yaml:"packets"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("ops: yaml: %w", err)
	}
	r := &Registry{Packets: doc.Packets,
		byOpSM: map[uint16]Packet{}, byOpCM: map[uint16]Packet{}, byName: map[string]Packet{}}
	for _, p := range doc.Packets {
		if p.Dir == "SM" {
			r.byOpSM[p.Op] = p
		} else {
			r.byOpCM[p.Op] = p
		}
		r.byName[p.Name] = p
	}
	return r, nil
}

// Lookup — имя пакета по опкоду и направлению ("SM"/"CM").
func (r *Registry) Lookup(op uint16, dir string) (Packet, bool) {
	if dir == "SM" {
		p, ok := r.byOpSM[op]
		return p, ok
	}
	p, ok := r.byOpCM[op]
	return p, ok
}

// Count — размер реестра (для канарейки-баннера, S7).
func (r *Registry) Count() int { return len(r.Packets) }
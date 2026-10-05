package server

import (
	"os"
	"strconv"
	"strings"
	"sync"
)

// IPList — useForbiddenIPList (etc/BlockIPs.txt): построчные IP "a.b.c.d",
// пустые строки и '#' — комментарии. Подсети не поддерживаются (TODO при нужде).
type IPList struct {
	mu   sync.RWMutex
	set  map[[4]byte]struct{}
	path string
}

func LoadIPList(path string) *IPList {
	l := &IPList{set: map[[4]byte]struct{}{}, path: path}
	if path == "" {
		return l
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return l
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if ip4, ok := parseIP4(line); ok {
			l.set[ip4] = struct{}{}
		}
	}
	return l
}

func (l *IPList) Blocked(ip [4]byte) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	_, ok := l.set[ip]
	return ok
}

func parseIP4(s string) ([4]byte, bool) {
	var out [4]byte
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || n > 255 {
			return out, false
		}
		out[i] = byte(n)
	}
	return out, true
}

// Package agent — Agent API для внешнего агента-разработчика (R2 роадмапа).
// POST /api/agent/run   — выполнить команду на VM (cmd|powershell), JSON-ответ.
// GET  /api/agent/file  — прочитать файл (b64, капа по размеру).
// POST /api/agent/file  — записать/дописать файл (b64).
// GET  /api/agent/ls    — листинг каталога.
// GET  /api/agent/log   — хвост лога из logs.files конфига ({{date}} подставляется).
// Доступ: заголовок X-Agent-Token (конфиг agent.token или env AIONOP_AGENT_TOKEN).
package agent

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"aion-op/internal/config"
)

const maxTimeoutSec = 900

type Handler struct {
	cfg *config.Config
	sem chan struct{} // не больше 2 параллельных exec
}

func Mount(mux *http.ServeMux, cfg *config.Config) {
	h := &Handler{cfg: cfg, sem: make(chan struct{}, 2)}
	mux.HandleFunc("POST /api/agent/run", h.auth(h.run))
	mux.HandleFunc("GET /api/agent/file", h.auth(h.readFile))
	mux.HandleFunc("POST /api/agent/file", h.auth(h.writeFile))
	mux.HandleFunc("GET /api/agent/ls", h.auth(h.ls))
	mux.HandleFunc("GET /api/agent/log", h.auth(h.tail))
}

// auth — токен (constant-time) + аудит-строка в лог op.
func (h *Handler) auth(next http.HandlerFunc) http.HandlerFunc {
	token := h.cfg.Agent.Token
	if v := os.Getenv("AIONOP_AGENT_TOKEN"); v != "" {
		token = v
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ok := token != "" && subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Agent-Token")), []byte(token)) == 1
		log.Printf("AGENT %s %s %s ok=%t", r.Method, r.URL.String(), r.RemoteAddr, ok)
		if !ok {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

// --- POST /api/agent/run ---

type runReq struct {
	Cmd        string `json:"cmd"`
	Shell      string `json:"shell"` // "cmd" (дефолт) | "ps"
	TimeoutSec int    `json:"timeout_sec"`
	Cwd        string `json:"cwd"`
}

type runResp struct {
	ExitCode  int    `json:"exit_code"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	Truncated bool   `json:"truncated"`
	Ms        int64  `json:"ms"`
	Timeout   bool   `json:"timeout,omitempty"`
}

type capWriter struct {
	buf   strings.Builder
	left  int
	cap_  bool
	total int64
}

func (c *capWriter) Write(p []byte) (int, error) {
	c.total += int64(len(p))
	if c.left > 0 {
		n := len(p)
		if n > c.left {
			n = c.left
			c.cap_ = true
		}
		c.buf.Write(p[:n])
		c.left -= n
	} else if len(p) > 0 {
		c.cap_ = true
	}
	return len(p), nil
}

func (h *Handler) run(w http.ResponseWriter, r *http.Request) {
	var req runReq
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil || strings.TrimSpace(req.Cmd) == "" {
		http.Error(w, `{"error":"bad json / пустой cmd"}`, http.StatusBadRequest)
		return
	}
	tmo := req.TimeoutSec
	if tmo <= 0 {
		tmo = h.cfg.Agent.TimeoutSec
	}
	if tmo > maxTimeoutSec {
		tmo = maxTimeoutSec
	}

	select {
	case h.sem <- struct{}{}:
		defer func() { <-h.sem }()
	default:
		http.Error(w, `{"error":"занято: уже выполняется другая команда"}`, http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(tmo)*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	switch req.Shell {
	case "ps":
		cmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", req.Cmd)
	default:
		cmd = exec.CommandContext(ctx, "cmd", "/c", req.Cmd)
	}
	if req.Cwd != "" {
		cmd.Dir = req.Cwd
	}

	capBytes := h.cfg.Agent.MaxOutMB * 1024 * 1024
	if capBytes <= 0 {
		capBytes = 8 << 20
	}
	so, se := &capWriter{left: capBytes}, &capWriter{left: capBytes}
	cmd.Stdout, cmd.Stderr = so, se

	start := time.Now()
	runErr := cmd.Run()
	resp := runResp{Ms: time.Since(start).Milliseconds()}

	if ctx.Err() == context.DeadlineExceeded {
		resp.Timeout = true
		if cmd.Process != nil { // добить дерево: cmd/powershell оставляют внуков
			kill := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid))
			_ = kill.Run()
		}
	} else if runErr != nil {
		if ee, ok := runErr.(*exec.ExitError); ok {
			resp.ExitCode = ee.ExitCode()
		} else {
			resp.ExitCode = -1
			resp.Stderr += "exec: " + runErr.Error()
		}
	}
	resp.Stdout, resp.Stderr = so.buf.String(), se.buf.String()
	resp.Truncated = so.cap_ || se.cap_
	writeJSON(w, resp)
}

// --- GET /api/agent/file?path=&max_mb= ---

func (h *Handler) readFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, `{"error":"path обязателен"}`, http.StatusBadRequest)
		return
	}
	maxMB := 4
	if v, err := strconv.Atoi(r.URL.Query().Get("max_mb")); err == nil && v > 0 && v <= 256 {
		maxMB = v
	}
	f, err := os.Open(path)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
		return
	}
	defer f.Close()
	st, _ := f.Stat()
	capBytes := int64(maxMB) << 20
	size := st.Size()
	buf := make([]byte, min64(size, capBytes))
	_, _ = io.ReadFull(f, buf)
	writeJSON(w, map[string]any{
		"path": path, "size": size, "truncated": size > capBytes,
		"content_b64": base64.StdEncoding.EncodeToString(buf),
	})
}

// --- POST /api/agent/file {path, content_b64, append} ---

func (h *Handler) writeFile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path       string `json:"path"`
		ContentB64 string `json:"content_b64"`
		Append     bool   `json:"append"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 192<<20)).Decode(&req); err != nil || req.Path == "" {
		http.Error(w, `{"error":"bad json / пустой path"}`, http.StatusBadRequest)
		return
	}
	data, err := base64.StdEncoding.DecodeString(req.ContentB64)
	if err != nil {
		http.Error(w, `{"error":"content_b64: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}
	if dir := filepath.Dir(req.Path); dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	flags := os.O_CREATE | os.O_WRONLY
	if req.Append {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	f, err := os.OpenFile(req.Path, flags, 0o644)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer f.Close()
	n, err := f.Write(data)
	writeJSON(w, map[string]any{"ok": err == nil, "bytes": n, "path": req.Path, "err": errStr(err)})
}

// --- GET /api/agent/ls?path= ---

func (h *Handler) ls(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "."
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
		return
	}
	type ent struct {
		Name    string `json:"name"`
		Dir     bool   `json:"dir"`
		Size    int64  `json:"size"`
		ModTime string `json:"modified"`
	}
	out := make([]ent, 0, len(entries))
	for _, e := range entries {
		info, ierr := e.Info()
		var sz int64
		var mt time.Time
		if ierr == nil {
			sz = info.Size()
			mt = info.ModTime()
		}
		out = append(out, ent{Name: e.Name(), Dir: e.IsDir(), Size: sz, ModTime: mt.Format("2006-01-02 15:04:05")})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	writeJSON(w, map[string]any{"path": path, "entries": out})
}

// --- GET /api/agent/log?name=<svc>&tail=N — хвост файла из logs.files ---

func (h *Handler) tail(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	tailN := 200
	if v, err := strconv.Atoi(r.URL.Query().Get("tail")); err == nil && v > 0 && v <= 10000 {
		tailN = v
	}
	var path string
	for _, f := range h.cfg.Logs.Files {
		if f.Svc == name {
			path = f.Path
			break
		}
	}
	if path == "" {
		http.Error(w, `{"error":"неизвестный svc (смотри logs.files в конфиге)"}`, http.StatusNotFound)
		return
	}
	path = strings.ReplaceAll(path, "{{date}}", time.Now().Format("2006-01-02"))
	data, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","path":"`+path+`"}`, http.StatusNotFound)
		return
	}
	if len(data) > 16<<20 { // капа чтения: хвост
		data = data[len(data)-16<<20:]
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(lines) > tailN {
		lines = lines[len(lines)-tailN:]
	}
	writeJSON(w, map[string]any{"path": path, "lines": lines})
}

func errStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

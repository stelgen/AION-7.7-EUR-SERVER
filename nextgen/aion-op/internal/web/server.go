// Package web — HTTP API + embedded UI. Phase 0: никаких управляющих роутов вообще.
package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"aion-op/internal/config"
	"aion-op/internal/core"
	"aion-op/internal/probe"
)

//go:embed ui/index.html ui/app.js ui/style.css
var uiFS embed.FS

type Server struct {
	cfg    *config.Config
	prober probe.Prober

	mu   sync.RWMutex
	snap probe.Snapshot
}

func New(cfg *config.Config, p probe.Prober) *Server {
	return &Server{cfg: cfg, prober: p}
}

func (s *Server) Run() error {
	s.snap = s.prober.Snapshot() // первый срез сразу
	go s.loop()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("GET /api/config", s.handleConfig)
	mux.HandleFunc("GET /static/app.js", s.serveUI("ui/app.js", "text/javascript; charset=utf-8"))
	mux.HandleFunc("GET /static/style.css", s.serveUI("ui/style.css", "text/css; charset=utf-8"))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		s.serveUI("ui/index.html", "text/html; charset=utf-8")(w, r)
	})

	addr := fmt.Sprintf(":%d", s.cfg.Operator.UIPort)
	log.Printf("http: слушаю %s", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) serveUI(name, ctype string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := uiFS.ReadFile(name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", ctype)
		_, _ = w.Write(b)
	}
}

func (s *Server) loop() {
	t := time.NewTicker(time.Duration(s.cfg.Operator.RefreshSec) * time.Second)
	defer t.Stop()
	for range t.C {
		snap := s.prober.Snapshot()
		s.mu.Lock()
		s.snap = snap
		s.mu.Unlock()
	}
}

// --- API-модели ---

type GroupView struct {
	ID       string           `json:"id"`
	Title    string           `json:"title"`
	Icon     string           `json:"icon"`
	Services []core.SvcStatus `json:"services"`
}

type WorldView struct {
	Conns         int    `json:"conns"`
	Expected      int    `json:"expected"`
	Collected     bool   `json:"collected"`
	LoadingWindow bool   `json:"loading_window"`
	PairBroken    bool   `json:"pair_broken"`
	PairNote      string `json:"pair_note,omitempty"`
}

type Summary struct {
	Running int `json:"running"`
	Warn    int `json:"warn"`
	Stopped int `json:"stopped"`
}

type Payload struct {
	When           string      `json:"when"`
	Source         string      `json:"source"`
	ProbeErr       string      `json:"probe_err,omitempty"`
	Mode           string      `json:"mode"`
	RefreshSec     int         `json:"refresh_sec"`
	VMMode         string      `json:"vm_mode"`
	ConsoleSession bool        `json:"console_session"`
	Summary        Summary     `json:"summary"`
	World          WorldView   `json:"world"`
	Groups         []GroupView `json:"groups"`
}

func (s *Server) build() Payload {
	s.mu.RLock()
	snap := s.snap
	s.mu.RUnlock()

	byGroup := map[string][]core.SvcStatus{}
	var flat []core.SvcStatus
	for _, svc := range s.cfg.Services {
		st := core.EvalService(svc, snap, s.cfg.WorldPair)
		byGroup[svc.Group] = append(byGroup[svc.Group], st)
		flat = append(flat, st)
	}

	groups := make([]GroupView, 0, len(s.cfg.Groups))
	ids := make([]string, 0, len(s.cfg.Groups))
	for id := range s.cfg.Groups {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		g := s.cfg.Groups[id]
		svcs := byGroup[id]
		sort.SliceStable(svcs, func(i, j int) bool { return svcs[i].Order < svcs[j].Order })
		groups = append(groups, GroupView{ID: id, Title: g.Title, Icon: g.Icon, Services: svcs})
	}

	broken, note := core.PairBroken(flat)
	sum := Summary{}
	for _, st := range flat {
		switch {
		case st.State == core.StateRunning:
			sum.Running++
		case st.State.Severity() >= 2:
			sum.Warn++
		default:
			sum.Stopped++
		}
	}

	return Payload{
		When:           snap.When.Format("15:04:05"),
		Source:         snap.Source,
		ProbeErr:       snap.Err,
		Mode:           s.cfg.Operator.Mode,
		RefreshSec:     s.cfg.Operator.RefreshSec,
		VMMode:         s.cfg.VM.Mode,
		ConsoleSession: snap.ConsoleSession,
		Summary:        sum,
		World: WorldView{
			Conns:         snap.Conns2002,
			Expected:      s.cfg.WorldPair.ExpectedConns,
			Collected:     snap.Conns2002 >= s.cfg.WorldPair.ExpectedConns,
			LoadingWindow: snap.Conns2002 > 0 && snap.Conns2002 < s.cfg.WorldPair.ExpectedConns,
			PairBroken:    broken,
			PairNote:      note,
		},
		Groups: groups,
	}
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.build())
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	// read-only вид топологии (без секретов — их в конфиге и нет).
	writeJSON(w, s.cfg)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

// Package web — HTTP API + embedded UI. Phase 1: в operate-режиме появляется
// POST /api/action (план → safety → confirm → исполнение/dry-run → audit).
package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"aion-op/internal/act"
	"aion-op/internal/agent"
	"aion-op/internal/alerts"
	"aion-op/internal/config"
	"aion-op/internal/core"
	"aion-op/internal/metrics"
	"aion-op/internal/probe"
	"aion-op/internal/sqlmon"
	"aion-op/internal/store"
)

//go:embed ui/index.html ui/app.js ui/style.css
var uiFS embed.FS

type Server struct {
	cfg       *config.Config
	prober    probe.Prober
	store     *store.Store
	engine    *alerts.Engine
	holder    *metrics.Holder
	exec      *act.Executor
	sqlHolder *sqlmon.Holder

	mu   sync.RWMutex
	snap probe.Snapshot
}

func New(cfg *config.Config, p probe.Prober, st *store.Store, eng *alerts.Engine,
	h *metrics.Holder, e *act.Executor, sh *sqlmon.Holder) *Server {
	return &Server{cfg: cfg, prober: p, store: st, engine: eng, holder: h, exec: e, sqlHolder: sh}
}

func (s *Server) Run() error {
	s.snap = s.prober.Snapshot() // первый срез сразу
	go s.loop()

	mux := http.NewServeMux()
	// R2: Agent API — монтируется только при agent.enabled + токене
	// (конфиг agent.token или env AIONOP_AGENT_TOKEN).
	if s.cfg.Agent.Enabled && (s.cfg.Agent.Token != "" || os.Getenv("AIONOP_AGENT_TOKEN") != "") {
		agent.Mount(mux, s.cfg)
		log.Printf("agent: API смонтирован (/api/agent/*)")
	} else if s.cfg.Agent.Enabled {
		log.Printf("agent: enabled, но токена нет (config/env) — API НЕ смонтирован")
	}
	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("GET /api/config", s.handleConfig)
	mux.HandleFunc("GET /api/events", s.handleEvents)
	mux.HandleFunc("GET /api/metrics", s.handleMetrics)
	mux.HandleFunc("GET /api/alerts", s.handleAlerts)
	mux.HandleFunc("GET /api/sql", s.handleSql)
	// Phase 1: управляющий роут монтируется ТОЛЬКО в operate-режиме;
	// в observe его физически нет (не disabled — отсутствует).
	if s.cfg.Operator.Mode == "operate" {
		mux.HandleFunc("POST /api/action", s.handleAction)
	}
	mux.HandleFunc("GET /static/app.js", s.serveUI("ui/app.js", "text/javascript; charset=utf-8"))
	mux.HandleFunc("GET /static/style.css", s.serveUI("ui/style.css", "text/css; charset=utf-8"))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		s.serveUI("ui/index.html", "text/html; charset=utf-8")(w, r)
	})

	addr := fmt.Sprintf("%s:%d", s.cfg.Operator.Bind, s.cfg.Operator.UIPort)
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

		// движок алертов: срез сервисов → открыть/закрыть
		var flat []core.SvcStatus
		for _, svc := range s.cfg.Services {
			flat = append(flat, core.EvalService(svc, snap, s.cfg.WorldPair))
		}
		broken, _ := core.PairBroken(flat)
		s.engine.Tick(flat, snap.Err, broken)
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
	When           string            `json:"when"`
	Source         string            `json:"source"`
	ProbeErr       string            `json:"probe_err,omitempty"`
	Mode           string            `json:"mode"`
	RefreshSec     int               `json:"refresh_sec"`
	VMMode         string            `json:"vm_mode"`
	ConsoleSession bool              `json:"console_session"`
	Summary        Summary           `json:"summary"`
	World          WorldView         `json:"world"`
	Groups         []GroupView       `json:"groups"`
	Events         []store.EventRow  `json:"events"`
	Alerts         []store.AlertRow  `json:"alerts"`
	Metrics        metrics.Snap      `json:"metrics"`
	Actions        []store.ActionRow `json:"actions"`
	DryRun         bool              `json:"dry_run"`
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
		Groups:  groups,
		Events:  s.store.RecentEvents(60),
		Alerts:  s.engine.Snapshot(),
		Actions: s.store.RecentActions(20),
		Metrics: s.holder.Get(),
		DryRun:  s.exec.DryRun(),
	}
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.build())
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	// read-only вид топологии (без секретов — их в конфиге и нет).
	writeJSON(w, s.cfg)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	limit := 300
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 2000 {
			limit = n
		}
	}
	writeJSON(w, s.store.RecentEvents(limit))
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	minutes := 240
	if v := r.URL.Query().Get("minutes"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 60*24*30 {
			minutes = n
		}
	}
	if name == "" {
		http.Error(w, "name обязателен (exe, напр. Server64.exe)", http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]any{"name": name, "points": s.store.Series(name, minutes)})
}

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.engine.Snapshot())
}

// handleSql — SQL-вкладка: срез + CCU-последние + gauge-история + waits-суммы.
func (s *Server) handleSql(w http.ResponseWriter, r *http.Request) {
	world, auth := s.store.LatestWorldAuth()
	writeJSON(w, map[string]any{
		"snap":       s.sqlHolder.Get(),
		"world":      world,
		"auth":       auth,
		"gauge_hist": s.store.GaugeHistory(240),
		"waits":      s.store.WaitSums(120, 8),
	})
}

// handleAction — Phase 1: план → safety → confirm → исполнение (или dry-run) → audit.
type actionReq struct {
	Action  string `json:"action"`
	ID      string `json:"id"`
	Confirm string `json:"confirm"`
}

func (s *Server) handleAction(w http.ResponseWriter, r *http.Request) {
	var req actionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.RLock()
	snap := s.snap
	s.mu.RUnlock()
	plan := s.exec.Build(req.Action, req.ID, snap)
	res := s.exec.Execute(r.Context(), plan, req.Confirm)
	writeJSON(w, res)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

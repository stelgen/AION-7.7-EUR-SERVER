// Package act — Phase 1: действия (start/stop/restart) с безопасностью best-practice:
// план → проверки → confirm для danger → исполнение → audit. Windows-канон:
// запуск = schtasks /run по задачам; стоп юзер-сессии = /IT kill-задачи
// (taskkill из SYSTEM/SSH по юзер-сессии молча не работает); стоп session-0 = taskkill /IM.
package act

import (
	"context"
	"fmt"
	"strings"
	"time"

	"aion-op/internal/config"
	"aion-op/internal/core"
	"aion-op/internal/probe"
	"aion-op/internal/store"
)

// Plan — то, что БУДЕТ исполнено (возвращается всегда: dry_run/audit/preview).
type Plan struct {
	Action      string   `json:"action"`
	Target      string   `json:"target"`
	Steps       []string `json:"steps,omitempty"` // команды по порядку
	Danger      bool     `json:"danger"`
	NeedConfirm string   `json:"need_confirm,omitempty"`
	Notes       []string `json:"notes,omitempty"`
	Rejected    string   `json:"rejected,omitempty"` // непусто = исполнение запрещено
}

// Result — исход исполнения.
type Result struct {
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
	Plan   Plan   `json:"plan"`
}

type Executor struct {
	cfg *config.Config
	st  *store.Store
	run func(ctx context.Context, cmd string) (string, error)
}

func New(cfg *config.Config, st *store.Store, run func(ctx context.Context, cmd string) (string, error)) *Executor {
	return &Executor{cfg: cfg, st: st, run: run}
}

func (e *Executor) DryRun() bool { return e.cfg.Operator.DryRun != nil && *e.cfg.Operator.DryRun }

// svc — сервис по id или "pair" → main.
func (e *Executor) svc(id string) (config.Service, bool) {
	if id == "pair" {
		return e.cfg.ByID("main")
	}
	return e.cfg.ByID(id)
}

// Build — план действия. world — последний срез для safety-проверок.
func (e *Executor) Build(action, id string, snap probe.Snapshot) Plan {
	p := Plan{Action: action, Target: id}
	svc, ok := e.svc(id)
	if !ok {
		p.Rejected = fmt.Sprintf("неизвестный сервис %q", id)
		return p
	}
	if svc.Locked {
		p.Rejected = "сервис под замком (мёртвый в 7.7-ките) — действий нет"
		return p
	}

	// Safety: проба должна быть живой
	if snap.Err != "" {
		p.Rejected = "проба VM недоступна (состояния UNKNOWN) — действия запрещены: " + firstLine(snap.Err)
		return p
	}

	switch action {
	case "start":
		if svc.Task == "" {
			p.Rejected = "задача запуска не задана в конфиге"
			return p
		}
		p.Steps = []string{"schtasks /run /tn " + svc.Task}
		p.Notes = append(p.Notes, "запуск в родной сессии через задачу "+svc.Task)

	case "stop":
		stop, ok := e.stopStep(svc)
		if !ok {
			p.Rejected = "стоп недоступен: нужна /IT kill-задача (kill_task в конфиге)"
			return p
		}
		p.Steps = []string{stop}

	case "restart":
		if svc.Pair != "" {
			p.Rejected = "члены пары рестартятся ТОЛЬКО парой (action=restart_pair, target=pair)"
			return p
		}
		stop, ok := e.stopStep(svc)
		if !ok {
			p.Rejected = "стоп недоступен: нужна /IT kill-задача (kill_task в конфиге)"
			return p
		}
		p.Danger = true
		p.NeedConfirm = "restart"
		p.Steps = []string{stop, "пауза 5с", "schtasks /run /tn " + svc.Task}

	case "restart_pair":
		if id != "pair" {
			p.Rejected = "restart_pair применим только к target=pair"
			return p
		}
		// Окно загрузки = рестарты запрещены (10–15 мин груз)
		main, _ := e.cfg.ByID("main")
		st := core.EvalService(main, snap, e.cfg.WorldPair)
		if st.State == core.StateLoading || st.State == core.StateDegraded && snap.Conns2002 > 0 {
			p.Rejected = fmt.Sprintf(
				"окно загрузки/деградация (%d/%d conns) — рестарт пары запрещён (кнопка сама разблокируется)",
				snap.Conns2002, e.cfg.WorldPair.ExpectedConns)
			return p
		}
		mainTask, _ := e.svc("main")
		npc, _ := e.cfg.ByID("npc")
		mainStop, mainOK := e.stopStep(mainTask)
		npcStop, npcOK := e.stopStep(npc)
		if !mainOK || !npcOK {
			p.Rejected = "нужны kill-задачи обоих членов пары (kill_task в конфиге)"
			return p
		}
		p.Danger = true
		p.NeedConfirm = "RESTART PAIR"
		p.Steps = []string{
			mainStop, // Server64
			npcStop,  // NPCSvr (умрёт graceful следом)
			"пауза 20с",
			"schtasks /run /tn " + npc.Task, // NPCSvr
			"ждать полной загрузки NPCSvr (~14.9 ГБ, 10–15 мин)",
			"schtasks /run /tn " + mainTask.Task, // Server64
			"ждать conns 16/16 на :2002 и LISTENING 7777",
		}
		p.Notes = append(p.Notes, "пара = единая единица; Server64 без RunAsDate (#180 байпасс)")

	default:
		p.Rejected = "неизвестное действие " + action
		return p
	}

	if e.cfg.Operator.DryRun != nil && *e.cfg.Operator.DryRun && p.Rejected == "" {
		p.Notes = append(p.Notes, "DRY_RUN: исполнение отключено в конфиге (operator.dry_run=true)")
	}
	return p
}

func (e *Executor) stopStep(s config.Service) (string, bool) {
	if s.KillTask != "" {
		return "schtasks /run /tn " + s.KillTask, true
	}
	if s.Interactive || s.Heavy {
		return "", false // юзер-сессия: голый taskkill не работает
	}
	if s.Exe != "" {
		return "taskkill /F /IM " + s.Exe, true // session-0 (SYSTEM-контекст)
	}
	return "", false
}

// Execute — confirm-гейт → исполнение шагов → audit. При dry_run шаги НЕ исполняются.
func (e *Executor) Execute(ctx context.Context, p Plan, confirm string) Result {
	audit := func(ok bool, executed bool, detail string) {
		e.st.AddAction(store.ActionRow{
			Ts: time.Now().Unix(), Action: p.Action, Target: p.Target,
			OK: ok, Executed: executed, Detail: detail,
		})
	}
	if p.Rejected != "" {
		audit(false, false, "rejected: "+p.Rejected)
		return Result{OK: false, Detail: p.Rejected, Plan: p}
	}
	if p.Danger && confirm != p.NeedConfirm {
		audit(false, false, "нет confirm")
		return Result{OK: false, Detail: fmt.Sprintf(
			"опасное действие: нужен confirm=%q (typed-confirm)", p.NeedConfirm), Plan: p}
	}
	if e.DryRun() {
		audit(true, false, "dry-run: план построен, исполнение отключено")
		return Result{OK: true, Detail: "DRY_RUN: исполнение отключено (operator.dry_run=true)", Plan: p}
	}

	var logw strings.Builder
	for _, step := range p.Steps {
		t := strings.TrimSpace(step)
		if strings.HasPrefix(t, "пауза") || strings.HasPrefix(t, "ждать") || strings.HasPrefix(t, "…") {
			logw.WriteString(step + " — (пометка плана, не команда)\n")
			continue
		}
		out, err := e.run(ctx, step)
		detail := strings.TrimSpace(firstLine(out))
		if err != nil {
			logw.WriteString(fmt.Sprintf("FAIL %s: %v %s\n", step, err, detail))
			audit(false, true, logw.String())
			return Result{OK: false, Detail: logw.String(), Plan: p}
		}
		logw.WriteString("OK " + step + "\n")
	}
	audit(true, true, logw.String())
	return Result{OK: true, Detail: logw.String(), Plan: p}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

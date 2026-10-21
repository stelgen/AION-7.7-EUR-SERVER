// Package watchdog — Phase 1.5: событийный автопилот (OP-6).
// Тикер-горутина: раз в interval_sec снимает срез, смотрит op-состояния:
//   - svc DOWN (не expected_down) → авторестарт с эскалацией backoff
//     (escalate: 1м→5м→30м, max_attempts; после — алерт + стоп автопилота
//     до ручного вбития ResetHalt);
//   - окно загрузки NPC (10–15 мин) — рестарты пары запрещены (тот же lock,
//     что в act.pairPlan);
//   - ночное окно (nightly.start..end) + тумблер user_toggle →
//     restart_pair npc+main один раз в сутки;
//   - каждое автодействие идёт в лог + SQLite audit-journal как ручное,
//     источник = "watchdog" (колонка actions.source).
//
// Дефолт enabled: false — включается осознанно после деплоя.
package watchdog

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"aion-op/internal/act"
	"aion-op/internal/config"
	"aion-op/internal/core"
	"aion-op/internal/probe"
	"aion-op/internal/store"
)

// Actions — то, что watchdog может делать (act.Executor удовлетворяет).
type Actions interface {
	Build(action, id string, snap probe.Snapshot) act.Plan
	ExecuteAs(ctx context.Context, p act.Plan, confirm, source string) act.Result
}

type Watchdog struct {
	cfg      config.WatchdogCfg
	wp       config.WorldPair
	services []config.Service
	actions  Actions
	prober   probe.Prober
	st       *store.Store
	logf     func(format string, args ...any)
	now      func() time.Time // подменяется в тестах

	mu           sync.Mutex
	attempts     map[string]int
	lastAttempt  map[string]time.Time
	halted       bool
	lastNightlyYMD string
}

func New(cfg *config.Config, actions Actions, prober probe.Prober, st *store.Store) *Watchdog {
	return &Watchdog{
		cfg: cfg.Watchdog, wp: cfg.WorldPair, services: cfg.Services,
		actions: actions, prober: prober, st: st,
		logf: log.Printf, now: time.Now,
		attempts: map[string]int{}, lastAttempt: map[string]time.Time{},
	}
}

// Loop — фоновая горутина; тик каждые cfg.IntervalSec (дефолт 30с).
func (w *Watchdog) Loop(ctx context.Context) {
	interval := time.Duration(w.cfg.IntervalSec) * time.Second
	if interval <= 0 {
		interval = 30 * time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.Tick(ctx)
		}
	}
}

// Tick — один проход проверки. Возвращает число выполненных автодействий.
// Тесты дергают Tick напрямую с fake-тикером (своим now/срезом).
func (w *Watchdog) Tick(ctx context.Context) int {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.cfg.Enabled || w.halted {
		return 0
	}
	snap := w.prober.Snapshot()
	if snap.Err != "" {
		return 0 // срез недоступен — ничего не дергаем (UNKNOWN)
	}
	n := 0
	if w.nightlyDue() {
		if w.restartPair(ctx, snap, "ночной рестарт пары") {
			n++
			w.lastNightlyYMD = w.now().Format("2006-01-02")
		}
	}
	for _, svc := range w.services {
		if w.autoRepairable(svc) == "" {
			continue
		}
		if w.checkDown(ctx, svc, snap) {
			n++
		}
	}
	return n
}

// autoRepairable — причина автопилота для сервиса или "" (не трогаем).
func (w *Watchdog) autoRepairable(s config.Service) string {
	if s.Locked || s.ObserveOnly || s.ExpectedDown || s.Task == "" {
		return ""
	}
	return s.ID
}

// checkDown — сервис в STOPPED (не expected_down) → авторестарт с эскалацией.
func (w *Watchdog) checkDown(ctx context.Context, s config.Service, snap probe.Snapshot) bool {
	st := core.EvalService(s, snap, w.wp)
	if st.State != core.StateStopped {
		// ожил: сбрасываем эскалацию и закрываем алерт
		if w.attempts[s.ID] > 0 {
			w.logf("watchdog: %s ожил, эскалация сброшена", s.ID)
			w.attempts[s.ID] = 0
			if w.st != nil {
				w.st.CloseAlert("watchdog:" + s.ID)
			}
		}
		return false
	}
	// NPC-окно защиты: пара грузится — рестарты пары отложены, не считаем попыткой
	if s.Pair != "" && w.loadingWindow(snap) {
		w.logf("watchdog: %s down, но окно загрузки пары — рестарт отложен", s.ID)
		return false
	}
	if w.attempts[s.ID] >= w.maxAttempts() {
		// предел: алерт + стоп автопилота до ручного вбития
		w.halted = true
		text := fmt.Sprintf("watchdog: %s не поднялся после %d автопопыток — автопилот остановлен, нужно ручное вмешательство", s.ID, w.maxAttempts())
		w.logf("%s", text)
		if w.st != nil {
			w.st.OpenAlert("watchdog:"+s.ID, 3, text)
		}
		return false
	}
	// backoff: попытка 1 сразу, далее escalate[0], escalate[1], ... между попытками
	if w.attempts[s.ID] > 0 {
		esc := w.cfg.EscalateSec
		idx := w.attempts[s.ID] - 1
		if idx >= len(esc) {
			idx = len(esc) - 1
		}
		delay := time.Duration(esc[idx]) * time.Second
		if w.now().Sub(w.lastAttempt[s.ID]) < delay {
			return false // еще рано
		}
	}
	w.attempts[s.ID]++
	w.lastAttempt[s.ID] = w.now()
	action, target, confirm := "restart", s.ID, "restart"
	if s.Pair != "" {
		action, target, confirm = "restart_pair", "pair", "RESTART PAIR"
	}
	p := w.actions.Build(action, target, snap)
	if p.Rejected != "" {
		// окно загрузки поймало нас в гонке — не считаем попыткой
		if strings.Contains(p.Rejected, "окно загрузки") {
			w.attempts[s.ID]--
		} else {
			w.logf("watchdog: план %s/%s отвергнут: %s", action, target, p.Rejected)
		}
		return false
	}
	res := w.actions.ExecuteAs(ctx, p, confirm, "watchdog")
	w.logf("watchdog: авто-%s %s → ok=%v: %s", action, target, res.OK, firstLine(res.Detail))
	return true
}

// restartPair — ночное окно + тумблер юзера → полный цикл пары.
func (w *Watchdog) restartPair(ctx context.Context, snap probe.Snapshot, why string) bool {
	if !w.cfg.UserToggle || !w.cfg.Nightly.RestartPair {
		return false
	}
	if !inWindow(w.now(), w.cfg.Nightly.Start, w.cfg.Nightly.End) {
		return false
	}
	if w.loadingWindow(snap) {
		w.logf("watchdog: ночное окно, но пара грузится — рестарт отложен")
		return false
	}
	p := w.actions.Build("restart_pair", "pair", snap)
	if p.Rejected != "" {
		w.logf("watchdog: ночной restart_pair отвергнут: %s", p.Rejected)
		return false
	}
	res := w.actions.ExecuteAs(ctx, p, p.NeedConfirm, "watchdog")
	w.logf("watchdog: %s → ok=%v: %s", why, res.OK, firstLine(res.Detail))
	return res.OK
}

func (w *Watchdog) nightlyDue() bool { return w.cfg.Nightly.RestartPair && w.cfg.UserToggle && w.lastNightlyYMD == "" }

func (w *Watchdog) maxAttempts() int {
	if w.cfg.MaxAttempts <= 0 {
		return 3
	}
	return w.cfg.MaxAttempts
}

// loadingWindow — NPC/main грузится (10–15 мин): тот же признак, что и lock в act.
func (w *Watchdog) loadingWindow(snap probe.Snapshot) bool {
	main, ok := configOf(w.services, "main")
	if !ok {
		return false
	}
	return core.EvalService(main, snap, w.wp).State == core.StateLoading
}

// Run — блокирующий цикл автопилота (goroutine в main). Тик → Tick();
// после halt-эскалации тишина до ResetHalt (ручное вбитие).
func (w *Watchdog) Run(ctx context.Context) {
	interval := time.Duration(w.cfg.IntervalSec) * time.Second
	if interval <= 0 {
		interval = 30 * time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	log.Printf("watchdog: Phase 1.5 запущен (interval=%s escalate=%v max=%d nightly=%s-%s)",
		interval, w.cfg.EscalateSec, w.cfg.MaxAttempts, w.cfg.Nightly.Start, w.cfg.Nightly.End)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.mu.Lock()
			halted := w.halted
			w.mu.Unlock()
			if halted {
				continue // жду ResetHalt
			}
			w.Tick(ctx)
		}
	}
}

// ResetHalt — ручное вбитие: снова включает автопилот и сбрасывает эскалацию.
func (w *Watchdog) ResetHalt() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.halted = false
	w.attempts = map[string]int{}
}

func inWindow(t time.Time, start, end string) bool {
	sh, sm, ok1 := parseHM(start)
	eh, em, ok2 := parseHM(end)
	if !ok1 || !ok2 {
		return false
	}
	cur, s, e := t.Hour()*60+t.Minute(), sh*60+sm, eh*60+em
	if s <= e {
		return cur >= s && cur < e
	}
	return cur >= s || cur < e // окно через полночь
}

func parseHM(s string) (int, int, bool) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	var h, m int
	if _, err := fmt.Sscanf(parts[0]+":"+parts[1], "%d:%d", &h, &m); err != nil {
		return 0, 0, false
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, false
	}
	return h, m, true
}

func configOf(services []config.Service, id string) (config.Service, bool) {
	for _, s := range services {
		if s.ID == id {
			return s, true
		}
	}
	return config.Service{}, false
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

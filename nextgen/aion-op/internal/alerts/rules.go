// Package alerts — правила алертов Phase 0.5. Пороги из эксплуатации 02–05.10:
// FreeCommit < 8 ГБ (ночь 03.10: тихая смерть процессов), хендлы Server64,
// Super-Lag/Intentional exception (крэш-прекурсоры CacheD), смерть сервисов.
package alerts

import (
	"fmt"
	"sync"
	"time"

	"aion-op/internal/core"
	"aion-op/internal/logs"
	"aion-op/internal/metrics"
	"aion-op/internal/sqlmon"
	"aion-op/internal/store"
)

const (
	freeCommitLowMB  = 8192  // алерт
	freeCommitOKMB   = 10240 // гистерезис закрытия
	leakDPMSustained = 20000 // дельта хендлов/мин для «утечки»
	leakDPMClean     = 5000
	leakTicksNeeded  = 3 // подряд
	rate2812Window   = 5 * time.Minute
	rate2812Max      = 50
	critLogKeep      = 15 * time.Minute
)

// Engine — активные алерты (map в памяти = истина, store = персистентность).
type Engine struct {
	mu      sync.Mutex
	st      *store.Store
	active  map[string]store.AlertRow
	sustain map[string]int // leak:<name> → тиков подряд

	ev2812   []time.Time // скользящее окно proc_missing
	lastCrit time.Time   // последний критичный лог (super_lag/crash/world_shutdown)
	lastMs   metrics.Snap
	lastG    sqlmon.Gauge
	sustainG map[string]int // sql:blocked → тиков подряд
}

func New(st *store.Store) *Engine {
	return &Engine{st: st, active: map[string]store.AlertRow{}, sustain: map[string]int{}, sustainG: map[string]int{}}
}

// FeedEvent — события логов (рейты 2812, критичные логи).
func (e *Engine) FeedEvent(ev logs.Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	switch {
	case ev.Kind == "proc_missing":
		e.ev2812 = append(e.ev2812, time.Now())
		if len(e.ev2812) > 500 {
			e.ev2812 = e.ev2812[len(e.ev2812)-500:]
		}
	case ev.Sev >= logs.SevCrit:
		e.lastCrit = time.Now()
	}
}

// FeedMetrics — последний срез метрик.
func (e *Engine) FeedMetrics(ms metrics.Snap) {
	e.mu.Lock()
	e.lastMs = ms
	e.mu.Unlock()
}

// FeedSql — последний SQL-срез (gauge).
func (e *Engine) FeedSql(g sqlmon.Gauge) {
	e.mu.Lock()
	e.lastG = g
	e.mu.Unlock()
}

// Tick — пересчёт правил: срез сервисов + проба. Открытие/закрытие через store.
func (e *Engine) Tick(statuses []core.SvcStatus, probeErr string, pairBroken bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	now := time.Now()

	set := func(open bool, idem string, sev int, text string) {
		if open {
			if _, ok := e.active[idem]; !ok {
				e.active[idem] = store.AlertRow{Idem: idem, Sev: sev, Text: text, OpenedTs: now.Unix()}
				e.st.OpenAlert(idem, sev, text)
			} else {
				e.st.OpenAlert(idem, sev, text) // обновить текст
			}
		} else {
			if _, ok := e.active[idem]; ok {
				delete(e.active, idem)
				e.st.CloseAlert(idem)
			}
		}
	}

	// 1. Проба VM (UNKNOWN-всё)
	set(probeErr != "", "probe", logs.SevCrit, "Проба VM не удалась: "+firstLine(probeErr))

	// 2. Пара разомкнута
	set(pairBroken, "pair", logs.SevCrit, "Пара NPC+MAIN разомкнута — рестартить только ПАРОЙ")

	// 3. Сервисы: down / degraded / loading
	for _, st := range statuses {
		if st.Locked {
			continue
		}
		switch st.State {
		case core.StateStopped:
			set(true, "down:"+st.ID, logs.SevMed, st.Display+" не запущен")
			set(false, "degraded:"+st.ID, 0, "")
		case core.StateDegraded:
			set(true, "degraded:"+st.ID, logs.SevMed, st.Display+": "+st.Detail)
			set(false, "down:"+st.ID, 0, "")
		case core.StateLoading:
			set(true, "loading:"+st.ID, logs.SevLow, st.Display+": окно загрузки — рестарты заблокированы")
			set(false, "down:"+st.ID, 0, "")
			set(false, "degraded:"+st.ID, 0, "")
		default:
			set(false, "down:"+st.ID, 0, "")
			set(false, "degraded:"+st.ID, 0, "")
			set(false, "loading:"+st.ID, 0, "")
		}
	}

	// 4. FreeCommit (гистерезис)
	fc := e.lastMs.Sys.FreeCommitMB
	switch {
	case fc > 0 && fc < freeCommitLowMB:
		set(true, "freecommit", logs.SevMed,
			fmt.Sprintf("FreeCommit %d МБ < %d — риск тихой смерти процессов (ночь 03.10)", fc, freeCommitLowMB))
	case fc >= freeCommitOKMB:
		set(false, "freecommit", 0, "")
	}

	// 5. Утечка хендлов (устойчивая 3 тика подряд)
	for _, p := range e.lastMs.Procs {
		idem := "leak:" + p.Name
		if p.HandlesDPM > leakDPMSustained {
			e.sustain[idem]++
			if e.sustain[idem] >= leakTicksNeeded {
				set(true, idem, logs.SevMed, fmt.Sprintf(
					"%s: хендлы +%d/мин устойчиво (%d шт) — кандидат утечки", p.Name, p.HandlesDPM, p.Handles))
			}
		} else {
			e.sustain[idem] = 0
			if p.HandlesDPM <= leakDPMClean {
				set(false, idem, 0, "")
			}
		}
	}

	// 6. Рейт 2812 (отсутствующие процы — compile-шум)
	cut := now.Add(-rate2812Window)
	n := 0
	for _, t := range e.ev2812 {
		if t.After(cut) {
			n++
		}
	}
	set(n > rate2812Max, "rate:2812", logs.SevLow,
		fmt.Sprintf("proc_missing ×%d за %v — compile-шум (2812)", n, rate2812Window))

	// 7. Критичный лог (Super-Lag / Intentional / Shutdown By NpcSocket) — активен 15 мин
	set(!e.lastCrit.IsZero() && now.Sub(e.lastCrit) < critLogKeep, "crit_log", logs.SevCrit,
		"Критичный лог: "+e.lastCrit.Format("15:04:05")+" (Super-Lag/Intentional/NpcSocket Close)")

	// 8. SQL blocking (уроки ночи 04-05: линейные блокировки = Super-Lag CacheD)
	if e.lastG.Blocked > 3 {
		e.sustainG["blocked"]++
		if e.sustainG["blocked"] >= 2 {
			set(true, "sql:blocked", logs.SevMed,
				fmt.Sprintf("SQL: %d заблокированных сессий (2 тика) — смотреть dm_exec_requests", e.lastG.Blocked))
		}
	} else {
		e.sustainG["blocked"] = 0
		set(false, "sql:blocked", 0, "")
	}

	// 9. Compile-очередь RESOURCE_SEMAPHORE (root ночи 04-05) — сразу
	set(e.lastG.ResqDepth > 0, "sql:resq", logs.SevMed,
		fmt.Sprintf("SQL: compile-очередь RESOURCE_SEMAPHORE (%d запросов, %d мс суммарно)",
			e.lastG.ResqDepth, e.lastG.ResqWaitMs))
}

// Snapshot — активные алерты (для UI).
func (e *Engine) Snapshot() []store.AlertRow {
	e.mu.Lock()
	defer e.mu.Unlock()
	res := make([]store.AlertRow, 0, len(e.active))
	for _, a := range e.active {
		res = append(res, a)
	}
	// sev DESC
	for i := 1; i < len(res); i++ {
		for j := i; j > 0 && res[j].Sev > res[j-1].Sev; j-- {
			res[j], res[j-1] = res[j-1], res[j]
		}
	}
	return res
}

func firstLine(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return s[:i]
		}
	}
	return s
}

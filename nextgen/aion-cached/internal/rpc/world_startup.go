// world_startup.go — R4.6 мир-init startup-цепочка (capture W5 23.10).
//
// ДОКАЗАНО (23.10, кросс-анализ worldinit-20261023.txt):
//   - Стартовый бёрст Server64 (W5: 288→O196/O90→RQ_LOAD→RQ_LOAD_ITEM→...)
//     = 120 O-кадров, из них 118 БАЙТ-В-БАЙТ равны world-flow login chain
//     (chain_worldinit_20261023.bin). Новые 2: повтор 0x7A (после C223/C15)
//     и 0xBF (после C15) — позиции #116/#118/#119 (конец цепочки).
//   - Кросс-оконные диффы (W5 vs W7/W11/W14/W15/W28) = ТОЛЬКО сессионные
//     поля: X BE @frame[9:13] + a @frame[13] (канон world_flow.go) + динамика
//     @frame[21]. Стартовый протокол детерминирован → chain-replay достаточен.
//   - Polling 0x3D (GP-опрос, 44/окно) входит в цепочку; тень v16b уже
//     отвечала на него байт-в-байт в W7 (S=44/O=44).
//   - Корень крэш-лупа R5 #3/#4: пер-конн флаг emittedWorldInit переживал
//     рестарт пары — Server64 шлёт НОВЫЙ 288 (новый X), тень молчала.
//     Решение: StartupTracker.OnSession(x) детектит новый X → сброс флагов
//     → повторная эмиссия цепочки с новыми подстановками.
//
// NPCSvr-профиль (W30: 228 O, 150×0xBB + 31 uniq) — ОТДЕЛЬНЫЙ трек R4.6.2.
package rpc

import (
	"embed"
)

//go:embed testdata/chain_worldstartup_20261023.bin
var worldStartupFS embed.FS

// GetWorldStartupChain — полная startup-цепочка W5 (120 кадров, 105408Б).
func GetWorldStartupChain() []byte {
	b, err := worldStartupFS.ReadFile("testdata/chain_worldstartup_20261023.bin")
	if err != nil {
		return nil
	}
	return b
}

// EmitWorldStartup — startup-цепочка с сессионными подстановками (sess1 → текущая).
// Формат входа/выхода = конкатенированные wire-кадры (как chain_rq_load_*).
func EmitWorldStartup(x uint32, charRef uint32, a byte) []byte {
	raw := GetWorldStartupChain()
	if raw == nil {
		return nil
	}
	var out []byte
	for _, f := range SplitWorldFlowFrames(raw) {
		out = append(out, SubstituteWorldFlow(f, WorldFlowSess1X, x, WorldFlowSess1CharRef, charRef, WorldFlowSess1A, a)...)
	}
	return out
}

// EmitWorldStartupLoad — load-сегмент startup-цепочки W5 (от O1 до конца,
// включая хвостовые 0x7A/0xBF) с сессионными подстановками.
// Preload (O196+O90) не входит — Server64 их игнорирует (live-факт: тень
// молчала на них и мир работал).
func EmitWorldStartupLoad(x, charRef uint32, a byte) []byte {
	_, load := SplitWorldFlowPreload(GetWorldStartupChain())
	var out []byte
	for _, f := range load {
		out = append(out, SubstituteWorldFlow(f, WorldFlowSess1X, x, WorldFlowSess1CharRef, charRef, WorldFlowSess1A, a)...)
	}
	return out
}

// StartupTracker — per-conn детектор циклов мир-init (новый X = новый бёрст).
type StartupTracker struct {
	lastX     uint32
	haveX     bool
	startedOK bool // startup-цепочка эмитилась в текущем цикле
}

// OnSession — вызывается на каждый cmd=288. true = НОВЫЙ цикл мир-init
// (первый 288 коннекта или X сменился — Server64 перезапустил мир) →
// вызывающий код должен сбросить emittedWorldInit/emittedStartup.
func (t *StartupTracker) OnSession(x uint32) bool {
	if !t.haveX || t.lastX != x {
		t.lastX = x
		t.haveX = true
		t.startedOK = false
		return true
	}
	return false
}

// MarkEmitted — startup-цепочка отэмиссена в текущем цикле.
func (t *StartupTracker) MarkEmitted() { t.startedOK = true }

// Done — startup отрабатывал в текущем цикле (без повторной эмиссии).
func (t *StartupTracker) Done() bool { return t.startedOK }

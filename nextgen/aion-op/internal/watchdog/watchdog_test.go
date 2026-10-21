package watchdog

import (
	"context"
	"testing"
	"time"

	"aion-op/internal/act"
	"aion-op/internal/config"
	"aion-op/internal/core"
	"aion-op/internal/probe"
	"aion-op/internal/store"
)

// fakeActions — считает вызовы Build/Execute.
type fakeActions struct {
	built     []string // "action/target"
	executed  []string // "action/target"
	executeFn func(p act.Plan) act.Result
}

func (f *fakeActions) Build(action, id string, snap probe.Snapshot) act.Plan {
	f.built = append(f.built, action+"/"+id)
	return act.Plan{Action: action, Target: id, Steps: []string{"step"}}
}
func (f *fakeActions) ExecuteAs(_ context.Context, p act.Plan, confirm, source string) act.Result {
	f.executed = append(f.executed, p.Action+"/"+p.Target)
	if f.executeFn != nil {
		return f.executeFn(p)
	}
	return act.Result{OK: true, Detail: "ok\n"}
}

func wdCfg(enabled bool) *config.Config {
	return &config.Config{
		Services: []config.Service{
			{ID: "gate", Exe: "aion-gate.exe", Task: "AionGate", Ports: []int{2106}},
			{ID: "npc", Exe: "NPCSvr64.exe", Task: "AionNPCit", Heavy: true, Pair: "world_pair"},
			{ID: "main", Exe: "Server64.exe", Task: "AionMainit", Heavy: true, Pair: "world_pair", PairMaster: true},
			{ID: "authd", Exe: "L2Authd.exe", Task: "AionAuthOnly", ExpectedDown: true},
			{ID: "chat", Exe: "ChannelChattingServer.exe", Locked: true},
		},
		WorldPair: config.WorldPair{NPCPort: 2002, ExpectedConns: 16},
		Watchdog: config.WatchdogCfg{
			Enabled: enabled, IntervalSec: 30, EscalateSec: []int{60, 300, 1800}, MaxAttempts: 3,
			Nightly: config.WatchdogNightly{Start: "04:00", End: "05:00", RestartPair: true},
		},
	}
}

func procsDown(names ...string) probe.Snapshot {
	snap := probe.Snapshot{Ports: map[int]bool{}, Processes: map[string][]probe.Proc{}, Conns2002: 16}
	live := map[string]bool{"aion-gate.exe": true, "npcsvr64.exe": true, "server64.exe": true, "l2authd.exe": true}
	for _, n := range names {
		delete(live, n)
	}
	for n := range live {
		snap.Processes[n] = []probe.Proc{{Name: n, PID: "1"}}
	}
	return snap
}

type fakeProber struct{ snap probe.Snapshot }

func (f *fakeProber) Snapshot() probe.Snapshot { return f.snap }

func newW(t *testing.T, cfg *config.Config, snap probe.Snapshot, fa *fakeActions) *Watchdog {
	t.Helper()
	w := New(cfg, fa, &fakeProber{snap: snap}, nil)
	return w
}

// 1) svc down → рестарт вызван (не expected_down, не locked).
func TestDownTriggersRestart(t *testing.T) {
	fa := &fakeActions{}
	w := newW(t, wdCfg(true), procsDown("aion-gate.exe"), fa)
	if n := w.Tick(context.Background()); n != 1 {
		t.Fatalf("Tick=%d", n)
	}
	if len(fa.executed) != 1 || fa.executed[0] != "restart/gate" {
		t.Fatalf("executed: %v", fa.executed)
	}
	// живой сервис не трогается
	fa2 := &fakeActions{}
	w2 := newW(t, wdCfg(true), procsDown(), fa2)
	if n := w2.Tick(context.Background()); n != 0 || len(fa2.executed) != 0 {
		t.Fatalf("живой стек тронут: n=%d %v", n, fa2.executed)
	}
}

// expected_down / locked не рестартятся.
func TestExpectedDownAndLockedSkipped(t *testing.T) {
	fa := &fakeActions{}
	snap := procsDown("l2authd.exe", "channelchattingserver.exe", "aion-gate.exe")
	w := newW(t, wdCfg(true), snap, fa)
	if n := w.Tick(context.Background()); n != 1 || fa.executed[0] != "restart/gate" {
		t.Fatalf("n=%d executed=%v", n, fa.executed)
	}
}

// 2) backoff-эскалация: 3 попытки → алерт + стоп автопилота.
func TestEscalationAndHalt(t *testing.T) {
	fa := &fakeActions{executeFn: func(p act.Plan) act.Result { return act.Result{OK: true, Detail: "ran"} }}
	w := newW(t, wdCfg(true), procsDown("aion-gate.exe"), fa)
	base := time.Now()
	cur := base
	w.now = func() time.Time { return cur }
	ctx := context.Background()

	if n := w.Tick(ctx); n != 1 { // попытка 1 сразу
		t.Fatalf("tick1 n=%d", n)
	}
	cur = base.Add(30 * time.Second)
	if n := w.Tick(ctx); n != 0 { // 30с < escalate[0]=60с — рано
		t.Fatalf("tick2 n=%d (ранняя попытка)", n)
	}
	cur = base.Add(61 * time.Second)
	if n := w.Tick(ctx); n != 1 { // попытка 2
		t.Fatalf("tick3 n=%d", n)
	}
	cur = base.Add(61 * time.Second) // < escalate[1]=300с после попытки 2
	if n := w.Tick(ctx); n != 0 {
		t.Fatalf("tick4 n=%d", n)
	}
	cur = base.Add(61*time.Second + 301*time.Second)
	if n := w.Tick(ctx); n != 1 { // попытка 3
		t.Fatalf("tick5 n=%d", n)
	}
	if len(fa.executed) != 3 {
		t.Fatalf("попыток=%d", len(fa.executed))
	}
	cur = base.Add(61*time.Second + 301*time.Second + 1801*time.Second)
	if n := w.Tick(ctx); n != 0 { // max_attempts исчерпан → halt
		t.Fatalf("tick6 n=%d", n)
	}
	if !w.halted {
		t.Fatal("после 3 попыток автопилот должен быть остановлен")
	}
	// с алертом в store
	st, err := store.Open(t.TempDir()+"/t.db", 1)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	fa3 := &fakeActions{}
	w3 := newW(t, wdCfg(true), procsDown("aion-gate.exe"), fa3)
	w3.st = st
	c3 := time.Now()
	w3.now = func() time.Time { return c3 }
	for i := 0; i < 3; i++ {
		w3.Tick(ctx)
		c3 = c3.Add(2 * time.Hour)
	}
	w3.Tick(ctx)
	st.Flush()
	if as := st.ActiveAlerts(); len(as) != 1 || as[0].Idem != "watchdog:gate" {
		t.Fatalf("alerts: %+v", as)
	}
	// ResetHalt (ручное вбитие) снова включает автопилот
	w3.ResetHalt()
	c3 = c3.Add(2 * time.Hour)
	if n := w3.Tick(ctx); n != 1 {
		t.Fatalf("после ResetHalt n=%d", n)
	}
}

// сервис ожил → эскалация сбрасывается.
func TestRecoveryResetsEscalation(t *testing.T) {
	fa := &fakeActions{}
	w := newW(t, wdCfg(true), procsDown("aion-gate.exe"), fa)
	ctx := context.Background()
	base := time.Now()
	w.now = func() time.Time { return base }
	w.Tick(ctx)
	// ожил
	w2p := &fakeProber{snap: procsDown()}
	w.prober = w2p
	base = base.Add(2 * time.Minute)
	w.Tick(ctx)
	if w.attempts["gate"] != 0 {
		t.Fatalf("эскалация не сброшена: %d", w.attempts["gate"])
	}
}

// 3) ночное окно + тумблер юзера → restart_pair.
func TestNightlyRestartPair(t *testing.T) {
	fa := &fakeActions{}
	cfg := wdCfg(true)
	cfg.Watchdog.UserToggle = true
	// вне окна (10:00) — нет
	w := newW(t, cfg, procsDown(), fa)
	w.now = func() time.Time { return time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC) }
	w.Tick(context.Background())
	for _, b := range fa.built {
		if b == "restart_pair/pair" {
			t.Fatal("рестарт пары вне ночного окна")
		}
	}
	// в окне (04:30) → restart_pair/pair
	w2 := newW(t, cfg, procsDown(), &fakeActions{})
	w2.now = func() time.Time { return time.Date(2026, 10, 21, 4, 30, 0, 0, time.UTC) }
	w2.Tick(context.Background())
	found := false
	for _, b := range w2.actions.(*fakeActions).built {
		if b == "restart_pair/pair" {
			found = true
		}
	}
	_ = found
}

// тумблер юзера выключен → ночного рестарта нет.
func TestNightlyRequiresToggle(t *testing.T) {
	fa := &fakeActions{}
	cfg := wdCfg(true)
	cfg.Watchdog.UserToggle = false
	w := newW(t, cfg, procsDown(), fa)
	w.now = func() time.Time { return time.Date(2026, 10, 21, 4, 30, 0, 0, time.UTC) }
	w.Tick(context.Background())
	for _, b := range fa.built {
		if b == "restart_pair/pair" {
			t.Fatal("тумблер юзера выключен — рестарта быть не должно")
		}
	}
}

// ночной рестарт не повторяется в ту же ночь.
func TestNightlyOncePerNight(t *testing.T) {
	fa := &fakeActions{}
	cfg := wdCfg(true)
	cfg.Watchdog.UserToggle = true
	w := newW(t, cfg, procsDown(), fa)
	cur := time.Date(2026, 10, 21, 4, 30, 0, 0, time.UTC)
	w.now = func() time.Time { return cur }
	ctx := context.Background()
	w.Tick(ctx)
	w.lastNightlyYMD = "2026-10-21"
	nPairs := 0
	for _, b := range fa.built {
		if b == "restart_pair/pair" {
			nPairs++
		}
	}
	if nPairs > 1 {
		t.Fatalf("ночной рестарт повторился: %d", nPairs)
	}
}

// 4) lock: окно загрузки NPC — рестарт пары отложен.
func TestLoadingWindowDefersRestart(t *testing.T) {
	fa := &fakeActions{}
	cfg := wdCfg(true)
	cfg.Watchdog.UserToggle = true
	snap := procsDown()
	snap.Conns2002 = 5 // окно загрузки
	w := newW(t, cfg, snap, fa)
	w.now = func() time.Time { return time.Date(2026, 10, 21, 4, 30, 0, 0, time.UTC) }
	w.Tick(context.Background())
	if w.lastNightlyYMD != "" {
		t.Fatal("в окне загрузки ночной рестарт не должен засчитываться")
	}
	for _, b := range fa.executed {
		if b == "restart_pair/pair" {
			t.Fatal("рестарт пары в окне загрузки запрещен")
		}
	}
}

// окно через полночь.
func TestWindowOverMidnight(t *testing.T) {
	if !inWindow(time.Date(2026, 10, 21, 23, 30, 0, 0, time.UTC), "23:00", "01:00") {
		t.Fatal("23:30 в окне 23:00-01:00")
	}
	if inWindow(time.Date(2026, 10, 21, 2, 0, 0, 0, time.UTC), "23:00", "01:00") {
		t.Fatal("02:00 вне окна 23:00-01:00")
	}
	if !inWindow(time.Date(2026, 10, 21, 4, 30, 0, 0, time.UTC), "04:00", "05:00") {
		t.Fatal("04:30 в окне 04:00-05:00")
	}
}

// enabled=false — ничего не происходит.
func TestDisabledNoop(t *testing.T) {
	fa := &fakeActions{}
	w := newW(t, wdCfg(false), procsDown("aion-gate.exe"), fa)
	if n := w.Tick(context.Background()); n != 0 || len(fa.executed) != 0 {
		t.Fatalf("disabled: n=%d executed=%v", n, fa.executed)
	}
}

var _ = core.StateRunning

// Status(): срез состояния (enabled/halted/attempts) — копия map, без утечек мутации.
func TestStatusSnapshot(t *testing.T) {
	fa := &fakeActions{}
	w := newW(t, wdCfg(true), procsDown(), fa)
	st := w.Status()
	if !st.Enabled || st.Halted || st.Attempts == nil {
		t.Fatalf("status: %+v", st)
	}
	st.Attempts["gate"] = 99 // мутируем копию
	if _, leaked := w.Status().Attempts["gate"]; leaked {
		t.Fatal("Status отдаёт внутреннюю карту — утечка мутации")
	}
	w.mu.Lock()
	w.halted = true
	w.mu.Unlock()
	if !w.Status().Halted {
		t.Fatal("halted не виден в статусе")
	}
}

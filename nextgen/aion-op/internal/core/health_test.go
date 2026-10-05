package core

import (
	"testing"

	"aion-op/internal/config"
	"aion-op/internal/probe"
)

func testService(id, exe string, ports []int, pair string, master bool) config.Service {
	return config.Service{ID: id, Group: "heavy", Display: id, Exe: exe, Task: "T" + id,
		Ports: ports, Pair: pair, PairMaster: master}
}

func baseSnap() probe.Snapshot {
	return probe.Snapshot{
		Processes: map[string][]probe.Proc{}, Ports: map[int]bool{}, Conns2002: 16,
		ConsoleSession: true, Source: "test",
	}
}

func addProc(s *probe.Snapshot, exe string) {
	s.Processes[exe] = append(s.Processes[exe], probe.Proc{Name: exe, PID: "111", MemKB: 1024 * 100})
}

func TestAllRunning(t *testing.T) {
	snap := baseSnap()
	addProc(&snap, "server64.exe")
	addProc(&snap, "npcsvr64.exe")
	snap.Ports[2002] = true
	snap.Ports[7777] = true

	wp := config.WorldPair{NPCPort: 2002, ExpectedConns: 16}
	main := EvalService(testService("main", "Server64.exe", []int{2002, 7777}, "world_pair", true), snap, wp)
	if main.State != StateRunning || main.Detail == "" {
		t.Fatalf("main: %s %q", main.State, main.Detail)
	}
	if len(main.PortsOK) != 2 {
		t.Fatalf("main ports_ok: %v", main.PortsOK)
	}
}

func TestLoadingWindow(t *testing.T) {
	snap := baseSnap()
	snap.Conns2002 = 5
	addProc(&snap, "server64.exe")
	addProc(&snap, "npcsvr64.exe")
	snap.Ports[2002] = true
	snap.Ports[7777] = true

	wp := config.WorldPair{NPCPort: 2002, ExpectedConns: 16}
	main := EvalService(testService("main", "Server64.exe", []int{2002, 7777}, "world_pair", true), snap, wp)
	if main.State != StateLoading {
		t.Fatalf("want LOADING, got %s", main.State)
	}
}

func TestDegradedPortDown(t *testing.T) {
	snap := baseSnap()
	addProc(&snap, "authgated.exe") // порт 2106 не слушает
	svc := EvalService(testService("gate", "AuthGateD.exe", []int{2106}, "", false), snap, config.WorldPair{})
	if svc.State != StateDegraded || len(svc.PortsBad) != 1 {
		t.Fatalf("gate: %s ports_bad=%v", svc.State, svc.PortsBad)
	}
}

func TestStoppedAndPairBroken(t *testing.T) {
	snap := baseSnap() // всё погашено
	wp := config.WorldPair{NPCPort: 2002, ExpectedConns: 16}
	main := EvalService(testService("main", "Server64.exe", []int{2002, 7777}, "world_pair", true), snap, wp)
	npc := EvalService(testService("npc", "NPCSvr64.exe", nil, "world_pair", false), snap, wp)
	if main.State != StateStopped || npc.State != StateStopped {
		t.Fatalf("want STOPPED/STOPPED, got %s/%s", main.State, npc.State)
	}
	if broken, _ := PairBroken([]SvcStatus{main, npc}); broken {
		t.Fatal("оба мертвы — пара НЕ разомкнута")
	}

	// Server64 жив, NPCSvr нет → пара разомкнута
	addProc(&snap, "server64.exe")
	snap.Ports[2002] = true
	snap.Ports[7777] = true
	main2 := EvalService(testService("main", "Server64.exe", []int{2002, 7777}, "world_pair", true), snap, wp)
	npc2 := EvalService(testService("npc", "NPCSvr64.exe", nil, "world_pair", false), snap, wp)
	broken, note := PairBroken([]SvcStatus{main2, npc2})
	if !broken || note == "" {
		t.Fatalf("want pair broken, got %v %q", broken, note)
	}
}

func TestLockedAndProbeFail(t *testing.T) {
	snap := baseSnap()
	svc := EvalService(testService("chat", "ChannelChattingServer.exe", []int{10254}, "", false), snap, config.WorldPair{})
	svc.Locked = true
	// замок в EvalService берётся из config.Service, тут проверим через флаг сервиса:
	cfgSvc := testService("chat", "ChannelChattingServer.exe", []int{10254}, "", false)
	cfgSvc.Locked = true
	got := EvalService(cfgSvc, snap, config.WorldPair{})
	if got.State != StateStopped || got.Detail == "" {
		t.Fatalf("locked: %s %q", got.State, got.Detail)
	}
	_ = svc

	fail := baseSnap()
	fail.Err = "ssh: connection refused"
	st := EvalService(cfgSvc, fail, config.WorldPair{})
	if st.State != StateUnknown {
		t.Fatalf("probe fail: want UNKNOWN, got %s", st.State)
	}
}

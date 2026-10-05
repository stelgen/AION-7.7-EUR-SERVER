package act

import (
	"testing"

	"aion-op/internal/config"
	"aion-op/internal/probe"
)

func testCfg() *config.Config {
	t := true
	return &config.Config{
		Operator: config.Operator{Mode: "observe", DryRun: &t},
		Services: []config.Service{
			{ID: "acc", Exe: "AccountCacheServer.exe", Task: "AionAcc", Ports: []int{2220}, Order: 1},
			{ID: "main", Exe: "Server64.exe", Task: "AionMainit", KillTask: "AionKickMain",
				Ports: []int{2002, 7777}, Heavy: true, Pair: "world_pair", PairMaster: true},
			{ID: "npc", Exe: "NPCSvr64.exe", Task: "AionNPCit", KillTask: "AionKickNPC2",
				Heavy: true, Pair: "world_pair"},
			{ID: "gate", Exe: "AuthGateD.exe", Task: "AionGate", Interactive: true},
			{ID: "chat", Exe: "ChannelChattingServer.exe", Locked: true},
		},
		WorldPair: config.WorldPair{NPCPort: 2002, ExpectedConns: 16},
	}
}

func snapOK() probe.Snapshot {
	procs := map[string][]probe.Proc{}
	for _, name := range []string{"server64.exe", "npcsvr64.exe", "authgated.exe", "accountcacheserver.exe"} {
		procs[name] = []probe.Proc{{Name: name, PID: "1", Session: "Console", MemKB: 100000}}
	}
	ports := map[int]bool{2002: true, 7777: true, 2106: true, 2220: true}
	return probe.Snapshot{Processes: procs, Ports: ports, Conns2002: 16}
}

func TestBuildDryRunPlan(t *testing.T) {
	e := New(testCfg(), nil, nil)
	p := e.Build("restart_pair", "pair", snapOK())
	if p.Rejected != "" {
		t.Fatalf("restart_pair rejected: %s", p.Rejected)
	}
	if !p.Danger || p.NeedConfirm != "RESTART PAIR" || len(p.Steps) < 5 {
		t.Fatalf("plan: %+v", p)
	}
	for _, n := range p.Notes {
		if len(n) > 4 && n[:7] == "DRY_RUN" {
			return // dry-run note есть
		}
	}
	t.Fatalf("нет DRY_RUN-note: %+v", p.Notes)
}

func TestBuildLoadingWindowReject(t *testing.T) {
	e := New(testCfg(), nil, nil)
	s := snapOK()
	s.Conns2002 = 5 // окно загрузки
	p := e.Build("restart_pair", "pair", s)
	if p.Rejected == "" {
		t.Fatal("рестарт в окне загрузки должен быть отвергнут")
	}
}

func TestBuildLockedAndPairMembers(t *testing.T) {
	e := New(testCfg(), nil, nil)
	if p := e.Build("start", "chat", snapOK()); p.Rejected == "" {
		t.Fatal("locked должен быть отвергнут")
	}
	if p := e.Build("restart", "main", snapOK()); p.Rejected == "" {
		t.Fatal("член пары не рестартится сольно")
	}
	if p := e.Build("stop", "main", snapOK()); p.Rejected != "" || p.Steps[0] != "schtasks /run /tn AionKickMain" {
		t.Fatalf("stop main: %+v", p)
	}
	if p := e.Build("start", "acc", snapOK()); p.Rejected != "" || p.Steps[0] != "schtasks /run /tn AionAcc" {
		t.Fatalf("start acc: %+v", p)
	}
	// gate: interactive без kill_task → stop недоступен (нужна /IT задача)
	if p := e.Build("stop", "gate", snapOK()); p.Rejected == "" {
		t.Fatal("stop gate без kill_task должен требовать задачу")
	}
}

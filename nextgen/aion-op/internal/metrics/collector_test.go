package metrics

import "testing"

func TestParsePSProcs(t *testing.T) {
	out := `"ProcessName","Id","Handles","MB"
"Server64","2036","827112","10289"
"NPCSvr64","6872","578451","14931"
`
	procs := parsePSProcs(out)
	if len(procs) != 2 {
		t.Fatalf("procs: %+v", procs)
	}
	if procs[0].Name != "Server64.exe" || procs[0].Handles != 827112 || procs[0].MemMB != 10289 {
		t.Fatalf("p0: %+v", procs[0])
	}
	if procs[1].PID != "6872" {
		t.Fatalf("p1: %+v", procs[1])
	}
}

func TestParsePSOS(t *testing.T) {
	out := `"FreePhysicalMemory","FreeVirtualMemory"
"10137600","45056000"
`
	os := parsePSOS(out)
	if os.FreePhysMB != 9900 || os.FreeCommitMB != 44000 {
		t.Fatalf("os: %+v", os)
	}
}

func TestHolder(t *testing.T) {
	h := &Holder{}
	h.Set(Snap{Err: "x"})
	if h.Get().Err != "x" {
		t.Fatal("holder roundtrip")
	}
}

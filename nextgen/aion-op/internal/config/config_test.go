package config

import "testing"

func TestLoadRealConfig(t *testing.T) {
	c, err := Load("../../config.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(c.Services) != 21 { // 20 + authdprod (OP-1, R6-конфиг)
		t.Fatalf("services: %d, want 21", len(c.Services))
	}
	ap, ok := c.ByID("authdprod")
	if !ok || ap.Task != "AionAuthdProd" || len(ap.Ports) != 2 {
		t.Fatalf("authdprod: %+v", ap)
	}
	if c.WorldPair.ExpectedConns != 8 || c.WorldPair.NPCPort != 2002 { // 8 = fix FALSE ALERT 07.10
		t.Fatalf("world_pair: %+v", c.WorldPair)
	}
	if c.Operator.Mode != "operate" || c.VM.Mode != "local" { // Phase 1: прод-конфиг в бою
		t.Fatalf("modes: op=%s vm=%s", c.Operator.Mode, c.VM.Mode)
	}
	main, ok := c.ByID("main")
	if !ok || !main.PairMaster || !main.Heavy {
		t.Fatalf("main: %+v", main)
	}
	if _, ok := c.Groups["heavy"]; !ok {
		t.Fatal("нет группы heavy")
	}
	locked, expectedDown := 0, 0
	for _, s := range c.Services {
		if s.Locked {
			locked++
		}
		if s.ExpectedDown {
			expectedDown++
		}
	}
	if locked != 6 { // 5 + logsrv (R5-конфиг)
		t.Fatalf("locked: %d, want 6", locked)
	}
	if expectedDown != 2 { // authd + gateorig (OP-3)
		t.Fatalf("expected_down: %d, want 2", expectedDown)
	}
}

func TestValidateRejects(t *testing.T) {
	c := &Config{Operator: Operator{Mode: "yolo"}}
	if err := c.Validate(); err == nil {
		t.Fatal("yolo-режим должен быть отвергнут")
	}
}

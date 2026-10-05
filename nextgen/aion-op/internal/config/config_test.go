package config

import "testing"

func TestLoadRealConfig(t *testing.T) {
	c, err := Load("../../config.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(c.Services) != 16 {
		t.Fatalf("services: %d, want 16", len(c.Services))
	}
	if c.WorldPair.ExpectedConns != 16 || c.WorldPair.NPCPort != 2002 {
		t.Fatalf("world_pair: %+v", c.WorldPair)
	}
	if c.Operator.Mode != "observe" || c.VM.Mode != "mock" {
		t.Fatalf("modes: op=%s vm=%s", c.Operator.Mode, c.VM.Mode)
	}
	main, ok := c.ByID("main")
	if !ok || !main.PairMaster || !main.Heavy {
		t.Fatalf("main: %+v", main)
	}
	if _, ok := c.Groups["heavy"]; !ok {
		t.Fatal("нет группы heavy")
	}
	locked := 0
	for _, s := range c.Services {
		if s.Locked {
			locked++
		}
	}
	if locked != 5 {
		t.Fatalf("locked: %d, want 5", locked)
	}
}

func TestValidateRejects(t *testing.T) {
	c := &Config{Operator: Operator{Mode: "yolo"}}
	if err := c.Validate(); err == nil {
		t.Fatal("yolo-режим должен быть отвергнут")
	}
}

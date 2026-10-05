// Package core — state machine и оценка здоровья: срез + топология → состояния.
package core

import (
	"fmt"
	"strings"

	"aion-op/internal/config"
	"aion-op/internal/probe"
)

type State string

const (
	StateUnknown  State = "UNKNOWN"
	StateStopped  State = "STOPPED"
	StateRunning  State = "RUNNING"
	StateLoading  State = "LOADING"
	StateDegraded State = "DEGRADED"
	StateFatal    State = "FATAL"
)

func (s State) Severity() int {
	switch s {
	case StateDegraded, StateFatal, StateLoading:
		return 2
	case StateStopped, StateUnknown:
		return 1
	default:
		return 0
	}
}

type SvcStatus struct {
	ID          string   `json:"id"`
	Group       string   `json:"group"`
	Display     string   `json:"display"`
	Task        string   `json:"task"`
	KillTask    string   `json:"kill_task"`
	State       State    `json:"state"`
	ProcCount   int      `json:"proc_count"`
	PIDs        []string `json:"pids"`
	MemMB       uint64   `json:"mem_mb"`
	PortsOK     []int    `json:"ports_ok"`
	PortsBad    []int    `json:"ports_bad"`
	Detail      string   `json:"detail"`
	Interactive bool     `json:"interactive"`
	Heavy       bool     `json:"heavy"`
	ObserveOnly bool     `json:"observe_only"`
	Locked      bool     `json:"locked"`
	Order       int      `json:"order"`
}

// EvalService — чистая функция: срез + сервис → статус.
func EvalService(s config.Service, snap probe.Snapshot, wp config.WorldPair) SvcStatus {
	st := SvcStatus{
		ID: s.ID, Group: s.Group, Display: s.Display, Task: s.Task, KillTask: s.KillTask,
		Interactive: s.Interactive, Heavy: s.Heavy,
		ObserveOnly: s.ObserveOnly, Locked: s.Locked, Order: s.Order,
		PortsOK: []int{}, PortsBad: []int{}, PIDs: []string{},
	}
	if snap.Err != "" {
		st.State = StateUnknown
		st.Detail = "проба недоступна: " + firstLine(snap.Err)
		return st
	}

	procs := snap.Processes[strings.ToLower(s.Exe)]
	st.ProcCount = len(procs)
	for _, p := range procs {
		st.PIDs = append(st.PIDs, p.PID)
		st.MemMB += p.MemKB / 1024
	}

	if s.Locked {
		st.State = StateStopped
		st.Detail = "не работает в 7.7-ките (замок, PLAN.md §4)"
		return st
	}
	if st.ProcCount == 0 {
		st.State = StateStopped
		switch s.ID {
		case "main":
			st.Detail = "Server64 не запущен"
		case "npc":
			st.Detail = "NPCSvr не запущен"
		default:
			st.Detail = "процесс не найден"
		}
		return st
	}

	for _, p := range s.Ports {
		if snap.Ports[p] {
			st.PortsOK = append(st.PortsOK, p)
		} else {
			st.PortsBad = append(st.PortsBad, p)
		}
	}
	if len(st.PortsBad) > 0 {
		st.State = StateDegraded
		st.Detail = fmt.Sprintf("процесс есть (PID %s), порт(ы) не слушают: %v",
			strings.Join(st.PIDs, ","), st.PortsBad)
		return st
	}

	// Пара NPC+MAIN: главный (Server64) смотрит на conns2002.
	if s.Pair == "world_pair" && s.PairMaster {
		if snap.Conns2002 >= wp.ExpectedConns {
			st.State = StateRunning
			st.Detail = fmt.Sprintf("мир собран: %d/%d conns на :%d",
				snap.Conns2002, wp.ExpectedConns, wp.NPCPort)
		} else {
			st.State = StateLoading
			st.Detail = fmt.Sprintf("окно загрузки/спавна: %d/%d conns — рестарты заблокированы (10–15 мин)",
				snap.Conns2002, wp.ExpectedConns)
		}
		return st
	}

	st.State = StateRunning
	st.Detail = fmt.Sprintf("PID %s, порты %v", strings.Join(st.PIDs, ","), st.PortsOK)
	return st
}

// PairBroken — пара разомкнута (один жив, другой нет): рестартить только ПАРОЙ.
func PairBroken(statuses []SvcStatus) (bool, string) {
	var main, npc *SvcStatus
	for i := range statuses {
		switch statuses[i].ID {
		case "main":
			main = &statuses[i]
		case "npc":
			npc = &statuses[i]
		}
	}
	if main == nil || npc == nil {
		return false, ""
	}
	switch {
	case main.State == StateStopped && npc.State == StateStopped:
		return false, ""
	case main.State == StateStopped && npc.State != StateStopped:
		return true, "Server64 мёртв, NPCSvr жив — при смерти Server64 NPCSvr умирает graceful; рестартить ПАРУ"
	case npc.State == StateStopped && main.State != StateStopped:
		return true, "NPCSvr нет, Server64 жив — пара разомкнута; рестартить ПАРУ"
	default:
		return false, ""
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

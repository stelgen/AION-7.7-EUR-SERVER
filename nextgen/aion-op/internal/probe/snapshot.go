// SnapshotVia — общая сборка среза состояния через Runner (ssh | local). Phase 1.
package probe

import (
	"context"
	"fmt"
	"time"

	"aion-op/internal/config"
)

// SnapshotVia — срез read-only: tasklist → процессы, netstat → порты/conns, quser → сессия.
func SnapshotVia(r Runner, cfg *config.Config, source string) Snapshot {
	ctx := context.Background()
	snap := Snapshot{
		When:      time.Now(),
		Source:    source,
		Processes: map[string][]Proc{},
		Ports:     map[int]bool{},
	}

	tl, err := r.Run(ctx, "tasklist /fo csv /nh")
	if err != nil {
		snap.Err = err.Error()
		return snap
	}
	snap.Processes = parseTasklistCSV(tl)

	ns, err := r.Run(ctx, "netstat -ano -p tcp")
	if err != nil {
		snap.Err = err.Error()
		return snap
	}
	snap.Ports, snap.Conns2002 = parseNetstat(ns, cfg.WorldPair.NPCPort)

	if qu, err := r.Run(ctx, "quser"); err == nil {
		snap.ConsoleSession = parseQuser(qu)
	}
	return snap
}

var _ = fmt.Sprintf // резерв

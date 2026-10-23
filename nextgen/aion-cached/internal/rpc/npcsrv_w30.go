// npcsrv_w30.go — R4.6.2: W30-бёрст NPCSvr-инициализации (capture 23.10).
//
// ДОКАЗАНО (worldinit-20261023.txt W30, 11:27:08→11:28:00):
//   - NPCSvr-коннект = C201 handshake БЕЗ session-288 → старый sessSet-гейт
//     делал весь NPCSvr-путь мёртвым кодом (S=0 — тень молчала полностью).
//   - Карта 1:1: C277→O187(RP_CHALLENGE_TASK) ×150; C62→пачка 45 кадров
//     (40×O18 + одиночные); C201→O28(1035Б); C211×2→[0,3] (авто-канон
//     «со 2-го вызова»); C179/C224 = fire-and-forget (0 кадров — канон).
//   - X-сигнатур в шапках НЕТ → replay без сессионных подстановок.
//   - 228 O-кадров / 338445Б полностью воспроизводимы по cseq (331 C-кадр).
package rpc

import (
	"embed"
	"encoding/json"
	"log"
	"sync"
)

//go:embed testdata/npcsrv_w30_20261023.bin
var npcW30FS embed.FS

//go:embed testdata/npcsrv_w30_20261023.json
var npcW30ManifestFS embed.FS

type npcW30Manifest struct {
	Source  string             `json:"source"`
	Frames  int                `json:"frames"`
	CSeq    []uint16           `json:"cseq"`
	Batches map[string][][]int `json:"batches"` // cmd -> occurrence -> индексы кадров
}

// npcW30 — общий state (read-only после init): кадры + батчи per-cmd.
var npcW30 = struct {
	sync.Once
	frames  [][]byte
	batches map[uint16][][][]byte // cmd -> occurrence -> кадры батча
}{}

func npcW30Init() {
	npcW30.Do(func() {
		raw, err := npcW30FS.ReadFile("testdata/npcsrv_w30_20261023.bin")
		if err != nil {
			log.Printf("[cached] npcsrv W30 bin не найден: %v", err)
			return
		}
		mb, err := npcW30ManifestFS.ReadFile("testdata/npcsrv_w30_20261023.json")
		if err != nil {
			log.Printf("[cached] npcsrv W30 manifest не найден: %v", err)
			return
		}
		var man npcW30Manifest
		if err := json.Unmarshal(mb, &man); err != nil {
			log.Printf("[cached] npcsrv W30 manifest parse: %v", err)
			return
		}
		all := SplitWorldFlowFrames(raw)
		npcW30.frames = all
		npcW30.batches = map[uint16][][][]byte{}
		for cs, occs := range man.Batches {
			var cmd uint16
			if _, err := fmtSscanf(cs, &cmd); err != nil {
				continue
			}
			var frames [][][]byte // occurrence → кадры
			for _, idxs := range occs {
				var batch [][]byte
				for _, i := range idxs {
					if int(i) < len(all) {
						batch = append(batch, all[i])
					}
				}
				frames = append(frames, batch)
			}
			npcW30.batches[cmd] = frames
		}
	})
}

// EmitNpcSrvW30 — батч ответов W30 по (cmd, occurrence). nil = канон-тишина
// (fire-and-forget либо occurrence исчерпан).
func EmitNpcSrvW30(cmd uint16, occurrence int) []byte {
	npcW30Init()
	occs, ok := npcW30.batches[cmd]
	if !ok {
		return nil
	}
	if occurrence >= len(occs) {
		return nil
	}
	var out []byte
	for _, f := range occs[occurrence] {
		out = append(out, f...)
	}
	return out
}

// NpcW30BatchCount — число captured-occurrence для cmd (для тестов/диагностики).
func NpcW30BatchCount(cmd uint16) int {
	npcW30Init()
	return len(npcW30.batches[cmd])
}

// fmtSscanf — парсинг десятичного ключа manifest в uint16.
func fmtSscanf(s string, out *uint16) (int, error) {
	v := 0
	if s == "" {
		return 0, errEmptyKey
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, errEmptyKey
		}
		v = v*10 + int(s[i]-'0')
		if v > 65535 {
			return 0, errEmptyKey
		}
	}
	*out = uint16(v)
	return 1, nil
}

type smallErr string

func (e smallErr) Error() string { return string(e) }

const errEmptyKey = smallErr("bad manifest key")

// GetNpcW30Blob — сырой W30-блоб (228 кадров конкатенацией).
func GetNpcW30Blob() []byte {
	npcW30Init()
	raw, _ := npcW30FS.ReadFile("testdata/npcsrv_w30_20261023.bin")
	return raw
}

// NpcW30CSeq — порядок C-кадров W30 (для offline-replay теста).
func NpcW30CSeq() []uint16 {
	npcW30Init()
	mb, err := npcW30ManifestFS.ReadFile("testdata/npcsrv_w30_20261023.json")
	if err != nil {
		return nil
	}
	var man npcW30Manifest
	if err := json.Unmarshal(mb, &man); err != nil {
		return nil
	}
	return man.CSeq
}

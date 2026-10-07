// Package tap — R4 fork-режим (FORK-SPEC): тень разбирает КОПИЮ трафика живого мира.
// Вход: hex-дампы направлений из tshark follow (см. tools/analysis/cap7777_decrypt.py).
// Логика: открытый SM_KEY первым фреймом S2C → восстановление ключей → дешифровка потока
// → сверка каждого пакета с ops.yaml → VERDICT (SAME=в реестре, DIFF/UNKNOWN=нет).
// Ориг при этом НЕ трогается: тень пассивный наблюдатель.
package tap

import (
	"encoding/binary"
	"fmt"
	"strings"

	"aion-main/internal/crypt"
	"aion-main/internal/ops"
	"aion-main/internal/wire"
)

// Stat — сводка tap-прогона.
type Stat struct {
	S2CFrames, C2SFrames   int
	S2CKnown, S2CUnknown   int
	C2SKnown, C2SUnknown   int
	Invalid                int
	UnknownS2C, UnknownC2S []string // опкоды вне реестра
	TopS2C, TopC2S         []string // "0xXXXX name xN"
}

func hex16(v uint16) string { return fmt.Sprintf("0x%04X", v) }

// Run — прогон tap по потокам; печатает вердикты, возвращает сводку.
func Run(s2c, c2s []byte, reg *ops.Registry, logf func(string, ...any)) *Stat {
	st := &Stat{}

	// --- S2C: первый фрейм = открытый SM_KEY ---
	sf, tail := wire.Split(s2c)
	if len(sf) == 0 {
		logf("tap: S2C пуст")
		return st
	}
	fk := binary.LittleEndian.Uint32(sf[0][5:9])
	base := crypt.BaseFromFalseKey(fk)
	srvKey := crypt.NewKeyPair(base)
	cnt := map[string]int{}
	for idx, body := range sf {
		if idx == 0 { // SM_KEY открытый
			op, _, ok := wire.ParseServerBody(body)
			if ok {
				cnt[hex16(op)]++
				st.S2CKnown++
			}
			continue
		}
		dec, ok := srvKey.Decrypt(body, crypt.S2C)
		if !ok {
			st.Invalid++
			continue
		}
		op, _, ok := wire.ParseServerBody(dec)
		if !ok {
			st.Invalid++
			continue
		}
		name := "UNKNOWN"
		if p, found := reg.Lookup(op, "SM"); found {
			name = p.Name
			st.S2CKnown++
		} else {
			name = "DIFF_" + hex16(op)
			st.S2CUnknown++
			st.UnknownS2C = append(st.UnknownS2C, hex16(op))
		}
		cnt[name+" "+hex16(op)]++
		st.S2CFrames++
	}
	for k, v := range cnt {
		if v >= 10 {
			st.TopS2C = append(st.TopS2C, fmt.Sprintf("%s x%d", k, v))
		}
	}

	// --- C2S: клиентский ключ (независимая копия, канон keys[CLIENT]) ---
	clKey := crypt.NewKeyPair(base)
	cf, tail2 := wire.Split(c2s)
	cntc := map[string]int{}
	for _, body := range cf {
		dec, ok := clKey.Decrypt(body, crypt.C2S)
		if !ok {
			st.Invalid++
			continue
		}
		op, _, ok := wire.ParseClientBody(dec)
		if !ok {
			st.Invalid++
			continue
		}
		name := "UNKNOWN"
		if p, found := reg.Lookup(op, "CM"); found {
			name = p.Name
			st.C2SKnown++
		} else {
			name = "DIFF_" + hex16(op)
			st.C2SUnknown++
			st.UnknownC2S = append(st.UnknownC2S, hex16(op))
		}
		cntc[name+" "+hex16(op)]++
		st.C2SFrames++
	}
	_ = tail
	_ = tail2
	for k, v := range cntc {
		if v >= 5 {
			st.TopC2S = append(st.TopC2S, fmt.Sprintf("%s x%d", k, v))
		}
	}

	logf("[TAP] S2C: %d кадров (known=%d unknown=%d) | C2S: %d (known=%d unknown=%d) | invalid=%d",
		st.S2CFrames, st.S2CKnown, st.S2CUnknown, st.C2SFrames, st.C2SKnown, st.C2SUnknown, st.Invalid)
	if len(st.UnknownS2C) > 0 {
		logf("[TAP] S2C вне реестра: %s", strings.Join(dedup(st.UnknownS2C), ","))
	}
	if len(st.UnknownC2S) > 0 {
		logf("[TAP] C2S вне реестра: %s", strings.Join(dedup(st.UnknownC2S), ","))
	}
	for _, s := range st.TopS2C {
		logf("[TAP-S2C] %s", s)
	}
	for _, s := range st.TopC2S {
		logf("[TAP-C2S] %s", s)
	}
	return st
}

func dedup(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

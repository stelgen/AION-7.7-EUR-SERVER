// Package records — разбор type-5 (статус-запись, body 194 const).
// Разметка по живому capture 05.10 (317 сэмплов: svc 301/302/309 — по тагу на клиента)
// + инварианты в тестах (testdata/status_records.txt). Маппинг в TBL_* — DB-слой (дальше).
package records

import (
	"encoding/binary"
	"fmt"
	"math"
)

// Len — размер тела status-записи.
const Len = 194

// StatusRecord — разобранная запись (поле = raw u32/float + имя по догадке только там,
// где capture подтверждает семантику).
type StatusRecord struct {
	SvcType  uint32 // @0: capture = 301/302/309 (таг клиента/сервиса)
	Metric1  uint32 // @4: ~13 значений (счётчик/объём), не mono
	TickRaw  uint64 // @8: движковое/системное время (НЕ mono в окне — семантика уточняется)
	Metric2  uint32 // @16
	Metric3  uint32 // @20
	Metric4  uint32 // @24 (биты 0x8000 встречаются)
	WorldID  uint32 // @28: 210010000 = мир; бит 0x80000000 = флаг записи
	X        float32
	Y        float32
	Z        float32
	Fields   [134]byte // @44..178: флаги 0xFFFFFFFF80000000-блоки, счётчики (raw)
	EngineMs uint32    // @188: ms-счётчик, МАНОТОНЕН в рамках клиента (тесты capture)
	SysTime  [8]uint16 // @178: SYSTEMTIME (8 WORD: год,мес,dow,день,ч,м,с,мс)
}

// ParseStatus — тело 194 → запись.
func ParseStatus(body []byte) (*StatusRecord, error) {
	if len(body) != Len {
		return nil, fmt.Errorf("records: body=%d, want %d", len(body), Len)
	}
	r := &StatusRecord{}
	r.SvcType = binary.LittleEndian.Uint32(body[0:4])
	r.Metric1 = binary.LittleEndian.Uint32(body[4:8])
	r.TickRaw = binary.LittleEndian.Uint64(body[8:16])
	r.Metric2 = binary.LittleEndian.Uint32(body[16:20])
	r.Metric3 = binary.LittleEndian.Uint32(body[20:24])
	r.Metric4 = binary.LittleEndian.Uint32(body[24:28])
	r.WorldID = binary.LittleEndian.Uint32(body[28:32])
	r.X = floatFrom(body[32:36])
	r.Y = floatFrom(body[36:40])
	r.Z = floatFrom(body[40:44])
	copy(r.Fields[:], body[44:178])
	for i := 0; i < 8; i++ {
		r.SysTime[i] = binary.LittleEndian.Uint16(body[178+i*2 : 180+i*2])
	}
	r.EngineMs = binary.LittleEndian.Uint32(body[188:192])
	return r, nil
}

// floatFrom — РЕИНТЕРПРЕТ бит в float32 (было int→float — координаты показывались мусором,
// найдено при подмене 05.10: pos=(1146813568.0) вместо ~887.0).
func floatFrom(b []byte) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(b))
}

// SysTimeString — "2026-10-05 09:15:49.696".
func (r *StatusRecord) SysTimeString() string {
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d.%03d",
		r.SysTime[0], r.SysTime[1], r.SysTime[3],
		r.SysTime[4], r.SysTime[5], r.SysTime[6], r.SysTime[7])
}

// World — мир без флаговых битов.
func (r *StatusRecord) World() uint32 { return r.WorldID &^ 0x80000000 }

// String — CSV-строка для per-day status-файла.
func (r *StatusRecord) String() string {
	return fmt.Sprintf("%d,%d,%d,%d,%d,%d,%d,%.3f,%.3f,%.3f,%d,%s",
		r.SvcType, r.Metric1, r.TickRaw, r.Metric2, r.Metric3, r.Metric4,
		r.World(), r.X, r.Y, r.Z, r.EngineMs, r.SysTimeString())
}

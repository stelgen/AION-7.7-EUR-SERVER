#!/usr/bin/env python3
"""patch_test_relay.py — TestRelayGQHandshakeE2E → relay-wire кадры (дизasm-канон 23.10)."""
import re

p = "internal/server/admin_e2e_test.go"
src = open(p, "rb").read().decode("utf-8")

old = """// e2e: GQ/GP MVP — NPRelay64 handshake GQ_LOGON(2) переключает conn в relay-режим,
// далее cmd=2 трактуется как GQ (silent, имя из .profile idx 6), без ответа.
func TestRelayGQHandshakeE2E(t *testing.T) {"""
new = """// e2e: relay-wire MVP (дизasm-канон 23.10) — NPRelay кадр [u16 total][type u8][0xBE]
// переключает conn в relay-режим, type 1-5 = silent (паритет NoPacket), без ответа.
func TestRelayGQHandshakeE2E(t *testing.T) {"""
assert old in src
src = src.replace(old, new, 1)

old2 = """\t// 1) handshake: GQ_LOGON(2)
\tif _, err := c.Write(buildTestFrame(2, make([]byte, 8))); err != nil {
\t\tt.Fatal(err)
\t}
\texpectSilence(t, c)

\t// 2) в relay-режиме cmd=2 = GQ_LOGON-семейство (silent, conn жив)
\tif _, err := c.Write(buildTestFrame(2, make([]byte, 8))); err != nil {
\t\tt.Fatalf("conn closed in relay mode: %v", err)
\t}
\texpectSilence(t, c)
}"""
new2 = """\t// 1) handshake: relay-wire type=2 (GQ_LOGON)
\tif _, err := c.Write(relayTestFrame(2, make([]byte, 8))); err != nil {
\t\tt.Fatal(err)
\t}
\texpectSilence(t, c)

\t// 2) relay-режим: type=2 silent (паритет NoPacket), conn жив
\tif _, err := c.Write(relayTestFrame(2, make([]byte, 8))); err != nil {
\t\tt.Fatalf("conn closed in relay mode: %v", err)
\t}
\texpectSilence(t, c)

\t// 3) type=3 (GQ_ITEMADD) — silent, conn жив; type=6 НЕ relay (аккache-кадр)
\tif _, err := c.Write(relayTestFrame(3, []byte{1, 2, 3, 4})); err != nil {
\t\tt.Fatalf("conn closed after ITEMADD: %v", err)
\t}
\texpectSilence(t, c)
}

// relayTestFrame — relay-wire кадр по дизasm-канону 23.10:
// [u16 total LE][type u8][0xBE][payload]; total = полная длина кадра.
func relayTestFrame(typ byte, payload []byte) []byte {
\ttotal := 4 + len(payload)
\tout := make([]byte, total)
\tbinary.LittleEndian.PutUint16(out, uint16(total))
\tout[2] = typ
\tout[3] = 0xBE
\tcopy(out[4:], payload)
\treturn out
}"""
assert old2 in src, "old2 not found"
src = src.replace(old2, new2, 1)

open(p, "wb").write(src.encode("utf-8"))
print("test patched OK")

#!/usr/bin/env python3
"""patch_server_relay.py — вставка relay pre-scan в handleConn + удаление старого handshake-блока."""
import re, sys

p = "internal/server/server.go"
src = open(p, "rb").read().decode("utf-8")

# 1) цикл: for len(pending) >= wire.HeaderWire { + Feed  →  for { + pre-scan + guard + Feed
old_head = "\t\tfor len(pending) >= wire.HeaderWire {\n\t\t\tconsumed, frame, ferr := dec.Feed(pending)\n"
new_head = (
    "\t\tfor {\n"
    "\t\t\t// R3 GQ/GP (дизasm-канон 23.10): relay-кадр = [u16 total][type u8≤5][0xBE]...\n"
    "\t\t\t// — короче HeaderWire и без 0xEB-маркера; Feed reject'нул бы его\n"
    "\t\t\t// (bad size/marker) и закрыл conn ⇒ детектим на СЫРОМ буфере до Feed.\n"
    "\t\t\tif ch == ChanServer && len(pending) >= 4 {\n"
    "\t\t\t\tif typ, total, ok := relayFrame(pending); ok {\n"
    "\t\t\t\t\tif len(pending) < total {\n"
    "\t\t\t\t\t\tbreak // ждём ещё байт\n"
    "\t\t\t\t\t}\n"
    "\t\t\t\t\tpayload := append([]byte(nil), pending[4:total]...)\n"
    "\t\t\t\t\tpending = pending[total:]\n"
    "\t\t\t\t\t_ = relayWireDispatch(&mode, typ, payload)\n"
    "\t\t\t\t\tcontinue\n"
    "\t\t\t\t}\n"
    "\t\t\t}\n"
    "\t\t\tif len(pending) < wire.HeaderWire {\n"
    "\t\t\t\tbreak\n"
    "\t\t\t}\n"
    "\t\t\tconsumed, frame, ferr := dec.Feed(pending)\n"
)
assert old_head in src, "old_head not found"
src = src.replace(old_head, new_head, 1)

# 2) удалить старый handshake-блок
pat = re.compile(
    r"\n\t\t\t// R3 GQ/GP: NPRelay64 handshake[^\n]*\n"
    r"\t\t\tif ch == ChanServer && maybeRelayHandshake\(&mode, frame\.Cmd\) \{\n"
    r"\t\t\t\tname := rpc\.Name\(6, frame\.Cmd\)[^\n]*\n"
    r"\t\t\t\t_ = relayGQDispatch\(&mode, ch, frame\.Cmd, frame\.Payload, name\)\n"
    r"\t\t\t\tcontinue[^\n]*\n"
    r"\t\t\t\}\n"
)
src, n = pat.subn("\n", src)
assert n == 1, f"handshake block replaced {n} times"

open(p, "wb").write(src.encode("utf-8"))
print("patched OK")

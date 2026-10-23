#!/usr/bin/env python3
"""patch_server_worldflow_v2.py — sess-guard + отключение WorldInitReplay fallback."""
import re

p = "internal/server/server.go"
src = open(p, "rb").read().decode("utf-8")

# 1) case 11: добавить sessSet-guard (emit только если session захвачена)
m11 = re.search(r'(\t+)if !emittedWorldInit \{\n', src)
assert m11, "world-init branch not found"
ins = (
    m11.group(1) + "\tif !sessSet {\n"
    + m11.group(1) + "\t\t// R5-урок 23.10: тень поднята в середине сессии — C288 не приходил,\n"
    + m11.group(1) + "\t\t// sess.X=0 → подстановки нулями. Честный A/B только при перелогине.\n"
    + m11.group(1) + "\t\tlog.Printf(\"[cached] %s RQ_LOAD БЕЗ session — skip world-init (нужен перелогин)\", name)\n"
    + m11.group(1) + "\t} else " + m11.group(1).rstrip('\t') + "if !emittedWorldInit {\n"
)
# проще: заменить 'if !emittedWorldInit {' → 'if sessSet && !emittedWorldInit {'
src = src.replace('if !emittedWorldInit {', 'if sessSet && !emittedWorldInit {', 1)

# 2) WorldInitReplay fallback → отключить (лог вместо эмиссии)
old_fb = """if worldInitReplay == nil {
\t\t\t\t\t\t\tworldInitReplay = rpc.NewWorldInitReplay()
\t\t\t\t\t\t}
\t\t\t\t\t\tif r := worldInitReplay.EmitWorldInit(frame.Cmd); len(r) > 0 {
\t\t\t\t\t\t\tresp = append(resp, r...)
\t\t\t\t\t\t\tlog.Printf("[cached] %s world-init cmd=%d → replay %dB", name, frame.Cmd, len(r))
\t\t\t\t\t\t}"""
new_fb = """log.Printf("[cached] %s unhandled cmd=%d (WorldInitReplay fallback DISABLED при активном world-flow, R5)", name, frame.Cmd)"""
# normalize whitespace
pat = re.compile(r'if worldInitReplay == nil \{[\s\S]*?EmitWorldInit\(frame\.Cmd\); len\(r\) > 0 \{[\s\S]*?\}', re.M)
src, n = pat.subn(new_fb.replace('\\', '\\\\'), src, 1)
assert n == 1, f"fallback replaced {n}"

open(p, "wb").write(src.encode("utf-8"))
print("patched v2: sessSet-guard@11 + WorldInitReplay fallback disabled")

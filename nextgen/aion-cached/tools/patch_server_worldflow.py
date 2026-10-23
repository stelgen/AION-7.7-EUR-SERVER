#!/usr/bin/env python3
"""patch_server_worldflow.py v3 — R5 мир-init в server.go (чистая версия)."""
import re

p = "internal/server/server.go"
src = open(p, "rb").read().decode("utf-8")
assert 'EmitWorldFlowPreload' not in src, "уже пропатчен"

# 1) preload после sessSet-блока
anchor1 = 'sessSet = true'
idx = src.find(anchor1)
assert idx > 0, "anchor1 not found"
log_end = src.find('\n', src.find('log.Printf("[cached] session captured', idx))
close_brace = src.find('\n\t\t\t\t\t}', log_end)
assert close_brace > 0, "closing brace not found"
insert_at = close_brace + len('\n\t\t\t\t\t}')
insert1 = (
    '\n\t\t\t\t\t\t// R5: preload мир-init (O196 ack + O90 CharacterList) сразу после session-open\n'
    '\t\t\t\t\t\tif pre := rpc.EmitWorldFlowPreload(sess.X, sess.A); len(pre) > 0 {\n'
    '\t\t\t\t\t\t\tresp = pre\n'
    '\t\t\t\t\t\t\temittedWorldInit = false\n'
    '\t\t\t\t\t\t\tlog.Printf("[cached] %s session 288 → world-flow preload %d bytes (charlist+ack)", name, len(pre))\n'
    '\t\t\t\t\t\t}'
)
src = src[:insert_at] + insert1 + src[insert_at:]

# 2) case 11 → заменить блок до case 14
m11 = re.search(r'(\t+)case 11:', src)
start = m11.start()
i14 = src.find('case 14:', start)
assert i14 > 0
new_block = (
    "case 11: // R5: RQ_LOAD → ПОЛНЫЙ мир-init load-сегмент (118 кадров capture-20261023)\n"
    "\t\t\t\t\tif !emittedWorldInit {\n"
    "\t\t\t\t\t\tloadChain := rpc.EmitWorldFlowLoad(sess.X, sess.A)\n"
    "\t\t\t\t\t\tif len(loadChain) > 0 {\n"
    "\t\t\t\t\t\t\tresp = append(resp, loadChain...)\n"
    "\t\t\t\t\t\t\temittedWorldInit = true\n"
    "\t\t\t\t\t\t}\n"
    "\t\t\t\t\t\tlog.Printf(\"[cached] %s RQ_LOAD → world-flow load %d bytes (session-applied, R5)\", name, len(loadChain))\n"
    "\t\t\t\t\t} else {\n"
    "\t\t\t\t\t\tloadChain := rpc.ApplySessionToChain(rpc.GetLoadChain(), sess)\n"
    "\t\t\t\t\t\tif len(loadChain) > 0 {\n"
    "\t\t\t\t\t\t\tresp = append(resp, loadChain...)\n"
    "\t\t\t\t\t\t}\n"
    "\t\t\t\t\t\tlog.Printf(\"[cached] %s RQ_LOAD → chain replay %d bytes (session-applied, fallback)\", name, len(loadChain))\n"
    "\t\t\t\t\t}\n"
    "\t\t\t\t\t"
)
src = src[:start] + new_block + src[i14:]

# 3) case 14 → обернуть тело в if !emittedWorldInit { ... }
m14 = re.search(r'(\t+)case 14:([^\n]*)\n', src)
indent = m14.group(1)
c14_start = m14.start()
body_start = m14.end()
i201 = src.find('case 201:', body_start)
assert i201 > 0
body_end = src.rfind('\n', 0, i201)  # конец строки перед case 201
body = src[body_start:body_end+1]
# dedent-не нужен: обёртка добавляет уровень, но Go не требует выравнивания
guarded = (
    src[c14_start:body_start]
    + indent + "\tif emittedWorldInit {\n"
    + indent + "\t\tlog.Printf(\"[cached] %s RQ_LOAD_ITEM skipped (world-init already emitted)\", name)\n"
    + indent + "\t} else {\n"
    + body.rstrip('\n') + "\n"
    + indent + "\t}\n"
)
src = src[:c14_start] + guarded + src[body_end+1:]

# 4) объявить emittedWorldInit рядом с worldInitReplay
anchor3 = "var worldInitReplay *rpc.WorldInitReplay"
idx3 = src.find(anchor3)
assert idx3 > 0
line_end = src.find('\n', idx3)
src = src[:line_end] + "\n\temittedWorldInit := false // R5: мир-init отработан в этом коннекте" + src[line_end:]

open(p, "wb").write(src.encode("utf-8"))
print("patched v3 OK: preload@288 + world-flow@11 + guard@14 + flag")

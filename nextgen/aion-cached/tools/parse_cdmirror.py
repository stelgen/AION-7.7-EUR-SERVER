#!/usr/bin/env python3
"""parse_cdmirror.py — парсер capture-лога accmirror v1.3 (cdmirror :2006).

Цель R3-хвосты: инвентарь GQ/GP (NPRelay-канал) по коннектам.
Формат строк:
  ... === CLIENT <addr>            — новый коннект
  ... C> frame len=N cmd=0xXXX(~0xYYY marker=0xA? ok=?) payload=M hex=...
  ... O> frame ...                 — ответ ориг-CacheD (копия-2016)
  ... S> ...                       — события тени/служебные
Wire (accache-канон): [len u16 total][cmd u16 LE][marker AE C2S/AF S2C][~cmd u16][payload]
"""
import re, sys, json, collections

LOG = sys.argv[1] if len(sys.argv) > 1 else "cdmirror-20261022-login.log"

CLIENT_RE = re.compile(r"^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}) === CLIENT (\S+)")
FRAME_RE = re.compile(
    r"^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}) ([CO])> frame len=(\d+) cmd=0x([0-9A-Fa-f]+)"
    r"\(~0x([0-9A-Fa-f]+) marker=0x([0-9A-Fa-f]+) ok=(\w+)\) payload=(\d+) hex=([0-9a-f]*)"
)

conns = []  # list of dict(addr, frames=[(ts,dir,cmd,inv,payload_hex)])
cur = None
for line in open(LOG, encoding="utf-8", errors="replace"):
    line = line.rstrip("\r\n")
    m = CLIENT_RE.match(line)
    if m:
        cur = {"addr": m.group(2), "t0": m.group(1), "frames": []}
        conns.append(cur)
        continue
    m = FRAME_RE.match(line)
    if m and cur is not None:
        ts, d, ln, cmd, inv, mk, ok, pl, hx = m.groups()
        cur["frames"].append({
            "ts": ts, "dir": d, "len": int(ln), "cmd": int(cmd, 16),
            "inv": int(inv, 16), "marker": mk, "ok": ok,
            "plen": int(pl), "hex": hx,
        })

print(f"connections={len(conns)} frames={sum(len(c['frames']) for c in conns)}")

# 1) первый cmd каждого коннекта (эвристика режима: RQ_VERSION=215 / GQ_LOGON=2)
first = collections.Counter()
for c in conns:
    cs = [f for f in c["frames"] if f["dir"] == "C"]
    first[cs[0]["cmd"] if cs else -1] += 1
print("first-C cmd histogram:", dict(sorted(first.items())))

# 2) коннекты, начинающиеся с 1/2 (кандидаты NPRelay: GQ_KICKOFF=1/GQ_LOGON=2)
relay = [c for c in conns if [f for f in c["frames"] if f["dir"] == "C"] and
         [f for f in c["frames"] if f["dir"] == "C"][0]["cmd"] in (1, 2)]
print(f"relay-candidate conns={len(relay)}")

# 3) инвентарь GQ (C>) и GP (O>) по relay-коннектам
gq = collections.Counter(); gp = collections.Counter()
gq_pay = collections.defaultdict(list); gp_pay = collections.defaultdict(list)
for c in relay:
    for f in c["frames"]:
        if f["dir"] == "C":
            gq[f["cmd"]] += 1
            gq_pay[f["cmd"]].append(f["hex"])
        else:
            gp[f["cmd"]] += 1
            gp_pay[f["cmd"]].append(f["hex"])

print("GQ ops:", {f"0x{k:04X}({k})": v for k, v in sorted(gq.items())})
print("GP ops:", {f"0x{k:04X}({k})": v for k, v in sorted(gp.items())})

# 4) дедуп payload'ов per-op
def uniq(pays):
    seen, out = set(), []
    for h in pays:
        if h not in seen:
            seen.add(h); out.append(h)
    return out

report = {"conns": [], "gq": {}, "gp": {}}
for i, c in enumerate(conns):
    cs = [f["cmd"] for f in c["frames"] if f["dir"] == "C"]
    report["conns"].append({
        "idx": i, "addr": c["addr"], "t0": c["t0"], "n": len(c["frames"]),
        "firstC": cs[0] if cs else None,
        "cmdsC": sorted(set(cs)),
        "cmdsO": sorted(set(f["cmd"] for f in c["frames"] if f["dir"] == "O")),
    })
for op, pays in sorted(gq_pay.items()):
    report["gq"][str(op)] = {"count": gq[op], "uniq": uniq(pays)}
for op, pays in sorted(gp_pay.items()):
    report["gp"][str(op)] = {"count": gp[op], "uniq": uniq(pays)}

json.dump(report, open("cdmirror_gqgp_report.json", "w"), indent=1)
print("written cdmirror_gqgp_report.json")

# 5) краткая таблица GQ: op, count, sizes, первый payload (обрезано)
print("\n== GQ per-op ==")
for op, pays in sorted(gq_pay.items()):
    sizes = sorted(set(len(h) // 2 for h in pays))
    print(f"op=0x{op:04X}({op:3d}) n={gq[op]:3d} uniq={len(uniq(pays)):3d} sizes={sizes}")
print("\n== GP per-op ==")
for op, pays in sorted(gp_pay.items()):
    sizes = sorted(set(len(h) // 2 for h in pays))
    print(f"op=0x{op:04X}({op:3d}) n={gp[op]:3d} uniq={len(uniq(pays)):3d} sizes={sizes}")

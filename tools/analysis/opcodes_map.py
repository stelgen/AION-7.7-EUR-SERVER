#!/usr/bin/env python3
"""opcodes_map.py — единая таблица опкодов Aion Game-протокола из 5 реестров:
  xml75   : Packet Samurai Game_7.5.x.xml (имя<->opcode)
  encom75 : encom-leak-7577 GameServer (AionPacketHandlerFactory CM + ServerPacketsOpcodes SM)
  mobius77: Mobius_AionEmu 7.7 (там же)
  ag78    : aion-germany AL-Game 7.8 (там же)
  ag58    : aion-germany AL-Game-5.8 (там же)
Выход: CSV (имя,dir,опкоды по источникам,теги версий,согласие) + сводка.
"""
import re, sys, csv, json
from collections import defaultdict

REF = "/home/user/STELGEN/projects/aion_server_2026-10-02/reference"
XML75 = REF + "/encom-leak-7577/tools/Packet Samurai/dist/protocols/Game_7.5.x.xml"
JAVAS = {
    "encom75": REF + "/encom-leak-7577/GameServer/src/com/aionemu/gameserver/network",
    "mobius77": REF + "/Mobius_AionEmu/java/com/aionemu/gameserver/network",
    "ag78": REF + "/aion-germany/AL-Game/src/com/aionemu/gameserver/network",
    "ag58": REF + "/aion-germany/AL-Game-5.8/src/com/aionemu/gameserver/network",
}

RX_XML = re.compile(r'<packet id="(0x[0-9A-Fa-f]+)"\s+name="(\w+)"')
RX_SM = re.compile(r'addPacketOpcode\((SM_\w+)\.class,\s*(0x[0-9A-Fa-f]+)\w*,\s*idSet\)\s*;\s*(//.*)?')
RX_CM = re.compile(r'addPacket\(new (CM_\w+)\((0x[0-9A-Fa-f]+)\w*,[^;]*\)\s*;\s*(//.*)?')

table = {}  # name -> {"dir":..., src:{src:op}, tags:{src:[tags]}}

def put(name, op, src, tag=""):
    d = "SM" if name.startswith("SM_") else ("CM" if name.startswith("CM_") else "?")
    row = table.setdefault(name, {"dir": d, "src": {}, "tags": {}})
    row["src"][src] = op
    if tag and src not in row["tags"]:
        row["tags"][src] = tag.strip("// ").strip()

def read(p):
    try:
        return open(p, encoding="utf-8", errors="replace").read()
    except OSError:
        return ""

# 1) XML 7.5
seen = set()
for op, name in RX_XML.findall(read(XML75)):
    if name not in seen:
        seen.add(name)
        put(name, int(op, 16), "xml75")

# 2..5) Java реестры
for src, netdir in JAVAS.items():
    txt_sm = read(netdir + "/aion/ServerPacketsOpcodes.java")
    for name, op, tail in RX_SM.findall(txt_sm):
        put(name, int(op, 16), src, tail or "")
    txt_cm = read(netdir + "/factories/AionPacketHandlerFactory.java")
    for name, op, tail in RX_CM.findall(txt_cm):
        put(name, int(op, 16), src, tail or "")

# 3) Сводка/согласие
SRC_ORDER = ["xml75", "encom75", "mobius77", "ag78", "ag58"]
def fmt_op(v):
    return f"0x{v:04X}" if v is not None else ""
rows = []
stats = defaultdict(int)
for name in sorted(table):
    r = table[name]
    srcs = [r["src"].get(s) for s in SRC_ORDER]
    present = [v for v in srcs if v is not None]
    distinct = len(set(present))
    agree = ("NA" if len(present) < 2 else ("FULL" if distinct == 1 else "DIFF"))
    stats[agree] += 1
    stats["n_" + str(len(present))] += 1
    tags = "|".join(f"{s}:{r['tags'][s]}" for s in SRC_ORDER if s in r["tags"])
    rows.append({
        "name": name, "dir": r["dir"],
        **{s: fmt_op(v) for s, v in zip(SRC_ORDER, srcs)},
        "agree": agree, "distinct": distinct, "tags": tags,
    })

out_csv = sys.argv[1] if len(sys.argv) > 1 else "/tmp/opcodes.csv"
with open(out_csv, "w", newline="") as f:
    w = csv.DictWriter(f, fieldnames=list(rows[0].keys()))
    w.writeheader()
    w.writerows(rows)

summary = {
    "total_names": len(table),
    "by_agree": dict(stats),
    "dirs": {
        "SM": sum(1 for r in rows if r["dir"] == "SM"),
        "CM": sum(1 for r in rows if r["dir"] == "CM"),
    },
    "src_counts": {s: sum(1 for r in rows if r[s] != "") for s in SRC_ORDER},
}
print(json.dumps(summary, indent=1))
full = [r for r in rows if r["agree"] == "FULL"]
diff = [r for r in rows if r["agree"] == "DIFF"]
print("FULL sample:", [(r["name"], r["xml75"], r["mobius77"], r["ag78"]) for r in full[:8]])
print("DIFF sample:", [(r["name"], r["xml75"], r["encom75"], r["mobius77"], r["ag78"]) for r in diff[:10]])
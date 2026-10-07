#!/usr/bin/env python3
"""R0 aion-npc: эталонный словарь AI-типов по 4 Java-клонам Aion-эмуляторов.

Вытаскивает: @AIName-реестр (имя AI -> класс), частоты ai="..." из npc_templates.xml,
наличие think/walker атрибутов. Запуск из корня репо: python3 tools/extract_ai_dict.py > /tmp/ai_dict.json
Клоны ждёт в ../../../reference/ относительно nextgen/aion-npc (вне гита).
"""
import re, os, collections, json

HERE = os.path.dirname(os.path.abspath(__file__))
R = os.path.normpath(os.path.join(HERE, "..", "..", "..", "..", "reference"))
CLONES = [
    ("7.7-Mobius", "Mobius_AionEmu/dist/game/data/scripts/system/handlers/ai",
     "Mobius_AionEmu/dist/game/data/static_data/npcs/npc_templates.xml"),
    ("7.8-AL", "AionLightning/AL-Game/data/scripts/system/handlers/ai",
     "AionLightning/AL-Game/data/static_data/npcs/npc_templates.xml"),
    ("7.8-AG", "aion-germany/AL-Game/data/scripts/system/handlers/ai",
     "aion-germany/AL-Game/data/static_data/npcs/npc_templates.xml"),
    ("5.8-AG", "aion-germany/AL-Game-5.8/data/scripts/system/handlers/ai",
     "aion-germany/AL-Game-5.8/data/static_data/npcs/npc_templates.xml"),
    ("4.8-BA", "aion-server/game-server/data/handlers/ai",
     "aion-server/game-server/data/static_data/npcs/npc_templates.xml"),
]

ann_re = re.compile(r'@AIName\("([^"]+)"\)')

def registry_of(path):
    """ai-name -> class: @AIName(...) затем первый class-декл на не-коммент-строке после неё."""
    reg = {}
    for root, _, files in os.walk(path):
        for f in files:
            if not f.endswith(".java"):
                continue
            lines = open(os.path.join(root, f), encoding="utf-8", errors="ignore").read().splitlines()
            for i, ln in enumerate(lines):
                m = ann_re.search(ln)
                if not m:
                    continue
                cls = None
                for ln2 in lines[i:i + 15]:
                    s = ln2.strip()
                    if s.startswith("*") or s.startswith("//"):
                        continue
                    mc = re.search(r'\bclass\s+(\w+)', ln2)
                    if mc:
                        cls = mc.group(1)
                        break
                if cls:
                    reg[m.group(1)] = cls
    return reg

report = {}
for ver, aidir, xml in CLONES:
    reg = registry_of(os.path.join(R, aidir))
    freq = collections.Counter()
    total = 0
    think_vals = collections.Counter()
    with open(os.path.join(R, xml), encoding="utf-8", errors="ignore") as fh:
        for line in fh:
            if "<npc_template " in line:
                total += 1
                m = re.search(r'\sai="([^"]*)"', line)
                if m:
                    freq[m.group(1)] += 1
    used, regnames = set(freq), set(reg)
    report[ver] = {
        "handlers_total": len(reg),
        "registry": reg,
        "npc_total": total,
        "ai_freq": dict(freq.most_common()),
        "ai_used_no_handler": sorted(used - regnames),
        "handlers_never_used": sorted(regnames - used),
    }

print(json.dumps(report, ensure_ascii=False, indent=1))

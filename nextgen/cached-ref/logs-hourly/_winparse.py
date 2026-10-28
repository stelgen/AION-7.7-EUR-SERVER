#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""ОФФЛАЙН-разбор capture-окон aion-cached: windows-analysis (28.10).
Разбор worldinit-20261023.txt + cdmirror-20261024-worldstart.log.
Без VM/кода/коммитов — только анализ."""
import re, json, collections, datetime, os

BASE = "/home/ot/STELGEN/projects/aion_server_2026-10-02/AION-7.7-EUR-SERVER/nextgen/cached-ref/logs-hourly"
F_WORLD = os.path.join(BASE, "worldinit-20261023.txt")
F_MIRROR = os.path.join(BASE, "cdmirror-20261024-worldstart.log")
OUT = os.path.join(BASE, "windows-analysis-20261028.md")

FRAME_RE = re.compile(
    r"^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})\s+(C>|O>)\s+frame len=(\d+) cmd=(0x[0-9A-Fa-f]+)\(([^)]*)\)\s+payload=(\d+)\s+hex=([0-9a-fA-F]*)")
WIN_RE = re.compile(r"^=== WORLDINIT_WINDOW (\d+) \(marker line (\d+), frames (\d+)\) ===")
CLIENT_RE = re.compile(r"^=== CLIENT (\S+)")
BACKUP_RE = re.compile(r"^BACK UP (\S+)")
STATS_RE = re.compile(r"^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}) STATS (.*)")

def parse_file(path, source):
    """Возвращает список событий в хронологическом порядке файла.
    Событие: dict(kind=frame|win|client|backup|stats, ...)"""
    events = []
    with open(path, encoding="utf-8", errors="replace") as f:
        for ln, line in enumerate(f, 1):
            line = line.rstrip("\n")
            if not line:
                continue
            m = WIN_RE.match(line)
            if m:
                events.append(dict(kind="win", n=int(m.group(1)), marker_line=int(m.group(2)),
                                   frames=int(m.group(3)), line=ln, source=source))
                continue
            m = CLIENT_RE.match(line)
            if m:
                events.append(dict(kind="client", addr=m.group(1), line=ln, source=source))
                continue
            m = BACKUP_RE.match(line)
            if m:
                events.append(dict(kind="backup", addr=m.group(1), line=ln, source=source))
                continue
            m = STATS_RE.match(line)
            if m:
                events.append(dict(kind="stats", ts=m.group(1), body=m.group(2), line=ln, source=source))
                continue
            m = FRAME_RE.match(line)
            if m:
                events.append(dict(kind="frame", ts=m.group(1), dir=m.group(2),
                                   length=int(m.group(3)), cmd=m.group(4).upper(),
                                   info=m.group(5), payload=int(m.group(6)),
                                   hex=m.group(7), line=ln, source=source))
                continue
            events.append(dict(kind="other", raw=line[:120], line=ln, source=source))
    return events

world = parse_file(F_WORLD, "worldinit")
mirror = parse_file(F_MIRROR, "cdmirror")

print(f"worldinit: {len(world)} events; cdmirror: {len(mirror)} events")
print("world kinds:", collections.Counter(e['kind'] for e in world))
print("mirror kinds:", collections.Counter(e['kind'] for e in mirror))

# --- Сегментация ---
segments = []  # list of dict(id, source, frames=[...])
def segmentize(events, source):
    segs = []
    cur = None
    seg_idx = 0
    for e in events:
        if e['kind'] == 'win':
            cur = dict(id=f"WIN{e['n']}", source=source, start_line=e['line'],
                       marker_line=e['marker_line'], declared_frames=e['frames'], frames=[])
            segs.append(cur)
        elif e['kind'] == 'client':
            seg_idx += 1
            cur = dict(id=f"CLIENT{seg_idx}@{e['addr']}", source=source, start_line=e['line'],
                       addr=e['addr'], declared_frames=None, frames=[])
            segs.append(cur)
        elif e['kind'] == 'frame':
            if cur is None:
                cur = dict(id=f"HEADER-{source}", source=source, start_line=1,
                           declared_frames=None, frames=[])
                segs.append(cur)
            cur['frames'].append(e)
    return segs

segs_world = segmentize(world, "worldinit")
segs_mirror = segmentize(mirror, "cdmirror")
all_segs = segs_world + segs_mirror

print(f"segments: worldinit={len(segs_world)} cdmirror={len(segs_mirror)}")

# --- Анализ каждого сегмента ---
def analyze_segment(seg):
    fr = seg['frames']
    res = dict(id=seg['id'], source=seg['source'], nframes=len(fr),
               first=None, firstC=None, firstO=None, push_on_accept=False,
               seq=[], pairs=collections.Counter(), o16_no_c215=False, o123_no_c215=False)
    firstC_idx = firstO_idx = None
    for i, e in enumerate(fr):
        if firstC_idx is None and e['dir'] == 'C>':
            firstC_idx = i
        if firstO_idx is None and e['dir'] == 'O>':
            firstO_idx = i
        res['seq'].append((e['dir'], e['cmd'], e['length']))
    if fr:
        res['first'] = (fr[0]['dir'], fr[0]['cmd'], fr[0]['length'], fr[0]['ts'])
    if firstC_idx is not None:
        res['firstC'] = (fr[firstC_idx]['cmd'], fr[firstC_idx]['length'], fr[firstC_idx]['ts'])
    if firstO_idx is not None:
        res['firstO'] = (fr[firstO_idx]['cmd'], fr[firstO_idx]['length'], fr[firstO_idx]['ts'])
        if firstC_idx is None or firstO_idx < firstC_idx:
            res['push_on_accept'] = True
    # пары: O -> ближайший предыдущий C в том же сегменте
    lastC = None
    c215_seen = False
    for e in fr:
        if e['dir'] == 'C>':
            lastC = e['cmd']
            if e['cmd'] == '0xD7':
                c215_seen = True
        else:
            if lastC:
                res['pairs'][(lastC, e['cmd'], e['length'])] += 1
            if e['cmd'] == '0x10' and not c215_seen:
                res['o16_no_c215'] = True
            if e['cmd'] == '0x7B' and not c215_seen:
                res['o123_no_c215'] = True
    return res

analyses = [analyze_segment(s) for s in all_segs]

# --- Первые 3 C-кадра каждого сегмента (фингерпринт) ---
def first3C(seg):
    out = []
    for e in seg['frames']:
        if e['dir'] == 'C>':
            out.append((e['cmd'], e['length']))
            if len(out) == 3:
                break
    return out

fingerprints = []
for seg in all_segs:
    f3 = first3C(seg)
    a = analyze_segment(seg)
    fingerprints.append(dict(id=seg['id'], source=seg['source'], addr=seg.get('addr'),
                             nframes=len(seg['frames']), first3C=f3,
                             first=a['first'], push=a['push_on_accept']))

print("\n=== ФИНГЕРПРИНТЫ СЕГМЕНТОВ ===")
for fp in fingerprints:
    print(f"{fp['id']:28s} nf={fp['nframes']:5d} first3C={fp['first3C']} push={fp['push']} first={fp['first'][:3] if fp['first'] else None}")

# --- Сводная таблица C->O пар по всем сегментам ---
pair_total = collections.Counter()
for a in analyses:
    pair_total.update(a['pairs'])
print("\n=== C->O ПАРЫ (по всем окнам) ===")
for (ccmd, ocmd, olen), cnt in sorted(pair_total.items(), key=lambda x: -x[1]):
    print(f"C{ccmd} -> O{ocmd} lenO={olen} x{cnt}")

# --- O16/O123 без C215 ---
flagged = [a for a in analyses if a['o16_no_c215'] or a['o123_no_c215']]
print(f"\nСегментов с O16/O123 БЕЗ предшествующего C215: {len(flagged)}")
for a in flagged[:10]:
    print(" ", a['id'], a['o16_no_c215'], a['o123_no_c215'])

# --- push_on_accept сегменты ---
pushed = [a for a in analyses if a['push_on_accept']]
print(f"\nСегментов с push_on_accept (O раньше первого C): {len(pushed)}")
for a in pushed[:10]:
    print(" ", a['id'], "firstO=", a['firstO'], "firstC=", a['firstC'])

# --- Сохраняем промежуточный JSON для генерации отчёта ---
json.dump(dict(
    segments=[dict(id=s['id'], source=s['source'], addr=s.get('addr'),
                   start_line=s['start_line'], marker_line=s.get('marker_line'),
                   declared_frames=s.get('declared_frames'), nframes=len(s['frames']),
                   analysis=analyze_segment(s), first3C=first3C(s))
              for s in all_segs],
    pair_total={f"{c}->{o}:{l}": n for (c, o, l), n in pair_total.items()},
    totals=dict(world_events=len(world), mirror_events=len(mirror),
                segs_world=len(segs_world), segs_mirror=len(segs_mirror)),
), open(os.path.join(BASE, "_windows-analysis-intermediate.json"), "w"), ensure_ascii=False, indent=1)
print("\nJSON saved.")

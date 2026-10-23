#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""R4.6.2 NPCSvr: генератор W30-бёрста из worldinit capture (канон NPCSvr-init).

Источник: cached-ref/logs-hourly/worldinit-20261023.txt, окно W30
(11:27:08→11:28:00, NPCSvr-коннект: C201 handshake → init-запросы).
ДОКАЗАНО: S=0 (тень молчала полностью — sessSet-гейт убивал NPCSvr-путь);
C277→O187 ×150 (1:1), C62→пачка 46 (40×O18), 179/224 = fire-and-forget;
X-сигнатур в шапках НЕТ — подстановки не нужны.

Выход:
  internal/rpc/testdata/npcsrv_w30_20261023.bin   — 228 O-кадров (конкатенация wire)
  internal/rpc/testdata/npcsrv_w30_20261023.json  — {cseq, batches: {cmd: [n_frames...]}}
"""
import re, os, json, collections, datetime

CAP = os.path.join(os.path.dirname(__file__), '..', '..', 'cached-ref', 'logs-hourly', 'worldinit-20261023.txt')
OUT_BIN = os.path.join(os.path.dirname(__file__), '..', 'internal', 'rpc', 'testdata', 'npcsrv_w30_20261023.bin')
OUT_JSON = os.path.join(os.path.dirname(__file__), '..', 'internal', 'rpc', 'testdata', 'npcsrv_w30_20261023.json')

pat = re.compile(r'^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}) ([COS])> frame len=(\d+) cmd=0x([0-9A-Fa-f]+)\(~0x[0-9A-Fa-f]+ marker=0x[0-9A-Fa-f]+ ok=\w+\) payload=(\d+) hex=([0-9a-f]+)')

frames = []
for line in open(CAP, errors='replace'):
    m = pat.match(line.strip())
    if m:
        frames.append({'t': m.group(1), 's': m.group(2), 'len': int(m.group(3)),
                       'cmd': int(m.group(4), 16), 'hex': m.group(6)})

def dt(t): return datetime.datetime.strptime(t, '%Y/%m/%d %H:%M:%S')
windows, cur = [], [frames[0]]
for a, b in zip(frames, frames[1:]):
    if (dt(b['t']) - dt(a['t'])).total_seconds() > 60:
        windows.append(cur); cur = []
    cur.append(b)
windows.append(cur)

# NPCSvr-окно: максимум O-кадров в окне, начинающемся с C201 и без C288
best, best_idx = None, -1
for i, w in enumerate(windows):
    cfirst = next((f['cmd'] for f in w if f['s'] == 'C'), None)
    o = [f for f in w if f['s'] == 'O']
    if cfirst == 201 and not any(f['cmd'] == 288 and f['s'] == 'C' for f in w):
        if best is None or len(o) > len(best):
            best, best_idx = o, i
assert best and len(best) == 228, f"ожидалось 228 O-кадров W30, получено {len(best) if best else 0}"

w = windows[best_idx]
cseq = [f['cmd'] for f in w if f['s'] == 'C']

# батчи: O-кадры между C_i и C_{i+1} ТОГО ЖЕ cmd (финал = до конца окна)
cur_batch = collections.defaultdict(list)   # cmd -> текущий батч (O-кадры)
batches = collections.defaultdict(list)     # cmd -> [батч=список O-кадров]
oblob = []
for f in w:
    if f['s'] == 'C':
        c = f['cmd']
        batches[c].append(cur_batch[c]); cur_batch[c] = []
    elif f['s'] == 'O':
        # O относится к последнему незакрытому C (по времени) — для канона
        # важно точное воссоздание ПОРЯДКА, поэтому копим глобально в порядке,
        # а принадлежность считаем по последнему C
        oblob.append(f)
        # принадлежность: последний C перед этим O
        # (последовательный проход: ищем последний C в отсортированном порядке)
for c, lst in batches.items():
    pass

# корректная атрибуция: для каждого O — последний C перед ним; O добавляем в батч этого C
cur = collections.defaultdict(list)
batch_map = collections.defaultdict(list)
last_c = None
for f in w:
    if f['s'] == 'C':
        if last_c is not None:
            batch_map[last_c].append(cur[last_c][:])
        last_c = f['cmd']
        cur[f['cmd']] = []
    elif f['s'] == 'O':
        cur[last_c].append(f)
if last_c is not None:
    batch_map[last_c].append(cur[last_c][:])

# ЯВНЫЕ индексы кадров (порядок O внутри окна) — без курсоров/неоднозначностей
o_index = {id(f): i for i, f in enumerate(best)}
manifest = {
    'source': f'worldinit-20261023.txt W{best_idx} (NPCSvr conn)',
    'frames': len(best),
    'cseq': cseq,
    'batches': {str(c): [[o_index[id(f)] for f in fr] for fr in lst]
                for c, lst in sorted(batch_map.items())},
}
os.makedirs(os.path.dirname(OUT_BIN), exist_ok=True)
blob = b''.join(bytes.fromhex(f['hex']) for f in best)
open(OUT_BIN, 'wb').write(blob)
open(OUT_JSON, 'w').write(json.dumps(manifest, indent=1))
total = sum(len(fr) for lst in manifest['batches'].values() for fr in lst)
print(f"OK: {OUT_BIN} {len(blob)}B {len(best)} кадров; cseq={len(cseq)} C; батчей={total} (проверка={total==len(best)})")

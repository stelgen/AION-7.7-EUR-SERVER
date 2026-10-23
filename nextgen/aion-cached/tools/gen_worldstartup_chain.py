#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""R4.6 мир-init: генератор startup-цепочки из worldinit capture (W5-канон).

Источник: cached-ref/logs-hourly/worldinit-20261023.txt, окно W5
(09:44:08→09:44:56, полный бёрст старта пары Server64: 288→196/90→11→14...).
Канон: O-кадры в порядке прихода, wire-формат [u16 len][cmd u16][0xAF][~cmd][payload].
Выход: internal/rpc/testdata/chain_worldstartup_20261023.bin (конкатенация 120 кадров).

Доказано (23.10): 118/120 кадров байт-в-байт = world-flow login chain;
новые: 0x7A (повтор после C223/C15), 0xBF (повтор после C15).
"""
import re, sys, json, os

CAP = os.path.join(os.path.dirname(__file__), '..', '..', 'cached-ref', 'logs-hourly', 'worldinit-20261023.txt')
OUT = os.path.join(os.path.dirname(__file__), '..', 'internal', 'rpc', 'testdata', 'chain_worldstartup_20261023.bin')
MAN = os.path.join(os.path.dirname(__file__), '..', 'internal', 'rpc', 'testdata', 'chain_worldstartup_20261023.json')

pat = re.compile(r'^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}) ([COS])> frame len=(\d+) cmd=0x([0-9A-Fa-f]+)\(~0x[0-9A-Fa-f]+ marker=0x[0-9A-Fa-f]+ ok=\w+\) payload=(\d+) hex=([0-9a-f]+)')

frames = []
for line in open(CAP, errors='replace'):
    m = pat.match(line.strip())
    if m:
        frames.append({'t': m.group(1), 's': m.group(2), 'len': int(m.group(3)),
                       'cmd': int(m.group(4), 16), 'hex': m.group(6)})

import datetime
def dt(t): return datetime.datetime.strptime(t, '%Y/%m/%d %H:%M:%S')
windows, cur = [], [frames[0]]
for a, b in zip(frames, frames[1:]):
    if (dt(b['t']) - dt(a['t'])).total_seconds() > 60:
        windows.append(cur); cur = []
    cur.append(b)
windows.append(cur)

# каноническое окно: максимум O-кадров из бёрстов с 288-стартом (W5-профиль)
best, best_idx = None, -1
for i, w in enumerate(windows):
    o = [f for f in w if f['s'] == 'O']
    c288 = any(f['s'] == 'C' and f['cmd'] == 288 for f in w)
    if c288 and (best is None or len(o) > len(best)):
        best, best_idx = o, i
assert best and len(best) == 120, f"ожидалось 120 O-кадров W5, получено {len(best) if best else 0}"

os.makedirs(os.path.dirname(OUT), exist_ok=True)
blob = b''.join(bytes.fromhex(f['hex']) for f in best)
open(OUT, 'wb').write(blob)

manifest = {
    'source': 'worldinit-20261023.txt W%d' % best_idx,
    'frames': len(best),
    'bytes': len(blob),
    'seq': [{'i': i, 'cmd': f['cmd'], 'len': f['len']} for i, f in enumerate(best)],
}
open(MAN, 'w').write(json.dumps(manifest, indent=1))
print(f"OK: {OUT} {len(blob)}B {len(best)} кадров; manifest {MAN}")

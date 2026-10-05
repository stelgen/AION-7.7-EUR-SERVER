#!/usr/bin/env python3
"""diag6: проба SM_INIT-констант (ресёрч 05-06.10: aioncore/aion-lightning SM_INIT,
beyond-aion 4.8, RaGEZONE CM_LOGIN) в ECB-dec(key1) plaintext welcome из capture.

Проверяем: 0x0000c621 (revision), 0x00030403 (197635), 0x00200000 (2097152),
0x3FCE09ED, нулевые 16b блоки; группируем по run (block0), ищем timestamp-корреляцию.
"""
import struct, sys, re
sys.path = [p for p in sys.path if p.rstrip('/\\') not in ('/tmp', '')]
import blowfish

BASE = '/home/user/STELGEN/projects/aion_server_2026-10-02/AION-7.7-EUR-SERVER'
KEY1 = bytes.fromhex('6b60cb5b82ce90b1cc2b6c556c6c6c6c')
SOURCES = ['/tmp/proxylog.txt', BASE + '/nextgen/aion-gate/testdata/handshake-capture-20261003.txt']

MAGICS = {
    'rev_c621':    bytes.fromhex('21c60000'),  # 0x0000c621 SM_INIT protocol revision [aioncore]
    'unk_197635':  bytes.fromhex('03040300'),  # 0x00030403 writeD(197635) [aioncore]
    'unk_2097152': bytes.fromhex('00002000'),  # 0x00200000 writeD(2097152) [aioncore]
    'unk_3FCE09ED':bytes.fromhex('ed09ce3f'),  # 0x3FCE09ED [beyond-aion 4.8]
}

def ecb_dec(data):
    c = blowfish.Cipher(KEY1)
    out = b''
    for off in range(0, len(data), 8):
        out += next(c.decrypt_ecb(data[off:off+8]))
    return out

welcomes = []  # (src, ts, pkt)
rx = re.compile(r'(\d\d:\d\d:\d\d).*?G->C len=194 hex=([0-9a-f]+)')
for src in SOURCES:
    for line in open(src, errors='replace'):
        m = rx.search(line)
        if m:
            welcomes.append((src.split('/')[-1], m.group(1), bytes.fromhex(m.group(2))))

print(f'welcome-пакетов всего: {len(welcomes)} (с дедупом)')
seen = {}
for src, ts, pkt in welcomes:
    assert pkt[:2] == b'\xc2\x00', pkt[:4].hex()
    dec = ecb_dec(pkt[2:])
    seen.setdefault(dec, []).append((src, ts))

plains = list(seen.keys())
print(f'уникальных plaintext (ECB-dec key1): {len(plains)}  (остальные = повторы тождественных)')

# --- 1) группировка по block0 (dword0,dword1) — подтверждение per-run констант
groups = {}
for p in plains:
    d0, d1 = struct.unpack('<II', p[0:8])
    groups.setdefault((d0, d1), []).append(p)
print(f'групп по block0=(d0,d1): {len(groups)}')
for (d0, d1), ps in sorted(groups.items()):
    uniq_rest = len({p[8:] for p in ps})
    print(f'  d0={d0:08x} d1={d1:08x}  плейнтекстов={len(ps)}  разных после байта 8: {uniq_rest}')

# --- 2) скан магических констант SM_INIT на ЛЮБЫХ смещениях (по всем plaintext)
print('\n=== скан констант SM_INIT (все смещения, все plaintext) ===')
hits = {}
for p in plains:
    for name, magic in MAGICS.items():
        off = p.find(magic)
        while off != -1:
            hits.setdefault(name, []).append(off)
            off = p.find(magic, off + 1)
if hits:
    for name, offs in hits.items():
        print(f'  НАЙДЕНО {name}: offsets={sorted(set(offs))} (в {len(set(offs))} смещ.)')
else:
    print('  ни одна константа SM_INIT НЕ найдена ни в одном plaintext')

# --- 3) нулевые блоки (>=8 нулей подряд) и их смещения
print('\n=== нулевые прогоны >=8 байт ===')
zr = {}
for p in plains:
    for m in re.finditer(rb'\x00{8,}', p):
        zr.setdefault((m.start(), len(m.group())), 0)
        zr[(m.start(), len(m.group()))] += 1
for (off, ln), cnt in sorted(zr.items()):
    print(f'  offset={off:3d} len={ln:2d}  в {cnt} plaintext')

# --- 4) timestamp-корреляция block0: unix-time / GetTickCount
print('\n=== block0 как время? (unix 2024-2027 или ms-tick) ===')
for (d0, d1) in sorted(groups):
    for name, v in (('d0', d0), ('d1', d1)):
        if 0x65000000 <= v <= 0x7C000000:   # unix 2024-09..2027-01
            print(f'  {name}={v:08x} похоже на unix-time!')
        if 0x0000E000 <= v <= 0x3B9AC000:   # 57s..16.6 дней в мс
            print(f'  {name}={v:08x} похоже на GetTickCount-ms ({v/1000/3600:.1f} ч)')
# разности dword1 между группами — кратны ли чему-то
ds = sorted(d1 for (d0, d1) in groups)
print('  dword1 всех групп:', ' '.join(f'{x:08x}' for x in ds[:10]), '...' if len(ds) > 10 else '')

# --- 5) где начинается расхождение между соединениями одного run
print('\n=== первый различающийся байт между plaintext одного run ===')
for (d0, d1), ps in list(groups.items()):
    if len(ps) >= 2:
        a, b = ps[0], ps[1]
        i = next((k for k in range(len(a)) if a[k] != b[k]), None)
        print(f'  d0={d0:08x}: n={len(ps)} первый diff @ байт {i}')
        break  # достаточно одной группы
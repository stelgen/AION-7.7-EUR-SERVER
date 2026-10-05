#!/usr/bin/env python3
"""diag7: где начинается цепочка скрамбла?

Модель (asm 0x417a20): new[k]=old[k]^S_k, S_k=cumsum(old[0..k]) (S_init=0 на старте цепочки).
Следствие: bit0(new[m+1]) == bit0(old[m]) == bit0(new[m]) — ЕДИНСТВЕННАЯ гарантированно
равная пара (m, m+1) при старте цепочки с dword m.
capture: пара (0,1) НЕ равна (0xd8≠0x23 у 7d521423) → гипотеза: цепочка стартует с m=2
(первые 8 байт plaintext = per-run сид ВНЕ цепочки; diag6: первый diff между сессиями @ байт 8).
"""
import struct, sys, re
sys.path = [p for p in sys.path if p.rstrip('/\\') not in ('/tmp', '')]
import blowfish

BASE = '/home/user/STELGEN/projects/aion_server_2026-10-02/AION-7.7-EUR-SERVER'
KEY1 = bytes.fromhex('6b60cb5b82ce90b1cc2b6c556c6c6c6c')
SOURCES = ['/tmp/proxylog.txt', BASE + '/nextgen/aion-gate/testdata/handshake-capture-20261003.txt']

def ecb_dec(data):
    c = blowfish.Cipher(KEY1)
    out = b''
    for off in range(0, len(data), 8):
        out += next(c.decrypt_ecb(data[off:off+8]))
    return out

plains = set()
rx = re.compile(r'G->C len=194 hex=([0-9a-f]+)')
for src in SOURCES:
    for line in open(src, errors='replace'):
        m = rx.search(line)
        if m:
            plains.add(ecb_dec(bytes.fromhex(m.group(1))[2:]))
print('уникальных plaintext:', len(plains))

# доля равенства bit0 по соседним парам (k,k+1) по всем 82
pairs = {}
for p in plains:
    dw = struct.unpack('<46I', p[:184])
    for k in range(45):
        eq = ((dw[k+1] ^ dw[k]) & 1) == 0
        pairs.setdefault(k, [0, 0])
        pairs[k][0 if eq else 1] += 1

print('пара  доля bit0(new[k+1])==bit0(new[k])')
for k in range(45):
    eq, ne = pairs[k]
    bar = '#' * int(round(20 * eq / len(plains)))
    flag = ' <== КАНДИДАТ СТАРТА ЦЕПИЧКИ' if eq == len(plains) else ''
    print(f'  ({k},{k+1})  {eq}/{len(plains)}  {bar}{flag}')

# также: константность XOR-разности бит0 пары (если S_init пер-ран сид с фикс. бит0)
print('\nXOR-разность бит0 по парам: сколько plaintext дают КОНСТАНТНОЕ значение разности')
for k in range(45):
    vals = set()
    for p in plains:
        dw = struct.unpack('<46I', p[:184])
        vals.add((dw[k+1] ^ dw[k]) & 1)
    if len(vals) == 1:
        print(f'  пара ({k},{k+1}): разность константна = {vals.pop()}')
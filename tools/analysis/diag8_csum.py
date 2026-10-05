#!/usr/bin/env python3
"""diag8: локализация цепочки скрамбла через чексумму.
csum хранится в dec[184:188]. Если csum = сумма старых dwords скрамбл-региона,
то найдётся окно [i..j) dwords с суммой == csum; консистентное i по всем 82
= старт цепочки. Плюс элементарные гипотезы (XOR, байтовая сумма, окно==dw[45])."""
import struct, sys, re
from collections import Counter
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

win_start_cnt = Counter()      # окна [i..46) сумма==csum
exact_cnt = Counter()          # точные гипотезы
M = 0xffffffff

for p in plains:
    dw = struct.unpack('<46I', p[:184])
    csum = struct.unpack('<I', p[184:188])[0]
    # префикс-суммы для всех окон
    ps = [0]
    for v in dw:
        ps.append((ps[-1] + v) & M)
    for i in range(47):
        for j in range(i + 1, 47):
            if (ps[j] - ps[i]) & M == csum:
                win_start_cnt[(i, j)] += 1
    # элементарные
    if csum == sum(dw[:45]) & M: exact_cnt['sum(d0..44)'] += 1
    if csum == sum(dw[:46]) & M: exact_cnt['sum(d0..45)'] += 1
    if dw[45] == sum(dw[:44]) & M: exact_cnt['dw45==sum(d0..43)'] += 1
    x = 0
    for v in dw[:46]: x ^= v
    if csum == x: exact_cnt['xor(d0..45)'] += 1
    if csum == (sum(p[:184]) & M): exact_cnt['bytesum(0..184)'] += 1
    if csum == (sum(p[2:186]) & M): exact_cnt['bytesum(2..186)'] += 1

print('\nточные гипотезы (сколько из 82):')
for k, v in exact_cnt.most_common():
    print(f'  {k}: {v}')

print('\nокна с суммой==csum, встречающиеся в >=3 plaintext (i,j → счёт):')
for (i, j), c in win_start_cnt.most_common(15):
    if c >= 3:
        print(f'  окно [{i}:{j}): {c}/82')
if not any(c >= 3 for c in win_start_cnt.values()):
    print('  (нет консистентных окон — csum не является суммой contiguous dwords dec)')
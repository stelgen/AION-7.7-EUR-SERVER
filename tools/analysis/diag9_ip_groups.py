#!/usr/bin/env python3
"""diag9: d0/d1 welcome-групп vs клиентский IP — корреляция (кто задаёт plaintext[0:8])."""
import struct, sys, re
sys.path = [p for p in sys.path if p.rstrip('/\\') not in ('/tmp', '')]
import blowfish

KEY1 = bytes.fromhex('6b60cb5b82ce90b1cc2b6c556c6c6c6c')
SOURCES = ['/tmp/proxylog.txt',
           '/home/user/STELGEN/projects/aion_server_2026-10-02/AION-7.7-EUR-SERVER/nextgen/aion-gate/testdata/handshake-capture-20261003.txt']

def ecb_dec(data):
    c = blowfish.Cipher(KEY1)
    out = b''
    for off in range(0, len(data), 8):
        out += next(c.decrypt_ecb(data[off:off+8]))
    return out

rx_cli = re.compile(r'(\d\d:\d\d:\d\d) --- new client \(\'([\d.]+)\', (\d+)\)')
rx_wel = re.compile(r'(\d\d:\d\d:\d\d) G->C len=194 hex=([0-9a-f]+)')
cur_ip = None
recs = []  # (ts, ip, d0, d1, port)
for src in SOURCES:
    for line in open(src, errors='replace'):
        m = rx_cli.search(line)
        if m:
            cur_ip = m.group(2)
            continue
        m = rx_wel.search(line)
        if m:
            d0, d1 = struct.unpack('<II', ecb_dec(bytes.fromhex(m.group(2))[2:])[:8])
            recs.append((m.group(1), cur_ip, d0, d1))

print('welcome с привязкой к IP:', len(recs))
by_ip = {}
for ts, ip, d0, d1 in recs:
    by_ip.setdefault(ip, set()).add((d0, d1))
print('\nIP → группы (d0,d1):')
for ip, gs in by_ip.items():
    print(f'  {ip}: {len(gs)} групп')
by_grp = {}
for ts, ip, d0, d1 in recs:
    by_grp.setdefault((d0, d1), set()).add(ip)
multi_ip = {g: ips for g, ips in by_grp.items() if len(ips) > 1}
print(f'\nгрупп всего: {len(by_grp)}; групп с НЕСКОЛЬКИМИ IP: {len(multi_ip)}')
for g, ips in list(multi_ip.items())[:10]:
    print(f'  {g[0]:08x}/{g[1]:08x}: {sorted(ips)}')
# АНТИ-проверка: один IP в нескольких группах?
one_ip_many = {ip: gs for ip, gs in by_ip.items() if len(gs) > 1}
print(f'IP с несколькими группами: {len(one_ip_many)}')
for ip, gs in one_ip_many.items():
    print(f'  {ip}: {sorted(gs)[:6]}')

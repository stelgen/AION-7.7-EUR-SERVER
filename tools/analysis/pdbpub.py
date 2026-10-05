#!/usr/bin/env python3
"""Скан S_PUB32 (0x110E) по сырому PDB: name -> (section, offset). Без разбора MSF.
Usage: pdbpub.py <file.pdb> [filter]"""
import struct, sys, re

NAME_RE = re.compile(rb'^[A-Za-z_?@$<>:.,\d\\/-]{4,220}$')

def scan(data):
    res = {}
    n = len(data)
    i = 0
    kind_le = b'\x0e\x11'  # 0x110E LE
    while True:
        i = data.find(kind_le, i)
        if i < 0 or i + 16 > n:
            break
        ln = struct.unpack_from('<H', data, i - 2)[0] if i >= 2 else 0
        if 26 <= ln <= 400 and i - 2 + ln <= n:
            flags, off, sec = struct.unpack_from('<IIH', data, i + 2)
            name = data[i + 12: i - 2 + ln]
            z = name.find(b'\x00')
            if z > 0:
                name = name[:z]
                if 4 <= len(name) <= 220 and NAME_RE.match(name) and 1 <= sec <= 32 and off < 0x1000000:
                    try:
                        nm = name.decode('ascii')
                        if nm not in res:
                            res[nm] = (sec, off, flags)
                    except UnicodeDecodeError:
                        pass
                    i = i - 2 + ln
                    continue
        i += 2
    return res

if __name__ == '__main__':
    data = open(sys.argv[1], 'rb').read()
    syms = scan(data)
    print(f'# publics={len(syms)}', file=sys.stderr)
    filt = (sys.argv[2] if len(sys.argv) > 2 else '').lower()
    for name, (sec, off, flags) in sorted(syms.items(), key=lambda kv: (kv[1][0], kv[1][1])):
        if filt and filt not in name.lower():
            continue
        print(f'{sec}\t{off:#x}\t{flags:#x}\t{name}')

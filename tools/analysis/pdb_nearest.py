#!/usr/bin/env python3
"""pdb_nearest.py — ближайший PDB-publics символ к VA.

Формат publics-файла (tools/analysis/pdbpub.py): `sec\toff\tsize\tname`,
ВАЖНО: off = parts[1] (НЕ parts[2] — это size!). sec '1' = .text:
VA = imagebase + 0x1000 + off.

Использование:
  python3 pdb_nearest.py <pdb-publics.txt> <VA> [VA...]
  python3 pdb_nearest.py <pdb-publics.txt> --classify 0x140302173 0x140460897 ...
"""
import bisect, sys

def load(path, imagebase=0x140000000, text_va=0x1000):
    syms = []
    for line in open(path, encoding='utf-8', errors='ignore'):
        parts = line.rstrip('\r\n').split('\t')
        if len(parts) >= 4 and parts[0] == '1':
            try:
                syms.append((int(parts[1], 16), parts[3]))
            except ValueError:
                pass
    syms.sort()
    return syms

def nearest(syms, va, imagebase=0x140000000, text_va=0x1000):
    offs = [s[0] for s in syms]
    key = va - imagebase - text_va
    i = bisect.bisect_right(offs, key) - 1
    if i < 0:
        return None, 0
    return syms[i], key - syms[i][0]

def main():
    path = sys.argv[1]
    syms = load(path)
    args = sys.argv[2:]
    classify = False
    if args and args[0] == '--classify':
        classify = True
        args = args[1:]
    for a in args:
        va = int(a, 16)
        s, delta = nearest(syms, va)
        if s is None:
            print(f"{hex(va)} -> <no symbol>")
            continue
        mark = ' [FUNC-START]' if delta == 0 else ''
        print(f"{hex(va)} -> {s[1]} +{hex(delta)}{mark}")

if __name__ == '__main__':
    main()
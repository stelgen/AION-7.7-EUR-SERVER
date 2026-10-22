#!/usr/bin/env python3
"""find_callers.py — поиск всех ссылок на VA-функцию в PE-бинаре (x64).

Покрывает: E8 rel32 (call), E9 rel32 (jmp-thunk), FF15/FF25 (call/jmp [rip+d]),
lea/mov rip-rel (48/4c 8d|8b /r mod=00 rm=101), ptr VA в .data/.rdata (таблицы).

Использование:
  python3 find_callers.py <binary.exe> <VA hex, напр. 0x140302173> [--mask-only]

Метод обкатан на Server64.exe (45МБ) и CacheD64.exe (22МБ), см.
nextgen/aion-binpatch/docs/recon-108-111-20261022.md.
"""
import struct, sys

def sections(data):
    pe = struct.unpack_from('<I', data, 0x3c)[0]
    nsec = struct.unpack_from('<H', data, pe+6)[0]
    opt = struct.unpack_from('<H', data, pe+20)[0]
    sec0 = pe + 24 + opt
    out = []
    for i in range(nsec):
        s = sec0 + i*40
        name = data[s:s+8].rstrip(b'\x00').decode(errors='ignore')
        vsize, vaddr, rsize, raddr = struct.unpack_from('<IIII', data, s+8)
        out.append((name, vaddr, raddr, rsize))
    return out

def va2off(secs, va, imagebase=0x140000000):
    rva = va - imagebase
    for name, vaddr, raddr, rsize in secs:
        if vaddr <= rva < vaddr + rsize:
            return raddr + (rva - vaddr)
    return None

def main():
    path, va = sys.argv[1], int(sys.argv[2], 16)
    data = open(path, 'rb').read()
    secs = sections(data)
    text = next(s for s in secs if s[0] == '.text')
    _, tva, traw, tsz = text
    target = va

    callers = []
    for off in range(traw, traw + tsz - 7):
        cva = 0x140000000 + tva + (off - traw)
        b = data
        if b[off] == 0xE8 or b[off] == 0xE9:                       # call/jmp rel32
            rel = struct.unpack_from('<i', b, off+1)[0]
            if cva + 5 + rel == target:
                callers.append((cva, 'call' if b[off] == 0xE8 else 'jmp'))
        if b[off] == 0xFF and b[off+1] in (0x15, 0x25):            # call/jmp [rip+d]
            d = struct.unpack_from('<i', b, off+2)[0]
            if cva + 6 + d == target:
                callers.append((cva, f'FF{b[off+1]:02x} [rip]'))
        if b[off] in (0x48, 0x4c) and b[off+1] in (0x8b, 0x8d) and (b[off+2] & 0xC7) == 0x05:
            d = struct.unpack_from('<i', b, off+3)[0]              # mov/lea reg, [rip+d]
            if cva + 7 + d == target:
                callers.append((cva, f'{b[off+1]:02x} [rip]'))

    print(f"code refs to {hex(target)}:")
    for cva, kind in callers:
        print(f"  {hex(cva)} {kind}")

    # ptr VA в секциях данных
    tb = struct.pack('<Q', target)
    for name, vaddr, raddr, rsize in secs:
        if name == '.text':
            continue
        seg = data[raddr:raddr+rsize]
        pos = 0
        while True:
            i = seg.find(tb, pos)
            if i < 0: break
            print(f"  ptr @ {name} VA {hex(0x140000000 + vaddr + i)}")
            pos = i + 1

if __name__ == '__main__':
    main()
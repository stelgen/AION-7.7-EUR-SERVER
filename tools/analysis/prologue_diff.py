#!/usr/bin/env python3
"""prologue_diff.py — якорный дифф функций между сборками (метод #111).

Алгоритм (обкатан на Server64 vs чужой 111-Mainserver64, см.
nextgen/aion-binpatch/docs/recon-108-111-20261022.md):
  1. Сверить размеры/MD5 бинарей — если разные, построчный дифф НЕВАЛИДЕН.
  2. Prologue (16Б) функции из нашей сборки (VA по нашему PDB) ищется в чужом exe.
     - 0 хитов  -> функция пересобрана (NO-PROLOGUE);
     - много    -> generic пролог (mov [rsp+8]...), ложные якоря;
     - 1 хит    -> якорь найден, тела диффятся.
  3. Дифф N байт тел -> кластеры -> если кластеры НЕ байт-в-байт, а структурные
     (другой поток/аргументы) => перенос невозможен, нужен семантический реверс.

Использование:
  python3 prologue_diff.py <ours.exe> <foreign.exe> pairs.json
pairs.json: {"Имя@Класс": {"ours_va": "0x1404338E0"}, ...}
"""
import json, struct, sys

def sections(data):
    pe = struct.unpack_from('<I', data, 0x3c)[0]
    nsec = struct.unpack_from('<H', data, pe+6)[0]
    opt = struct.unpack_from('<H', data, pe+20)[0]
    sec0 = pe + 24 + opt
    out = []
    for i in range(nsec):
        s = sec0 + i*40
        vsize, vaddr, rsize, raddr = struct.unpack_from('<IIII', data, s+8)
        out.append((vaddr, raddr, rsize))
    return out

def va2off(secs, va, imagebase=0x140000000):
    rva = va - imagebase
    for vaddr, raddr, rsize in secs:
        if vaddr <= rva < vaddr + rsize:
            return raddr + (rva - vaddr)
    return None

def main():
    ours_path, foreign_path, pairs_file = sys.argv[1], sys.argv[2], sys.argv[3]
    ours = open(ours_path, 'rb').read()
    foreign = open(foreign_path, 'rb').read()
    osecs, fsecs = sections(ours), sections(foreign)
    print(f"ours={len(ours)}B foreign={len(foreign)}B "
          f"{'SIZE-MATCH' if len(ours)==len(foreign) else 'SIZE-MISMATCH: построчный дифф невалиден, только якорный'}")

    N = int(sys.argv[4]) if len(sys.argv) > 4 else 512
    pairs = json.load(open(pairs_file, encoding='utf-8'))
    for name, spec in pairs.items():
        va = int(spec['ours_va'], 16)
        off = va2off(osecs, va)
        if off is None:
            print(f"{name}: ours VA вне секций"); continue
        pro = bytes(ours[off:off+16])
        pos, found = 0, []
        while True:
            i = foreign.find(pro, pos)
            if i < 0: break
            found.append(i); pos = i + 1
        tag = 'UNIQUE' if len(found) == 1 else (f'x{len(found)} generic' if found else 'NO-PROLOGUE (пересобрана)')
        print(f"\n{name}: prologue={pro.hex()} -> {tag}")
        if len(found) != 1:
            continue
        foff = found[0]
        a, b = ours[off:off+N], foreign[foff:foff+N]
        diffs = [k for k in range(N) if a[k] != b[k]]
        clusters = []
        for k in diffs:
            if clusters and k - clusters[-1][1] <= 8:
                clusters[-1][1] = k
            else:
                clusters.append([k, k])
        span = sum(c1-c0+1 for c0, c1 in clusters)
        identical = (a == b)
        print(f"  тела: identical={identical}, diff-кластеры={len(clusters)}, span={span}Б")
        for c0, c1 in clusters[:4]:
            print(f"    @{hex(va+c0)}..{hex(va+c1)} ours={a[c0:c1+1].hex()} foreign={b[c0:c1+1].hex()}")
        if not identical and (len(clusters) > 3 or span > N//4):
            print("  ВЕРДИКТ: тела структурно расходятся — байтовый перенос невозможен")

if __name__ == '__main__':
    main()
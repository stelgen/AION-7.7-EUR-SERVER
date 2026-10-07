#!/usr/bin/env python3
"""cap7777_decrypt.py — дешифровка capture 7777 (7.5-7.8 XOR-крипта) и сверка опкодов.
Wire: [u16 size LE self-inclusive][body]. S2C body=[E][0x56][~E][payload], E=(op+0xD8)^0xD9.
C2S body=[A][0x75][~A][payload]. Rolling key: 8Б = [baseKey LE][A1 6C 54 87], key+=len(body) после пакета.
"""
import struct, sys, csv
from collections import Counter
from scapy.all import PcapReader, TCP, IP

PCAP = sys.argv[1] if len(sys.argv) > 1 else "/home/user/STELGEN/tmp/cap7777/cap7777-0810.pcapng"
CSVP = "/home/user/STELGEN/projects/aion_server_2026-10-02/AION-7.7-EUR-SERVER/nextgen/aion-main/docs/opcodes-unified-0810.csv"

STATIC_KEY = b"nKO/WctQ0AVLbpzfBkS6NevDYT8ourG5CRlmdjyJ72aswx4EPq1UgZhFMXH?3iI9"
assert len(STATIC_KEY) >= 64, len(STATIC_KEY)
CLI_CODE, SRV_CODE = 0x75, 0x56

def xor_crypt(data, key8, decrypt):
    out = bytearray(data)
    prev_enc = out[0]
    out[0] ^= key8[0]
    for i in range(1, len(out)):
        cur_enc = out[i]
        out[i] ^= STATIC_KEY[i & 63] ^ key8[i & 7] ^ (prev_enc if decrypt else out[i - 1] if False else prev_enc)
        # NB: encrypt side: prev = new encrypted byte; decrypt side: prev = old encrypted byte
        if decrypt:
            prev_enc = cur_enc
        else:
            prev_enc = out[i]
    return out

def roll_key(key8, size):
    v = struct.unpack("<Q", key8)[0]
    v = (v + size) & 0xFFFFFFFFFFFFFFFF
    return struct.pack("<Q", v)

def reassemble(pcap):
    """Корректная склейка TCP-потока: перекрытия (retransmissions) обрезаются."""
    flows = {}
    with PcapReader(pcap) as rd:
        for pkt in rd:
            if TCP not in pkt or IP not in pkt:
                continue
            t = pkt[TCP]
            key = (pkt[IP].src, t.sport, pkt[IP].dst, t.dport)
            payload = bytes(t.payload)
            if not payload:
                continue
            flows.setdefault(key, []).append((t.seq, payload))
    out = {}
    for key, segs in flows.items():
        segs.sort(key=lambda sp: sp[0])
        base = segs[0][0]
        data = bytearray()
        covered = 0  # длина уже записанного с base
        for seq, payload in segs:
            off = seq - base
            if off + len(payload) <= covered:
                continue  # полный дубль
            if off > covered:
                # разрыв в последовательности (потерянные сегменты) — вставляем нули-маркер
                data.extend(b"\x00" * (off - covered))
                covered = off
            cut = covered - off
            data.extend(payload[cut:])
            covered = off + len(payload)
        out[key] = bytes(data)
    return out

def split_frames(stream):
    frames = []
    i = 0
    while i + 2 <= len(stream):
        size = struct.unpack_from("<H", stream, i)[0]
        if size < 2 or i + size > len(stream):
            break  # хвост/некратность
        frames.append(stream[i + 2:i + size])
        i += size
    return frames, len(stream) - i

def main():
    import os
    if len(sys.argv) >= 4 and sys.argv[2].endswith(".hex"):
        # режим: s2c.hex / c2s.hex из tshark follow
        s2c = bytes.fromhex("".join(open(sys.argv[2]).read().split()))
        c2s = bytes.fromhex("".join(open(sys.argv[3]).read().split()))
    else:
        flows = reassemble(PCAP)
        # выбираем НАИБОЛЬШИЙ поток по каждому направлению (в 7777-фильтр попадают и сканеры)
        best_c2s, best_s2c = (b"", 0), (b"", 0)
        for (sip, sp, dip, dp), data in flows.items():
            if dp == 7777 and len(data) > best_c2s[1]:
                best_c2s = (data, len(data))
            if sp == 7777 and len(data) > best_s2c[1]:
                best_s2c = (data, len(data))
        c2s, s2c = best_c2s[0], best_s2c[0]
    stats = Counter()
    log = []

    # --- S2C ---
    sf, tail = split_frames(s2c)
    keyS = None
    keyC = None
    first_open = True
    for idx, body in enumerate(sf):
        if first_open:
            E = struct.unpack_from("<H", body, 0)[0]
            code = body[2]
            negE = struct.unpack_from("<H", body, 3)[0]
            op = ((E ^ 0xD9) - 0xD8) & 0xFFFF
            log.append(f"S2C#{idx} OPEN E=0x{E:04X} code=0x{code:02X} ~E_ok={(E ^ 0xFFFF) & 0xFFFF == negE} op=0x{op:04X} len={len(body)}")
            if code == SRV_CODE and ((E ^ 0xFFFF) & 0xFFFF) == negE:
                if op == 0x48 and len(body) >= 9:  # SM_KEY
                    falseKey = struct.unpack_from("<I", body, 5)[0]
                    baseKey = ((falseKey - 0x3FF2CCDF) & 0xFFFFFFFF) ^ 0xCD92E4D9
                    keyS = struct.pack("<I", baseKey) + bytes([0xA1, 0x6C, 0x54, 0x87])
                    keyC = keyS  # client key = копия server key на старте (rolling независимый)
                    log.append(f"  SM_KEY: falseKey=0x{falseKey:08X} baseKey=0x{baseKey:08X} -> key={keyS.hex()}")
                first_open = False
                stats[("S2C", op)] += 1
            else:
                # не SM_KEY в начале? — считаем всё же открытым, но подозрение
                stats[("S2C", op)] += 1
            continue
        dec = xor_crypt(body, keyS, decrypt=True)
        E = struct.unpack_from("<H", dec, 0)[0]
        code = dec[2]
        negE = struct.unpack_from("<H", dec, 3)[0]
        valid = code == SRV_CODE and ((E ^ 0xFFFF) & 0xFFFF) == negE
        op = ((E ^ 0xD9) - 0xD8) & 0xFFFF
        stats[("S2C", op if valid else None)] += 1
        if not valid:
            log.append(f"S2C#{idx} INVALID raw={body[:10].hex()} len={len(body)}")
        else:
            log.append(f"S2C#{idx} op=0x{op:04X} len={len(body)}")
        keyS = roll_key(keyS, len(body))

    # --- C2S ---
    cf, tail2 = split_frames(c2s)
    # keyC уже установлен в S2C-блоке (= ключ из SM_KEY)
    first_c = False
    for idx, body in enumerate(cf):
        if first_c and keyC is None:
            # пробуем открытый (до SM_KEY)
            A = struct.unpack_from("<H", body, 0)[0]
            negA = struct.unpack_from("<H", body, 3)[0]
            if body[2] == CLI_CODE and ((A ^ 0xFFFF) & 0xFFFF) == negA:
                stats[("C2S", A)] += 1
                log.append(f"C2S#{idx} OPEN op=0x{A:04X} len={len(body)}")
                continue
            first_c = False
        if keyC is None:
            log.append(f"C2S#{idx} до SM_KEY-момента — ключ ещё не установлен? raw={body[:8].hex()}")
            continue
        dec = xor_crypt(body, keyC, decrypt=True)
        A = struct.unpack_from("<H", dec, 0)[0]
        negA = struct.unpack_from("<H", dec, 3)[0]
        valid = dec[2] == CLI_CODE and ((A ^ 0xFFFF) & 0xFFFF) == negA
        if valid:
            stats[("C2S", A)] += 1
            log.append(f"C2S#{idx} op=0x{A:04X} len={len(body)}")
            keyC = roll_key(keyC, len(body))
        else:
            stats[("C2S", None)] += 1
            log.append(f"C2S#{idx} INVALID raw={body[:10].hex()} len={len(body)}")

    # --- Сверка с таблицей ---
    table = {}
    for r in csv.DictReader(open(CSVP)):
        table[r["name"]] = r
    print(f"S2C фреймов {len(sf)} (tail {tail}), C2S фреймов {len(cf)} (tail {tail2})")
    known = unknown = invalid = 0
    unmatched = []
    for (d, op), n in sorted(stats.items(), key=lambda kv: -kv[1]):
        if op is None:
            invalid += n
            continue
        hexop = f"0x{op:04X}"
        m = next((nm for nm, r in table.items() if r.get("ag78", "") == hexop or r.get("mobius77", "") == hexop or r.get("xml75", "") == hexop), None)
        if m:
            known += n
        else:
            unknown += n
            unmatched.append((d, hexop, n))
    print(f"\n== ИТОГ == known={known} unknown={unknown} invalid={invalid}")
    print("TOP S2C:", [(f"0x{o:04X}", n) for (d, o), n in stats.most_common(15) if o is not None and d == "S2C"])
    print("TOP C2S:", [(f"0x{o:04X}", n) for (d, o), n in stats.most_common(15) if o is not None and d == "C2S"])
    print("UNKNOWN:", unmatched[:20])
    print("C2S полный:", sorted([(f"0x{o:04X}", n) for (d, o), n in stats.items() if d == "C2S" and o is not None], key=lambda x: -x[1]))
    open("/home/user/STELGEN/tmp/cap7777/decrypt-log.txt", "w").write("\n".join(log))

if __name__ == "__main__":
    main()
#!/usr/bin/env python3
"""Разбор welcome из capture: ECB-dec(key1) → unscramble (битовый DP) → структура."""
import struct, sys
sys.path = [p for p in sys.path if p.rstrip('/\\') not in ('/tmp','')]  # чтобы не подхватился /tmp/blowfish.py
import blowfish  # pip-модуль

CAP='/home/user/STELGEN/projects/aion_server_2026-10-02/AION-7.7-EUR-SERVER/nextgen/aion-gate/testdata/handshake-capture-20261003.txt'
LUT=open('/home/user/STELGEN/projects/aion_server_2026-10-02/AION-7.7-EUR-SERVER/nextgen/aion-gate/internal/proto/testdata/gate-lut.bin','rb').read()
lut=[struct.unpack('<I',LUT[i*4:i*4+4])[0] for i in range(256)]
KEY1=bytes.fromhex('6b60cb5b82ce90b1cc2b6c556c6c6c6c')
M=0xffffffff

def solve(a,y,limit=64):
    """все x (32бит): x ^ (a+x) == y"""
    cur=[(0,0)]
    for i in range(32):
        ai=(a>>i)&1; yi=(y>>i)&1; nxt=[]
        for x,c in cur:
            for xb in (0,1):
                s=ai+xb+c
                if (xb^(s&1))==yi:
                    nxt.append((x|(xb<<i), s>>1))
        cur=nxt
        if len(cur)>limit: cur=cur[:limit]
        if not cur: return []
    return sorted(set(x for x,c in cur))

def ecb_dec(data):
    c=blowfish.Cipher(KEY1)
    out=b''
    for off in range(0,len(data),8):
        out+=next(c.decrypt_ecb(data[off:off+8]))
    return out

def unscramble(new):
    n=len(new)//4
    dw=[struct.unpack('<I',new[i*4:i*4+4])[0] for i in range(n)]
    sums=[0]*n
    sols=[None]*n  # список кандидатов на dword (или список списков — DFS)
    # DFS с отсечками структуры
    results=[]
    def dfs(k,S,acc):
        if len(results)>8: return
        if k==n:
            results.append(list(acc)); return
        cands=solve(S,dw[k]) if k>0 else [dw[0]]
        for old in cands:
            ns=S+old
            ok=True
            if k>0:
                # проверка структуры: new_dword = old ^ ns
                pass
            acc.append(old)
            dfs(k+1,ns,acc)
            acc.pop()
            if len(results)>8: return
    dfs(0,0,[])
    return results

def inv_mod(m):  # инверсия scrambleModulus (чистый XOR — обратим)
    b=bytearray(m)
    for i in range(0x40,0x80): b[i]^=b[i-0x40]
    for i in range(4): b[0x0d+i]^=b[0x34+i]
    for i in range(0x40): b[i]^=b[0x40+i]
    for i in range(4): b[i],b[0x4d+i]=b[0x4d+i],b[i]
    return bytes(b)

sessions=[]
for line in open(CAP):
    if ' G->C len=194 ' in line:
        hx=line.split(' len=194 ')[1].strip()
        sessions.append(bytes.fromhex(hx))
print('welcome-пакетов в capture:', len(sessions))

for idx in (0,1,2):
    pkt=sessions[idx]
    assert pkt[:2]==b'\xc2\x00'
    dec=ecb_dec(pkt[2:])
    assert len(dec)==192
    csum=struct.unpack('<I',dec[184:188])[0]
    sols=unscramble(dec[:184])
    print('решений unscramble:', len(sols))
    for plain in sols[:3]:
        p=b''.join(struct.pack('<I',x) for x in plain)
        cs=sum(plain[:45])&M
        if cs!=csum:
            print('  csum MISMATCH: cumsum=%08x vs %08x — решение неверно'%(cs,csum)); continue
        d0=plain[0]
        print(f'  dword0={d0:08x} plaintext[0]={p[0]:02x} bytes1-4={p[1:5].hex()}')
        mod=inv_mod(p[9:137])
        gg=p[137:153]; key2=p[153:169]; lt=p[169]; B=p[170:173]; extra=p[173:177]
        zeros=all(b==0 for b in mod[:96])
        r=None
        for rr in range(256):
            k2=b''.join(struct.pack('<I',lut[x]) for x in (rr,rr&0xc1,rr&0xf2,rr&0x23))
            if k2==key2: r=rr; break
        print(f'  mod: 96 нулей={zeros} mod32={mod[96:].hex()}')
        print(f'  GG нули={all(b==0 for b in gg)} key2={key2.hex()} LUT-r={r}')
        print(f'  loginType={lt} B0..B2={B.hex()} extra4={extra.hex()}')
        # authdSession: байты 5..9
        print(f'  authdSession bytes5-9={p[5:9].hex()}')

#!/usr/bin/env python3
"""DFS unscramble paddb (векторное байтовое накопление) с отсечкой mod-окна off=9."""
import struct, sys
sys.path=[p for p in sys.path if p.rstrip('/\\') not in ('/tmp','')]
import blowfish

CAP='/home/user/STELGEN/projects/aion_server_2026-10-02/AION-7.7-EUR-SERVER/nextgen/aion-gate/testdata/handshake-capture-20261003.txt'
LUT=open('/home/user/STELGEN/projects/aion_server_2026-10-02/AION-7.7-EUR-SERVER/nextgen/aion-gate/internal/proto/testdata/gate-lut.bin','rb').read()
lut=[struct.unpack('<I',LUT[i*4:i*4+4])[0] for i in range(256)]
KEY1=bytes.fromhex('6b60cb5b82ce90b1cc2b6c556c6c6c6c')

def ecb_dec(data):
    c=blowfish.Cipher(KEY1)
    return b''.join(next(c.decrypt_ecb(data[o:o+8])) for o in range(0,len(data),8))

def inv_mod(m):
    b=bytearray(m)
    for i in range(0x40,0x80): b[i]^=b[i-0x40]
    for i in range(4): b[0x0d+i]^=b[0x34+i]
    for i in range(0x40): b[i]^=b[0x40+i]
    for i in range(4): b[i],b[0x4d+i]=b[0x4d+i],b[i]
    return bytes(b)

def solve8(a,y):
    cur=[(0,0)]
    for i in range(8):
        ai=(a>>i)&1; yi=(y>>i)&1; nxt=[]
        for x,c in cur:
            for xb in (0,1):
                s=ai+xb+c
                if (xb^(s&1))==yi: nxt.append((x|(xb<<i),s>>1))
        cur=nxt
    return sorted(set(x for x,c in cur))

def dfs(new, results, limit=40):
    n=len(new)//4
    dw=[new[i*4:i*4+4] for i in range(n)]
    acc=[0,0,0,0]
    acc_chosen=[]
    def rec(k):
        if len(results)>=limit: return
        if k==n:
            results.append(list(acc_chosen)); return
        if k==0:
            acc_chosen.append(dw[0])
            for b in range(4): acc[b]=(acc[b]+dw[0][b])&0xff
            rec(k+1)
            for b in range(4): acc[b]=(acc[b]-dw[0][b])&0xff
            acc_chosen.pop(); return
        cands=[solve8(acc[b], dw[k][b]) for b in range(4)]
        if any(len(c)==0 for c in cands): return
        for o0 in cands[0]:
            for o1 in cands[1]:
                for o2 in cands[2]:
                    for o3 in cands[3]:
                        old=bytes((o0,o1,o2,o3))
                        acc_chosen.append(old)
                        for b in range(4): acc[b]=(acc[b]+old[b])&0xff
                        rec(k+1)
                        for b in range(4): acc[b]=(acc[b]-old[b])&0xff
                        acc_chosen.pop()
                        if len(results)>=limit: return
    rec(0)

sess=[bytes.fromhex(l.split(' len=194 ')[1].strip()) for l in open(CAP) if ' G->C len=194 ' in l]
idx=1
dec=ecb_dec(sess[idx][2:])
res=[]
dfs(dec[:184], res)
print('решений:', len(res))
found=0
for plain_dws in res:
    p=b''.join(plain_dws)
    m=inv_mod(p[9:137])
    if not all(b==0 for b in m[:96]): continue
    gg=p[137:153]; key2=p[153:169]
    r=None
    for rr in range(256):
        k2=b''.join(struct.pack('<I',lut[x]) for x in (rr,rr&0xc1,rr&0xf2,rr&0x23))
        if k2==key2: r=rr; break
    if r is None: continue
    found+=1
    print(f'✅ plaintext[0]={p[0]:02x} sid?={p[1:5].hex()} authd?={p[5:9].hex()}')
    print(f'   mod32={m[96:].hex()} gg0={all(b==0 for b in gg)} key2r={r}')
    print(f'   lt={p[169]} B={p[170:173].hex()} extra4={p[173:177].hex()}')
    if found>=3: break
print('структурных решений:', found)

#!/usr/bin/env python3
"""Перебор вариантов скрамбла: fwd/bwd x dword/byte x with-self/without-self."""
import struct, sys
sys.path=[p for p in sys.path if p.rstrip('/\\') not in ('/tmp','')]
import blowfish

CAP='/home/user/STELGEN/projects/aion_server_2026-10-02/AION-7.7-EUR-SERVER/nextgen/aion-gate/testdata/handshake-capture-20261003.txt'
LUT=open('/home/user/STELGEN/projects/aion_server_2026-10-02/AION-7.7-EUR-SERVER/nextgen/aion-gate/internal/proto/testdata/gate-lut.bin','rb').read()
lut=[struct.unpack('<I',LUT[i*4:i*4+4])[0] for i in range(256)]
KEY1=bytes.fromhex('6b60cb5b82ce90b1cc2b6c556c6c6c6c')
M=0xffffffff

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

def solve8(a,y):  # 8-бит: x ^ (a+x) == y
    cur=[(0,0)]
    for i in range(8):
        ai=(a>>i)&1; yi=(y>>i)&1; nxt=[]
        for x,c in cur:
            for xb in (0,1):
                s=ai+xb+c
                if (xb^(s&1))==yi: nxt.append((x|(xb<<i),s>>1))
        cur=nxt
    return sorted(set(x for x,c in cur))

def check(p,tag):
    mod=inv_mod(p[9:137])
    ok=all(b==0 for b in mod[:96])
    gg=all(b==0 for b in p[137:153])
    r=None
    for rr in range(256):
        k2=b''.join(struct.pack('<I',lut[x]) for x in (rr,rr&0xc1,rr&0xf2,rr&0x23))
        if k2==p[153:169]: r=rr; break
    stat=f'{tag}: mod0={ok} gg0={gg} key2r={r} p0={p[0]:02x} lt={p[169]} B={p[170:173].hex()} extra={p[173:177].hex()} a5_9={p[5:9].hex()}'
    print(stat)
    return ok and gg and r is not None

sess=[bytes.fromhex(l.split(' len=194 ')[1].strip()) for l in open(CAP) if ' G->C len=194 ' in l]

def try_all(idx):
    pkt=sess[idx]; dec=ecb_dec(pkt[2:])
    nb=184  # байт данных (без csum)
    # 1) dword without-self fwd: old[k]=new[k]^S; S+=old
    dw=[struct.unpack('<I',dec[i*4:i*4+4])[0] for i in range(45)]
    S=0; pl=[]
    for x in dw: o=x^S; pl.append(o); S=(S+o)&M
    check(b''.join(struct.pack('<I',v) for v in pl),'dword-noself-fwd')
    # 2) byte without-self fwd
    S=0; pb=bytearray()
    for x in dec[:nb]: o=x^S; pb.append(o); S=(S+o)&0xff
    check(bytes(pb),'byte-noself-fwd')
    # 3) byte with-self fwd (DP по байтам)
    pb=bytearray(); S=0; ok=True
    for x in dec[:nb]:
        c=solve8(S,x)
        if len(c)!=1: ok=False; pb+=bytes([c[0] if c else 0]); S=(S+c[0] if c else S)&0xff
        else: pb.append(c[0]); S=(S+c[0])&0xff
    if ok: check(bytes(pb),'byte-self-fwd')
    else: print('byte-self-fwd: неоднозначности')
    # 4) dword without-self BWD: идём с конца: S=сумма уже решённых (справа)
    S=0; pl=[0]*45
    for k in range(44,-1,-1):
        o=dw[k]^S; pl[k]=o; S=(S+o)&M
    check(b''.join(struct.pack('<I',v) for v in pl),'dword-noself-bwd')
    # 5) byte without-self BWD
    S=0; pb=bytearray(nb)
    for k in range(nb-1,-1,-1):
        o=dec[k]^S; pb[k]=o; S=(S+o)&0xff
    check(bytes(pb),'byte-noself-bwd')

for i in (0,1):
    try_all(i)

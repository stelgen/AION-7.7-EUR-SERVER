#!/usr/bin/env python3
"""Поиск смещения mod-поля: inv_mod(p[off:off+128])[:96] == нули, для разных моделей unscramble."""
import struct, sys
sys.path=[p for p in sys.path if p.rstrip('/\\') not in ('/tmp','')]
import blowfish

CAP='/home/user/STELGEN/projects/aion_server_2026-10-02/AION-7.7-EUR-SERVER/nextgen/aion-gate/testdata/handshake-capture-20261003.txt'
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

sess=[bytes.fromhex(l.split(' len=194 ')[1].strip()) for l in open(CAP) if ' G->C len=194 ' in l]

def models(dec):
    nb=184
    out={}
    dw=[struct.unpack('<I',dec[i*4:i*4+4])[0] for i in range(45)]
    S=0; pl=[]
    for x in dw: o=x^S; pl.append(o); S=(S+o)&M
    out['dword-noself']=b''.join(struct.pack('<I',v) for v in pl)
    S=0; pb=bytearray()
    for x in dec[:nb]: o=x^S; pb.append(o); S=(S+o)&0xff
    out['byte-noself']=bytes(pb)
    return out

for idx in (0,1,2,3):
    dec=ecb_dec(sess[idx][2:])
    for tag,p in models(dec).items():
        hits=[]
        for off in range(0,49):
            m=inv_mod(p[off:off+128])
            if all(b==0 for b in m[:96]):
                hits.append((off,m[96:]))
        if hits:
            for off,m32 in hits:
                print(f'sess{idx} {tag}: off={off} mod32={m32.hex()}')
        else:
            print(f'sess{idx} {tag}: mod-окно не найдено (off 0..48)')

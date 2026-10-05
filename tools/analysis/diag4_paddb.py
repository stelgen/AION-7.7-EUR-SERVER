#!/usr/bin/env python3
"""Модель скрамбла с SIMD-накоплением (paddb/paddw, без межбайтовых переносов)."""
import struct, sys
sys.path=[p for p in sys.path if p.rstrip('/\\') not in ('/tmp','')]
import blowfish

CAP='/home/user/STELGEN/projects/aion_server_2026-10-02/AION-7.7-EUR-SERVER/nextgen/aion-gate/testdata/handshake-capture-20261003.txt'
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

def paddb_unscramble(new):  # with-self, байтовое padd-накопление
    out=bytearray(); S=[0]*len(new)  # побайтовый аккумулятор
    acc=bytearray(len(new))
    for k in range(len(new)):
        # old_b = new_b ^ S_b (S_b = acc байтово); уравнение x ^ (acc_b + x) = new_b
        c=solve8(acc[k], new[k])
        if len(c)!=1: return None, k, len(c)
        o=c[0]
        out.append(o)
        for i in range(len(new)):  # paddb: acc += old (байтово, весь вектор)
            acc[i]=(acc[i]+o)&0xff
        # нет: paddb на ВСЕМ векторе — acc скалярно по позициям вектора, а old — ВЕСЬ вектор? нет:
    return None,None,None

def paddb_vec(new):  # старый[k] — ВЕСЬ вектор данных? скрамбл DWORD LE: k — номер DWORD, аккумулятор — вектор из 4 байт
    # data как dword-массив; S — 4-байтовый вектор; S_k = S_{k-1} +_paddb old[k]; new[k]=old[k]^S_k
    n=len(new)//4
    dw=[new[i*4:i*4+4] for i in range(n)]
    S=bytearray(4); res=[]
    for k in range(n):
        if k==0:
            res.append(dw[0]); S=bytearray(dw[0]); continue
        # уравнение побайтово: x_b ^ (S_b + x_b) = new_b
        old=bytearray(4); ok=True
        for b in range(4):
            c=solve8(S[b], dw[k][b])
            if len(c)!=1: ok=False; break
            old[b]=c[0]
        if not ok: return None,k
        res.append(bytes(old))
        for b in range(4): S[b]=(S[b]+old[b])&0xff
    return b''.join(res),None

def paddw_vec(new):
    n=len(new)//4
    dw=[struct.unpack('<H',new[i*2:i*2+2])[0] for i in range(len(new)//2)]
    S=[0,0]; res=[]
    words=[struct.unpack('<H',new[i*2:i*2+2])[0] for i in range(len(new)//2)]
    # с self: old_w ^ (S_w + old_w) = new_w
    out=bytearray()
    Sw=[0,0]
    for k in range(len(words)):
        if k==0:
            out+=struct.pack('<H',words[0]); Sw=[words[0]&0xffff,(words[0]>>16)&0xffff] if False else [struct.unpack('<H',new[0:2])[0],struct.unpack('<H',new[2:4])[0]]
            continue
        o=[0,0]; ok=True
        for b in range(2):
            c=solve8(Sw[b],words[k*2+b]) if False else None
        # 16-бит DP нужен — пропустим (пока paddb)
        return None
    return bytes(out),None

sess=[bytes.fromhex(l.split(' len=194 ')[1].strip()) for l in open(CAP) if ' G->C len=194 ' in l]
for idx in (0,1,2):
    dec=ecb_dec(sess[idx][2:])
    p,bad=paddb_vec(dec[:184])
    if p is None:
        print(f'sess{idx}: paddb неоднозначность на dword {bad}'); continue
    print(f'sess{idx}: p0={p[0]:02x}')
    hits=[]
    for off in range(0,49):
        m=inv_mod(p[off:off+128])
        if all(b==0 for b in m[:96]): hits.append(off)
    print(f'  mod-окно: {hits}')
    if hits:
        off=hits[0]; m=inv_mod(p[off:off+128])
        print(f'  mod32={m[96:].hex()}')
        print(f'  до окна={p[:off].hex()} после={p[off+128:off+160].hex()}')

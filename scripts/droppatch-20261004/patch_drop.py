# -*- coding: utf-8 -*-
# AION drop patch: A3(K multiplier in CDI) + B(adj 1->100) + C(direct prob xK) + D(cash xM)
# READ source, write temp, then in-place rewrite preserving inode/hardlink.
# usage: patch_drop.py cdi <file> | patch_drop.py npc <file> [--apply]
# default = DRY (counts only, writes nothing). --apply writes.
import io, re, sys, json, os

K=200; FLOOR=5000000; MAXP=10000000; CASH_MULT=10
APPLY = '--apply' in sys.argv
path=sys.argv[2]

def openx(p):
    h=open(p,'rb').read(2)
    enc='utf-16' if h in (b'\xff\xfe',b'\xfe\xff') else 'utf-8'
    return io.open(p,'r',encoding=enc,errors='replace',newline=''), enc

re_prob=re.compile(r'<prob>(\d+)</prob>')
re_adj=re.compile(r'<common_drop_adjustment>(\d+)</common_drop_adjustment>')
re_min=re.compile(r'<min_cash_amount>(\d+)</min_cash_amount>')
re_max=re.compile(r'<max_cash_amount>(\d+)</max_cash_amount>')
re_cdp=re.compile(r'<cash_drop_prob>(\d+)</cash_drop_prob>')

cnt={'prob_x':0,'prob_clamp':0,'prob_skip_ge5M':0,'prob_quest_skip':0,'adj1':0,'adj_seen':0,'cash_blocks':0,'cash_min':0,'cash_max':0}
re_item=re.compile(r'<item>(.*?)</item>')
re_questlike=re.compile(r'(?<!con)quest|q_',re.I)

def mult_prob(m):
    v=int(m.group(1))
    if v>=FLOOR:
        cnt['prob_skip_ge5M']+=1
        return m.group(0)
    nv=v*K
    if nv>MAXP: nv=MAXP; cnt['prob_clamp']+=1
    cnt['prob_x']+=1
    return '<prob>%d</prob>'%nv

def proc_cdi(r,w):
    in_items=False; d=[]
    for line in r:
        s=line.strip()
        if s=='<items>': in_items=True
        elif s=='</items>': in_items=False
        elif in_items:
            if s=='<data>': d=[line]; continue
            if d:
                d.append(line)
                if s=='</data>':
                    block=''.join(d); d=[]
                    im=re_item.search(block)
                    key=im.group(1) if im else ''
                    if re_questlike.search(key):
                        cnt['prob_quest_skip']+=len(re_prob.findall(block))
                    else:
                        block=re_prob.sub(mult_prob,block)
                    w.write(block)
                continue
            w.write(line); continue
        w.write(line)

def proc_npc(r,w):
    block=[]
    state='pre'
    def flush():
        if not block: return
        text=''.join(block)
        # C: direct item probs (inside <items>...</items>)
        def items_sub(m):
            return re_prob.sub(mult_prob, m.group(0))
        text=re.sub(r'<items>.*?</items>', items_sub, text, flags=re.S)
        # B: adj exactly 1 -> 100
        def adj_sub(m):
            cnt['adj_seen']+=1
            if m.group(1)=='1':
                cnt['adj1']+=1
                return '<common_drop_adjustment>100</common_drop_adjustment>'
            return m.group(0)
        text=re_adj.sub(adj_sub,text)
        # D: cash min/max xM if prob is ~100%
        cd=re_cdp.search(text)
        if cd:
            pv=int(cd.group(1))
            if pv in (9999998,9999999,10000000):
                mn=re_min.search(text); mx=re_max.search(text)
                if (mn and int(mn.group(1))>0) or (mx and int(mx.group(1))>0):
                    def mm_min(m):
                        v=int(m.group(1))
                        if v>0: cnt['cash_min']+=1
                        return '<min_cash_amount>%d</min_cash_amount>'%(v*CASH_MULT)
                    def mm_max(m):
                        v=int(m.group(1))
                        if v>0: cnt['cash_max']+=1
                        return '<max_cash_amount>%d</max_cash_amount>'%(v*CASH_MULT)
                    text=re_min.sub(mm_min,text, count=1)
                    text=re_max.sub(mm_max,text, count=1)
                    cnt['cash_blocks']+=1
        w.write(text)
        block.clear()
    for line in r:
        s=line.strip()
        if state=='pre':
            w.write(line)
            if s=='<npc>': state='in'
            continue
        block.append(line)
        if s=='</npc>':
            flush(); state='pre'
    flush()

mode=sys.argv[1]
r,enc=openx(path)
if APPLY:
    tmp=path+'.patched'
    w=io.open(tmp,'w',encoding=enc,newline='')
else:
    w=io.open(os.devnull,'w',encoding=enc,newline='')
if mode=='cdi': proc_cdi(r,w)
else: proc_npc(r,w)
w.close(); r.close()
if APPLY:
    size_new=os.path.getsize(tmp)
    # in-place rewrite preserving inode
    with open(tmp,'rb') as g, open(path,'r+b') as f:
        while True:
            chunk=g.read(1<<22)
            if not chunk: break
            f.write(chunk)
        f.truncate()
    os.remove(tmp)
    cnt['new_file_size']=size_new
cnt['mode']=mode; cnt['file']=path; cnt['apply']=APPLY
print(json.dumps(cnt,ensure_ascii=False))

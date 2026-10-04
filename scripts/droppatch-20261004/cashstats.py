# -*- coding: utf-8 -*-
# DRY-RUN cash: correct triple parse (min,max,prob buffered per items_info)
import io, re, sys, json, collections
def openx(p):
    h=open(p,'rb').read(2)
    return io.open(p,'r',encoding='utf-16' if h in (b'\xff\xfe',b'\xfe\xff') else 'utf-8',errors='replace')
buf=None; stats=collections.Counter(); mins=0; maxs=0
lev=0; examples=[]
for line in openx(sys.argv[1]):
    l=line.strip()
    if buf is None:
        if l=='<items_info>': buf=[]; lev=0
        continue
    if l=='</items_info>':
        prob=None; mn=0; mx=0
        for x in buf:
            m=re.search(r'<min_cash_amount>(\d+)</min_cash_amount>',x)
            if m: mn=int(m.group(1))
            m=re.search(r'<max_cash_amount>(\d+)</max_cash_amount>',x)
            if m: mx=int(m.group(1))
            m=re.search(r'<cash_drop_prob>(\d+)</cash_drop_prob>',x)
            if m: prob=int(m.group(1))
        if prob is not None:
            if prob in (9999998,9999999,10000000):
                stats['prob100_'+('sum0' if mn==0 and mx==0 else 'sumset')]+=1
                if mn>0 or mx>0:
                    mins+=1; maxs+=1
                    if len(examples)<6: examples.append({'prob':prob,'min':mn,'max':mx})
            elif prob>0: stats['prob_mid']+=1
        buf=None; continue
    m=re.search(r'<level>(\d+)</level>',l)
    if m and lev==0: lev=int(m.group(1))
    buf.append(l)
print(json.dumps({'file':sys.argv[1],'prob100_sumset':stats.get('prob100_sumset',0),
    'prob100_sum0':stats.get('prob100_sum0',0),'prob_mid':stats.get('prob_mid',0),
    'examples':examples},ensure_ascii=False))

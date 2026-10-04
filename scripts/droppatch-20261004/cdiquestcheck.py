# -*- coding: utf-8 -*-
# CDI quest-key scan: any item keys looking like quests
import io, re, sys, json
def openx(p):
    h=open(p,'rb').read(2)
    return io.open(p,'r',encoding='utf-16' if h in (b'\xff\xfe',b'\xfe\xff') else 'utf-8',errors='replace')
import collections
hits=collections.Counter(); total=0; in_items=False
for line in openx(sys.argv[1]):
    l=line.strip()
    if l=='<items>': in_items=True; continue
    if l=='</items>': in_items=False; continue
    if in_items:
        m=re.match(r'<item>(.*?)</item>',l)
        if m:
            total+=1
            k=m.group(1)
            if re.search(r'quest|q_|_qv|IDRAKSHA|WRAP_QUE',k,re.I): hits[k]+=1
print(json.dumps({'item_keys_total':total,'questlike':len(hits),'top':dict(list(hits.items())[:20])},ensure_ascii=False))

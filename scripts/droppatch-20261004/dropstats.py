# -*- coding: utf-8 -*-
# READ-ONLY drop statistics
import io, re, sys, json, collections

def openx(p):
    h=open(p,'rb').read(2)
    return io.open(p,'r',encoding='utf-16' if h in (b'\xff\xfe',b'\xfe\xff') else 'utf-8',errors='replace')

mode=sys.argv[1]; path=sys.argv[2]

if mode=='cdi':
    total=0; p500=0; pools=0; pools_aff=0; aff_rec=0
    b=collections.Counter()
    cur=None; in_items=False
    for line in openx(path):
        l=line.strip()
        if l=='<commondrop>': cur=0; continue
        if cur is None: continue
        if l=='</commondrop>':
            pools+=1
            if cur>0: pools_aff+=1; aff_rec+=cur
            cur=None; continue
        if l=='<items>': in_items=True; continue
        if l=='</items>': in_items=False; continue
        if not in_items: continue
        m=re.search(r'<prob>(\d+)</prob>',l)
        if m:
            v=int(m.group(1)); total+=1; cur+=1
            if v==500000: p500+=1
            if v>10000000: k='gt10M'
            elif v==10000000: k='=10M'
            elif v>=5000000: k='5M-10M'
            elif v>=2500000: k='2.5M-5M'
            elif v>=1000000: k='1M-2.5M'
            elif v>=500000: k='500k-1M'
            elif v<50000: k='lt50k'
            else: k='50k-500k'
            b[k]+=1
    out={'mode':mode,'file':path,'total_records':total,'records_prob_500000':p500,
         'buckets':dict(b),'pools_total':pools,'pools_with_rec_lt_5M':pools_aff,
         'records_lt_5M_in_those_pools':aff_rec}
    print(json.dumps(out,ensure_ascii=False))

elif mode=='npc':
    # per npc file: cash stats + direct items <prob> distribution
    cash_probs=collections.Counter(); cash_min=collections.Counter(); cash_max=collections.Counter()
    npcs=0; npcs_ii=0
    item_probs=collections.Counter(); direct_records=0
    cur=None; ii=None; in_items=False; in_dg=False; in_cdx=False; d=None
    for line in openx(path):
        l=line.strip()
        if l=='<npc>':
            npcs+=1; ii=None; in_items=in_dg=in_cdx=False; d=None
            continue
        if cur is not None and l=='</npc>': cur=None; continue
        cur=1
        if l=='<items_info>': ii={'cash':None}; npcs_ii+=1; continue
        if ii is None: continue
        if l=='</items_info>':
            if ii is not None and ii.get('cash') is not None:
                c=ii['cash']
                cash_probs[c['prob']]+=1
                if c['min'] not in ('0',None): cash_min[c['min']]+=1
                if c['max'] not in ('0',None): cash_max[c['max']]+=1
            ii=None; continue
        if l=='<drop_groups>': in_dg=True; continue
        if l=='</drop_groups>': in_dg=False; continue
        if l=='<items>': in_items=True; continue
        if l=='</items>': in_items=False; continue
        if l=='<common_drops>': in_cdx=True; continue
        if l=='</common_drops>': in_cdx=False; continue
        if l=='<data>': d={}; continue
        if l=='</data>': d=None; continue
        if ii['cash'] is None:
            m=re.search(r'<cash_drop_prob>(-?\d+)</cash_drop_prob>',l)
            if m:
                ii['cash']={'prob':m.group(1),'min':'0','max':'0'}; continue
        m=re.search(r'<min_cash_amount>(-?\d+)</min_cash_amount>',l)
        if m and ii['cash'] is not None: ii['cash']['min']=m.group(1); continue
        m=re.search(r'<max_cash_amount>(-?\d+)</max_cash_amount>',l)
        if m and ii['cash'] is not None: ii['cash']['max']=m.group(1); continue
        if in_items and d is not None:
            m=re.search(r'<prob>(\d+)</prob>',l)
            if m:
                v=int(m.group(1)); direct_records+=1
                if v>10000000: k='gt10M'
                elif v==10000000: k='=10M'
                elif v>=5000000: k='5M-10M'
                elif v>=2500000: k='2.5M-5M'
                elif v>=1000000: k='1M-2.5M'
                elif v>=500000: k='500k-1M'
                elif v<50000: k='lt50k'
                else: k='50k-500k'
                item_probs[k]+=1
    for npc_ii in [None]:
        pass
    out={'mode':mode,'file':path,'npcs':npcs,'npcs_with_items_info':npcs_ii,
         'direct_item_records':direct_records,'direct_prob_buckets':dict(item_probs),
         'cash_drop_prob_dist':dict(cash_probs.most_common(12)),
         'cash_probs_top':dict(cash_probs.most_common(12)),
         'cash_prob_nonzero_npcs':sum(v for k,v in cash_probs.items() if k not in ('0',None)),
         'cash_min_top':dict(cash_min.most_common(8)),'cash_max_top':dict(cash_max.most_common(8))}
    print(json.dumps(out,ensure_ascii=False))

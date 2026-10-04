# -*- coding: utf-8 -*-
# DRY-RUN (no writes): plan scenarios for A3/B/C/D
import io, re, sys, json, collections

def openx(p):
    h=open(p,'rb').read(2)
    return io.open(p,'r',encoding='utf-16' if h in (b'\xff\xfe',b'\xfe\xff') else 'utf-8',errors='replace')

mode=sys.argv[1]; path=sys.argv[2]
KS=[50,100,200,300,500]
FLOOR=5000000; MAX=10000000

if mode=='cdi':
    probs=[]
    cur=None; in_items=False
    for line in openx(path):
        l=line.strip()
        if l=='<commondrop>': cur=None; in_items=False; continue
        if l=='<items>': in_items=True; continue
        if l=='</items>': in_items=False; continue
        if cur is None:
            if l=='<commondrop>': continue
            if in_items:
                m=re.search(r'<prob>(\d+)</prob>',l)
                if m: probs.append(int(m.group(1)))
            # detect commondrop start robustly
            if l=='<commondrop>': cur=1
            continue
    # NOTE: simpler robust parse below
    if not probs:
        in_items=False
        for line in openx(path):
            l=line.strip()
            if l=='<items>': in_items=True; continue
            if l=='</items>': in_items=False; continue
            if in_items:
                m=re.search(r'<prob>(\d+)</prob>',l)
                if m: probs.append(int(m.group(1)))
    probs.sort()
    n=len(probs)
    def pct(p): return probs[min(n-1,int(n*p/100.0))]
    scen={}
    for K in KS:
        affected=sum(1 for v in probs if v<FLOOR)
        become_ge50=sum(1 for v in probs if v<FLOOR and v*K>=FLOOR)
        clamped=sum(1 for v in probs if v<FLOOR and v*K>MAX)
        scen[str(K)]={'affected':affected,'become_ge50pct':become_ge50,'clamped_to_100pct':clamped}
    print(json.dumps({'mode':'cdi-dry','file':path,'records':n,
        'p10':pct(10),'p25':pct(25),'p50':pct(50),'p75':pct(75),'p90':pct(90),'p99':pct(99),
        'ge5M_now':sum(1 for v in probs if v>=FLOOR),'scenarios':scen},ensure_ascii=False))

elif mode=='npc':
    adj=collections.Counter(); adj1_pools=set(); cd_names=set()
    in_cd=False; d=None; cur_cd=None
    in_items=False; item_probs=[]; item_lt5M=0
    lev=None; lev_have=0; npcs=0
    cash_cand=0; cash_prob100=0; cash_minmax_nonzero=0
    cur_prob=None
    for line in openx(path):
        l=line.strip()
        if l=='<npc>': npcs+=1; lev=None; continue
        m=re.search(r'<level>(\d+)</level>',l)
        if m and lev is None: lev=int(m.group(1)); lev_have+=1
        if l=='<common_drops>': in_cd=True; continue
        if l=='</common_drops>': in_cd=False; continue
        if in_cd:
            if l=='<data>': d={}; continue
            if l=='</data>':
                if d is not None:
                    a=d.get('common_drop_adjustment')
                    if a is not None:
                        adj[a]+=1
                        if a=='1' and d.get('common_drop'): adj1_pools.add(d['common_drop'])
                    if d.get('common_drop'): cd_names.add(d['common_drop'])
                d=None; continue
            if d is not None:
                for t in ('common_drop','common_drop_adjustment'):
                    m=re.search(r'<%s>(.*?)</%s>'%(t,t),l)
                    if m and t not in d: d[t]=m.group(1)
            continue
        if l=='<items>': in_items=True; continue
        if l=='</items>': in_items=False; continue
        if in_items:
            m=re.search(r'<prob>(\d+)</prob>',l)
            if m:
                v=int(m.group(1)); item_probs.append(v)
                if v<FLOOR: item_lt5M+=1
            continue
        # cash
        m=re.search(r'<cash_drop_prob>(\d+)</cash_drop_prob>',l)
        if m:
            cur_prob=m.group(1)
            if cur_prob in ('9999998','9999999','10000000'): cash_prob100+=1
            continue
        if cur_prob is not None:
            m=re.search(r'<min_cash_amount>(\d+)</min_cash_amount>',l)
            if m:
                if m.group(1)!='0': cash_minmax_nonzero+=1
                if cur_prob in ('9999998','9999999','10000000') and m.group(1)=='0': cash_cand+=1
                continue
    item_probs.sort(); n=len(item_probs)
    def pct(p): return item_probs[min(n-1,int(n*p/100.0))] if n else 0
    qlike=[p for p in adj1_pools if re.search(r'quest|q_|IDRAKSHA',p,re.I)]
    print(json.dumps({'mode':'npc-dry','file':path,'npcs':npcs,'lev_known':lev_have,
        'adj_dist':dict(adj.most_common(10)),'adj_total':sum(adj.values()),
        'adj1_pools':len(adj1_pools),'adj1_pool_questlike':qlike[:12],'adj1_pool_questlike_cnt':len(qlike),
        'pools_total_names':len(cd_names),
        'direct_records':n,'direct_lt5M':item_lt5M,'direct_p50':pct(50),'direct_p75':pct(75),
        'cash_prob_ge_9999998':cash_prob100,'cash_sum0_candidates':cash_cand,
        'cash_minmax_nonzero_somewhere':cash_minmax_nonzero},ensure_ascii=False))

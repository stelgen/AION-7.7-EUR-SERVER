# -*- coding: utf-8 -*-
# READ-ONLY: common_drop_adjustment distribution per npc file
import io, re, sys, json, collections
def openx(p):
    h=open(p,'rb').read(2)
    return io.open(p,'r',encoding='utf-16' if h in (b'\xff\xfe',b'\xfe\xff') else 'utf-8',errors='replace')
adj=collections.Counter(); in_cd=False; npcs=0; in_ii=False
for line in openx(sys.argv[1]):
    l=line.strip()
    if l=='<npc>': npcs+=1; continue
    if l=='<common_drops>': in_cd=True; continue
    if l=='</common_drops>': in_cd=False; continue
    if in_cd:
        m=re.search(r'<common_drop_adjustment>(-?\d+)</common_drop_adjustment>',l)
        if m: adj[m.group(1)]+=1
out={'file':sys.argv[1],'npcs':npcs,'adj_dist':dict(adj.most_common(15)),'adj_total':sum(adj.values())}
print(json.dumps(out,ensure_ascii=False))

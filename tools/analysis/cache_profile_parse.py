import re, glob, os
D="/home/user/STELGEN/projects/aion_rev_2026-10-05/artifacts/pdb-big/CacheD64/CacheServer/log"
f=os.path.join(D,"2026-10-07.profile")
lines=open(f,encoding="utf-8",errors="replace").read().splitlines()
sec_re=re.compile(r"id\s+\|\s+(\S+) Protocol")
row_re=re.compile(r"^\s*(\d+)\s+\|\s+(\S+)\s+\|\s+(\d+)\s+\|\s+([\d.]+)")
cur=None; tables={}
for ln in lines:
    m=sec_re.search(ln)
    if m:
        cur=m.group(1)
        if cur not in tables:
            tables[cur]={}
        continue
    if cur:
        m=row_re.match(ln)
        if m:
            tables[cur][int(m.group(1))]=m.group(2)
out=open("/tmp/cache-opcode-map.md","w")
out.write("# CacheD64 opcode maps из .profile (DBProfiler) — живая нумерация\n\nИсточник: `log/*.profile` (первый полный цикл в файле). Формат: `id | имя`.\n\n")
for sec in ["DB2Server","Server2DB","Log2Server","Server2Log","IC2DB","DB2IC","NPRelay2Server","Server2NPRelay"]:
    if sec not in tables:
        continue
    out.write(f"## {sec} ({len(tables[sec])} ops)\n\n| id | opcode |\n|---|---|\n")
    for i in sorted(tables[sec]):
        out.write(f"| {i} | {tables[sec][i]} |\n")
    out.write("\n")
out.close()
print("sections:", {k:len(v) for k,v in tables.items()})
agg={}
for fp in glob.glob(os.path.join(D,"*.profile")):
    cur=None
    for ln in open(fp,encoding="utf-8",errors="replace"):
        m=sec_re.search(ln)
        if m:
            cur=m.group(1)
            continue
        if cur:
            m=row_re.match(ln)
            if m:
                name=m.group(2); cnt=int(m.group(3)); tr=float(m.group(4))
                a=agg.setdefault((cur,name),[0,0.0]); a[0]+=cnt; a[1]+=tr
rows=[(c,tr,s,n) for (s,n),(c,tr) in agg.items() if c>0]
rows.sort(reverse=True)
print("== TOP used ==")
for c,tr,s,n in rows[:60]:
    print(f"{s:15s} {n:45s} count={c:>9} kb={tr:.0f}")
print("total used pairs:",len(rows))

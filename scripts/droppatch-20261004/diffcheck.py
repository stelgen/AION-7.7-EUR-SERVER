# -*- coding: utf-8 -*-
# diff verify: orig vs patched (line-level)
import io, sys, json
def openx(p):
    h=open(p,'rb').read(2)
    return io.open(p,'r',encoding='utf-16' if h in (b'\xff\xfe',b'\xfe\xff') else 'utf-8',errors='replace')
a=openx(sys.argv[1]); b=openx(sys.argv[2])
diffs=[]; la=lb=0
import itertools
for x,y in itertools.zip_longest(a,b):
    la+=1
    if x!=y:
        lb+=1
        if len(diffs)<12:
            diffs.append({'orig':(x or '')[:180],'new':(y or '')[:180]})
print(json.dumps({'orig_lines':la,'new_lines':lb,'diffs':diffs},ensure_ascii=False))

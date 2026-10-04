import sys

files = [
    (r'D:\AION_LIVE_SERVER\AuthGateD\etc\config.txt', r'D:\AION_LIVE_SERVER\AuthGateD\etc\config.txt.bak-back2106',
     [(b'serverPort = 2107', b'serverPort = 2106')]),
    (r'D:\AION_LIVE_SERVER\MainServer\config.xml', r'D:\AION_LIVE_SERVER\MainServer\config.xml.bak-restore7777',
     [(b'<clientAcceptPort>7778</clientAcceptPort>', b'<clientAcceptPort>7777</clientAcceptPort>')]),
    (r'D:\AION_LIVE_SERVER\MainServer\common.xml', r'D:\AION_LIVE_SERVER\MainServer\common.xml.bak-pet2107',
     [(b'<petitionServerPort>21055</petitionServerPort>', b'<petitionServerPort>2107</petitionServerPort>')]),
]

for path, bak, subs in files:
    data = open(bak, 'rb').read()
    nonascii = sum(1 for b in data if b > 0x7F)
    hits = []
    for a, b in subs:
        n = data.count(a)
        data = data.replace(a, b)
        hits.append((a.decode(), n))
    open(path, 'wb').write(data)
    print(path, '| nonascii_bytes_in_orig=', nonascii, '| replacements:', hits)

# verify
for path, _, subs in files:
    d = open(path, 'rb').read()
    for a, b in subs:
        print(path, '->', b.decode(), 'present:', d.count(b), '| old gone:', d.count(a) == 0)

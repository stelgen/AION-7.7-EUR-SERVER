# -*- coding: utf-8 -*-
p = r'D:\AION_LIVE_SERVER\AuthGateD-2109\etc\config.txt'
d = open(p, 'rb').read()
d2 = d.replace(b'serverPort = 2106', b'serverPort = 2109')
open(p, 'wb').write(d2)
print('port-replaced:', d != d2)
p = r'C:\Temp\gate.bat'
d = open(p, 'rb').read()
d = d.replace(b'wait 2107', b'wait 2106').replace(b'AuthGateD 2107 not up', b'AuthGateD 2106 not up')
open(p, 'wb').write(d)
print('gate.bat 2107 left:', d.count(b'2107'))

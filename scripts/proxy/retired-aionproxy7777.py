import socket, threading, datetime

LOG = open('C:/Temp/proxylog7777.txt', 'a', buffering=1)

def log(s):
    LOG.write(datetime.datetime.now().strftime('%H:%M:%S') + ' ' + s + '\n')

def pump(src, dst, tag):
    try:
        while True:
            d = src.recv(65536)
            if not d:
                break
            log(tag + ' len=' + str(len(d)) + ' hex=' + d[:128].hex())
            dst.sendall(d)
    except Exception as e:
        log(tag + ' ERR ' + str(e))
    try:
        src.close(); dst.close()
    except Exception:
        pass
    log(tag + ' closed')

def handle(c, addr):
    log('--- new client ' + str(addr))
    g = None
    for t in (('192.168.0.125', 7778), ('127.0.0.1', 7778)):
        try:
            g = socket.create_connection(t, 3)
            log('upstream ' + str(t) + ' OK')
            break
        except Exception as e:
            log('upstream fail ' + str(t) + ' ' + str(e))
            g = None
    if g:
        threading.Thread(target=pump, args=(c, g, 'C->W'), daemon=True).start()
        threading.Thread(target=pump, args=(g, c, 'W->C'), daemon=True).start()

log('=== world proxy started, 7777 -> 7778 ===')
s = socket.socket()
s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
s.bind(('0.0.0.0', 7777))
s.listen(10)
while True:
    c, addr = s.accept()
    threading.Thread(target=handle, args=(c, addr), daemon=True).start()

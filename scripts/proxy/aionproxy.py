import socket, threading, datetime

LOG = open('C:/Temp/proxylog.txt', 'a', buffering=1)

def log(s):
    LOG.write(datetime.datetime.now().strftime('%H:%M:%S') + ' ' + s + '\n')

def pump(src, dst, tag):
    buf = b''
    try:
        while True:
            d = src.recv(65536)
            if not d:
                break
            buf += d
            while len(buf) >= 2:
                ln = int.from_bytes(buf[:2], 'little')
                if ln < 2 or ln > 65000:
                    log(tag + ' BADFRAME raw=' + buf[:64].hex())
                    dst.sendall(buf)
                    buf = b''
                    break
                if len(buf) < ln:
                    break
                frame, buf = buf[:ln], buf[ln:]
                log(tag + ' len=' + str(ln) + ' hex=' + frame.hex())
                dst.sendall(frame)
    except Exception as e:
        log(tag + ' ERR ' + str(e))
    try:
        src.close(); dst.close()
    except Exception:
        pass
    log(tag + ' closed')

def handle(c):
    try:
        g = socket.create_connection(('127.0.0.1', 2107))
        threading.Thread(target=pump, args=(c, g, 'C->G'), daemon=True).start()
        threading.Thread(target=pump, args=(g, c, 'G->C'), daemon=True).start()
    except Exception as e:
        log('handle ERR ' + str(e))

log('=== proxy started, 2106 -> 2107 ===')
s = socket.socket()
s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
s.bind(('0.0.0.0', 2106))
s.listen(10)
while True:
    c, addr = s.accept()
    log('--- new client ' + str(addr))
    threading.Thread(target=handle, args=(c,), daemon=True).start()
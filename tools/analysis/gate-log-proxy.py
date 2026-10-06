#!/usr/bin/env python3
# -*- coding: utf-8 -*-
# gate-log-proxy: listen 2109 -> forward 2106, hex-log of [u16 LE] frames both ways.
# usage: C:\Temp\py\python.exe gate-log-proxy.py <listen_port> <target_port> <logfile>
import socket, struct, threading, sys, time

LISTEN = int(sys.argv[1]) if len(sys.argv) > 1 else 2109
TARGET = int(sys.argv[2]) if len(sys.argv) > 2 else 2106
LOGF   = sys.argv[3] if len(sys.argv) > 3 else r'C:\Temp\gate-proxy.log'
lock = threading.Lock()

def log(msg):
    with lock:
        with open(LOGF, 'a', encoding='ascii', errors='replace') as f:
            f.write(time.strftime('%H:%M:%S ') + msg + '\n')

def pump(src, dst, tag):
    buf = b''
    try:
        while True:
            data = src.recv(65536)
            if not data:
                log(f'{tag} EOF')
                break
            buf += data
            while len(buf) >= 2:
                ln = struct.unpack('<H', buf[:2])[0]
                if ln < 2 or ln > 0x2100:
                    log(f'{tag} RAW-nonframe len-field={ln} hex={buf[:64].hex()}')
                    buf = b''
                    break
                if len(buf) < ln:
                    break
                frame = buf[:ln]
                buf = buf[ln:]
                hexs = frame[:200].hex()
                log(f'{tag} len={ln} hex={hexs}{"..." if ln > 200 else ""}')
            dst.sendall(data)
    except Exception as e:
        log(f'{tag} ERR {e}')
    try:
        dst.close()
    except Exception:
        pass

def handle(cs, addr):
    log(f'--- new client {addr[0]}:{addr[1]}')
    try:
        up = socket.create_connection(('127.0.0.1', TARGET), timeout=5)
    except Exception as e:
        log(f'UPSTREAM FAIL {e}')
        cs.close()
        return
    t1 = threading.Thread(target=pump, args=(up, cs, 'G->C'), daemon=True)
    t2 = threading.Thread(target=pump, args=(cs, up, 'C->G'), daemon=True)
    t1.start(); t2.start(); t1.join(); t2.join()
    log('--- session closed')

srv = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
srv.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
srv.bind(('0.0.0.0', LISTEN))
srv.listen(8)
log(f'proxy started {LISTEN} -> {TARGET}')
while True:
    cs, addr = srv.accept()
    threading.Thread(target=handle, args=(cs, addr), daemon=True).start()
#!/usr/bin/env python3
"""accparse v2 — парсер capture-лога accmirror v1.1 (aion-accache R1/R2.5).

КАНОН WIRE (дизasm OnRead/GetCmd_ACQ + capture 10.10, 0 bad frames):
  [u16 N][u16 cmd][u8 marker][u16 ~cmd][payload]
  N = ПОЛНАЯ длина кадра вкл. len-поле (тело = N-2);
  marker: C2S=0xEB, S2C=0xEC; ~cmd = инверсия cmd.

Использование:
  python3 accparse.py raw.log          # разбор лога accmirror (строки C>/O> hex=)
  python3 accparse.py --hex C.hex O.hex
"""
import re, struct, sys

NAMES = {1:'VERSION',2:'TEST_INSERT',3:'TEST_SELECT',4:'FIRST_LOAD_ACCOUNT_INFO',5:'LOAD_BM_PACK',
6:'UPDATE_BM_PACK',7:'LOAD_TRIAL_ACCOUNT_DATA',8:'UPDATE_TRIAL_ACCOUNT_DATA',9:'SYNC_PACKET_TEST',
10:'SAVE_CAHR_CUSTOM',11:'LOAD_CAHR_CUSTOM',12:'LOAD_CHAR_CUSTOM_BY_ITEM',13:'SAVE_CHAR_CUSTOM_TO_ITEM',
14:'LOG_INFO',15:'CHAR_CREATED',16:'CHAR_LOGIN',17:'CHAR_LOGOUT',18:'CHAR_DELETE',
19:'CHAR_DELETE_COMPLETED',20:'CHAR_INFO_REFRESH',21:'CHAR_LEVEL_CHANGED',22:'GAP(22)',
23:'UPDATE_LOGIN_EVENT_RECORD',24:'DELETE_LOGIN_EVENT_RECORD',25:'UPDATE_HIDDEN_FATIGUE',
26:'LOAD_LUNA',27:'UPDATE_LUNA',28:'CONFIRM_LUNA_REWARD',29:'DECREASE_LUNA_KEY',
30:'UPDATE_BOARD_BM_STATE',31:'ASK_CAN_MAKE_JUMPING_CHARACTER',32:'LOAD_PREV_PLAYTIME',
33:'UPDATE_PREV_PLAYTIME',34:'LOAD_PREV_PLAYTIMES',35:'UPDATE_PREV_PLAYTIMES',
36:'RESET_LUNA_REWARD',37:'TRANSFORM_OPERATION',38:'MONSTER_CORE_UPDATE_VALUE',
39:'MONSTER_CORE_UPGRADE'}

def streams_from_log(path):
    st = {'C': bytearray(), 'O': bytearray()}
    for line in open(path, encoding='utf-8', errors='replace'):
        m = re.match(r'\S+ \S+ (C|O)> .*?hex=([0-9a-f]+)', line)
        if m:
            st[m.group(1)] += bytes.fromhex(m.group(2))
    return st

def parse(buf, label):
    off, bad = 0, 0
    print(f'--- {label} ({len(buf)}B) ---')
    while off + 2 <= len(buf):
        n = struct.unpack_from('<H', buf, off)[0]
        if n < 7 or off + n > len(buf):
            print(f'@{off:4d} SHORT/TAIL: {buf[off:off+32].hex()}')
            break
        body = buf[off+2:off+n]
        cmd = struct.unpack_from('<H', body, 0)[0]
        m = body[2]
        xc = struct.unpack_from('<H', body, 3)[0]
        ok = m in (0xEB, 0xEC) and xc == ((~cmd) & 0xFFFF)
        if not ok:
            bad += 1
        print(f'@{off:4d} {"OK " if ok else "BAD"} cmd={cmd:3d} {NAMES.get(cmd, "?"):28s} '
              f'm={m:#04x} pay={body[5:].hex()}')
        off += n
    print(f'--- {label}: bad={bad} ---')

if __name__ == '__main__':
    if len(sys.argv) >= 2 and sys.argv[1] == '--hex':
        parse(bytes.fromhex(open(sys.argv[2]).read().strip()), 'C>')
        parse(bytes.fromhex(open(sys.argv[3]).read().strip()), 'O>')
    else:
        st = streams_from_log(sys.argv[1])
        parse(bytes(st['C']), 'C> Server64->ACS')
        parse(bytes(st['O']), 'O> ACS->Server64')

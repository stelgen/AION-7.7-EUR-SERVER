import struct, json

C = bytes.fromhex(open('/tmp/c.hex').read())
O = bytes.fromhex(open('/tmp/o.hex').read())

names = {1:'VERSION',2:'TEST_INSERT',3:'TEST_SELECT',4:'FIRST_LOAD_ACCOUNT_INFO',5:'LOAD_BM_PACK',
6:'UPDATE_BM_PACK',7:'LOAD_TRIAL_ACCOUNT_DATA',8:'UPDATE_TRIAL',9:'SYNC_TEST',10:'SAVE_CAHR_CUSTOM',
11:'LOAD_CAHR_CUSTOM',12:'LOAD_CHAR_CUSTOM_BY_ITEM',13:'SAVE_CHAR_CUSTOM_TO_ITEM',14:'LOG_INFO',
15:'CHAR_CREATED',16:'CHAR_LOGIN',17:'CHAR_LOGOUT',18:'CHAR_DELETE',19:'CHAR_DELETE_COMPLETED',
20:'CHAR_INFO_REFRESH',21:'CHAR_LEVEL_CHANGED',23:'UPDATE_LOGIN_EVENT',24:'DELETE_LOGIN_EVENT',
25:'UPDATE_HIDDEN_FATIGUE',26:'LOAD_LUNA',27:'UPDATE_LUNA',28:'CONFIRM_LUNA_REWARD',29:'DECREASE_LUNA_KEY',
30:'UPDATE_BOARD_BM',31:'ASK_JUMPING_CHAR',32:'LOAD_PREV_PLAYTIME',33:'UPDATE_PREV_PLAYTIME',
34:'LOAD_PREV_PLAYTIMES',35:'UPDATE_PREV_PLAYTIMES',36:'RESET_LUNA_REWARD',37:'TRANSFORM_OP',
38:'MONSTER_CORE_UPD',39:'MONSTER_CORE_UPG'}

def parse(buf, label):
    off, out, orphan = 0, [], 0
    while off < len(buf):
        if off+2 > len(buf): break
        ln = struct.unpack_from('<H', buf, off)[0]
        mk = buf[off+2] if off+2 < len(buf) else None
        if mk in (0xEB, 0xEC) and off+5 <= len(buf):
            xlen = struct.unpack_from('<H', buf, off+3)[0]
            if xlen == ((~ln) & 0xFFFF):
                total = ln + 2
                payload = buf[off+5:off+total]
                out.append(('CTRL', ln, mk, payload))
                off += total
                continue
        # RPC
        total = ln + 2
        if off+7 <= len(buf) and total >= 7 and off+total <= len(buf):
            cmd = struct.unpack_from('<H', buf, off+2)[0]
            m = buf[off+4]
            xc = struct.unpack_from('<H', buf, off+5)[0]
            if m in (0xEB, 0xEC) and xc == ((~cmd) & 0xFFFF):
                out.append(('RPC', cmd, m, buf[off+7:off+total]))
                off += total
                continue
        out.append(('ORPHAN', 0, 0, buf[off:off+min(8,len(buf)-off)]))
        orphan += 1
        off += 1
    print(f'--- {label} ({len(buf)}B, orphans={orphan}) ---')
    for i, e in enumerate(out):
        kind = e[0]
        if kind == 'RPC':
            _, cmd, m, pl = e
            print(f'{i:2d} RPC  cmd={cmd:3d} {names.get(cmd,"?"):26s} m={m:#04x} pay={pl.hex()}')
        elif kind == 'CTRL':
            _, ln, m, pl = e
            print(f'{i:2d} CTRL len-2={ln:3d} {"":26s} m={m:#04x} pay={pl.hex()}')
        else:
            print(f'{i:2d} ORPH {e[3].hex()}')

parse(C, 'C> Server64->ACS')
parse(O, 'O> ACS->Server64')

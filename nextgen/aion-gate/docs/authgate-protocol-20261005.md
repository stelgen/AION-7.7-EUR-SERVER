# AuthGateD протокол (реверс-док, 05.10.2026 вечер)

> Статус: **модель шифрования ПОДТВЕРЖДЕНА живым capture** (см. §2 верификация).
> Реверс по PDB publics (1010, `/tmp/authpub.txt`) + дизasm (`tools/analysis/AuthGateD_disasm.asm`, 62k строк)
> + живой capture 03.10 (`/tmp/proxylog.txt`, 150КБ, копия на VM C:\Temp\proxylog.txt).
> Прод-эталон: `tools/analysis/AuthGateD_original.exe` MD5 `fb8e66081543706598989c1b9c69c0bb` == прод.

## 0. Артефакты и адресная карта

- ImageBase 0x400000; .text 0x401000, .rdata 0x42c000, .data 0x437000
- PDB: `~/STELGEN/projects/aion_rev_2026-10-05/artifacts/pdb-small/AION_LIVE_SERVER/AuthGateD/AuthGateD.pdb`
- publics: `python3 tools/analysis/pdbpub.py <pdb> > /tmp/authpub.txt` (1010 записей; в выводе уже VA)
- LUT-таблица 256 DWORD @0x437160 (в exe off 0x37160): `/tmp/gate_lut.bin`
- Конфиг гейта байтово: `/tmp/gate-config.txt` (50914 байт, 48707 не-ASCII CP949)
- Классы: CClientSocket (сессия клиента), CClientPacket (хендлеры), CClientSocketContainer,
  CAuthSocket/CAuthPacket (wire 2110), CAuthGateSvcMgr, CRSAKeyPool, CBlowfishWrapper,
  CCSAuth2 (GameGuard), CBadUser (brute), CIPList (BlockIPs), CConfig

### Ключевые функции (VA)
| VA | Функция |
|---|---|
| 0x406230 | RecvLogin (type-0; RSA-dec + CheckSessionId + сборка authd-пакета) |
| 0x406500/406540/406560 | RecvServerList / RecvLogout / RecvServerListExt |
| 0x406660/4066f0/406730 | RecvGameGuard / RecvUpdateSession / RecvExternalTokenLogin |
| 0x406b20/406c50/406c70/406dc0 | RecvAuxAuthentication / RecvAgreementCheck / RecvAgreement / RecvTokenLogin |
| 0x4061c0 | ByPassPacket |
| 0x4079d0 | CheckSessionId(sid) — m_iSessionId @this+0xfc; 7 call-сайтов |
| 0x407ac0 | GG-auth обработчик после CheckSessionId (16B @data+132 → CCSAuth2) |
| 0x407e80 | CClientSocket::Send(fmt, ...) — шлёт клиенту, fmt "cc" = отказ |
| 0x407d50 | **Welcome-сборка** (Assemble + ECB ctx#1) |
| 0x407550 | GenerateInitialKey(seed WORD) — key1 @this+0x2194 (12B LUT + 4×0x6c) |
| 0x4075d0 | rand-ключ key2 @this+0x21a4 (4 DWORD из LUT по маскам) |
| 0x407fa0 | CClientSocket::OnCreate (SendCltConnect → GG init → ключи → welcome) |
| 0x417a20 | BlowfishWrapper::EncryptPrimary (скрамбл + чексумма + ECB) |
| 0x417ad0 | BlowfishWrapper::DecryptSecondary (ECB-dec + XOR-чексумма проверка) |
| 0x4178d0 | Blowfish_Init (стандартный; P @0x437570, S @0x4375b8) |
| 0x417070/0x417470 | Blowfish encrypt/decrypt block (стандартные) |
| 0x417b60 | CRSAKeyPool::Decrypt(keyIdx, buf, inLen[, max]) — 128B in → 32B out |
| 0x417c50 | **scrambleModulus** (перемешивание RSA-модуля для welcome) |
| 0x417d40 | CRSAKeyPool::GenKey (rsakpMake, beecrypt) |
| 0x417b40 | GetKey: counter++ % 5 (пул 5 ключей, структура 0x4c байт) |
| 0x406000/0x406050/0x4060a0 | CAuthSocket::SendCltConnect / SendCltDisconnect / SendCltPacket |
| 0x405d40 | CAuthSocket::OnRead (приём от authd) |
| 0x40e740→0x40e540 | Assemble(dest, size, fmt, va_list) — спеки 'S'/'b'/'c'/'d'/'s'/'w' |
| 0x419c70 | memcpy |
| 0x416650/0x416640/0x416660 | IsBlockedTrial / IsBlockedIP / AddBadUser (brute) |

### Поля CClientSocket (сессия)
+0x88 remoteIP, +0xfc m_iSessionId, +0x104 Blowfish ctx#1 (key1 static, шифр ГЕЙТ→КЛИЕНТ),
+0x114c Blowfish ctx#2 (key2 rand, дешифр КЛИЕНТ→ГЕЙТ), +0x2194 key1 (16B), +0x21a4 key2 (16B),
+0xf4 RSA key idx (0..4), +0x9c CCSAuth2, +0x100 флаг GG-authorized, +0x101 флаг,
+0xcc..0xdc GG query data (16B), +0xdc GG auth copy (16B)

## 1. Криптосхема (верифицировано)

### Blowfish
- **Стандартный** Blowfish ECB; Blowfish_Init @0x4178d0: P-массив из 0x437570 (пи-константы),
  S-boxes из 0x4375b8 (стандартные), key-XOR стандартный, key-schedule стандартный.
- ctx = 0x1048 байт: S[4][256] @+0, P[18] @+0x1000.
- Чек-сумма/скрамбл в EncryptPrimary @0x417a20:
  1) len_out = (len+7)&~7 (округление до 8)
  2) скрамбл DWORD LE: data[0] не меняется; для k≥1: S_k = cumsum(data_old[0..k]), data_new[k] = data_old[k] ^ S_k
  3) data[n_dw] = S (кумулятивная сумма ВСЕХ dword) — чексумма за данными
  4) len2 = n + 8; ECB(данные, len2) — шифруется на 8 байт больше, чем было данных
- DecryptSecondary @0x417ad0: len кратно 8; ECB-dec; проверка: XOR(data[0..(len-8)/4)) == data[(len-8)/4]

### Ключи сессии (LUT @0x437160, /tmp/gate_lut.bin)
- **key1 (static)** = GenerateInitialKey(0x04bd): dh=0x04, dl=0xbd, base=dl^dh=0xb9;
  k[i] = LUT[0xbd] ^ LUT[(0xb9^i)&0xff] ^ LUT[0x04] (i=0..2, LE) + 4×0x6c =
  **`6b60cb5b82ce90b1cc2b6c556c6c6c6c`** — ЭТО ЖЕ ИЗВЕСТНЫЙ СТАТИЧЕСКИЙ КЛЮЧ AION-клиента.
- **key2 (rand)** @0x4075d0: r = rand()&0xff; key2 = 4 DWORD LE:
  LUT[r], LUT[r&0xc1], LUT[r&0xf2], LUT[r&0x23]. Отдаётся клиенту ВНУТРИ welcome.

### Верификация capture (03.10, /tmp/proxylog.txt)
- ECB-dec(welcome, key1): dword0 и dword1 ОДИНАКОВЫ у пары сессий одного запуска гейта
  (0x7d521423, 0x634692d8), dword2+ меняются — модель ключа/скрамбла ПОДТВЕРЖДЕНА.
- data_new[0] = plaintext DWORD[0..4) = 0x7d521423 → байты `23 14 52 7d`:
  plaintext[0] = **0x23** (НЕ 0 — значит первый 'c'-спек получает константу 0x23, а пуш 0 —
  другой вар; точная привязка варов — открытая позиция §5.1).
- sid (0x7d5214) ОДИНАКОВ у сессий 1/2 и у 3/4 (пары по запускам гейта) → m_iSessionId
  растёт НЕ на каждый коннект; генератор @0x4041b8 (время+база) / 0x4076f3→0x408070.
- Welcome 194 = [2b len=0xC2,0x00][24 ECB-блока] — сходится с моделью Assemble 173B
  → +4 = 177 → round-up 184 → +8 = 192 → 194 total ✓

## 2. Клиентский протокол 2106 (поток)

```
клиент подключился → гейт:
  1. OnCreate: SendCltConnect(sessionId, remoteIP) → authd (2110)
  2. GenerateInitialKey(0x4bd) → key1; 0x4075d0 rand → key2
  3. Blowfish_Init(ctx#1, key1, 16); Blowfish_Init(ctx#2, key2, 16)
  4. welcome 194b = [2b len][ECB(key1): Assemble("cddbbbcccc", wargs) скрамбл+чексумма]
     wargs(13): 0, sessionId, [authd_sock+0xa0], 0x80, scrambleModulus(keyIdx) 128B,
     0x10, GG-query+4 16B (нули при GG-off), 0x10, key2 16B, loginType, b0, b1, b2
     (b0/b1/b2 = байты @0x43c1b0/b1/b2, companyCode-зона)
клиент 34b (RSA-обмен): гейт RSA-dec → 32B X (ключ сессии клиента?) → ответ 42b =
  [2b][ECB(key2): echo X 32B + чексумма] (в capture блок X виден одинаково в 34 и 42)
клиент 186b/314b (логин: классика/LoginEx): [2b][184B/312B]
  RecvLogin@0x406230: RSA-блок 128B/256B (глобал 0x43c1b8=false/true) → в стек,
  len -= блок; 0x417b60(0x43fcbc, [sock+0xf4], buf, 34) — первые 34Б → decbuf
  → sessionId=DWORD@data+128 → CheckSessionId 0x4079d0 → GG-check 0x407ac0 (16B @data+132)
  → поле [data+148] → IsBlockedTrial → сборка authd-пакета fmt "cbdb"
  → SendCltPacket(sessionId, buf) → от authd приходит ответ → гейт шлёт 74b/26b/42b
фейлы: Send("cc", 1, N) текст-пакет (22 = blocked IP, 45 = wrong loginType) + DelayedClose
```

- Фрейминг: [u16 LE total][payload]; ГЕЙТ→КЛИЕНТ шифрован ctx#1?? — ВНИМАНИЕ:
  capture показывает что 42/74/26-пакеты содержат БЛОК X (RSA-выдачу) повторённым →
  шифрованы сессионным ключом. Точный ctx (key1 vs key2) для каждого типа ответа —
  открытая позиция (§5.2). Welcome точно ECB(key1).
- Пакетная таблица m_PacketTable @0x437110 (19 типов, §0): type = первый байт plaintext
  после расшифровки (предположение — верифицировать живым клиентом).

## 3. Wire 2110 (gate ↔ authd) — БАЙТ-В-БАЙТ (дизasm 05.10 ~20:45)

- Гейт держит ОДНУ коннекцию к authd (глобал 0x43b438); authReconnectInterval=0 —
  при потере authd гейт молчит и НЕ переподключается (рестартить гейт после authd).
- Assemble @0x40e540(dest, size, fmt, va): c=1B, h=2B LE, d=4B LE, b=(len,ptr) blob,
  s/S=строки (двухуровневая jump-table из exe). fmt-строки .data:
  0x42c780="cdd" 0x42c784="cd" 0x42c788="cdh" 0x42c7ac="cb" 0x42c7dc="cbdb" 0x42c7f8="cc".
- ГЕЙТ→AUTHD (пул-буфер, len в buf+0x2000, send 0x40c460):
  - CltConnect @0x406000:    "cdd"(0,sid,IP)   → [00][4B sid][4B IP]           (9B)
  - CltDisconnect @0x406050: "cd"(1,sid)       → [01][4B sid]                  (5B)
  - CltPacket @0x4060a0:     "cdh"(2,sid,L-5)  → [02][4B sid][2B len]payload,
    len = payload+2 — САМОИНКЛЮЗИВНЫЙ u16 (payload строится Assemble с +7).
- AUTHD→GATE (CAuthSocket::OnRead @0x405d40, ring [sock+0x78], state [sock+0xac] 0/1/2):
  - [01][4B id] → [sock+0xa4]=id, регистрация 0x408670(0x43b440,id)
  - [03][4B sessionId] → [sock+0xa0]=sessionId (authd назначает после CltConnect)
  - [02][4B id][2B len-2][type][payload] → тело; len=(buf[6]<<8|buf[5])-2 —
    тоже САМОИНКЛЮЗИВНЫЙ (симметрия с "cdh"), ≤0x1ffb иначе лог+close; state=0 после пакета.
  - type-byte ≥0x15 → лог+close; ∈[0..0x14]; тип 4 = push serverlist
    (payload берем из capture; IP мира+region в байтах).
  - поток пакета: alloc 0x405450 → ctor 0x405430 (id/buf/size/sock в +0xc..+0x18) →
    очередь → OnIOCallback @0x405460 (0x4086e0; при fail — "cc").
- RecvLogin @0x406230 relay в authd: Assemble(pkt+7,"cbdb", 0, 34, decbuf,
  dword@data+148, len-24, data+152) → [00][34B decbuf][4B dword@148 (|0x80000000
  при 0x43c18a!=0 && session[0x101]==0)][len-24B @152]; authd-фрейм = 7+1+34+4+(len-24).
- cc-отказы: 0x407e80(session,"cc",1,code): 22 = IP в блок-листе (0x416640 на [sock+0x88]),
  45 (0x2d) = статус гейта 0x43c1ac не готов; после — DelayedClose, timeout [0x43c1b4]*1000мс.
- brute 20/60/120 = менеджер 0x43fb68 (методы 0x416640/50/60), константы внутри.
- Обёртки relay типов 1/2/3/5 @0x406500/520/540/560 → 0x4061c0: Assemble(pkt+7,"cb",[byte][blob]).
- Коды World→Auth: f2030000+NN (authlog-analysis-0410.md) — константы В payload, не во фрейминге.
- ⚠ УРОК ПАТЧЕЙ p1–p5: authd хрупкий — наш гейт wire 1-в-1, фейлы клиентов гасим сами.

## 4. Конфиг-зеркало (etc/config.txt, ver 41007) и эксплуатация

serverPort=2106; authAddr=127.0.0.1; authPort=2110; numThread=32; numIOThread=96;
acceptCallNum=200; socketLimit=2000; sessionTimeout=5 (мин); useForbiddenIPList=true
(etc/BlockIPs.txt); logDirectory="log"; checkGameGuard=false; useGameGuard=false;
useNotifyCSResult=true; loginType=2; companyCode="2"; logdport=3999 (useLogd=0);
tryInterval=60; tryCount=20; tryBlockInterval=120; dumpPacket=true; ggNumActive=50;
useGCSideExtendAccount=true; appLaunchBanDelay=5; useReportMail=false; authReconnectInterval=0.

- brute: 20 попыток / 60 с → блок 120 с (CBadUser; AddBadUser → DelayedClose(ban×1000))
- гейт живёт ТОЛЬКО в интерактивной юзер-сессии (Console 1); задача AionGate (/IT) →
  C:\Temp\gate.bat v3 (ждёт 2104 → start → ждёт 2106 → retry)
- порядок старта: SQL → AccountCache → authd → гейт
- aion-op: id=gate, display=AuthGateD, exe=AuthGateD.exe, task=AionGate, kill_task=AionKickGate,
  tailer D:/AION_LIVE_SERVER/AuthGateD/log/{{date}}.err — после свитча заменить exe на aion-gate.exe
- логи гейта: log\YYYY-MM-DD.NN.log (сводка инита) + .mon (монитор)

## 5. Открытые позиции (верифицировать в следующей сессии)

1. **Точная раскладка welcome-plaintext**: dword0 = `23 14 52 7d` → plaintext[0]=0x23.
   ✅ Проверено 05.10 ~20:30 чтением байтов из exe (offset=VA-0x400000):
   @0x43c1b0/b1/b2 = 65 00 72 00 = UTF-16 «er » (текстовая строка) — 0x23 НЕ из этой зоны,
   источник первого байта ищем в пуш-порядке Assemble (0x407d50). Доразобрать 0x407d50 пуш-порядок.
   Обратный скрамбл через DP-решатель x^(x+S)=y сломался (уравнение имеет неоднозначные
   биты) — для РЕАЛИЗАЦИИ не нужен (шифруем вперёд), нужен только для полной верификации.
2. **Как определяется type клиентского пакета**: первый байт plaintext после ECB-dec(key2)?
   В capture 34/186/314-пакеты шифрованы; хендлеры dispatch по m_PacketTable[type].
3. ✅ ЗАКРЫТО (05.10 ~21:40): 0x417b60 = beecrypt **rsapricrt** (импорты IAT:
   mpnzero/mpnsetbin/rsapricrt/mpnfree/i2osp): вход = 128Б login-данных как BE-число
   (mpnsetbin len=0x80, ведущие нули = выравнивание поля modulus128), ключ = пул
   0x43fcbc записей 0x4c, keyIdx = [sock+0xf4]; результат → i2osp BE в тот же буфер
   + memmove-выравнивание к началу → decbuf 32Б. Go: RSA.DecryptBlock(≤128Б).
   Осталось: длина decbuf 32 vs 34 (arg3=0x22) и поля plaintext LoginEx — живой клиент.
4. ✅ ЗАКРЫТО дизasmом (05.10 ~20:45) — §3 переписан байт-в-байт: gate→authd "cdd"/"cd"/"cdh",
   authd→gate state-машина [01]/[02]/[03], len самоинклюзивный u16 с обеих сторон.
   Остаток: dispatch-table по type-byte (брать из capture, не дизasm).
5. **42-ответ**: кто шлёт, каким ctx, точный plaintext ([echo X] + чексумма?).
6. sid = 0x7d5214 (8215060) — семантика генератора @0x408070.

## 6. План реализации (следующая сессия)

> СТАТУС 05.10 ~20:20 (актуализация): шаги 1–5 НЕ выполнены (Go-кода нет, в aion-gate только testdata/); позиции §5.1–5.6 все открыты. «го» на свитч дано заранее, юзер логиниться не будет — тесты фейками, свитч без живого логина. Каждый шаг = коммит+пуш+дельта в память.

1. Доверить позиции §5.1/§5.3/§5.4 дизasmом (1-2 ч).
2. nextgen/aion-gate (Go): internal/proto (framing/Blowfish/скрамбл/RSA-256/LUT-ключи/welcome),
   internal/server (сессии, CheckSessionId-логика, brute 20/60/120, BlockIPs, cc-отказы),
   internal/authdclient (wire 2110 1-в-1), internal/ship (копия из aion-logd), internal/config
   (yaml-зеркало config.txt + ship.*).
3. Тесты: фейк-клиент (handshake+LoginEx+фейлы+brute), фейк-authd (wire-фикстуры),
   byte-в-byte welcome против capture (fixture из /tmp/proxylog.txt).
4. Параллельный прогон на :21055 + фейк-клиент e2e.
5. Свитч: D:\SAION\aion-gate\ (exe+config.yaml+run.cmd), schtasks /change /tn AionGate
   /tr "D:\SAION\aion-gate\run.cmd" + /run; откат = retarget C:\Temp\gate.bat.
6. aion-op config: display+exe → aion-gate.exe, рестарт AionOp.

## 7. ЧЕКПОИНТ byte-exact welcome (05.10 ~22:10, НЕ ЗАКРЫТ — противоречие)

- Прогнано 82 welcome из capture: ECB-dec(key1=6b60cb5b) → 49 групп (d0,d1) —
  d0/d1 константны В РАМКАХ запуска гейта (пары/тройки сессий), рандом между запусками.
- Группа d0=7d521423/d1=634692d8 (3 сессии) = proxylog-запуск, plaintext[0]=0x23 ✓.
- У ДРУГИХ групп plaintext[0] ≠ 0x23 (0xcc, 0xc5, 0xfa…) → «0x23» — НЕ константа,
  а значение динамического вара (или ключ запуска иной — но 49 запусков за сутки маловероятны).
- Модель скрамбла (new[k]=old[k]^(S_prev+old[k]), csum=cumsum) подтверждена дизasm 0x417a20
  (прочитан лично, 1-в-1) — НО capture её нарушает: жёсткое тождество y_0==a_0 (бит0 dword1
  == бит0 dword0) НЕ выполняется (0xd4≠0xcc и т.д.). GG-зона-нулей (серии равных dword) в dec
  НЕ найдено ни на одном смещении → layout варов и/или позиция скрамбла в буфере — не те.
- solve32: x^(a+x)=y решается линейно (x_i детерминирован переносами; решение может
  отсутствовать/быть множественным — см. tools/analysis/diag*.py).
- СЛЕДУЮЩИЙ ШАГ: дизasm **0x407d50 (welcome-билдер)** — точный порядок варов, размер
  буфера, где welcomeExtra4(+4), и что реально передаётся в EncryptPrimary (возможно
  скрамбл идёт с offset-заголовка, или ECB↔скрамбл в другом порядке). Без этого
  byte-exact welcome невозможен, а наш welcome клиент НЕ расшифрует — БЛОКЕР СВИТЧА.
- Скрипты: tools/analysis/{unscramble_welcome,diag2_scramble,diag3_modwin,diag4_paddb,diag5_paddb_dfs}.py

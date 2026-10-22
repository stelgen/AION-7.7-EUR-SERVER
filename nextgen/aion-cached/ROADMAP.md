# ROADMAP — aion-cached (Go-фреймворк CacheD64)

> Журнал теорий / леджер решений для трека reverse-engineering CacheD64.
> Фазы: R1 каркас → R2 дизasm → R3 Go MVP → R4 A/B → R5 свитч.

---

## Фазы

| Фаза | Статус | Дата | Что сделано |
|---|---|---|---|
| R1 — Go-каркас wire-фреймворка | ✅ | 10.10.2026 | wire.go (Reader/Encoder, канон CServerSocket.cpp), handlers.go (map[uint16]Handler), ship.go, server.go (TCP 2006/2007/2009), конфиг, 9 тестов зелёные |
| R2 — дизasm DummyCrypt | ⏳ | — | Требуется точная форма XOR-стрима из generated/Cached.c + PDB publics |
| R3 — Go MVP (реальные хандлеры) | ⬜ | — | Замена stub на рабочие RQ/RP, GQ/GP, NPC-DB |
| R4 — A/B тест с Live CacheD64 | ⬜ | — | Захват и сравнение пакетов (probe/forkprobe/fake-клиенты) |
| R5 — свитч (замена бинарника) | ⬜ | — | Полная замена CacheD64.exe на проде |

---

## Журнал теорий / леджер решений

| # | Гипотеза | Способ проверки | Статус |
|---|---|---|---|
| 1 | Wire-канон L2 C1 CacheD применим к Aion CacheD64 (2006/2007/2009 — 1-в-1 каналы) | Сопоставление packetNNN_*.h серверных листенеров L2 C1 с нашими каналами | ✅ канон (R1 каркас построен) |
| 2 | DummyCrypt = `payload[i] ^= byte((m_key + uint32(i)) & 0xFF)` | Дизasm из `generated/Cached.c` или capture-сверка | ⏳ R2 |
| 3 | Rolling key катится при браке csum (m_key += m_packetSize до проверки) | Чтение CServerSocket.cpp: m_key += m_packetSize идёт до csum-check | ✅ канон |
| 4 | Outbound крипта отсутствует (L2 C1 Send НЕ шифрует ответы) | Чтение CServerSocket::Send — нет вызова DummyCrypt.Encrypt | ✅ канон |
| 5 | Opcode на проводе — 1 байт (L2 C1), Go-тип Frame.Opcode — uint16 | Соответствие L2 C1 wire-формату | ✅ канон |
| 6 | map[uint16]Handler для future-proofing (Aion CacheD64 может использовать 2-байтные опкоды) | Требование ТЗ R1 | ✅ принято |
| 7 | Frame.Opcode = 1 байт на проводе, но Go-тип uint16 | Не наруушает wire-канон, расширяет Go API | ✅ принято |
| 8 | Конфиг = key=value текст (не yaml/json), R1 минимализм | ТЗ: «Минимум для R1» | ✅ принято |

---

## Следующий шаг

R2 — дизasm DummyCrypt из `generated/Cached.c` / PDB publics, сверка с формулой `payload[i] ^= byte((m_key + uint32(i)) & 0xFF)`.

## R2 ЗАКРЫТ (21.10): DummyCrypt = NO-OP, wire = plaintext ✅

**Два независимых доказательства:**

1. **L2 C1 сурсы (MasterToma)** — `L2Core/src/crypt/DummyCrypt.cpp` (символ L2CacheD 0x00435050):
   тело `Decrypt(uint8_t* data, uint64_t key, int length)` **ПОЛНОСТЬЮ ПУСТОЕ**
   (`UNUSED(data); UNUSED(key); UNUSED(length);`) — «Dummy» в буквальном смысле.

2. **Aion 7.7 CacheD64.exe (дизasm по .map, exe+pdb локально)** — крипто-фэмили Utils.obj:
   | Символ | VA | |
   |---|---|---|
   | ?Decrypt@@YAXPEAEPEBKH@Z | 0x1401b0360 | Decrypt(uint8_t*, const uint32_t* key, int) |
   | ?Encrypt@@YAXPEAEPEBKH@Z | 0x1401b03e0 | |
   | ?Encrypt@@YAXPEAEPEBEPEBKH@Z | 0x1401b0440 | Encrypt(uint8_t*, const uint8_t*, const uint32_t*, int) |
   | ?Decrypt2@@YAXPEAEPEBKH@Z | 0x1401b04c0 | |
   | ?Encrypt2@@YAXPEAEPEBKH@Z | 0x1401b0570 | |

   **Ноль ссылок** на всю семью в бинаре (сканы: E8 rel32, E9 thunks, FF15 call [rip],
   lea rip-rel, ptr VA в .rdata+полный .data 19МБ) = **мёртвый код, не в wire-пути**.

**Бонус-факт (алгоритм мёртвой семьи, на случай реанимации в будущих версиях):**
дизasm Decrypt показал XOR-цепочку: `data[i] = staticTable[i&0x3F] ^ key[i&0x7] ^ data[i] ^ data[i-1]`
(staticTable 64Б ASCII-алфавит "nKO/WctQ0AVLbpzf...?3iI9" @0x1402dccb0; i=0 — только key[0]).
Связь с m_key (scalar в L2) НЕ найдена — Aion Decrypt берёт key-УКАЗАТЕЛЬ.

**Вывод для aion-cached:** wire 2006/2007/2009 = plaintext; единственная защита кадра =
csum-хвост (u16 LE wireLen). DummyCrypt в wire.go = no-op (API сохранён, seed = vestigial).
Конфиг seed теперь не влияет на провод — ОСТАВИТЬ (не ломать канон L2-эталона).

**Открытое:** позиция rolling m_key += m_packetSize в Aion OnRead не проверялась
(нет decrypt-вызова → вероятно rolling тоже отсутствует) — непринципиально (key не используется).

| 8 | Opcode на проводе Aion может быть 2 байта: Server2DB = 381 ops > 255 (1-байт не влезает) | live capture 2006 (R4); в wire.go добавлен флаг OpcodeWidth (1=L2-канон дефолт / 2=LE) + BuildW + Decoder.SetOpcodeWidth + тесты | ⏳ |
| 9 | RQ_VERSION: S2C-форма "cd" доказана L2 C1 (packet000: Send("cd", op, isEqual)), но cachedProtocolVersion Aion НЕизвестна ($Revision: 311 = L2) | live capture первого version-обмена | ⏳ — наш hVersion отвечает ТИШИНОЙ (не рискуем isEqual) |
| 10 | RQ_PING (108) → RP_PING (45) not-found форма "cdddddddd" = L2 C1 packet144 канон: Send("cdddddddd", PingTest, p2,p3,p4,p1,ts,0,0,0) | live capture | ✅ реализовано (MVP, not-found ветка — штатная у ориг) |
| 11 | NPC-DB (2009) словаря опкодов в .profile НЕТ — только 8 каналов (2006×2/2051×2/2305×2/2007×2) | живой capture 2009 | ⏳ |

## R3 ПРОГРЕСС (22.10): MVP-диспетчер ✅, read-путь раскладок = R4/capture

| Кусок | Статус |
|---|---|
| internal/rpc/opcodes.go — 765 опкодов, 8 протоколов (генератор tools/analysis/gen_cached_opcodes.py из profile-opcode-map.md) | ✅ |
| Route(): имя из профиля на каждый кадр + ship-события (raw-first) | ✅ |
| hPing: RP_PING not-found форма (L2 C1 канон packet144) | ✅ |
| hVersion: лог, тишина (теория #9) | ✅ |
| Топ-путь silent: RQ_LOAD(11)/RQ_LOAD_ITEM(14)/RQ_LEAVE_WORLD(15)/RQ_U_SAVE(16) | ✅ лог, ⏳ раскладки |
| wire OpcodeWidth 1/2 (теория #8) | ✅ флаг+тесты |
| DB-слой 781 procs + RAM-модели User/Item | ⬜ (после capture-раскладок R4) |
| GQ/GP MVP (2007 admin) | ⬜ |

| 12 | Голые имена procs в 781-инвентаре редки: aion_GetCharInfo существует ТОЛЬКО версионированным (5 суффиксов _20XXXXXX), aion_CreateGuild вообще нет (GetGuild есть) — bare-имена НЕ угадывать, только из списка | — | ✅ учтено в тестах |
| 13 | 140 exe-only procs: наш CallWorldProc отвечает ErrProcMissing ДО SQL (поведение-маркер A/B: у ориг fail уйдёт на SQL-уровень) | A/B R4 | ✅ реализовано |

## R3 DB-СЛОЙ (22.10, часть 2): инвентарь 781+140+132 ✅

| Кусок | Статус |
|---|---|
| internal/db/procs.go — СГЕНЕРИРОВАНО (gen_cached_dbprocs.py): 781 WorldProcs (домен+versioned) / 140 ExeOnlyProcs / 132 DbOnlyProcs | ✅ |
| Домены: Misc 402 · Guild 65 · VendorItem 48 · User 34 · CharInfo 31 · House 31 · ItemEnchant 26 · Quest 22 · Buddy 22 · Skills 21 · Abyss 20 · Block 15 · Mail 14 · Auction 14 · AddedService 13 · NPC 3 | ✅ |
| CallWorldProc: ErrProcMissing ДО SQL (exe-only 140) / ErrProcUnknown / {call dbo.<name>(?,...)} / Query→Exec-фолбэк | ✅ |
| SQLStore (sql.Open, MaxOpenConns=10 = numberOfDBThreads) — заготовка под connStr с VM | ✅ каркас |
| db_test: 781/155-versioned/140/132 инвентарные + QueryForm + ExecFallback | ✅ 6/6 |

| 14 | **2007 interactive = ТЕКСТОВЫЙ протокол** (L2 C1 CAdminSocket.cpp: строки до '\n', packetId первым полем через ReadSocketArgs "dsss", текстовые ответы Send("0%d",3)) — НЕ бинарный GQ/GP! GQ/GP-опкоды из .profile = канал NPRelay (отдельный) | capture 2007 | ✅ реализовано: admin_text.go textline-ридер + dispatch по packetId → GQ-имена (теория-приближение), silent-ответы (формы = capture) |
| 15 | **КАНОН-WIRE РЕВОЛЮЦИЯ**: R1-каркас был по L2 C1 (op 1Б, без маркера) — НЕСОВМЕСТИМ с golden-кадрами Aion. Правильный канон = ACCACHE-ФОРМАТ [len u16 total][cmd u16 LE][EB/EC][~cmd u16][payload] БЕЗ хвост-суммы (golden 0f000100ebfeff... = 15Б ровно, len-поле = контроль). u16-cmd закрывает теорию #8 (381>255 влезает) | golden-кадры (доказано) | ✅ **wire.go переписан 22.10**: BuildC2S/BuildS2C/Feed/Validate + TestGoldenAccacheFrame |

## R3 GQ/GP + 2007 (22.10, часть 3)

| Кусок | Статус |
|---|---|
| 2007 = текстовый interactive (L2 C1 канон): admin_text.go textline + packetId-dispatch (GQ-имена, теория #14) | ✅ MVP |
| e2e-тесты: TestAdminServerPingE2E (бинарный ping), TestAdminTextE2E (2007 textline), TestServerUnknownSilent | ✅ 3/3 |
| wire-канон РЕВОЛЮЦИЯ → accache-формат (теория #15): wire.go переписан, golden-кадр тест | ✅ |
| серверные тест-хуки: SetListenerForTest/ServeOnTest(ch) | ✅ |
| GQ/GP бинарный NPRelay-канал (порт из конфига NPCRelay — TBD) | ⬜ |
| Текстовые ответы interactive (Send("0%d",3)-формы) | ⏳ capture |

## R3 GQ/GP NPRelay-канал (22.10, часть 4) — АРХИТЕКТУРА РАСКРЫТА

**КАНОН (прод-факты):**
- NPRelay64 common.xml: `cacheServerPort=2006` — релей подключается к **ТОМУ ЖЕ 2006**, что и Server64 (второй тип клиента, НЕ отдельный порт)
- NPRelay64.exe = protobuf к MainServer (`NpRelayToMainProtocol.pb.cc`) + GQ/GP к CacheD
- CacheD64: `Listener<RelaySocket>` (`ProtocolVersion@0x14008d940` = **xor eax,eax; ret** — NO-OP заглушка)
- id-пространства RQ/GQ ПЕРЕСЕКАЮТСЯ (RQ_U_UPDATE_COUNT=2 = GQ_LOGON=2) ⇒ **per-conn режим**: NPRelay64 handshake GQ_LOGON(2)/GQ_KICKOFF(1) переключает соединение в relay-диспетчер

| Кусок | Статус |
|---|---|
| relay_handlers.go: per-conn mode (modeServer/modeRelay), maybeRelayHandshake, relayGQDispatch (silent MVP, GQ-имена из профиля) | ✅ |
| handleConn: relay-ветка (Server64-путь не меняется — NPRelay64 первым кадром шлёт GQ_LOGON(2), не RQ_TEST) | ✅ |
| e2e: TestRelayGQHandshakeE2E (handshake → relay-режим → silent, conn жив) | ✅ |
| Гоча: double-consume pending[consumed:] (panic [15:0]) — фикс | ✅ |
| GQ-раскладки payload (54 ops) | ⏳ capture R4 |
| GP-ответы | ⏳ capture R4 |
| 17 | **ГОТЧА-канон e2e**: `body = frame[2:]` ⇒ payload@body[**5**] (HeaderWire=7, минус len-поле 2 = 5; НЕ body[7]!). ts в p[16:20], нули p[20:32]. Поймано e2e-пингом (echo-поля сдвигались на 2) | любой S2C-парсинг в тестах | ✅ закодировано в TestAdminServerPingE2E (+комментарий) |

📊 сосед узнал (22.10, от accache R5/live A/B): (1) accache СВИТЧЕН В ПРОД — :2220 теперь наш aion-accache (ориг отведён); для cached это значит: common.xml accountCachedPort=2220 ведёт к НАШЕМУ сервису, его wire/SQL-паттерны (EXEC dbo.proc @pN — Т22 go-mssqldb) переиспользуем в aion-cached DB-слое; (2) op-events proc_missing у CacheD64 (aion_GetItemCollection*/aion_GetItemCollectionExpiredList/aion_GetItemCollectionComplete*/aion_LoadFameInfo/aion_LoadReinventInfo/aion_GetItemCollectionLevelList/aion_getItemAttributeDeltaListAll_20190919/aion_GetItemAttributeDeltaListAllVendorDark|Light_20190919/aion_DeleteItemByDate_20191206) — живое подтверждение exe-only procs из инвентаря 140 (теория #13: ErrProcMissing как A/B-маркер верна); (3) accache lesson: write-раскладки не проверяются read-only тенью — для cached R5 ставить A/B на живые WRITE тоже (или стенд с write).

## R3-семантика из hourly-логов (22.10, Т24) — read-only, без «го»

**ПОПРАВКА R0-факта:** «log/*.log = RPC-строки с параметрами» — НЕподтвердилось: hourly .log
маленькие (10-14КБ/час), RPC-трассировки НЕ содержат; .err = ODBC-диагностика/сироты/AlivePacket
(RQ_ строк: 0). Ценность другая — СЕМАНТИЧЕСКИЕ СЛЕДЫ хендлеров (см. cached-ref/logs-hourly/):

| След | Канон |
|---|---|
| `[CharacterList] ServerId:1 (AccID:1010) (charID:1003) (charCount:2)` | на КАЖДЫЙ чар-селект (после logout-флоу ACS); charID в логе = последний контекст ⏳ |
| `[char]enter world 1010/1002 lev80 110010000(pos)` / `[char]leave world ...` | вход/выход из мира; ID+позиция = персистентные из БД (последнего выхода) |
| `LoadClientQuickBar ::== characterId(100X)` | quickbar-загрузка |
| `thahn client setting:RP_LOAD_SKILL_COOLTIME...` | RP-имена сверяются со словарём .profile ✓ |
| Старт-цепочка | CheckDatabase → CleanUp ×13 → VendorDb::CreateIndex → Poll → IOThread → 2007/2009 ready → 2006 ready → log server |
| Память | VendorDb.cpp pool ≈90% TOTAL RAM — вендор-данные в RAM (для RAM-моделей R3) |

**ВЫВОД:** wire-раскладки read-пути из логов НЕ выводятся — ТОЛЬКО live capture
(fork-сдвиг 2006→2016 по «го» / pktmon). Семантика-порядок = Т24 (канон выше).
Мир при тестах юзера сыпет proc_missing (ItemCollection*/LoadFameInfo/LoadReinventInfo) —
exe-only procs подтверждены живьём (теория #13).

## R4 FORK-СДВИГ ВЫПОЛНЕН (22.10 ~08:43-09:00) — канон 2006 снят живьём 🔥

Схема (accache-паттерн): копия CacheServer-2016 (ports 2016/2017, common.xml правлен байтово)
+ cdmirror (accmirror v1.3: front :2006 → back :2016, shadow-tap → тень aion-cached :2018)
+ рестарт пары (Server64 умер от потери 2006 — ожидаемо; NPCSvr пережил; мир собран 8/8 через
mirror→копия). Замечания: копия висит при живом ориг (single-instance как ACS — стартовать соло);
NPCSvr тоже клиент 2006 (поправка R0 «единственный Server64»).

### Т26 — КАНОН 2006 (Server64→CacheD) ✅ live (332 C-кадра)
`[u16 len total][cmd u16 LE][marker 0xAE C2S / 0xAF S2C][~cmd u16 = ^cmd][payload]` — plaintext.
Маркер-пара = ПО КАНАЛУ: accache EB/EC, cached AE/AF (канон-каркас един: len/cmd/marker/~cmd,
инверсия ✓ на 100% кадров). Опкод-пространство u16 (теория #8 подтверждена: RQ_VERSION=215,
cmd=282 и пр.). DummyCrypt отсутствует. RQ_VERSION-хендшейк в потоке ✓ (cmd=215, len=17).

### Т27 — NPCSvr = второй клиент 2006 ✅ (469 кадров, свой поток)
cmd=170 ×261: UTF-16 строки `tl_dp_id:NNN, calc_prob:...`, `tl_abyss_id:1014` (drop/abyss-статы);
cmd=200 ×32: UTF-16 `[SAVE]v01/ctrl`; cmd=179/224/106/299... Marker тот же 0xAE. NPCSvr и Server64
РАЗДЕЛЯЮТ порт 2006 и маркер-пару — РАЗЛИЧЕНИЕ = по первому кадру/содержимому (⏳ фингерпринт).

### Т28 — Server64 фоновый тикер ✅
cmd=179 (0xB3) ×147 (~6с период): [0000][accId=1010][0000][01000000][zeros][0x89][0x02] — статус-тикер;
cmd=277 (0x115) ×150: короткие [03 06 e9 03 0000 02]. Раскладки логина (RQ_LOAD=11/LOAD_ITEM=14/
U_SAVE=16) в этот capture НЕ попали (юзер был уже в мире) — СЛЕДУЮЩИЙ ЛОГИН ЮЗЕРА их снимет
(capture пишется непрерывно: D:\SAION\aion-cached-capture\cdmirror.log).

### Статус
R4-раскладки: capture ЖИВОЙ, ждём логин юзера → раскладки → Go-хендлеры → RAM-модели → A/B.
Тень (aion-cached MVP) режет 0xAE-кадры (bad marker) — доработка wire маркеров = следующий код-шаг.

## R4 РАСКЛАДКИ READ-ПУТИ СНЯТЫ (22.10 09:09-09:10, логин юзера через capture-стенд) 🔥🔥

Сессионный контекст: **X = 0x06F38F70** (хендл сессии, константа в рамках conn; появляется в cmd=288 — первый кадр), **a = 4** (константа). Все кадры marker C2S=0xAE/S2C=0xAF, plaintext.

### C2S-раскладки (канон, hex живого логина)

| cmd | Кадр | Payload |
|---|---|---|
| 288 (первый) | 20Б | [X][a=4][accId][pad 00] — открытие сессии |
| 175 | 22Б | [X][a=4][accId][01 01 00] |
| **RQ_LOAD (11)** | 23Б | [X][a=4][charId][accId] |
| **RQ_LOAD_ITEM (14)** ×2 | 28Б | [X][a=4][charId][категория u32: 02/03][accId][page u8] |
| **RQ_LEAVE_WORLD (15)** | 35Б | [accId][charId][4×u32 поз+heading][pad 00000000] |
| **RQ_U_SAVE (16)** ×2 | 857Б | [accId][charId][...большой сейв...] — полный разбор = отдельный шаг |
| 320/54/184 | 19Б | [X][a][ts] семейство |
| 21 (enter world) | 19Б | [10e160e9=мирID?][01000000][1000] ⏳ |
| 194/135/340/351/357/375/369/373/258/272/240/277... | выход-кластер | — |

### S2C-ответы (ориг-копия)

- **cmd=90 len=1571 = CharacterList** (UTF-16 «SteLGeN», charRef `ea030002`).
- **RP cmd=1 len=1156** = загрузка чара: [X][a][01000000 00][ff40529f][charRef][позиция e7096e15...][979a2402]... полный разбор = шаг.
- **RQ_LOAD_ITEM → RP cmd=34/127/77 (echo-структура [charRef][X][a][charId][кат][page]) → RP cmd=109 ×7 len=8021 (item-блоки) + cmd=61 серия (8003/5658/30×N — счётчики 0x30-0x4D!) + RP cmd=0x0E len=7194 ×3 с [page u8] 00/01/02 в ~cmd-позиции — payload = ZLIB (78 9c...)** ⚡ item-данные жаты!
- RP-команды-подтверждения ×20+ (cmd=9/24/29/44/47/62/91/97/99/101/110/128/132/139/150/178/189/219/225/228/234/235/240-248...).

### Вывод
Read-путь логина = RQ_VERSION → 288/175 (сессия) → CharacterList(90) → RQ_LOAD → RP cmd=1 (1156Б) → RQ_LOAD_ITEM ×2 (кат 2/3) → item-пачки (zlib!) → enter world (cmd=21) → фоновый тикер 179/277 → U_SAVE → LEAVE_WORLD. Этого достаточно для Go-хендлеров read-пути MVP (кроме полного RP cmd=1/109-полей — детальный разбор = следующий шаг, ZLIB-пачки распаковывать zlib-ом).

## R4 OFFLINE-РАЗБОР RP cmd=1/0x0E/109 (22.10, zlib+структура — материал в гите) 🔥🔥🔥

### RP cmd=1 (RP_LOAD, 1156Б) — ответ на RQ_LOAD:
[X=0x06f38f70 echo][a=4][01][ff40529f][159][charRef 0x20003ea][позиция e7096e15][...][level=0x50=80 @off48][...] —
структура чара 1149Б (полный разбор полей = следующий шаг). X ЭХО-ИТСЯ обратно — сессия подтверждается.

### RP cmd=0x0E (RP_LOAD_ITEM_DATA) ×3 — **item XML ZLIB!**:
- page 00: compressed 7160Б → decompressed **28578Б** (ratio 4.0x) — **UTF-16LE XML** `<?xml version="1.0" encoding="UTF-16"?>`
- page 01: compressed 7160Б → decompressed **3098Б** (ratio 0.4x — иного характера)
- page 02: compressed 7160Б → decompressed **744Б**
- Префикс перед zlib (27Б): [page u8][X][a=4][charRef 0x20003ea][03][0000][1c1112/2f02/e700][counts]
- **Item/скилл-данные Server64 жат ZLIB → XML UTF-16LE** — Server64 читает XML-конфиг чара через CacheD!

### RP cmd=109 (item-блоки) ×12 по 8021Б = ~96КБ item-данных:
[charRef echo][X][a=4][03][1100 flag][0x197=407 count][0134df0a][12504b44 binary...] — бинарные item-записи БЕЗ zlib.

### Вывод для MVP
Read-путь Server64: RQ_VERSION → 288/175 (сессия) → cmd=90 (CharacterList) → RQ_LOAD → **RP cmd=1 (char struct)** → RQ_LOAD_ITEM ×2 → **RP cmd=0x0E ×3 (zlib XML page 0/1/2)** → **RP cmd=109 ×12 (binary items)** → enter world.
Наш MVP: эмиттеры RP cmd=1 (скелет [X][a][01][...][charRef][позиция][level]) + RP cmd=0x0E (zlib XML из БД/кэша) + RP cmd=109 (item-блоки из БД) → A/B.

## R4 DEEP ANALYSIS: RP cmd=0x0E = CHARACTER SETTINGS XML STORE (22.10) 🔥🔥🔥

**ЗАПРОС-ОТВЕТ ПОДТВЕРЖДЁН РАСПАКОВКОЙ:** RP cmd=0x0E (RQ_LOAD_ITEM по словарю) содержит
НЕ items, а **НАСТРОЙКИ КЛИЕНТА** — zlib-сжатый UTF-16LE XML! Три страницы:

| Page | Decompressed | Содержимое |
|---|---|---|
| 0 | 28578Б | `<options>`: abyss_fort_info_pos, auto_use_*_item, boost_weapon, cam_*, ch_emotion_history, and more |
| 1 | 3098Б | `<options>`: botting, chainskill, custom_chainskill, devanion_skill_setting, **quickbar (qb ×9)** |
| 2 | 744Б | `<options>`: favorite_monster_achievement, housing_favorite_buddy_list, indun_cooltime_favorite_list, item_collection_favorite_list, title_favorite_list |

**CacheD = RAM-кэш аккаунтов + ХРАНИЛИЩЕ НАСТРОЕК КЛИЕНТА (XML) + item-инвентарь.**
Это объясняет почему AccountCacheServer существует: клиент сохраняет свои настройки
(chainskills, quickbars, UI options) через Server64→CacheD, а при следующем логине
CacheD возвращает их. XML генерируется КЛИЕНТОМ и хранится CacheD'ом в RAM (потом в БД).

Для нашего MVP: **кэшировать zlib-XML per-charId** (принять от клиента при сейве,
отдавать при логине). Простейший подход: capture-replay per-charId + БД-персист.

### Полная карта опкодов из capture (block#0=Server64 ticker, block#1=Server64+NPCSvr)

Block#0 (первое соединение): cmd=277 ×150 + cmd=179 ×147 (тикер), RQ_VERSION ×1.
Block#1 (второе соединение): cmd=170 ×301 (tl_ строки), RQ_LOAD ×4, LOAD_ITEM ×8, 
U_SAVE ×8, LEAVE ×8, cmd=135 ×16, RP cmd=61 ×175, cmd=109 ×22, cmd=14 ×12.

ДВА соединения на :2006: Server64 (логин+тикер) и NPCSvr (tl_ строки+метрики).
Marker един (0xAE/0xAF), канон един. РАЗЛИЧЕНИЕ по первому кадру/содержимому (⏳).

### RP cmd=1 (1156Б) — CHARACTER STATE DUMP (полная карта non-zero блоков)

[X@a echo][01][ff40529f][159] → [charRef][позиция e7096e15] → [979a2402][02] →
[level=80][02] → [counters e35d/c72b] → [0x3C=60][0x16=22] → [1000 + "DVM-<hash>" UTF-16]
→ [второе имя] → [координаты ×3] → [timestamps] → [назад: "SteLGeN-<hash>" UTF-16] →
[статы/данные 0x30-0x4D счётчики] → [timestamps] → ...

Это ПОЛНОЕ СОСТОЯНИЕ ЧАРА: identity + position + level + имя + статы + тайминги.
1156Б = сериализованный объект чара для быстрой загрузки (CacheD держит его в RAM).

## Т31 — RP cmd=1 ПОЛНЫЙ БАЙТОВЫЙ РАЗБОР (22.10, offline из capture 1.5МБ) 🔥

1156Б frame, payload 1149Б. Byte-level map (не dword-aligned!):

| Offset(payload) | Size | Content | DB Source |
|---|---|---|---|
| 0-3 | 4 | X (session handle echo) | session |
| 4-7 | 4 | a=4 | constant |
| 8-9 | 2 | 01 | constant |
| 12 | 1 | 00 | separator (НЕ dword!) |
| 13-16 | 4 | ff40529f | hash/unknown |
| 17-20 | 4 | zeros | |
| 21-24 | 4 | charRef [charId u16][slot u16] | char_id |
| 25-28 | 4 | e7096e15 | position/timestamp |
| 29-40 | 12 | zeros | |
| 41-44 | 4 | 979a2402 | counter |
| 49 | 1 | 0x50 = LEVEL! | user_level (single byte!) |
| 55-56 | 2 | 02 | |
| 63-64 | 2 | e35d = 23995 | stat |
| 67-68 | 2 | c72b = 11207 | stat |
| 75-76 | 2 | 3c = 60 | stat |
| 83-84 | 2 | 16 = 22 | stat |
| 130-149 | 20 | 1000 + charRef byte + "DVM-" + hash (UTF-16 internal name) | |
| 317-327 | 10 | "DVM-" + hash (repeat) | |
| 399-460 | 62 | coordinates ×3 (position trail) | position |
| 468-477 | 9 | timestamps | |
| 903-927 | 25 | **"SteLGeN-" + hash UTF-16** = FULL CHARACTER NAME | char full name |
| 955-958 | 4 | timestamp 0x6AD99F0A | |
| 963-1003 | 41 | stat bytes | |
| 1005-1035 | 30 | binary data | |
| 1036-1119 | 84 | timestamps/values | |

**Character full name найден!** offset 903-927: UTF-16LE "SteLGeN-" + binary hash = полное
внутреннее имя чара (account-prefix + server-hash suffix). "DVM-" на 133-149 и 317-327 =
внутренние идентификаторы (⏳ роль).

Level = SINGLE BYTE at payload[49] (frame[56]) — не dword! Fix в GenerateRPLoadDB: out[56].

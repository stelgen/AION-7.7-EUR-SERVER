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

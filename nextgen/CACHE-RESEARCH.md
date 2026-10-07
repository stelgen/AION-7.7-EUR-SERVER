# CACHE-RESEARCH: CacheD64 (Aion CacheServer64 7.7) — ресёрч и план MVP

Дата: 08.10.2026. Статус: R0-разведка ЗАВЕРШЕНА. Прод не тронут.

## 1. Что это за приложение

`CacheD64.exe` (внутреннее имя **AION CacheServer64 77.20.0604.15625**, сборка 2020-06-04 16:25:41 — та же волна билдов, что Server64/ACS) — **RAM-кэш мира и единственный ODBC-писатель в БД мира** `_AionWorldNew114_rc`. Это «мирный» брат AccountCacheServer (2220): ACS кэширует аккаунты, CacheD64 — мир (чары/предметы/гильдии/почту/аукцион/вендоров/хаусы/абисс-рейтинг/инстансы/квесты/капчу-инфу и т.д.).

Сервер64 НЕ пишет в мир-БД напрямую — все мутации мира проходят через CacheD64 (RPC), который держит данные в RAM и асинхронно льёт их в SQL через ~781 процедуру `aion_*`. Отсюда требование `max text repl size=65664` (без него CacheD64 самоубивается — README) и логика «SQL-ошибки процедур, abyss-циклы» в его логах.

## 2. Порт-карта и кто с кем общается

| Порт | Назначение | Клиенты |
|---|---|---|
| **2006** serverPort | основной RPC (мир) | **единственный TCP-клиент = Server64** (127.0.0.1, ESTABLISHED, netstat live) |
| **2007** interactivePort | интерактивный канал | не наблюдался в netstat (спящий/по требованию) |
| **2009** | третий листенер (назначение TBD — дизasm) | — |
| 2305 (ICServer) | CacheD = клиент Interchange | исходящее соединение |
| 2051 (LogServer) | телеметрия (LogClientToLogServer/LogSvc) | исходящее |
| SQL (local) | `_AionWorldNew114_rc` (aionworld_new.dsn), `AionAccounts` (L2Conn.dsn), `Aion_log` (aiongm.dsn), PetitionDB/BkPetitionDB.dsn | ODBC |

DSN-набор на VM: `aionworld_new.dsn` → (local)\_AionWorldNew114_rc; `L2Conn.dsn` → AionAccounts (Trusted); `aiongm.dsn` → Aion_log; + PetitionDB.dsn/BkPetitionDB.dsn.

Конфиги: `common.xml` (country=7, numberOfDBThreads=10, serverPort=2006, interactivePort=2007, logserver 127.0.0.1:2051, logDirectory), `config.xml` (serverTitle «Gardarika Cache Server», serverID=1), `DBLogDetail`/`DBLogSummary`, `perfmon.ini` — всё снято в артефакты.

## 3. Символика — ПОЛНАЯ

- `CacheD64.exe` 22 526 464 B, MD5 `15e213947931fbd0bf346c952ff236fe` (Jun 2020)
- **`CacheD64.pdb` 106 377 216 B, MD5 `979ae355d1fda7fb666e9cd6a05a7adc`** — полная символика, лежала рядом с exe
- `CacheD64.map` 10 173 037 B, MD5 `022066814de26ffb7b417a80163e4d72`
- Локально: `~/STELGEN/projects/aion_rev_2026-10-05/artifacts/pdb-big/CacheD64/CacheServer/` (exe+pdb+map+конфиги+лог-сэмплы+log/ 356МБ)
- **pdbpub: 14 281 публичных символа** (ACS было 8035) → `nextgen/cached-ref/pdb-publics-14281.txt`
- Framework тот же, что ACS: `..\..\Shared\IoCompletion.cpp` / `Main.cpp ListenThread`, deadlock-dumper («Intentional exception») в `AIONErr.txt` — 4 краша 04-05.10 (AV @+0x18c31f) = это ДАМПЕР дедлоков IOThread, не баг (audit-linked-loopback-20261005.md).

## 4. Класс-карта (из publics, `cached-ref/class-counts.txt` / `cached-rpc-map.md`)

RPC-направления (глобальные функции, стилистика ACS Decode*/Encode*):

| Класс | Функций | Смысл |
|---|---|---|
| **DbToServer** | 303 | ответы CacheD → Server64 (Encode*-ответы) |
| **ServerToDb** | 192 | запросы Server64 → CacheD (Decode*) |
| **ServerToDb_Update** | 31 | update-операции |
| **AdminToCache** | 33 | GM/web-админ канал (MakeBuilder, SendMail, GuildOperation, Captcha, Housing, AionTV, Poll*, Forbidden, Tournament...) |
| **CacheToAdmin** | 17 | ответы админу |
| **ICToCache / CacheToIC / ICClient / ICSocket** | 7/7/9/3 | Interchange-канал (DecodeImportItem/ExportItemResult/ReqGuildBasisInfo/ReqTnmtInfo/ReqRankInfo/VersionAck/ShutdownOtherServer) |
| **DBConn / DBConnSMS / DBProfiler** | 26/8/10 | ODBC-слой (AllocSQLPool/ExecuteInsert/BindInputParam...) |
| Домены: **ItemDb 40, UserDb 23, VendorDb 21, GuildDb 16, AccountDb 11, HousingDb 12, UserGloryPointDB 6, User 20, PollInfo 20** | | RAM-модели кэша |
| LogSvc 15, LogClientSocket 6, LogClientToLogServer 5, EMailLog, FileLog | | лог-канал 2051 |
| OverseasEventSystem 6, OverseasEvent 4, OverseasQuestEvent 3 | | overseas-ивенты (country) |
| CIO*/CIOSocket/CIOServer/CPacket/Socket/PacketProfiler | | сетевой фреймворк (общий с ACS/authd) |

Примеры сигнатур с параметрикой (mangled в publics):
```
?DecodeImportItem@ICToCache@@YAX...AEA_JAEAUItemDbData@@AEAPEAVUser@@AEAPEAVItem@@AEAIAEAW4ItemChangeContext@@AEAW4WarehouseType@@12@Z
?DecodeMakeBuilder@AdminToCache@@YAX...AEAFAEAI2AEAD@Z
?DecodeSendMail@AdminToCache@@...PEA_W...AEA_KAEAHAEA_J61@Z
```
→ полная параметрика RPC восстановима без дизasmа dispatch.

## 5. Словарь опкодов (строки, `cached-ref/rpc-opcodes-and-procs.md`)

| Семейство | Кол-во | Канал |
|---|---|---|
| **RQ_** | **382** | запросы мира (Server64→CacheD) |
| **RP_** | **255** | ответы мира |
| **GQ_ / GP_** | 55 / 53 | builder/GM-канал (Admin) — GQ_CAPTCHA/GQ_MAKEBUILDER/GQ_ITEMADD/... |
| **ACQ_ / ACP_** | 39 / 28 | встроенный ACS-клиент протокол (shared lib — тот же набор, что в AccountCacheServer) |

Всего RPC-команд ~590+ — **в ~8 раз больше ACS (74)**. Полные списки в артефактах.

## 6. DB-контракт

- exe ссылается на **789 уникальных `aion_*` процедур** (`cached-ref/exe-procs-789.txt`)
- в прод-БД `_AionWorldNew114_rc` существует **781** (`cached-ref/db-procs-world-781.txt`)
- отсутствуют **140** (`cached-ref/procs-missing-140.txt`) — в основном СТАРЫЕ версионные суффиксы (`_20090423/_20101025/...` — бинарь держит мультиверсионный dispatch, в БД лежат новые версии); реальные дырки — известный список (LoadFameInfo(3003) и др.)
- тела процедур можно снять с VM read-only (OBJECT_DEFINITION) — R3

## 7. Публичный передний край (web-ресёрч)

- **Ноль публичных реализаций/портов Aion CacheD64**: github пуст (CacheD64/l2cached/aionworld — ложные совпадения), L2J вообще не имеет CacheD-слоя.
- RageZone: 5.8 PTS leak тред (402 реплая), «Aion 7.7 C++ server files» 1205286 (PTS-бинари с PDB, ссылки под логином), 4.6 PTS треды, AKllX cracked matchmaking 4.6. HTML-копии тредов 1197401/1205208/1205286(p1-7)/1211744 + mmo-dev cached-тред сохранены в `nextgen/cached-ref/ragezone/`.
- mmo-dev: настройки conn CacheD/L2AuthD в **реестре** (PROJECT_L2/NCSoft) — PTS-паттерн; наш CacheD64 хранит connStr аналогично (как ACS: Load/SaveConnStrToReg).
- L2-референс: **L2 CacheD MasterToma C1** (локально `~/STELGEN/tmp/authd-research/artifacts/l2_c1/l2_c1/CacheD`, 692 файла): `src/{model(CUser/CItem/CPledge/CTransaction/CWarehouse...),network(CServerSocket/CNpcDbSocket/CCacheDServer/CAdminServer),threads,config,data}`, `reversed/Cached.h` 6504 строки IDA-типлибы, `generated/Cached.c`, RPC **без крипты** (DummyCrypt), handler-таблицы. Это архитектурный образец, НЕ паритет.
- Вывод: **мы идём первыми** — методика authd/ACS (fork-proxy capture + pdbpub + дизasm + Go) уже отработана 4 раза.

## 8. План MVP (R0-R6)

- **R0 ✅ (этот чат)**: артефакты сняты, словари вскрыты, DB-контракт посчитан.
- **R1 capture**: wire 2006 НЕ трогая мир — `pktmon` на VM (filter port 2006 → pcapng → парс офлайн). Фрейминг ожидается `[u16 len][u16 opcode]` (как 2110/2220). Плюс анализ готовых логов `CacheServer/log/*.log` (171 файл, 356МБ — там RPC-строки с параметрами!).
- **R2 дизasm**: dispatch-таблица ServerToDb по opcode + фрейминг; objdump + map (метод AuthGateD).
- **R3 `nextgen/aion-cache` (Go)**: wire 2006 + MapStore RAM-моделей (User/Item/Guild/...) + DB-слой (тела 781 procs снять в гит) + Admin-канал + Log-клиент 2051 + IC-клиент 2305.
- **R4 A/B**: второй инстанс на копии порта + pktmon-сверка трафика.
- **R5 свитч по «го»**: common.xml cacheServerPort / правка конфига Server64; откат — вернуть конфиг.
- **R6 наблюдение**: abyss-цикл 60с, логины, трейд/аукцион.

Оценка объёма: RPC 590+ команд против 74 у ACS → самый большой компонент трека B (оценка 2-4 недели итерациями). Стратегия снижения риска: MVP сначала только read-путь (char login: RQ_CHARACTER_LIST/RQ_GET_ITEM...) + write-путь транзитом в SQL, остальное — постепенно; Server64 при отсутствии ответа ждёт (как мир при «8/16»).

## 9. Шансы: **~85%**

- ✅ Полная символика (pdb+map) — редкий случай даже среди NCsoft-компонентов
- ✅ DB-контракт целиком в нашей БД (781/789)
- ✅ Словари RQ/RP/GQ/GP сняты, параметрика Decode* в publics
- ✅ Методика отработана 4 раза (logd, captcha, gate, authd) + ACS-аналог изучен
- ⚠ Неизвестно: wire-фрейминг 2006 (закрывается R1 pktmon), семантика RAM-мутаций и порядок записи, роль 2009, IC-подпротокол
- ⚠ Самый объёмный компонент — риск не в сложности, а в количестве команд

## 10. Охота за сурсами CacheD (итог 08.10, вторая волна)

**Публичный вердикт подтверждён ещё раз:** сурсов/эмуляторов Aion CacheD НЕ существует (GitHub repo-search `l2cached` = 0 репо; все Aion-эмуляторы пишут в БД DAO-слоем напрямую — Aion-Core 4.7.5 = AC-Game/AC-Login/AC-Chat, MySQL, БЕЗ кеш-компонента; L2J без CacheD-слоя). Единственный публичный reversed CacheD во всём NCsoft-наследии = **L2 CacheD C1 (MasterToma)** — он уже был в гите (authd-ref/l2-c1-mastertoma), теперь **скопирован самодостаточно в `cached-ref/l2-c1-cached/`** (7.7МБ, src+reversed+generated).

### 10.1 L2 CacheD C1 = живой шаблон нашей архитектуры (1-в-1 соответствие каналов)

| L2 CacheD C1 (MasterToma) | Aion CacheD64 (наш) |
|---|---|
| `serverHandlers/` (GameServer↔CacheD): packet000_CacheVersion, 001_LoadCharacter, 002_CreateCharacter, 003_CreateItem, 005_LoadItems, 009_SaveCharacter, 010_SaveItems, 018-022 Warehouse, 029/030 CharacterLogin/Logout, Pledge/Castle/Agit... | **2006**: ServerToDb/DbToServer (RQ_/RP_ 382/255) |
| `adminHandlers/` (GMServer): 01_CheckCharacter, 02_SetCharacterLocation, **03_SetBuilderCharacter**, 04_ChangeCharacterName, 06/07/08 Add/Del/ModSkill, 12_AddItem... | **2007 interactive**: AdminToCache/CacheToAdmin (**GQ_/GP_ 55/53** — GQ_MAKEBUILDER/GQ_ITEMADD/GQ_SET_BUILDER_CHAR 1-в-1 аналоги!) |
| `npcDbHandlers/` (L2NPC): 01_NpcDbVersion, 02_LoadNpcRequest, 03_SaveNpcInfo, 04_UpdateBossNpcValue | **2009 (гипотеза закрепилась!)**: третий листенер = NPC-DB канал NPCSvr64 |

### 10.2 Wire L2 C1 (приор для нашего R1) — из CServerSocket.cpp

```cpp
m_packetSize = (buf[off+1] << 8) + buf[off] - 2;  // [u16 LE] ДЛИНА САМОИНКЛЮЗИВНАЯ (как наш 2110!)
DummyCrypt::Decrypt(payload, m_key, m_packetSize); // XOR-стрим с rolling-ключом
m_key += m_packetSize;                             // ключ двигается на длину пакета
// хвост: [lastByte][preLastByte] == checksum == m_packetSize
```
Фрейм: `[u16 LE self-len][opcode][payload][2Б checksum]`, XOR-крипта с прокруткой. Наш 2006 вероятно эволюция той же схемы (крипта/чексум могут отличаться — pktmon закроет за минуты).

### 10.3 Что скачать юзеру (бусты, которые мне не дотянуться)

1. **RZ-аттачменты PTS-паков Aion** (нужен твой логин): 5.8 PTS VM (Mantios), 7.7 C++ pack (1205286/fyyre), свежий 2.7 PTS (июль 2026) — в каждом родной CacheServer64 своей версии → кросс-версионный дифф строк (эволюция словарей ACQ/RQ) для верификации dispatch. Выложить на шару 192.168.0.248:3923 — заберу.
2. **L2 PTS паки новых хроник** (HighFive/Gracia) с L2CacheD.exe — если в строках новых L2 CacheD появятся RQ_/RP_, маппинг L2↔Aion станет прямым по именам.
3. НЕ нужно качать: Aion-Core 4.7.5 (семантика слабее наших Mobius 7.7/beyond-aion 4.8), любые «aion cached» — их нет.
4. Главный буст вообще не скачивается: **R1 pktmon capture 2006**.

## 11. Артефакты

- `nextgen/cached-ref/` — publics, rpc-map, словари, procs-списки, strings, ragezone/mmo-dev HTML
- `~/STELGEN/projects/aion_rev_2026-10-05/artifacts/pdb-big/CacheD64/` — exe+pdb+map+конфиги+log (вне гита, 140МБ)
- Обновить `manifest-bin.md`: добавить CacheD64 строки.

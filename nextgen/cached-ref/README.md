# cached-ref — референс-сурсы и артефакты ресёрча CacheD64 (порт 2006/2007/2009)

> Ресёрч: [aion-cache/RESEARCH.md](../aion-cache/RESEARCH.md) (R0 закрыт 08.10, шанс ~85%). Цель: `aion-cache` (план) — RAM-кэш мира, единственный ODBC-писатель в `_AionWorldNew114_rc`.
> Самый большой компонент трека B: ~590 RPC-команд (8× ACS) — MVP = read-путь (char login/item load) + write-транзит в SQL.

| Файл/каталог | Что это |
|---|---|
| `cached-rpc-map.md` | ПОЛНАЯ класс-карта RPC: DbToServer 303 / ServerToDb 192 (+Update 31) / AdminToCache 33 + CacheToAdmin 17 (GM/web: MakeBuilder/SendMail/Captcha/Housing/Poll*…) / ICToCache+CacheToIC 7+7 / ICClient 9 / DBConn 26 / домены (ItemDb 40, UserDb 23, VendorDb 21, GuildDb 16…) / LogSvc 15 / OverseasEventSystem 6 |
| `rpc-opcodes-and-procs.md` | словари опкодов из strings: RQ_ 382 / RP_ 255 (мир-канал), GQ_ 55 / GP_ 53 (builder/GM), ACQ_ 39 / ACP_ 28 (встроенный ACS-клиент) |
| `pdb-publics-14281.txt` | полные publics из CacheD64.pdb (106МБ на VM, MD5 `979ae355`; локально `aion_rev/artifacts/pdb-big/CacheD64/`) |
| `class-counts.txt` | счётчики publics по классам |
| `db-procs-world-781.txt` / `exe-procs-789.txt` / `procs-missing-140.txt` | DB-контракт: exe ссылается 789 aion_* procs, в прод-БД есть 781, 140 старых версионных отсутствуют (exe держит мультиверсионный dispatch) |
| `strings-ascii.txt` / `strings-u16.txt` | строки exe (ASCII + UTF-16LE) — источники словарей/конфигов |
| `l2-c1-cached/` | **L2 CacheD C1 MasterToma — единственный публичный reversed CacheD во всём NCsoft-наследии** (скопирован самодостаточно): src/model (CUser/CItem/CPledge/CTransaction/CWarehouse), network (CServerSocket/CNpcDbSocket/CCacheDServer/CAdminServer — карта каналов 1-в-1 к нашим 2006/2007/2009), reversed/Cached.h (6504 строки IDA), generated/Cached.c, DummyCrypt (L2 C1 cache-RPC БЕЗ крипты) |
| `ragezone/` | HTML-копии тредов (5.8 PTS leak 402 реплая, 7.7 C++ server files 1205286, AKllX 4.6 matchmaking) |

## Карта каналов (гипотеза закреплена 08.10)

| L2 CacheD C1 (MasterToma) | Наш CacheD64 | Что |
|---|---|---|
| serverHandlers (283) | :2006 (RQ/RP) | мир-канал Server64 |
| adminHandlers | :2007 (GQ/GP) | builder/GM interactive |
| npcDbHandlers | :2009 | третий листенер — NPC-DB канал NPCSvr64. ⚠️ 08.10 live-netstat: гипотеза «клиент = NPCSvr64» ОТПАДЕНА — NPCSvr ходит в CacheD по 2006; 2009 в бою без клиентов (aion-npc ROADMAP леджер); кто клиент 2009 — ⏳ |

L2 wire: `[u16 self-len LE][opcode][payload][2Б csum]` + rolling-XOR DummyCrypt — вероятно эволюция в наш 2006; закрыть capture R1 (pktmon).

## Готчи R1 (акту. 10.10)

- ⚠ **pktmon на VM НЕ пишет loopback** (доказал aion-npc R1 на 2002: pktmon comp = только VirtIO) — 2006/2007/2009 = loopback → pktmon-план R0-эпохи ОТМЕНЁН. Обходы: fork-стенд копия CacheD (:2016) + Server64 на неё (по «го»), тест-мир LAN, wire-каркас из логов.
- Готовый материал без capture: `log/` 171 файл 356МБ — **.profile = нумерация всех 8 протоколов** (profile-opcode-map.md), **.err = SQL-транзит с полными параметрами** — локально в `aion_rev/artifacts/pdb-big/CacheD64/CacheServer/log/`.
| `profile-opcode-map.md` | **НУМЕРАЦИЯ опкодов всех 8 протоколов из log/*.profile (DBProfiler)**: DB2Server 238 RP / Server2DB 381 RQ / Log2Server 6 + Server2Log 13 (LP_) / IC2DB 11 + DB2IC 9 / NPRelay2Server 55 + Server2NPRelay 52 |
| `diff-dict-58-77.md` | дифф словарей 5.8 vs 7.7: append-only (0 удалений; RQ+31/RP+19/GP+1) → dispatch = superset |
| `58-cached/` | конфиги 5.8 CacheD (config/common.xml + DBLogDetail/DBLogSummary); exe 5.8 вне гита: `aion_rev/artifacts/kits-58/Cached/` |
| `ragezone/mmodev-1164-cached-ex-packets.html` | mmo-dev 1164 «Метод добавления новых расширенных пакетов в CacheD» (kick, hook Bind @0x460547) — публичная часть |

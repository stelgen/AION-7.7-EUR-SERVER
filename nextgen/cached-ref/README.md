# cached-ref — референс-сурсы и артефакты ресёрча CacheD64 (порт 2006/2007/2009)

> Ресёрч: [../CACHE-RESEARCH.md](../CACHE-RESEARCH.md) (R0 закрыт 08.10, шанс ~85%). Цель: `aion-cache` (план) — RAM-кэш мира, единственный ODBC-писатель в `_AionWorldNew114_rc`.
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
| npcDbHandlers | :2009 | третий листенер — NPC-DB канал NPCSvr64 |

L2 wire: `[u16 self-len LE][opcode][payload][2Б csum]` + rolling-XOR DummyCrypt — вероятно эволюция в наш 2006; закрыть capture R1 (pktmon).

## Готчи R1

- НЕ mirror-прокси на 2006 (рестарт Server64 дорогой) — **pktmon filter port 2006** на VM.
- Готовый материал без capture: `log/` 171 файл 356МБ (RPC-строки с параметрами, .memory/.itemload/.leak) — локально в `aion_rev/artifacts/pdb-big/CacheD64/CacheServer/log/`.

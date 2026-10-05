# 🔬 Замена LogServer64 → `aion-logd` (Go): протокол вскрыт, оценка

> Анализ 05.10.2026. Метод: LogServer64.pdb (MD5 сверен) → strings + mini-PDB-publics-парсер
> (`~/STELGEN/tmp/pdbpub.py`) → objdump-дизasm ключевых функций. Прод не трогался.
> Артефакты: `pdb-big/AION_LIVE_SERVER/LogServer/LogServer64.pdb`, дизasm `/tmp/logsrv.asm`.

## 1. Архитектура NC-логгера (общак)

Единый фреймворк `..\Shared\LogClientSocket.cpp` — **один и тот же класс во всех NC-бинарях**
(Server64/NPCSvr/CacheD/ICServer...): `LogClient → LogClientSocket → TCP 2051 → LogServer`.
LogServer64 = тонкий приёмник: **именно он пишет** per-service `.err/.log` файлы (формат
`%s\%04d-%02d-%02d.err` найден в нём!), держит GUI-окно, шлёт email (`aionlogd@ncsoft.com`),
и пишет в БД. Вывод: **замена логгера = замена LogServer64; сервисы не трогаются вообще**.

## 2. Протокол wire (вскрыт дизasm'ом OnRead/GetCmd_LP/обработчиков)

Фрейминг (тот же стиль, что у auth-гейта):
```
[u16 LE total_len][u8 type][0xBB][u8 ~type][payload...]   len ≤ 0x2000, type < 5
```
GetCmd_LP проверяет маркер: `expected marker=0xBB, got=..., packetType=%d` + инверсия `~type`.

Диспетчер = `logClientPacketTable` (.rdata 0x1400920E0, 5 записей):
| type | handler | суть |
|---|---|---|
| 0 | 0x140027B90 | **Version**: builderNumber u32 **≥ 10003** (0x2713), иначе лог «incorrect version builderNumber» |
| 1 | 0x140027E70 | строит ответ сервера: `[13 00][02][BB][FD][qword]` = **VersionAndTime** |
| 2 | 0x140027F20 | error-печаталка (код 0x80) |
| 3 | 0x140027B80 | заглушка (xor eax,eax; ret) |
| 4 | 0x140027F50 | сброс поля 0x1d4 + лог |

Handshake: клиент шлёт `Version(builderNumber)` → сервер отвечает `VersionAndTime(qword)`.
Живость: `AlivePacket / C_PING / C_RECONNECT` в строках; клиент ретраит коннект
(`Can't connect to log server` — **безопасный роллбэк: недоступность logd не роняет сервисы**).

## 3. Что LogServer делает с данными

- **Файлы**: per-service ежедневные `%s\%04d-%02d-%02d.err` / `.log` (+подкаталог), ротация по
  `CheckLogFileTime/CreateNewLogFile(5×int)`, `FlushLogFile`. **Именно поэтому** наши тейлеры
  в aion-op читают `CacheServer\log\*.err` — пишет их LogServer.
- **Разбор**: `ProcessLog/ProcessLogBatch(LogSvcType, SYSTEMTIME&, wchar_t*)` +
  `ParseLogData(wchar*, char*...)` + спец-декодер `DecodeBotLog`.
- **БД (ODBC)**: таймеры `{call Log_TblGameServerInfo_UpdateLogfreedisk(%d,%d)}`,
  `{call ...UpdateServerstatus(%d,%d,%d)}`, `{call Log_TblGameWorldInfo_InitializeCount(%d)}`
  — наши Phase-A процы; bulk логов: `EXECUTE aion_BulkInsertWide '%s','%s'`,
  `EXECUTE aion_SetInserted '%s','%s',%d...`; BCP-механизм с ретрай-очередью
  (`unsent\batch` — тот самый растущий каталог!).
- **НЕ переносим**: GUI (Log Suppressed и пр.), email-алерты, BCP-файлы, perf/hit-trace каталоги.

## 4. Оценка замены

**Scope**: Go-бинарь `aion-logd`: TCP:2051 (accept ~10 сервисов) → handshake (Version ≥10003 →
VersionAndTime) → dispatch 0..4 → лог-батчи (LogSvcType + wide text) → per-service ежедневные
файлы (в те же директории!) + TDS-процы (go-mssqldb: 3 таймера + 2 bulk) + alive/retry.
SQLite-зеркало логов опционально (одно место для поиска — GUI перекрывается вкладкой aion-op).

**Одна кодовая база**: TCP+файлы+SQLite+go-mssqldb — всё кроссплатформенное, `GOOS=windows/linux`
из одного исходника. На VM логи шлются по TCP — logd может жить где угодно в локалке (в т.ч. Linux).

**Разгадки двух живых итераций (05.10, capture-окна с подменой LogServer):**
1. **Сервер говорит ПЕРВЫМ** (итерация 1): клиенты подключаются к :2051 и ЖДУТ Version —
   `LogServerSocket::OnCreate` сам шлёт Version+VersionAndTime при accept; 3 клиента ретрайнулись за ~90с.
2. **Клиент валидирует SYSTEMTIME в Version** (итерация 2): `logClientPacketTable` — КЛИЕНТСКАЯ таблица
   (общий код, компилируется и в LogServer64 для self-клиента); хендлер 0 @0x140027B90:
   `min ≥ 10003` → копирует body+8..24 как **SYSTEMTIME (16 байт, 8 WORD)** → `SystemTimeToFileTime`
   → сравнение с локальным (мэджик 0x68DB8BAD = /10^7) — **источник «Time difference with Log-Server»**
   (RunAsDate-инцидент Server64!). Наш FILETIME+нули = год-мусор → клиент молча не продолжает
   (все 3 клиента ESTABLISHED, 0 байт). ФИКС: `proto.SysTime()` — настоящий SYSTEMTIME, локальное время VM → diff≈0.

**Ещё pending**: payload лог-батча (кандидат type 4 / ProcessLog путь), клиентская реакция после
валидного Version — покажет итерация 3 (live capture).

**Трудозатраты**: реализация 2 д + протокол-тесты (фейк-клиенты по типам) 1 д + параллельный
прогон (logd на :2052, diff файлов с оригиналом за сутки) 1 д + переключение/откат 0.5 д + запас
1 д ≈ **4–6 дней до прод-замены**. Код ~1000–1500 строк.

**Риски**: (а) payload лог-батча — закрывается дизasm'ом; (б) клиент может проверять ответ
сервера — тоже дизasm; (в) кодировка текста по wire (ожидаемо UTF-16LE raw — `wchar_t*` в API);
(г) роллбэк тривиален: вернуть задачу AionLog, сервисы ретраят.

**Ценность**: ACP=28591/DSN-трипанема умирает, логи — наши (SQLite+файлы), `unsent\batch`-
мусор исчезает, aion-op работает без изменений (файлы те же).

## 4.1 ИТОГИ ЖИВЫХ ИТЕРАЦИЙ + NPCSvr.pdb-дизasm (протокол ЗАКРЫТ на 95%)

**Маркеры по направлению**: клиент→сервер = **0xBA**, сервер→клиент = **0xBB**; инверсия = ^type в обоих.

**Пакеты (все подтверждены capture и/или дизasm'ом NPCSvr64-энкодера):**
| Тип | Направление | Layout | Суть |
|---|---|---|---|
| 0 Version | C→S: **13** total `[00][BA][FF][builder u32=200604][min u32=10003]` (БЕЗ SYSTEMTIME) | S→C: **29** total `[00][BB][FF][builder u32][min u32][SYSTEMTIME 16]` (OnCreate) + S→C VT-2: **13** `[02][BB][FD][qword FILETIME]` | handshake; клиент валидирует **SYSTEMTIME** сервера (мэджик 0x68DB8BAD=/10^7) — источник «Time difference with Log-Server» |
| 3 ServerStarted | C→S: **17** total, body 12 = 3×u32 (capture: (1,2,0),(1,3,0),(1,4,-22)) | старт-нотификация |
| 4 Control | C→S: **21** total, body 16 = **конст 0x644=1604** + 3×u32 (строитель @0x1402c8860 NPCSvr) | контрол-пакет (TBD) |
| 5 Status/Record | C→S: **body 194 CONST** (~317 за 4 мин ≈ 1/2с от каждого из 3 клиентов) | структурированная запись: `u32@0 = LogSvcType` (capture: **301/302/309** — по типу на сервис), u64@8 engtick, **f32@32/36/40 = X/Y/Z координаты**, u32@28 = WorldID (210010000!), флаги 0x80000000, счётчики; **хвост: SYSTEMTIME (16) на off 162 + u16 seq** — это данные для TBL_GAME_WORLD/SERVER_INFO |
| 11 Alive | C→S: **13** total `[0B][BA][F4][qword]` (строитель @0x1402aa5d8, EncodeAlive; #180 патчил байт-параметр) | ping |

**ГЛАВНЫЙ АРХИТЕКТУРНЫЙ ВЫВОД**: в LogSvc (клиентский фреймворк) есть СВОИ `CreateNewLogFile/LogFileUnLock/gLogDir` — **текстовые .err-логи пишут САМИ СЕРВИСЫ ЛОКАЛЬНО**; LogServer64 их не получает и не пишет! Он принимает только: handshake, ServerStarted, type-5 записи (для TBL_GAME_* в Aion_log + GUI), Alive, Ctrl.

**Следствия для замены**: (1) aion-op тейлеры .err вообще не зависят от замены — файлы пишет CacheD/Server64/NPCSvr сами; (2) logd-замене нужен только приём type-5 записей → раскладка в TBL_GAME_WORLD/SERVER_INFO + aion_SetInserted/aion_BulkInsertWide; (3) роллбэк тривиален; (4) «ненужность» LogServer для мира доказана живыми окнами (4-8 мин × 3, клиенты ретраят, мир цел).

**Capture-артефакт**: `~/STELGEN/tmp/logd-io/io/2026-10-05.io.hex` (1 МБ, 317 type-5 записей + handshakes) — samples для юнит-тестов парсера.

## 5. Что дальше (по «го»)

1. Добить дизasm (3 пункта выше) → точная спецификация payload.
2. `aion-logd` скелет + фейк-клиенты-тесты.
3. Параллельный прогон → свитч. Каждый шаг по «го».

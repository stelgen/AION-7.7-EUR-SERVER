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

**Осталось дизasm'ить перед кодом (~2–3 ч)**: точный payload лог-батча (LogBuffer/SendIOBuffer/
ParseLogData), клиентская проверка ответа сервера (LogClientSocket::OnRead — приватная, берётся
через xref), семантика EncodeAlive (алайв-интервал; #180 патчил его байт — параметр, не крипта).

**Трудозатраты**: реализация 2 д + протокол-тесты (фейк-клиенты по типам) 1 д + параллельный
прогон (logd на :2052, diff файлов с оригиналом за сутки) 1 д + переключение/откат 0.5 д + запас
1 д ≈ **4–6 дней до прод-замены**. Код ~1000–1500 строк.

**Риски**: (а) payload лог-батча — закрывается дизasm'ом; (б) клиент может проверять ответ
сервера — тоже дизasm; (в) кодировка текста по wire (ожидаемо UTF-16LE raw — `wchar_t*` в API);
(г) роллбэк тривиален: вернуть задачу AionLog, сервисы ретраят.

**Ценность**: ACP=28591/DSN-трипанема умирает, логи — наши (SQLite+файлы), `unsent\batch`-
мусор исчезает, aion-op работает без изменений (файлы те же).

## 5. Что дальше (по «го»)

1. Добить дизasm (3 пункта выше) → точная спецификация payload.
2. `aion-logd` скелет + фейк-клиенты-тесты.
3. Параллельный прогон → свитч. Каждый шаг по «го».

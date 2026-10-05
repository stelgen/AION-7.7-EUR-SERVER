# 🧾 ПОЛНЫЙ СНИМОК СТАТУСА — aion-logd (замена LogServer64) на 05.10.2026

> Консолидация ВСЕХ находок итераций 1–6. Прод: **наш logd в бою на :2051** (SYSTEM-задача
> AionLogCap → D:\SAION\aion-logd\run.cmd, откат = schtasks /run AionLog).

## ✅ ОБНОВЛЕНИЕ 05.10 (вечер): Л1–Л4 ЗАКРЫТЫ И ПОДМЕНЕНЫ НА ПРОДЕ

**ПРОД: 2 подмены (12:28 и 12:36), финальный exe MD5 6c8d2623, PID 7220 (SYSTEM-задача AionLogCap →
D:\SAION\aion-logd\run.cmd). Верифицировано: 3 клиента ESTABLISHED, InitializeCount(svc=3),
мир жив (7777), parse-ошибок нет, координаты человеческие (fix floatFrom bits→float).
Бекапы: aion-logd.exe.bak-0510 + logd.log.bak-0510 в D:\SAION\aion-logd\. Готча: после /end
ждать смерти процесса (copy валился «file in use» при 3с — нужно до 10с wait-loop).**

- **SHIP E2E ЧЕРЕЗ НАСТОЯЩИЙ RSYSLOG 8.2504 (валидация, приёмник потом убран по требованию):**
  UDP (датаграммы) и TCP (octet-counted RFC6587) — оба транспорта долетели 100%: start/conn.up/
  version/server.started/status×3/text/parse.err(+raw)/conn.down/self/stop; PRI корректные
  (14 info / 12 notice / 11 err); self-статус с sent/dropped/ошибками. Прод до приёмника не шипит
  (ship.enabled=false в прод-конфиге — секция на месте, включить одной строкой когда rsyslog на LAN).
- **ДИЗASM ТЕЙЛА type-9 (LogServer64.exe+pdb локально):** найден конвертер MsgId→wire-type
  @0x1400130ca: **0x644→4 (Control), 0x645→5 (Status), 0x646→6, 0x647→8, 0x648→9 (TextLog)** —
  клиент строит записи с внутренним MsgId, фрейминг общий (Shared\LogClient.cpp). type-6/8
  существуют (type-8 видел в capture). Полная раскладка хвоста (тик/floats/флаги) набивается на
  call-sites **Server64** — отдельная сессия с его PDB (284МБ); у нас уже: entries/онлайн, world,
  stamp, tick, floats — всё шипится + raw сохранён. Anchor'и: LogBuffer ctor 0x14023290,
  CreateFileLogBufferCollectorThread 0x14023680, SendServerStarted 0x14026820 (VA=ImageBase+off!).
  Disasm: /tmp/logsrv.asm 136k строк, publics /tmp/logpub.txt (5417).

- **Л1 type-9 парсер ГОТОВ** (`internal/textlog`): layout подтверждён живыми сэмплами —
  `[u32 id=928][ {u32 key][wchar name NUL]... }[tail][SYSTEMTIME 8WORD = последние 16 байт]`;
  entries = онлайн-сессии (1002,"SteLGeN")(1010,"Stelgen") — char+аккаунт; 223b = 1 сессия,
  251b = 2 сессии; мир-подобный u32 в хвосте (210010000/210040000) выделяется; недекодированный
  хвост — в hex+raw. Fixture'ы = боевые capture + mirror-эталон (testdata/textlog_records.txt).
  Пишутся per-service .err/.log (каталог = svc из ServerStarted коннекта, fallback textlog).
- **Л2 TBL-сверка**: READ-ONLY проверка прод-БД — TBL в схеме **aiongm_ur** (не dbo!);
  проц UpdateMainStatus/UpdateTotalMainStatus на проде ОТСУТСТВУЮТ (в Aion_log только 3 наши
  Log_Tbl*) ⇒ оригинал-то их звал в вечный 2812 — зона-счётчики НЕ писал НИКОГДА на нашем
  деплое, мы уже на паритете. Методы UpdateMainStatus/UpdateTotalMainStatus добавлены в logdb
  (позиционные {call}), вызов ЗАГЛУШЕН до деплоя REF58-проц + подтверждения маппинга metric1-4.
- **Л3 InitializeCount**: теперь на ServerStarted **svc=3** (MAIN/мир, mirror: data-коннект
  (1,3,0)), config `server.init_svc` (0 = старое поведение «первый»).
- **Л4 retention**: writer.Sweep — чистка base_dir старше `server.retention_days` (default 14),
  при старте + каждые 30 мин, событие ev=sweep.
- **ship-стандарт ГОТОВ** (`internal/ship`, ~450 строк, без зависимостей): syslog RFC5424
  udp/tcp(octet-counted) + HTTP ndjson + локальный ndjson-фолбэк (только явно); неблокирующая
  очередь + drop-счётчики, recover на каждой отправке, self-статус ev=self каждые 300с,
  события: start/stop/conn.up/conn.down/version/server.started/status/text/parse.err/db/db.err/sweep/self.
  Спека для ВСЕХ переписей: nextgen/TELEMETRY-SPEC.md. По умолчанию ship disabled — прод без приёмника.
- Тесты: textlog на живых fixture'ах (3 реальных пакета), ship (UDP/TCP/file/недоступный-синк),
  все зелёные; exe: linux+windows (build/), новый exe staged на VM C:\Temp\logd-new\ (НЕ запущен).

## ТЕКУЩИЙ ЖИВОЙ СТАТУС (10:00+ VM)

- **:2051 = наш aion-logd.exe** (PID 6180, SYSTEM-задача AionLogCap, каталог C:\logd-capture\)
- 3 клиента ESTABLISHED: NPCSvr 2232, Server64 7348, CacheD 992
- **ИЗВЕСТНЫЙ ОСТАТОЧНЫЙ БАГ: флап клиентов каждые ~153с** (6 реконнектов за 10:47–10:51)
  — периодика type-4 у нас 120с, у оригинала 90с; ProcessAliveResponse-гипотеза не закрыта
- **DB-слой РАБОТАЕТ**: InitializeCount(world=1) выполнен ×2, FREE_DISK=89 записан (был 56)
- status-CSV пишутся: logs\{svc301,svc302,svc309,io,capture.raw,badstatus.raw}\2026-10-05.status.csv
- mirror-io.hex (3.8 МБ) = ЭТАЛОННЫЙ capture оригинала (mirror-прокси 2051→2053,
  оригинал временно на :2053 через common.xml — ВОССТАНОВЛЕН на 2051)
- capture.yaml на VM: capture_all=true + logdb.enabled=true (боевой прогон-конфиг)

## АРХИТЕКТУРА (вскрыто полностью)

NC-логгер: общий фреймворк `..\Shared\LogClientSocket.cpp` во ВСЕХ бинарях.
**КРИТИЧНОЕ ОТКРЫТИЕ: .err/.log-файлы пишут САМИ СЕРВИСЫ локально** (LogSvc::CreateNewLogFile
есть в клиенте) — LogServer64 их не получает. Он принимает ТОЛЬКО телеметрию → TBL_GAME_*
(Aion_log) + GUI/email/BCP. Следствие: aion-op тейлеры не зависят от замены; замена = приём
type-5 → DB. Мир вообще не зависит от LogServer (доказано 6 подменами-окнами).

## ПРОТОКОЛ 100% (mirror + live capture + дизasm NPCSvr64.pdb/LogServer64.pdb)

**Фрейм**: `[u16 LE total][u8 type][marker][^type][payload]`, total≤0x2000.
**Маркеры по направлению: C→S=0xBA, S→C=0xBB** (WriteTypeMarker @0x1402aa780 NPCSvr:
[type][BA][~type]; SendIOBuffer=@0x1402aa460 NPCSvr, 5 вызовов).

| Тип | Напр. | Size | Body | Суть |
|---|---|---|---|---|
| 0 Version | C→S | 13 | [builder u32=200604][min u32=10003] | БЕЗ SYSTEMTIME (строитель @0x1402aa680: builder из глобала 0x14179972c) |
| 0 Version | S→C | 29 | [builder u32=200601][min u32][SYSTEMTIME 16] | OnCreate LogServer шлёт ПЕРВЫМ + сразу type-2 |
| 2 VerAndTime | S→C | 13 | [qword FILETIME] | клиент FileTimeToSystemTime→сравнение с локальным (мэджик 0x68DB8BAD=/10^7) |
| 3 ServerStarted | C→S | 17 | 3×u32: (id=1, svc, x) | svc=2(NPC)/3(MAIN)/4? x=0/-141/-182; строители @0x1402aa6d0/aa680 |
| 4 Control | S→C | 13 | 8 нулей | **ПЕРИОДИКА ОРИГИНАЛА КАЖДЫЕ 90с** (mirror: 10:43:24/10:44:54) — наш 120с не спас |
| 5 Status/Record | C→S | 199 | 194 const: SvcType@0 (301/302/309 = по клиенту!), Metric1@4, TickRaw u64@8 (НЕ mono), M2-4@16-28, WorldID@28 (флаг 0x80000000; 210010000/220010000!), f32 XYZ@32/36/40, Fields[134]@44, SYSTEMTIME[8]u16@178, EngineMs@188 (MONO!) | ~1/2с от каждого клиента; данные TBL_GAME_* |
| 9 **TEXT-ЛОГИ** | C→S | var (218 в capture) | UTF-16 записи: [len u32][charid u32][name wchar][...] | `(928,"SteLGeN")(1010,"SteLGeN")` — онлайн-таблица/текст-лог (14 шт за окно) |
| 11 Alive | C→S | 13 | [qword] | EncodeAlive; **оригинал НЕ отвечает** (mirror: s2c-11=0); #180 патчил его параметр |

Handshake: accept → СЕРВЕР шлёт Version(29)+VerAndTime(13) первым; клиент валидирует
**SYSTEMTIME из Version** (8 WORD — FileTimeToSystemTime от него) — источник
«Time difference with Log-Server» (RunAsDate-инцидент!). Наш SysTime(t) = локальное время VM.

## КОД (nextgen/aion-logd, ~1000 строк Go, один исходник win+linux)

- proto: Build/BuildC/Parse (оба маркера, type≤12, ErrShort/ресинк), VersionBody(+SysTime 8WORD),
  ClientVersion, ServerStarted, VersionAndTime (putFileTime: локальное-как-UTC!),
  AliveReply (в коде есть, в свитче убран — оригинал не отвечает)
- server: serve() = server-initiates-first; накопление по len; бad-packet ресинк; per-conn
  goroutine type-4 каждые 120с (⚠ ИЗМЕНИТЬ НА 90с!); dispatch: Version(ответ+VT)/VTReq(ответ)/
  ServerStarted(logdb.InitializeCount retry-atomic)/Status(ParseStatus→WriteStatus CSV)/
  Alive(молча)/default(.raw)
- records: StatusRecord.ParseStatus (194) — см. таблицу; String()→CSV; fixture 317 живых сэмплов
  testdata/status_records.txt; тесты-инварианты: EngineMs mono per-client, seq mono, год 2026
- writer: per-day .err/.log (формат оригинала) + WriteStatusCSV (svc301/302/309/*.status.csv)
  + WriteRaw (.tN.hex) + WriteIO (rx/tx ДО парсинга — ключевой инструмент capture)
- logdb: UpdateLogfreedisk/UpdateServerstatus/InitializeCount — **Sprintf-интами в строку**
  (go-mssqldb НЕ умеет ?-плейсхолдеры в exec; named @p=? тоже не взял; инъекция невозможна — числа);
  полная квалификация **Aion_log.dbo.*** (conn с database=master!); RunTimers (freedisk 300с
  FreeMB→ГБ кламп 255/tinyint, serverstatus 600с); InitializeCount по первому ServerStarted,
  retry-atomic при провале; FreeMB: windows GetDiskFreeSpaceExW / linux Statfs (build tags)
- тесты: proto (layout/roundtrip/оба маркера), records (317 живых), logdb (SQL-строки на фейке),
  server e2e (handshake/разрезанные пакеты/.raw); бинари: linux 5МБ, win 8МБ + mirror 8МБ

## DB (сделано на проде)

- Логин aionop_ro (был для aion-op): GRANT EXECUTE на **dbo.**Log_TblGameServerInfo_UpdateLogfreedisk/
  UpdateServerstatus/Log_TblGameWorldInfo_InitializeCount (Aion_log) — ПЕРЕВЫДАНЫ (схема dbo!),
  логDB-grants.sql/logdb-grants2.sql на VM C:\aionop\; GRANT UPDATE на aiongm_ur.TBL_GAME_WORLD_INFO/
  TBL_GAME_SERVER_INFO (ownership chaining не работает cross-schema — прямой UPDATE обязателен)
- ЖИВАЯ ПРОВЕРКА под aionop_ro: EXEC InitializeCount = 193 rows affected; EXEC freedisk(56) = OK
- ФАКТЫ DB: процы в dbo, таблицы в aiongm_ur; FREE_DISK = ГБ (tinyint, кламп 255);
  conn-string logd: database=master (если менять — только на Aion_log)

## УРОВНИ ИТЕРАЦИЙ (вся хронология)

1. Скелет+первая подмена: клиенты ретрайнули (~90с), capture пуст → сервер говорит ПЕРВЫМ
2. Server-first: все 3 клиента EST, но 0 байт → SYSTEMTIME-валидация (мс-поле 696)
3. WriteIO-дамп до парсинга → всё видно: маркеры 0xBA/0xBB, client Version 13b, type5 194b
4. Парсер type5 на живых сэмплах (317 fixture) + CSV per-day
5. DB-слой: Sprintf-EXEC,Qualifier Aion_log.dbo., InitializeCount выполнен, FREE_DISK=89
6. Mirror-прокси (оригинал на :2053): полный протокол (type-4 периодика 90с, type-9 текст-логи,
   builder 200601/200604, alive-ответ НЕ нужен) + прогон с периодикой 120с → флап остался

## ЧТО ОСТАЛОСЬ (след. сессия)

1. **Флап 153с**: периодика 90с (не 120!) — первое, что попробовать; если не спасёт —
   дизasm ProcessAliveResponse (пуб @0x1402a8d60 NPCSvr — начало функции не найдено чисто,
   тело в /tmp/npcs.asm 1.2M строк; adb: поиск по 0x68DB8BAD в /tmp/npcs.asm) или сравнение
   таймингов mirror (ориг. type-4 @10:41:54+90с → reconnect НЕ случился у оригинала)
2. **Текст-логи type-9**: парсер UTF-16-записей → .err файлы (сейчас только CSV type-5;
   type-9 сыпется в .raw — но это «кто онлайн»/текст — нужно добрать layout до конца)
3. **Свитч**: после нуля флапа — оставить logd на :2051 навсегда (задача AionLogCap),
   AionLog disable; откат = schtasks /run AionLog
4. Инициализация: InitializeCount теперь на ServerStarted — но оригинал зовёт при СТАРТЕ МИРА
   (пересмотреть: может на первом conns-пакете от каждого типа клиента)

## ФАЙЛЫ/АРТЕФАКТЫ

- Репо: nextgen/LOGD-REWRITE-ANALYSIS.md (анализ+4.1), nextgen/aion-logd/ (код)
- VM: C:\logd-capture\ (exe/yaml/логи/mirror-io.hex 3.8МБ/run-cap.cmd/mirror.cmd/common.xml+orig)
- Локально: /tmp/logd-mirror/mirror-io.hex, ~/STELGEN/tmp/logd-io/io/2026-10-05.io.hex (1 МБ)
- Артефакты: /tmp/logsrv.asm (LogServer64 полный), /tmp/npcs.asm (NPCSvr64 полный 1.2М строк),
  /tmp/logpub.txt /tmp/npcpub.txt (publics-дампы), pdbpub.py в tools/analysis/
- Коммиты: ea0a024→0579272→d296dc9→487dbca→f55394d→edb10f8 (+ НЕ ЗАПУШЕНЫ правки: logdb Sprintf,
  прогон-конфиг) — ПУШИТЬ СЕЙЧАС

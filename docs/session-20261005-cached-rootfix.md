# Сессия 04–05.10.2026: ROOT CAUSE крашей CacheD + полный цикл фиксов

## Хронология ночи (VM 109, время VM)
| Время | Событие |
|---|---|
| 23:11 | Ребут VM: применён pagefile-фикс 16 ГБ (вариант Б), лимит коммита 44 ГБ |
| 23:13:51 | SQL старт; порт готов ~23:14:30 |
| 23:14–23:52 | Батник v5 падал на [1/12] Wait SQL; автостарт-задачи Disabled |
| 23:42:17 | Краш CacheD64 #1: AV 0xc0000005 @+0x18c31f (после Deadlock detect 23:42:08, IOThread 9, 186с) |
| 23:52–00:14 | Деплой лаунчеров NPC/MAIN/CACHE на десктоп; рестарт стека |
| 00:09:59 | Юзер вошёл в мир (lev18), играл ~5 мин |
| 00:10:03 | CacheD Too slow 3312ms LoadItemsPacket; IOThread виснет |
| 00:13:41→52 | Deadlock detect (IOThread 2, 181с) → AV краш #2; Server64 «Shutdown By NpcSocket Close» 00:13:56→00:14:54 |
| 01:20–02:03 | Деплой CatchBlocked (threshold=5); SQL-твики; LOG/CACHE/MAIN лаунчеры |
| 01:31:45 | Стек поднят, Server64 коннект к CacheD; юзер заходит |
| 01:36:06→16 | Краш #3: Deadlock (IOThread 2, 193с) → AV; живой снимок: session 262 NOP 100% CPU 321с + X-лок db7 |
| 02:03:52 | Краш #4 — тот же паттерн; после KILL 262+274 цикл разорван |

## ROOT CAUSE (финал)
`aion_ProcessWithdrawAccount` (_AionWorldNew114_rc): внутри `BEGIN TRAN` + `XACT_ABORT ON`
3 ссылки на LOOPBACK linked server `[RC-AIONAUTHDB].aionaccdeldb.dbo.del_account`.
Вызов таймером CacheD → компиляция/исполнение зависает (command=NOP, 100% CPU,
self-deadlock compile-семафора с вложенной loopback-сессией) → IOThread CacheD
занят вечно → `CheckIOThreadDeadlock` (порог ~180с) → «Intentional exception» →
AV-краш CacheD → Server64 «Shutdown By NpcSocket Close» → мир падает.
Скрин LogServer64 подтвердил: его CheckIOThreadDeadlock сыпал по IOThread 3/5/9 —
все SQL-клиенты стека вставали в одну compile-очередь.

## Применённые фиксы (все с бекапами)
1. SQL config (RECONFIGURE, без рестарта): max server memory 2048→4096; optimize for ad hoc workloads=1; blocked process threshold (s)=5.
2. XE-сессия CatchBlocked: EVENT blocked_process_report, ring_buffer 8192KB, STARTUP_STATE=ON — поимка блокировщика (сработала: 305+ отчётов, 83 за эпизод).
3. `Log_TblGameWorldInfo_InitializeCount` восстановлена из REF58_aion_log (1-в-1: обнуление счётчиков зоны по @world_id). Бекап: Aion_log-20261005-preinitcount.bak (13.7МБ).
4. `aion_ProcessWithdrawAccount` de-loopback ALTER: 3× `[RC-AIONAUTHDB].` убраны. Бекап: AionWorld-20261005-prealter.bak (15МБ compressed). Оригинал: C:\Temp\proc_..._ORIGINAL_20261005.sql. Smoke EXEC — мгновенно.
5. AionKillLog — одноразовая SYSTEM-задача, потушила зависший LogServer64 (паттерн kill session-0).
6. KILL 262 + KILL 274 (loopback-сессия) — разрыв self-deadlock вручную.

## Файлы/артефакты
- Десктоп VM: AION-START-LOG.bat (77C0A20341B0A492E52D3ABCCDFAF05C), AION-START-CACHE.bat (8034F5FB232B7F4BDCBD504F480C0A16), AION-START-NPC.bat (70E79BA225E15234BCA8BD4245290632), AION-START-MAIN.bat (A5B05A2DBD2B3474A269223850E3D437), AION-START-ALL.bat v5 (919C49DDD4D26E66EC86B23EED901BF2)
- Порядок подъёма: LOG → CACHE → NPC (10-15 мин) → MAIN
- Локально: STELGEN/tmp/aion-vm/ — dumpwin.py/dumpstr.py/crashwin.asm (дизасм), rbfull.ps1/rbwin2.ps1/rbc.ps1/bp2.ps1/bp3.ps1 (CatchBlocked парсеры), deploy-bp.ps1, alterproc2.ps1, audit-ls.ps1, hotspot.ps1
- VM C:\Temp: xesnap.sql, rb.xml, proc_..._ORIGINAL/ALTERED_20261005.sql, dumpwin.py, suppress.py

## Живой профиль после фикса (02:39)
- Стек полный, мир собран (16 conns); **ProcessWithdrawAccount отработал ×2 без падения — фикс подтверждён в проде**
- Ходовые процы: aion_SetServerInfo ×27 (16ms), UpdateLogfreedisk ×7, ap_SetConcurrentUserStatistics ×6, GetDeletedCharList ×2
- Горячие таблицы: house_addrinfo 12360r/6180w, abyss 2154/1074, server_info 649/307, TBL_GAME_SERVER_INFO 152/151, challenge_task ×3 (300r/0w), user_count 228w, user_data 136/13, user_item 96/11
- Waits: RESOURCE_SEMAPHORE 8689с и LCK_M_X 4116с = накопленный мусор ночных эпизодов (следить динамикой)
- Live-метрики: NPCSvr 14.63ГБ/23 потока/578k хендлов; Server64 10.12ГБ/49 потоков/827k хендлов (⚠️ следить за утечкой хендлов); CAPTCHA 178с CPU (рисует капчи); ICServer 1.67ГБ idle; CacheD VendorDb-пул 196МБ/1.4М объектов

## Аудит loopback/linked servers (после фикса)
- sys.servers: RC-AIONAUTHDB (localhost, loopback), WIN-0883A4UBOEC (legacy remote)
- Ссылок [RC-AIONAUTHDB] в проц-телах: **0** (была 1 — устранена)
- WIN-0883A4UBOEC в коде: 0
- OPENQUERY/OPENROWSET: 5 проц мёртвой шардовой архитектуры (PutUserDataFromServer, SyncUserDataFromServer, 3× MoveChar_*_ORS) — 0 вызовов в логах, не трогать

## Разбор «что жрёт систему» (логика нагрузки)
- Все компоненты = один SHARED C++-фреймворк NC (IoCompletion.cpp/Log.cpp/MemoryMan.h/SyncPacket.cpp) — поэтому баги клонированы
- Логика живёт в данных: NPCSvr 14.6 ГБ = состояние мира (карты/NPC/спавны/дроп), Server64 10.1 ГБ = игроки/инстансы, ScriptDLL64.dll 87.7 МБ = квесты/AI
- Сервер НЕ рендерит мир (клиент рендерит) — CPU 7-8% при 14 ГБ RAM: сервер «судит и рассылает дельты»
- CAPTCHAImageServer 178с CPU = рисует капчи; ICServer 1.67 ГБ idle-буферы
- Оптимизация-теория: следить за хендлами (827k), ночной рестарт пары под утечку Abyss.cpp (~600k блоков/сессия), Ghidra-тишина луперов/String-спама, опция выноса SQL на отдельную машину (архитектурно предусмотрено)

## Исправленные неверные гипотезы (важно!)
1. «Compile storm от нехватки памяти» → ЧАСТИЧНО НЕВЕРНО: MEMORYCLERK_SQLQUERYCOMPILE=0; корень = loopback self-deadlock (память лишь усугубляла)
2. «Процы специально убрали из БД» → НЕВЕРНО: NC-версионирование с суффиксами-датами (_20190919/_20191206 — поколение 7.x, БД кита = 5.8-поколение со старыми версиями без суффиксов)
3. «SQL-ошибки процедур (2812) вызывали краши» → НЕВЕРНО: ошибки быстрые и безвредные; краш = зависание IOThread (в т.ч. из-за ProcessWithdrawAccount)
4. «Твики памяти закроют краши» → НЕДОСТАТОЧНО: эпизоды 01:36 и 02:03 были после твиков; root — loopback
5. «LogServer невиноват» → УТОЧНЕНО: LogServer — жертва той же очереди (скрин окна: его CheckIOThreadDeadlock сыпал по IOThread 3/5/9)

## Гит
- 039aa92: InitializeCount restore + 2 pending REF58-проц
- c1924a6: de-loopback ALTER (root fix)
- 9ec98d1: audit linked/loopback
- этот коммит: сессионный отчёт + 4 лаунчер-батника

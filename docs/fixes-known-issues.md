# AION 7.7 PTS EU — известные баги и применённые фиксы

> Обновлено: 02.10.2026 (вечерний лог-скан VM 109 `AION-77-PTS-EU`, 192.168.0.125)
> См. также: `docs/fixes-community.md` (community-фиксы из RaGEZONE 1211744), `docs/roadmap.md`

## 1. Починено и проверено (по логам)

| # | Проблема (из логов) | Фикс | Статус |
|---|---|---|---|
| 1 | CacheD64 самоубивался при больших пакетах | `max text repl size = 65664` в SQL | ✅ работает |
| 2 | HY000/556 «недопустимый файл DSN ""» на всё | SQL Native Client 11.0 + полные пути FILEDSN в `C:\DSN\` | ✅ работает |
| 3 | LogServer64: `Invalid current ANSI code-page` → падение (ACP=1252) | Реестр `Nls\CodePage\ACP = 28591` (⚠️ вступает после ребута) | ⏳ применён 02.10, ждёт ребут |
| 4 | LogServer: `Invalid file path D:\_AION_log\batchlog\` | Создан `D:\_AION_log\batchlog` | ✅ |
| 5 | SQL: `Could not find stored procedure 'Log_TblGameServerInfo_UpdateServerstatus'` | Создана заглушка-процедура в `Aion_log` (worldId/status/serverType, update если таблица есть) | ✅ 02.10, проверено вызовом |
| 6 | SQL-алерты каждые 5 мин: `Index 'IX_delete_complete_date' on 'user_data' does not exist` (aion_GetDeletedCharList) | `CREATE INDEX IX_delete_complete_date ON user_data(delete_complete_date, delete_date) INCLUDE(char_id,user_id,account_id,account_name,guild_id,guild_rank)` в `_AionWorldNew114_rc` | ✅ 02.10, процедура протестирована |
| 7 | SQL-алерты: `Could not find server 'RC-AIONAUTHDB' in sys.servers` (aion_ProcessWithdrawAccount) | `sp_addlinkedserver RC-AIONAUTHDB → localhost, catalog=AionAccounts` + логин sa; + создана БД-заглушка `aionaccdeldb` (Korean_Wansung_CI_AS) с таблицей `del_account(seq, account_id, stat)` — только колонки, реально используемые процедурой | ✅ 02.10, процедура протестирована |
| 8 | Server64 лупер: `Can't connect to Captcha server at 127.0.0.1:-22206` | **Баг signed int16**: 43330 > 32767 → 43330−65536 = −22206. Исправлено в `MainServer\common.xml` (`captchaServerPort 43330→22206`) и `CAPTCHAImageServer\config.ini` (`SrcPort 43330→22206`); бэкапы `*.bak-captcha43330` | ✅ конфиг; CAPTCHA слушает 22206. Полный эффект — после рестарта Server64 |
| 9 | Server64/CacheD64 луперы: `Can't connect to Interchange server 2005/2305` | ICServer.exe не был запущен (нет в старт-батнике). Запущен; слушает 2005+2305, 8 ESTABLISHED-соединений | ✅ 02.10 |
| 10 | Мир не стартует: `CondSpwnTimeMgr, NPCSvr hasn't connected` | NPCSvr64 был закрыт в 22:00 (graceful, не краш). Перезапущен 23:17 | ✅ запущен, ~10 мин загрузка |
| 11 | AccountCacheServer молча умер после 22:07 | Перезапущен через задачу `AionAcc`; порт 2220 слушает, Server64 подключён (дубль PID убит) | ✅ 02.10 |
| 12 | Одноразовые задачи `AionAcc/AionAuth/AionGate/AionLog/AionCache/AionNPC/AionMain/AionMain2/AionRAD/AionStep/AionL2` (C:\Temp\*.bat) стартовали сегодня в 23:54–23:58 → дубли экземпляров; `main.bat`/`main2.bat` запускают Server64.exe **без RunAsDate** (сломает время) | Все задачи `DISABLE` (батники сохранены в C:\Temp) | ✅ 02.10 |
| 13 | SQL-контекст: установка SQL из SSH падала CryptographicException (DPAPI) | Только интерактивный запуск (schtasks /IT / GUI) | ✅ известно |

Скрипт фиксов №5–7: `scripts/sql/fix-db-2026-10-02.sql` (идемпотентный, безопасно перезапускать).

## 2. Известные баги сборки (НЕ починены, ждать/рокадми)

- **Отсутствующие stored-процедуры**: `aion_SetCharInfo_20160818` (сейв персонажа), `aion_GetItemCollection*`, `aion_LoadFame*` и др. → БД неполная (~85%). Структуру параметров `aion_SetCharInfo_20160818` дал klon22 (RaGEZONE #110) — можно воссоздать. БД 5.8 PTS ближе к 7.7 (RaGEZONE #201) — оттуда можно подтягивать.
- **Дома после lvl 9**, **коллекции после 78 без рецептов**, **нулевой дроп Heiron LF3 10–20 lvl** — требуют IDA-патчей Server64 (community-пост #201).
- **Питомцы / свитки призыва спрайтов** — кит «updated» от a7741288 (март 2023, пост #28) чинил это на его стороне; у нас бинарник 2020-06-01. Вариант — сверить версии кита.
- **Песочные часы (2 новые карты недоступны)** — нужен «integration server», даже a7741288 не поднял (пост #32, #39).
- **Манастоуны-стекинг**: у «другого» MainServer64 был фикс manastones (бесконечный стек статов), но сломан матчмейкер; у AKllX — фикс матчмейкера (JZ→JNZ на IsEventServer) без manastones. ENIGMA-обфусцированные сборки от third-party = риск бэкдора (пост #106–108). Наш путь: чистый Server64 + свои патчи (наш: date-check bypass от Angry Catster #180).
- **NPCSvr64 утечка памяти** (Abyss.cpp(1211): 597 228 leaked-блоков к шатдауну, RSS растёт до ~10 ГБ) — косметика; лечится рестартом по расписанию.
- **Server64 луперы безвредных сервисов**: `shop agent server 10100`, `petition server 2107`, `ChannelChat 10254` — соответствующие сервисы в ките отсутствуют (или часть Maxx/PA-инфраструктуры). На логин/мир не влияют (подтверждено постами #116–117: NPRelay/Log «not important to login»).
- **NPRelayServer** требует config.xml (в ките нет) — генерить вручную; некритично.
- **LogServer64 альтернатива**: патченые exe с изменённой кодовой страницей — `LogServer_RU_EU_Region.rar` (TheReverend #106, требует 28591/регион) и `LogServer64.rar` (#117, требует ACP=1251). Наш путь (ACP=28591 системно) работает без чужих exe; альтернативы в `docs/links.md`.
- **RunAsDate**: обязателен для Server64 (билд 04.06.2020). Альтернатива — свой date-check bypass (у нас в репо, пост #180) или «crack authorization time» (FliesQQ #119).

## 3. Операционный статус после скана 02.10.2026 ~23:26

```
AccountCacheServer.exe  ✅ (порт 2220, Server64 подключён)
L2Authd.exe             ✅ 2104/2108/2110
AuthGateD.exe           ✅ 2106
CacheD64.exe            ✅ 2006/2007/2009 (interchange-лупер исчез)
Server64.exe            ✅ (через RunAsDate, 8.6 ГБ; ждёт NPCSvr)
NPCSvr64.exe            🔄 загрузка ~10-15 мин (8-13 ГБ RSS) → мир поднимется
ICServer.exe            ✅ 2005/2305 (Interchange починен)
CAPTCHAImageServer.exe  ✅ 22206 (Server64 подцепит после рестарта)
LogServer64.exe         ❌ ждёт ребут (ACP=28591 применён)
NPRelay/Ranking/PA      ⏸ не запускались (некритично)
```

Остаточные луперы в Server64.err: Captcha (до рестарта Server64), shop agent 10100, petition 2107, ChannelChat 10254 — все некритичны.

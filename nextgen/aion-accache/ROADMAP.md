# ROADMAP aion-accache — AccountCacheServer (Go) — живой план

Обновлено: 07.10.2026. Факты/ресёрч: RESEARCH.md; артефакты: accountcache-ref/.
Общие правила эксплуатации: ../ROADMAP.md §4 (ssh-PowerShell готчи, «го» на прод-действия,
бекапы, секреты не в гит, TELEMETRY-SPEC).

## Прогресс

| Фаза | Статус | Что |
|---|---|---|
| R0 | ✅ | VM-разведка: PDB 92МБ+map+exe скачаны, тела 101 procs+21 таблица, клиент 2220=Server64 (07.10) |
| R0.5 | ✅ | Dispatch-таблица дизasmом: wire-фрейм [len-2][cmd][0xEB][~cmd], T1 cmds 0-39, T2 0-7, cmd 22 не занят (07.10) |
| R2 | ✅ | Каркас nextgen/aion-accache: proto+dispatch+cache+db-интерфейс+server+ship, тесты зелёные (07.10) |
| R1 | ✅ | **capture + wire-канон 10.10**: accmirror v1.1 (:2220→:2221) + копия ACS-2221; 2 логина; дизasm OnRead/GetCmd_ACQ → **len=total** (офф-бай-2 в v1.0/каркасе исправлен), маркеры EB/EC, 0 bad frames на всём capture (RESEARCH §11). Стенд ОСТАВЛЕН работать (дозахват) |
| R2.5 | ✅ | **Раскладки 10.10**: internal/payload (Version/FirstLoad/BM/Luna/CharLogin/Logout/Fatigue с UTF-16 stamp) на живых golden-кадрах, тесты зелёные; tools/accparse.py v2; frames.txt в capture-артефактах. Хвосты в R3: роли полей Fatigue/Trial (PDB I,H,I,I,I), push 13/15/20/21, T2-канал |
| R3 | ✅ код 22.10 | SQLStore-каркас (go-mssqldb, MaxOpenConns=10); R3 22.10: db.GetUserInfo (SELECT last_login_time + MAX(user_level) FROM global_user_data — read-proc в 101 procs REF58 НЕТ, проверено; DATEDIFF → int64), db.SetLoginUser/SetLogoutUser_20120703 (char_id, server_id, cur_server_id, login/logout_time datetime — тела REF58), config db.server_id (дефолт 1 кодом); хендлеры: hCharLogin = GetUserInfo (pre-login значение) → resp+пуши → SetLoginUser; hCharLogout → SetLogoutUser; тесты db (формы/аргументы) + e2e login/logout SQL. Живой прогон на AionAccountCacheD_rc = вместе с live A/B (деплой по «го») |
| R4 | ✅ offline + live 22.10 | **offline: replay 16/16 (100%)**; **live A/B (3 раунда на стенде :2220→:2221, тень :2222 read-only): VERDICT — все наблюдаемые cmds байт-в-байт** (таблица в README); единственный байт-дифф = ACP_20[5] lastLogin — структурный (Т10-М2, механизм freeze совпадает, значения = возраст entry) |
| R5 | ✅ 22.10 | **СВИТЧ ВЫПОЛНЕН**: прод = aion-accache :2220 (задача AionAccacheProd + AionKickAccache), ориг отведён (AionAccMirror/AionAcc = откат); Server64 переподключился сам (~8с), VERSION ✓; контрольный прод-логин чист (0 db.err); write-урок Т23 (trade затёрт→восстановлен→исправлен→подтверждён). Откат = `/end AionAccacheProd; /run AionAccMirror` |
| R6 | 🟡 22.10 старт | **Наблюдение 24ч идёт** (старт 22.10 ~08:25, чек 23.10 ~08:25): baseline — pid 2780 (RAM 10.3МБ, handles 157, dpm 0), логины 3/логауты 2, setlogin 3/3, setlogout 2/2, db.err=0, parse.err=0, мир 8/8; харвестер `D:\SAION\tmp\accstats.ps1`; op-конфиг ✅ (OP-7: сервис accache + acc/forkd expected_down, alerts=[]) |

## ⏭️ Следующий шаг (R6 наблюдение)

R5 свитч выполнен (22.10): прод = наш aion-accache :2220, ориг отведён, контрольный логин
чист (0 db.err; ll/lo = сек-в-сек штампов наших записей; trade=70 сохранён после Т23-фикса).
Дальше:
1. R6 наблюдение 24ч: логины/перелогины/ночной цикл, мир 8/8, ошибки .err/логов тени.
2. op-конфиг: сервис accache в config-vm.yaml (task AionAccacheProd, порт 2220; старую acc-строку
   пометить expected_down) — чтобы op-дашборд не врал.
3. Открытые ⏳: Т10-М2 точное правило RAM-freeze (дизasm), Т15 вариант SetLoginUser ориг
   (20120703 у нас; глобал-колонки ориг не переписывал — следить), Т19 (ACP_15 dword3 3/22),
   Т23-нотка (P6=70 в последнем логине — decompose семантика ⏳).
4. Мусор: probe-акки probetest1-14 (uid ~1007-1018) — удалить по «го».

## R1-детали (архив 10.10 — выполнено)

1. fork-proxy Go (паттерн forkauthd): front :2220 → ориг-порт... ⚠ прод-ACS слушает 2220 НАПРЯМУЮ;
   схема = копия ACS на :2221 (байтовая правка common.xml serverPort, паттерн captcha-mirror) +
   наш proxy на :2220 → forward :2221 + копия hex в лог. Ориг НЕ трогать, юзер ничего не замечает.
   Альтернатива (если Server64 капризен к смене порта) — pktmon/netsh trace на 2220.
2. Capture-сценарий: юзер логинится ×N (Server64 сам наливает чар-флоу: FIRST_LOAD/CHAR_LOGIN/SAVE_CUSTOM/LUNA).
3. Разбор: payload-раскладка per-cmd (арбитр = PDB-сигнатуры Decode*), ACP-номера (vtable SendIOBuffer),
   T2-канал (кто второй клиент: возможен отдельный сокет того же Server64).
4. Промпт: PROMPT.md.

## Критерии готовности R4

- Все наблюдаемые cmds парсятся без ошибок (нет unhandled/parse.err в логах)
- FIRST_LOAD/CHAR_LOGIN/LOGOUT ответы байт-в-байт == ориг (fork-дифф)
- БД-запись: account_data/hidden_fatigue апдейты == ориг (SQL-дифф по времени)

## Откаты

- R5-свитч: schtasks /change /tn AionAcc /tr обратно (C:\Temp\acc.bat) + /run
- Бекапы: ориг-каталог D:\AION_LIVE_SERVER\AccountCacheServer НЕ трогается

## 🧪 Журнал теорий (R4-хвосты, 21.10)

| # | Гипотеза | Статус |
|---|---|---|
| Т1 | LOAD_LUNA reply ts = limit_play_reset_time (account_data) | ✅ ПРОД-ФАКТ: acc=1010 колонка = 0x6AC5DFE0 == capture ts байт-в-байт. Реализовано |
| Т2 | ACP_21 (S2C 0x15) = push после LOAD_BM_PACK [acc][02][01][8×0] | ✅ capture ×2, байты идентичны. Реализовано (позиция async) |
| Т3 | S2C 0x06 = ответ на LOAD_TRIAL (НЕ UPDATE_BM_PACK!): payload содержит sell=917/trade=70 из trial_account_data acc=1010 (прод-факт) | ✅ Реализовано: [acc][X echo][reset][sell][CONST0][trade][decompose][gather][extract][pad]; dword3=CONST0 (роль TBD, единственный кадр) |
| Т4 | FIRST_LOAD при пустой строке account_data → point=1 | ⚠ эмпирика golden acc=7 (строки нет, ответ point=1); альтернатива — строка существовала и удалена после capture. Реализовано как capture-канон |
| Т5 | FIRST_LOAD u8@12/u16@17 = производные limit_play_accum_time | ⚠ гипотеза (все golden = 0; byte(limit_play_reset_time)=0xE0 исключён). Реализовано accum-based |
| Т6 | UPDATE_HIDDEN_FATIGUE point НЕ зеркалится ACS в account_data | ⚠ golden#2 point=1 несмотря на UPDATE point=0; prod сейчас point=0 (пост-capture активность). Fake replay не мутирует point |
| Т7 | ACP_13 (S2C 0x0d, 4Б) — значение-константа | ✅ РЕШЁН 22.10: payload = [ts u32] = 0x640831AE = 2023-03-08 06:56:46 UTC — КОНСТАНТА в обоих corpus (10.07 + 22.10, 2 чара, 4 логина). Триггер = VERSION (1:1:1: каждый VERSION C2S получает VERSION-ответ + ACP_13; соединения без VERSION — без пуша). Роль ts ⏳ (Т13) |
| Т8 | Logout-кластер S2C 0x0f+0x14+0x1c | ✅ РЕШЁН 22.10 (corpus ×2 чара × 3 логина): НЕ logout — кластер ПРИ CHAR_LOGIN (порядок 15→20→28, между ними ответы 16/27 — ACS многопоточен). Раскладки: 0x0F = [charRef][0x000A07EA][3]; 0x14 = 56Б [charRef][16×0][lastLogin][12×0][1][8×0][flag][maxLevel]; 0x1C = [accId][charId=charRef&0xFFFF][u16 0]. Реализовано (16/16) |
| Т10 | ACP_20[5] = global_user_data.last_login_time (RAM-момент ориг) | ✅ источник: SQL AionAccountCacheD_rc.global_user_data (колонки char_id/account_id/create_date/last_login_time/last_logout_time/delete_date/delete_completed_date/user_level/server_id/cur_server_id/global_char_id/uid/class_type/race_type/is_special_server/is_jumping_character): 0x6AC5E7B9 = 10.07 06:33:29Z (логин#1 после старта ACS), 0x6AD988A7 = 06:53:11Z = секунда-в-секунду логин#1 (у логинов #2/#3) — RAM-race ориг. Точное правило обновления RAM ⏳ |
| Т11 | ACP_20[13] = maxLevel аккаунта (не per-char level) | ✅ 0x50=80 у ОБОИХ чаров, при user_level=1 у чара 1003 в БД — паттерн maxLevAmongNormalServers из proc user_board_bm (SELECT MAX(user_level) FROM global_user_data WHERE account_id=@a AND is_special_server=0 AND delete_date=0) |
| Т12 | ACP_20[12] (flag): наблюдения 0(10.07)/1/1/0(22.10) | ⏳ гипотезы: «уровень изменился с прошлого refresh» (CHAR_LEVEL_CHANGED ×5 в 22.10) / «pending luna reward». Арбитр = следующий capture с level-up |
| Т13 | ACP_13 ts 0x640831AE = 2023-03-08 06:56:46Z | ⏳ константа (вероятно sentinel/дата сборки 7.7); на паритет не влияет |
| Т14 | ACP_15[4] = 0x000A07EA (657386) | ⏳ константа во всех кадрах; не acc/char/ts. На паритет не влияет |
| Т15 | Какой вариант SetLoginUser зовёт ориг: 20120703 (4 арг) vs 20121120/20121206 (+global_char_id/class_type/race_type; 20121206 ещё и delete_date=0) | ⏳ арбитр = live A/B SQL-дифф: наши 20120703-вызовы НЕ трогают global_char_id/class/race — если у ориг после логина они меняются, variant = 20121206 (тогда источники аргументов = RAM-модель ACS) |
| Т16 | Таймзона штампа Server64 в C2S CHAR_LOGIN/LOGOUT («2026-10-22T06:53:11.620») | ⏳ парсим как UTC (time.Parse без зоны); арбитр = live A/B SQL-дифф last_login_time vs штамп |
| Т17 | SetLoginUser ставит last_logout_time = @login_time (тело 20120703/20121206!) — инвариант ll==lo после логина | ✅ из тела proc + LIVE-БД (22.10): ll = секунда-в-секунду штамп логина, lo = штамп логаута (после выхода ll≠lo — как и должно) |
| Т18 | FIRST_LOAD point = RAM-sticky/clamp(1) | ✅ live 22.10: DB point=0 → ориг отвечает 1 (3 раунда); реализовано (clamp+RAM) |
| Т19 | ACP_15[2] (dword3): 3 (существующие чары) vs 22 (новосозданный чар 1005) | ⏳ роль неизвестна (не acc/char/ts/level); арбитр = следующий capture с созданием чара / дизasm |
| Т20 | UPDATE_LUNA-префикс = [acc&0xFF][00][01][00] | ✅ live (1010→F2, 1023→FF); «константа F2» из capture-20261007 = артефакт одного аккаунта |
| Т21 | C2S CHAR_CREATED (0x0F) = сессионная раскладка [1][charRef][stamp][tail] | ✅ live (новый акк 1023/чар 1005); proc SetCreateUser-аргументы = другой канал ⏳; старый парс дал бы мусорную запись — read_only спас прод |
| Т22 | go-mssqldb НЕ поддерживает ODBC {call ...} | ✅ live-инцидент (Incorrect syntax near '{'); эталон синтаксиса = aion-authd (EXEC dbo.proc @pN) |
| Т9 | CHAR_LOGIN reply приходит РАНЬШЕ replies на предыдущий батч (UPDATE_LUNA/LOAD_LUNA после CHAR_LOGIN) | ✅ ACS многопоточен (numberOfDBThreads=10) — порядок replies недетерминирован; выравнивание по cmd |

## R4 итог 21.10

**12/16 golden S2C (75%) байт-в-байт.** Реализованы: ts-фикс LOAD_LUNA (Т1), ACP_21 push (Т2), trial-ответ S2C 0x06 (Т3), point=1 дефолт (Т4), accum-based X/Y (Т5), stateful replayExec (updatetime мутится Exec'ом, trial-row 917/70). Остаток = 4 кадра с непроверенными данными (Т7/Т8) — эмитить до живого capture опасно для Server64.


## R4 ФИНАЛ (22.10): capture-прогон выполнен, новый corpus в гите

**Новый corpus:** accountcache-ref/capture-20261022/ (254КБ, 60 C> + 44 O> кадров, 31 уникальный cmd, 2 чара, 2+ логина + logout-кластер ×3)

| Кадр | Было (07.10 corpus) | Стало (22.10 corpus) | Статус |
|---|---|---|---|
| VERSION/FIRST_LOAD/BM_PACK/CHAR_LOGIN/LUNA | ✅ 12/16 | ✅ те же | байт-в-байт |
| ACP_13 (0x0D) | ❌ ts=0x640831AE не в БД | **×4 в новом capture** | ⏳ раскладка |
| ACP_15 (0x0F) | ❌ 0x000A07EA без источника | **×3, 2 чара** | ⏳ раскладка |
| ACP_20 (0x14) | ❌ ts 0x6AC7E7B9 | **×3, level=80** | ⏳ раскладка |
| CONFIRM_LUNA_REWARD (0x1C) | ❌ | **×3, 2 чара** | ⏳ раскладка |

*(история, утро 22.10)* **R4 replay был 12/16 (75%)** — план «распарсить 4 XFAIL из нового
corpus (два чара = сравнение полей) → 16/16» ВЫПОЛНЕН В ЭТОМ ЖЕ ДНЕ (см. ниже).

**Capture живёт бесконечно:** accmirror пишет каждый кадр; юзер ходит на разных чарах.
Следующий repl забор = по требованию (aiongetb64 accmirror.log).

## R4 OFFLINE ЗАКРЫТ (22.10, вечер): 16/16 = 100% ✅

**Раскладки 4 XFAIL выведены из capture-20261022 (2 чара × 3 логина, сравнение полей)**
и закреплены golden-тестами (TestR4PushGoldens — оба чара, обе сессии):

| Пуш | Триггер | Payload |
|---|---|---|
| ACP_13 (0x0D) | VERSION (1:1:1 оба corpus) | `[ts]` = 0x640831AE константа |
| ACP_15 (0x0F) | CHAR_LOGIN (кластер 15→20→28) | `[charRef][0x000A07EA][3]` |
| ACP_20 (0x14) | CHAR_LOGIN | 56Б `[charRef][16×0][lastLogin][12×0][1][8×0][flag][maxLevel]` |
| ACP_28 (0x1C) | CHAR_LOGIN | `[accId][charId=charRef&0xFFFF][u16 0]` |

- Эмиттеры: payload.go (BuildCharCustomSavedPush/BuildCharCreatedPush/BuildCharInfoRefreshPush/BuildConfirmLunaRewardPush) + хендлеры hVersion/hCharLogin (server.go) + cache.UserInfo (RAM-срез global_user_data: lastLogin/maxLevel; SQLStore R3 — SELECT из global_user_data).
- Тесты: payload + server — все зелёные, vet чисто; **TestR4ReplayCapture: our=16 frames, matched 16/16 (100%)**.
- Остаток R4 (live): fork A/B с живым оригом (после R3 SQLStore) + T2-канал (опц.).

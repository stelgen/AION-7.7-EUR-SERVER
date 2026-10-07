# 🛰 AUTHD-ROADMAP — L2Authd → свой authd (Go)

> Верхнеуровневый план 07.10.2026. Полный ресёрч сурсов: [AUTHD-RESEARCH.md](RESEARCH.md).
> Промпт для разработки: [PROMPT-AUTHD.md](PROMPT.md).
>
> **СТАТУС 07.10 ночь: R1-R4 ГОТОВЫ + R5 fork-стенд ЖИВОЙ НА ПРОДЕ** (гейт 2106 → forkauthd 2116 →
> ориг 2110 + копия в shadow aion-authd 2117; юзер залогинился и в игре — живой путь через ориг).
> Каноны сняты fork'ом (type=3/4/7/fail — см. aion-authd/docs/authd-wire-20261007.md §2.5);
> **PA обязателен** (без него SYSTEM_ERROR 20). Прод-топология ОСТАВЛЕНА КАК ЕСТЬ (fork-shadow постоянно,
> по решению юзера) = R6-свитч свёл­ся к: наш authd в shadow до R0 (procs AionAccounts, роль 2104) →
> затем переключение живого пути на наш (fork: orig→shadow режим).
> **R0-PROCS ✅ 09.10**: инвентарь AionAccounts снят (31 proc + тела → `../authd-ref/procs-aionaccounts-77.rpt`):
> логин = `ap_GPwdWithFlag` (автосоздание ВНУТРИ SQL: нет акка + ASCII `[a-zA-Z0-9]+` → `ap_AutoReg`, flag=3, pwd=0x00×16);
> `ap_GStat` = uid/payStat/loginFlag/warnFlag/blockFlag/blockFlag2/subFlag/lastworld/block_end_date/forbidden_servers + UPDATE last_login (maddaemon fix 08);
> serverlist = `ap_GetServers` (таблица `server`: id/name/ip/inner_ip/ageLimit/pk_flag/kind/port/region);
> блок = `ap_GetRestriction` (block_msg: reason,msg по uid); `ap_SLog`/`ap_GUserTime` канон C1; `ap_SUserData` = стаб (SELECT 9).
> → SQLStore (mssql-стор) переводится на ВЫЗОВ ЭТИХ procs по имени вместо inline-C1 SQL.
> **R0-2104 ✅ 09.10 (протокол-first, решение юзера)**: дизasm НЕ нужен — L2Authd сам пишет полный wire-дамп
> `etc/log/*.packet`; словарь 2104 снят живым логином юзера: heartbeat 60с (type 2 A→W / 5 W→A),
> релей логина type 0 (uid+аккаунт в мир + ack), uid-эхо-события 13-44, type 35 = char_id+lev.
> Словарь+корпус: docs/authd-2104-recon-20261009.md + ../authd-ref/logs-2104/. Мир сам реконнектится.
> MVP 2104 = heartbeat + type0-релей + квитанции.
> **MVP-2104 РЕАЛИЗОВАН 09.10**: `internal/world` (Go) — листенер gsPort (дефолт 0=выкл),
> greeting/heartbeat/релей type-0/статус мира (type5 = users/limit), хук: type=3 на 2110 → релей в мир;
> golden-тесты из живого корпуса (greeting 110Б/ping/relay 107Б/X=total-1). Квитанции gsAcks=false (T2).
> **MSSQL-СТОР 09.10**: SQLStore дефолты = реальные procs (ap_GPwdWithFlag+ap_GStat/ap_AutoReg/
> ap_GetRestriction); created = флаг 3 + нулевой pwd; не-ASCII = NULL → ErrNotFound.
> **АРБИТРАЖ FORK-ЛОГА 09.10**: FIFO пар по ключу ОТВЕТА (sid,type) — вердикт только при полной
> паре, N-ONLY-гонки исключены (race-тест); SINGLE/ARBITRATION-DROP (>60с).
> **ДЕПЛОЙ НА VM 09.10 (окно без юзера на 2106, через op)**: новые aion-authd.exe + forkauthd.exe
> (scp → .new → stop forkd/authdn → copy → start) — forkd/authdn RUNNING, гейт реконнектнулся.
> Shadow: mem-store + gsPort=0 (мир-канал выкл до R6). ОСТАТОК ДО R6: fork-сверка 2104/квитанций/tail
> на живых логинах юзера (новый арбитраж даёт чистые пары) + mssql-стор на shadow (переключение
> driver: mssql) — потом «го» на свитч.

## 🧪 Журнал теорий (2104)

| Дата | Гипотеза | Проверка | Статус |
|---|---|---|---|
| 09.10 | wire 2104 = тот же семейственный фрейминг `[u16][type]`, X = total−1 (packetSizeType=2) | C1 WorldSrvSocket.cpp (сорцы, НЕ дизasm): `m_packetSize = buf[0]+buf[1]<<8+1−2` | ✅ канон |
| 09.10 | greeting мира = C1 OnCreate `Send("cdd",3,build,1)` | Server64-лог: «authVersion:2017012601, protocolVersion:1» — 1-в-1 | ✅ канон |
| 09.10 | type5 W→A = status [users u16][limit u16] | корпус: `0000f401` → `0100f401` (users 0→1 после логина юзера); 500 = лимит | 🟡 частично (семантика полей — на fork-диффе) |
| 09.10 | tail type-0 релея [49:107] (554da0b8/ff×24/50c2366b×2) = времена/expire | корпус статичен за 2 логина | ⏳ fork-дифф 2104 |
| 09.10 | квитанции A→W 19/16/44/14/13/31 = реакции на события мира | соответствие по корпусу в одном флоу | ⏳ fork-дифф 2104 |
> Сессия-док: [docs/session-20261007-authd-mvp.md](docs/session-20261007-authd-mvp.md).
> Метод-референс: треки aion-logd → aion-captcha → aion-gate (метод отработан 3 раза).
> Приложение-цель: `L2Authd.exe` (1,198,592 Б) — **2104** (serverPort), **2110** (serverExPort → AuthGateD),
> 2108 (GM interactive), 10062 (QMAS); конфиг `etc\config.txt`; БД `AionAccounts` через `L2Conn.dsn`;
> клиенты: AuthGateD (2110), AccountCache (2220), PA (10057, DISABLE навсегда).
> Логи authd: `winlog/packet/dual` (уже тейлерятся aion-op).

## 1. Полный реестр сурсов (что есть и что из каждого берём)

| # | Сурс | Где | Что берём |
|---|---|---|---|
| S1 | **Wire 2110 live-эталон** | `../aion-gate/README.md` §Канонические факты + `D:\SAION\aion-gate\gate-prod.log` (RAW A>G/G>A hex) | Фреймы `[00][sid][IP-be]`/`[01][sid]`/`[02][sid][len][blob 191Б asm-форма]` → назад `[03][sid]`(V=0xc621)/`[02][id][len][type][payload]`, type=3/4/7. Это УЖЕ 100% образец трафика — главный актив |
| S2 | **L2Auth (C1, полный декомпил)** | `../authd-ref/L2Auth-chaospaladin/` + `l2-c1-mastertoma/L2Auth/` | Архитектура authd: CAuthServer/CAuthSocket, WorldSrvServer (gs-wire), CAccount (ODBC-процедуры, block_msg, payStat), OneTimeLogOut/AutokickAccount семантика, crypt-модули |
| S3 | **Схема БД authd** | `l2-c1-mastertoma/DBScript/` (ReleaseAuthDBSchema.sql: `ap_GPwd/ap_GStat/ap_GUserTime/ap_SLog/ap_SUserTime`, lin2comm 44 procs) | Формат DB-слоя; сверить с нашими procs в AionAccounts (sp_helptext, схема/TBL в aiongm_ur) |
| S4 | **L2Authd.pdb** | Локально (малый PDB, manifest-pdb-big.md: «локально уже есть малые PDB (L2Authd/AuthGateD/...)») + бинарь 1,198,592 на VM | Метод LOGD-REWRITE-ANALYSIS: `pdbpub.py` → publics → objdump дизasm ключевых функций (проверка гипотез по wire 2104/10062/2220) |
| S5 | **Эталон семантики** | `reference/Mobius_AionEmu` (LoginServer: AccountController, AionAuthResponse) | Коды фейлов 0-22 (уже в aion-gate `authfail.go`), kick/ALREADY_LOGGED_IN, онлайн-флаг TTL |
| S6 | **Классика L2AuthD (архив)** | `authd-ref/l2auth-legacy-2008/`, `L2AuthHost-csharp/`, Ruk33/l2auth (локально) | Вторичные проекции: IP-фильтры, режимы хостинга, клиентский wire C4 |
| S7 | **Инфраструктурные заготовки** | `../aion-logd/internal/ship`, aion-gate (config/ship/логгеры) | ship-телеметрия по TELEMETRY-SPEC копируется как есть; стиль конфигов/деплоя D:\SAION |

## 2. Фазы (верхнеуровнево; каждая = коммит+пуш+дельта в память)

| Фаза | Что | Результат | Оценка |
|---|---|---|---|
| **R0 Разведка** (read-only) | Инвентарь VM: `D:\AION_LIVE_SERVER\L2Authd\etc\config.txt` (полный), L2Conn.dsn, логи winlog/packet/dual; кто держит 2104/2108/10062/2220; PDB L2Authd → `pdbpub.py` publics; сверка procs AionAccounts vs S3 — ✅ 09.10 (`procs-aionaccounts-77.rpt`); осталась роль 2104 | Карта зависимостей + перечень DB-вызовов оригинала | 1 день |
| **R1 Протокол-фундамент** | Разбор wire 2110 по gate-prod.log (S1) — фрейминг/типы/HEX-эталоны; дизasm S4 для 2104/2220/QMAS если встречаются | Док `docs/authd-wire-20261007.md` + golden-фреймы в тестах | 1-2 дня |
| **R2 Каркас** `nextgen/aion-authd/` | Go: main/config/ship (копия из logd), ODBC-слой (L2Conn.dsn-строка), state-машина сессий, листенер 2110 | Эхо-сервер 2110: принимает [00]/[01]/[02], отвечает по логам (replay-режим) | 2-3 дня |
| **R3 Логика authd** | Порт логики S2: логин-проц (blob 191Б → user/pwd/otp) → автосоздание (пароль НЕ проверяется — live-факт) → block_msg → online-флаг (TTL 2-6 мин!) → [03] V=0xc621, [02][type]3/4/7 (serverlist/74Б, 42b, 26b) | Паритет ответов с ориг по golden-фреймам | 3-5 дней |
| **R4 DB-слой** | Реализация вызовов РЕАЛЬНЫХ procs AionAccounts (тела сняты R0-09.10: `ap_GPwdWithFlag`→`ap_AutoReg`, `ap_GStat`, `ap_GetServers`, `ap_GetRestriction`, `ap_SLog`): uid+payStat, SLog-запись, OneTimeLogOut | Тесты с локальным MSSQL-контуром/фейком | 2-3 дня |
| **R5 fork-proxy A/B** | **fork-proxy на 2110**: наш гейт → fork-proxy → ориг L2Authd (живой путь) + КОПИЯ фреймов в наш authd; diff-лог O-vs-N на каждый фрейм (как в gate-треке: C>/O>/N>) | 100% паритет диффа; НЕ слать ориг мусор (L2Authd хрупкий — умирает от кривых пакетов) | 2-3 дня |
| **R6 Свитч прод** | По «го»: 2110 → наш (задача AionAuth ретаргет), откат = возврат L2Authd (schtasks + restart-auth.ps1 ритуал); наблюдение 24ч (логины/relogin/онлайн-флаг/night-циклы) | Свитчнут + session-док + ROADMAP/README дельта | 1 день |

**Суммарно:** ~2-4 недели. **Шанс успеха ~85%** (было ~70% в ROADMAP до ресёрча; подняли сурсы S1-S4).
Главный риск — скрытые ветки authd (QMAS/OTP/AccountCache-клиент 2220) → закрываются S4-дизasm'ом,
usage у нас минимальный (PA вырублен, OTP off, QMAS мёртв?). Второй риск — точные процы AionAccounts
(имена могут отличаться от C1) → R0 sp_helptext-инвентарь.

## 3. Железные правила (наследие трека B)

- Прод трогать только по «го»; fork-фаза вообще невидима для прод-игрока (гейт продолжает ходить в ориг).
- TELEMETRY-SPEC обязателен (ship из aion-logd).
- fork-proxy скромничает: копирует, НЕ модифицирует; при смерти L2Authd — рестарт-ритуал `C:\Temp\restart-auth.ps1`.
- Probe-логины ЛОЧАТ акки на 2-6 мин (онлайн-флаг) — тестовые креды держать пулом, не долбить.
- Секреты (connStr/пароли) — только в конфиге на VM.

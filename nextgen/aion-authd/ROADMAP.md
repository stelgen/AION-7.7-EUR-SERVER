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
| **R0 Разведка** (read-only) | Инвентарь VM: `D:\AION_LIVE_SERVER\L2Authd\etc\config.txt` (полный), L2Conn.dsn, логи winlog/packet/dual; кто держит 2104/2108/10062/2220; PDB L2Authd → `pdbpub.py` publics; сверка procs AionAccounts vs S3 | Карта зависимостей + перечень DB-вызовов оригинала | 1 день |
| **R1 Протокол-фундамент** | Разбор wire 2110 по gate-prod.log (S1) — фрейминг/типы/HEX-эталоны; дизasm S4 для 2104/2220/QMAS если встречаются | Док `docs/authd-wire-20261007.md` + golden-фреймы в тестах | 1-2 дня |
| **R2 Каркас** `nextgen/aion-authd/` | Go: main/config/ship (копия из logd), ODBC-слой (L2Conn.dsn-строка), state-машина сессий, листенер 2110 | Эхо-сервер 2110: принимает [00]/[01]/[02], отвечает по логам (replay-режим) | 2-3 дня |
| **R3 Логика authd** | Порт логики S2: логин-проц (blob 191Б → user/pwd/otp) → автосоздание (пароль НЕ проверяется — live-факт) → block_msg → online-флаг (TTL 2-6 мин!) → [03] V=0xc621, [02][type]3/4/7 (serverlist/74Б, 42b, 26b) | Паритет ответов с ориг по golden-фреймам | 3-5 дней |
| **R4 DB-слой** | Реализация procs-вызовов (или прямых SQL) поверх AionAccounts: uid+payStat, SLog-запись, OneTimeLogOut | Тесты с локальным MSSQL-контуром/фейком | 2-3 дня |
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

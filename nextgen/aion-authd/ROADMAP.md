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
> **FORK-ВЕРИФИКАЦИЯ НА ЖИВОМ ЛОГИНЕ ЮЗЕРА 09:58 ✅** (sid=4, stelgen, юзер в игре): greeting SAME;
> type=4 server-info **SAME байт-в-байт**; type=3 DIFF только в динамике: **uid** (ориг 1010 из БД vs
> наш mem-seed 1) и token/unk1 (Rnd) — структура 52Б паритетна; type=7 DIFF: pk1 у ориг ДИНАМИЧ
> (09:33→2, 09:58→4; наш статич 1) + наш pk2 = наш uid. ВЫВОД: R6-стенд требует mssql-стор на shadow
> (реальный uid) — код готов, включить driver: mssql. pk1-динамика = T2-канон.
>
> **🧪 ТЕХ-ДОЛГ (юзер, 09.10): клиент RU 7.7 всегда видит 0 персонажей на сервере.**
> Java-эталон EU-линии (aion-germany AL-Login SM_SERVER_LIST op=0x04): счётчик чаров акка = В ХВОСТЕ
> пакета: [servers.size][lastServer][per-server: id/ip/port(D)/age/pvp/cur(H)/max(H)/status/bits(D)/brackets]
> + [maxId+1 (H)][01][49 нулей][writeC charCount per server]; источник = GS-репорты (CM_GS_CHARACTER).
> Наш/ориг-2110 type=4 = 26Б — зоны счётчиков НЕТ (клиент читает 0). «74Б serverlist» из логов 06.10
> = похоже полный вариант с зоной счётчиков. ПЛАН: (1) снять клиентский G>C из gate-prod.log — есть ли
> 74Б-вариант и где зона; (2) реализовать сборку полного serverlist с counts из БД (user_data по аккаунту /
> ap_GetAccountGameSlot) в нашем authd/gate; (3) сверка живым клиентом. Не блокер R6 (косметика клиента).

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

## 📡 Дельта 09.10 вечер: юзер-сессия «зашёл-поиграл-вышел» — новые каноны

- **MSSQL-стор на тени ВЕРИФИЦИРОВАН ЖИВЬЁМ** (config-shadow: driver mssql, connStr
  `sqlserver://localhost?database=AionAccounts` = L2Conn.dsn Trusted_Connection, SSPI из
  SYSTEM-задачи прошёл): probe probeacc2 → type=3 payload с **РЕАЛЬНЫМ uid=1020** (ap_GPwdWithFlag→
  ap_AutoReg), канон 52Б. На юзер-логине type=3 больше не DIFF по uid.
- **type=7 pk1 = ТОЧНОЕ ЭХО World-ack** (W→A type=0 [uid][N] → A→Gate type=7 pk1=N):
  три логина 2→2 / 4→4 / 8→8. Реализовано: наш authd на CM_PLAY (op=0x02) при включённом
  мире шлёт relay type=0 (A→W) и ЖДЁТ ack → OnPlayAck → SendPlayOK (type=7 pk1=ack);
  при выключенном мире — старый путь pk1=1 (fallback). A→W type=0 = релей PLAY,
  НЕ логина (login мир не касается — type=3/4 идут до мира).
- **LOGOUT-флоу 2104 (юзер вышел корректно)**: W→A type=40 (uid) → W→A type=3
  (uid + 76000000 + char_id + lev) → heartbeat type=5 users 1→0 (`0100f401`→`0000f401` —
  theory users/limit → ✅ канон). C1-семантика: packet03_userQuits / packet04_userDropped.
- **forkauthd: переподключение тени** (ensureShadow, кулдаун 5с): рестарт shadow-authd больше
  не отключает копию до конца gate-сессии (баг 10:48 — fork держал мёртвый сокет, N-стороны
  не было, ARBITRATION-DROP отработал штатно).
- Деплой: aion-authd.exe (`3c0458c8`) + forkauthd.exe (`3bd24ff3`) на VM, gate d5b9c708
  (serverListCharCount=0 — паритет); forkd/authdn/gate RUNNING, мир 8/8.
- ОСТАТОК: fork type=3/7 SAME на СЛЕДУЮЩЕМ юзер-логине (тень уже на mssql + pk1-эхо) → R6 «го».

## ✅ ВЕРДИКТ ПАРИТА (09.10 поздний, юзер-логин 10:56, сессия sid=3, тень на mssql)

- **type=3: ПОЛНЫЙ ПАРИТ** — uid `f2030000`(1010) = у ориг И у нас (mssql-стор!); DIFF остался
  только в token/unk1 — рандом per-session (у каждого свой — КЛИЕНТ берёт из нашего пакета,
  совпадение с ориг не требуется) → семантический SAME.
- type=4 (SM_SERVER_LIST): SAME байт-в-байт ✓.
- type=7: pk1 у тени = 1 (fallback) — ожидаемо: у тени gsPort=0 (мира на тени нет);
  pk1-эхо-цепочка заработает при R6-свитче (мир перейдёт на наш 2104). Канон pk1 = эхо ack.
- **ВЫХОД ИЗ ИГРЫ (висяк клиента ~10 мин)** — сервер-стор ЧИСТАЯ: мир отпускает акк за 5с
  (10:58:38 type=40 → 10:58:43 type=3 quit → type=5 users 1→0), клиент молча рвёт LS-сокет
  (CM_LOGOUT op=03 НЕ шлёт — наш гейт в стороне). Висяк/краш при выходе = клиентская болячка 7.7
  (прецедент minion-crash) → ТЕХ-ДОЛГ ниже среднего: гипотезы — клиентский wait на не-SM-ответ
  / GameGuard / клиентский утечка-деструктор; триггер ревизии — жалобы юзера.

## 📋 ТЕХ-ДОЛГ (актуальный лист, приоритеты)

1. **[ниже среднего] Выход из игры: клиент висит ~10 мин / иногда умирает в процессах**
   — сервер-стор чистая (факты 10:58); клиент-локально. Ревизия по триггеру от юзера.
2. **[ниже среднего] charcount на сервер-селекте** — см. docs/techdebt-charcount-20261009.md (2 эксперимента,
   гипотезы a/b/c; НЕ наша регрессия — у ориг тоже пусто).
3. **[T2] pk1-эхо-верификация на живом логине** — после R6 (мир на нашем 2104): type=7 SAME.
4. **[T2] 2104-квитанции/tail релея** — fork-дифф возможен только с миром на нашем канале (R6).

## 🚀 R6 СВИТЧ ВЫПОЛНЕН (09.10, «го» юзера) — ЖИВОЙ ПУТЬ = НАШ AUTHD

- Топология: `клиент → aion-gate (2106, authPort=2110) → НАШ aion-authd (:2110 + мир-канал :2104)`;
  Server64 реконнектнулся к нашему 2104 (ESTABLISHED 3580→6740 ✓ — greeting/heartbeat приняли);
  гейт ↔ наш 2110 ESTABLISHED (5480→6740). Мир 8/8 не пострадал, PA жив, тень 2117 жива.
- Прод-конфиг: `D:\SAION\aion-authd\config-prod-authd.yaml` (serverPort 2110, gsPort 2104,
  db mssql trusted); задача **AionAuthdProd** (SYSTEM); лог `authd-prod.log`.
- forkauthd 2116 ОСТАВЛЕН запущенным (без клиентов) — часть отката.
- **ОТКАТ (одна цепочка)**: `D:\SAION\aion-authd\authd-rollback.cmd` (taskkill наш + старт ориг
  AionAuthOnly) → `C:\Temp\rollback-gate.ps1` (authPort 2110→2116) → `aionact restart gate restart`.
  Бекап гейт-конфига: config-prod.yaml.bak-0910-preR6.
- Гонка старта гейта повторилась (kick → пауза → первый /run иногда не поднимает; второй /run поднял
  PID 5480) — известная, учесть в op (задача «пауза 5с» маловата).
- Наблюдение 24ч: юзер логинится туда-сюда; смотреть authd-prod.log (login OK uid, world relay/ack,
  play-ok pk1=эхо), мир 8/8, relogin-тишину, онлайн-TTL свипы.

## ✅ 2104-КАНАЛ ОЖИВ (12:17, фикс X=body+2 задеплоен)

- RAW-лог: наш greeting `0c0003792b39780100000000` → **через 60с ответ мира** `0700050000f401`
  (pong type=5, users=0, limit=500) — Server64 ПРИНИМАЕТ наш формат: формула X=body+2
  (самоинклюзивный len, та же что 2110; C1 packetSizeType=3) подтверждена живьём с обеих сторон.
- Root-cause каскада: X=body+1 → Server64-парсер молчал → нет ack → нет type=7 → клиент умирал
  на сервер-селекте (и в R6-логине 11:23).
- ГОТОВО К ЛОГИНУ: ожидание = login OK → serverlist → world relay (ip-дворд) → **world ack** →
  type=7 pk1=эхо → вход в мир.

## 🔧 12:22: RAW-loop slice fix (деплой 12:24)

- Мир ответил ack на relay: RAW `[0b00][00][f2030000 0a000000]` = uid 1010, **pk1=10** — но
  raw-loop парсил payload со сдвигом +2 (ack обрезался до 4Б → OnPlayAck не сработал → type=7
  не ушёл → висяк на выборе сервера 12:19). Формула подтверждена: total = X (2 + body),
  payload = buf[3:x]. Фикс + тест зелёные (`7f9bce4`), деплой 12:24 (authd PID 3320).
- Цепочка play теперь полная: CM_PLAY → relay type=0 → ack (pk1) → type=7 pk1=эхо → мир.

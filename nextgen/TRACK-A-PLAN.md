# 🎛 Трек A — `aion-op`: план работ + карта кнопок

> Детализация раздела 3 из [PLAN.md](PLAN.md). Дата: 05.10.2026.
> **Принцип фазы: ничего не меняем на проде** — Phase 0/0.5 полностью read-only, кнопки физически заблокированы.
> Канон запуска/рестартов взят из эксплуатации 02–05.10 (см. docs/app-architecture.md, session-20261005).

---

## 1. Best-practice правила оператора (фундамент дизайна)

1. **State machine на каждый сервис**: `STOPPED → STARTING → RUNNING → DEGRADED → STOPPING → FATAL`. Кнопка = переход состояния, а не «shell-команда».
2. **Кнопки идемпотентны**: `start` на живом = no-op с сообщением; `stop` на мёртвом = no-op. Никогда не создаём дублей (урок: Server64 задваивался через AionMain+AionMain2).
3. **Группы + зависимости + приоритеты**: старт по порядку канона, стоп в обратном. Пара NPC+MAIN — ЕДИНАЯ единица управления (смерть одного ⇒ graceful смерть второго, рестартить только парой).
4. **Health = 3 уровня**: (a) процесс есть, (b) порт слушает, (c) лог-ливерность/маркер («*new world server connection», «NPC Server Started», 16 conns на 2002).
5. **Lock-и**: одно действие на сервис; глобальный lock на пару; рестарт запрещён в окне загрузки NPCSvr (10–15 мин, детект: `conns2002<16` или private < 14 ГБ).
6. **Опасные действия — с typed-confirm** (рестарт мира = 10–15 мин простоя). Все действия пишутся в audit-journal (кто/когда/что/результат).
7. **Режимы**: `observe` (дефолт, Phase 0) vs `operate` (кнопки разблокированы, Phase 1+). Переключатель осознанный, с логом.
8. **Никаких секретов в репо/конфиге** — SSH-ключи и SQL-логин только в локальном YAML вне гита.
9. **Windows-специфика (из крови)**: `taskkill` из SSH/SYSTEM не убивает юзер-сессию Console-1; L2Authd/AuthGateD живут ТОЛЬКО в интерактивной сессии; значит управляющие действия = `schtasks /run` по /IT-задачам (канон) или Phase-1 агент в юзер-сессии.
10. **Консоль должна быть скучной**: всё зелёное/тихое; красное появляется только когда реально случилось (SCADA-HMI правило).

---

## 2. Группы сервисов и карта кнопок

Порты-здоровье: SQL 1433 · AccountCache 2220 · L2Authd 2104 · Gate 2106 · LogServer 2051 · ICServer 2005 · CacheD 2006/2007 · Server64 2002+7777 · CAPTCHA 22206 · PA 10057 · (chat 10254 — известный мёртвый).

### Группа L «Лёгкие» (SYSTEM, session-0, почти не трогаем)
| Кнопка | Действие на VM | Предусловие | Пост-проверка |
|---|---|---|---|
| Start | `schtasks /run` AionAcc/AionLog/AionICSrv/AionCAPTCHA/AionPA | порт не слушает | порт открыт ≤3 мин (acc: ждать SQL 1433 → 2220) |
| Stop | через SYSTEM-задачу-киллер | — | порт закрыт |
| Restart | stop→start | — | порт снова открыт |

### Группа AUTH «Аутентификация» (только интерактивная сессия, /IT)
| Кнопка | Действие | Предусловие | Пост-проверка |
|---|---|---|---|
| Start | `schtasks /run /tn AionAuth` → wait 2104 → `AionGate` → wait 2106 | SQL+AccountCache живы | 2104+2106, gate↔authd ESTABLISHED |
| Stop / Restart | AionGate/AionAuth киллер-задачи | — | порты закрылись/открылись |

⚠ Кнопки вызывают ТОЛЬКО обёртки-задачи (auth.bat/gate.bat), не голые exe — голый exe = диалоги серийника и мгновенная смерть вне сессии.

### Группа WORLD «Пара» (главный блок, все действия danger-confirm)
| Кнопка | Действие | Предусловие | Пост-проверка |
|---|---|---|---|
| START WORLD (guided) | NPC-обёртка → ждать маркер загрузки → MAIN-обёртка | группа L+AUTH зелёные | conns2002=16, 7777 LISTENING, winlog «*new world server connection», NPCSvr private ≈15 ГБ |
| RESTART PAIR | stop MAIN → stop NPC → (пауза 20с) → START WORLD | вне окна загрузки | как выше |
| STOP PAIR | MAIN → NPC (graceful, taskkill-задачи) | — | оба процесса+порты ушли |
| KICK & RESTART | = RESTART PAIR с галкой «игроков выкинет» | typed-confirm | — |
| Nightly restart | тумблер авто-рестарта пары (анти-утечка Abyss.cpp/handles) | cron-окно в YAML | журнал запусков |

### Группа SQL «Наблюдение» (кнопок управления НЕТ)
| Кнопка | Действие |
|---|---|
| Snapshot waits | dm_os_wait_stats дифф за окно (RESOURCE_SEMAPHORE/LCK_M_X динамика) |
| Read CatchBlocked | вычитать ring-buffer XE-сессии CatchBlocked → событие в журнал |
| Snapshot blocking | живой срез dm_exec_requests/blocking в файл (канон liveblk.ps1) |

### Группа SVC «Не работает / не поддерживается» (кнопки locked)
ChannelChat (10254), NPRelay, PetitionD, ShopAgent, Ranking — замок с подписью «мёртвые в 7.7-ките, см. PLAN.md §4» — чтоб рука не тянулась.

### Глобальные кнопки (верхняя панель)
- **START ALL** — guided: LOG → CACHE → NPC (10–15 мин) → MAIN, пошагово с port-wait (канон v5-батника).
- **STOP ALL** — обратный порядок + err-чистка.
- **🚨 EMERGENCY SNAPSHOT** —一键 diag-пак: tasklist+netstat+хвосты *.err+blocking+CatchBlocked+FreeCommit → файл+вкладка. (Это то, что в ночи 04–05.10 делалось руками по 20 минут.)

---

## 3. Этапы работ

| Фаза | Что делаем | Кнопки | Критерий готовности |
|---|---|---|---|
| **0. Skeleton** | YAML-конфиг (сервисы/порты/группы/порядок), state-machine ядро, SSH-пробы (tasklist/netstat/ps → health), embedded web-UI: Overview+вкладки | все disabled, observe | Дашборд показывает реальный живой стек и корректно краснеет на поднятой/убитой службе |
| **0.5. Глаза** ✅ 05.10 | Лог-тейлеры+парсер (2812, Too slow, DeadLock/Super-Lag, AboutToPlayerTimer, Shutdown By NpcSocket), метрики: RAM/handles по процессам, FreeCommit/FreePhys, SQLite-WAL кольцо 30 дней; pprof. Отложено на Phase 1: CCU и SQL-waits (нужен SQL-доступ) | observe | Парсер 13 правил + 7 алерт-правил, mock-сценарии в демо; алерты: leak/freecommit/rate2812/crit_log протестированы |
| **1. Руки (агент)** | Go-агент ~2 МБ в юзер-сессии VM (ЕДИНСТВЕННАЯ инсталляция на прод, по «го»): выполняет кнопки через schtasks-задачи, шлёт health/log-стримы; кнопки разблокируются (operate) | активны | RESTART PAIR делает пару за 1 клик с полной пост-проверкой; месяц без ручных «пожаров» |
| **1.5. Автопилот** | Watchdog-правила (авто-рестарт по политике юзера, ночной рестарт пары, алерт-эскалация) | тумблеры | Ночь 04–05.10 («куча крашей») прошла бы без участия юзера |
| **2. Ghidra-кандидаты** | Импорт Server64.pdb+exe (#180): порог CheckIOThreadDeadlock, авторизацион-тайм → патч-лист на СТЕНДЕ (копия MainServer_backup), не прод | — | Список патчей + стенд-прогоны |

## 4. Тех-стек (фиксируем)
Go 1.2x (`~/STELGEN/go-dist`) · embedded HTML без зависимостей · SQLite WAL · `golang.org/x/crypto/ssh` (Phase 0) · go-mssqldb read-only-логин (SQL-вкладка) · агент: gopsutil + локальный TCP-чек. YAML — единственный источник истины о топологии.

## 5. Риски/грабли (уже известные, зашиваем в дизайн)
1. Окно спавна NPC 10–15 мин — блокиратор рестартов (см. lock-и).
2. Смешанный authd-стейт при кривом рестарте — Restart AUTH = полный цикл gate→authd с ожиданием 2110 ESTABLISHED.
3. Pagefile/commit 44 ГиБ — алерт по FreeCommit < 8 ГБ (ночь 03.10: тихая смерть процессов).
4. Динамика хендлов Server64 (827k) и NPCSvr-утечка — метрики-динамика, не абсолюты.
5. Автологон VM обязателен для /IT-задач — health-чек «десктоп-сессия есть» (quser console).
6. SQL-диалоги при старте лёгких раньше SQL — порядок в YAML + wait 1433.

# aion-op — оператор стека AION 7.7 (Phase 1: observe + глаза + SQL-вкладка; прод = VM)

Единый Go-бинарь. **Фаза 1**: read-only пробы + лог-парсер + метрики + SQLite + алерты
+ CCU/SQL-waits (read-only логин aionop_ro). Управление: act-слой с dry-run=true по
умолчанию; POST /api/action существует только в operate-режиме.
Прод-деплой: `C:\aionop\` + задача AionOp — см. [DEPLOY.md](DEPLOY.md).
План: [../TRACK-A-PLAN.md](../TRACK-A-PLAN.md), постановка: [../PLAN.md](../PLAN.md).

## Запуск

```bash
# демо без VM (mock-стек: всё зелёное)
go build -o aion-op . && ./aion-op -config config.yaml
# → http://127.0.0.1:10200 (+ pprof http://127.0.0.1:10201/debug/pprof/)

# демо «краснеет»:
AIONOP_MOCK_DOWN=main,gate ./aion-op            # мёртвые сервисы + pair_broken
AIONOP_MOCK_CONNS=4 ./aion-op                   # окно загрузки NPC (рестарты заблокированы)
AIONOP_MOCK_LEAK=main,npc ./aion-op             # хендлы +120k/мин → алерт утечки ×3 тика
AIONOP_MOCK_EVENT_EVERY=1 ./aion-op             # темп синтетических лог-событий

# прод (read-only по SSH; кнопок всё равно нет)
# vm.mode: ssh в config.yaml + доступ по ключу
./aion-op -config config.yaml
```

## Что уже есть (Phase 0 + 0.5)

**Phase 0 (скелет):**
- state machine (`RUNNING/LOADING/DEGRADED/STOPPED/UNKNOWN`), health = процесс + порт + conns-маркер;
- пара NPC+MAIN = единая единица: разрыв пары и окно загрузки (conns<16) детектируются;
- режим `observe` жёстко в конфиге; управляющих HTTP-роутов НЕТ (только GET);
- топология — единственный YAML-источник истины, секретов в нём нет.

**Phase 0.5 (глаза):**
- **лог-парсер** 13 правил → события: `super_lag`/`intentional_exception`/`world_shutdown` (crit),
  `too_slow`/`login_wait`/`date_mismatch` (med), `proc_missing`(2812)/`session_mismatch` (low),
  `login`/`world_registered`/`npc_started` (info); шум `Strings DB unexpted id` — дропается;
- **тейлеры**: ssh (`Get-Content -Tail` по `{{date}}.err`-файлам, дедуп по хэшам строк) | mock (ротация сценариев);
- **метрики**: RAM + handles + Δхендлов/мин по процессам, FreePhys/FreeCommit (powershell Get-Process/CimInstance, read-only);
- **store**: SQLite WAL (`modernc.org/sqlite`, без CGO), retention 30 дней, часовая чистка, один писатель;
- **алерты** (7 правил): смерть сервиса, разрыв пары, FreeCommit<8 ГБ (гистерезис 10 ГБ),
  утечка хендлов >20k/мин ×3 тика, рейт 2812 >50/5мин, критичный лог (окно 15 мин), проба VM;
- **pprof** на loopback — без роста за 48 ч.

**Phase 1 (руки + SQL-вкладка):**
- режим `local` (оператор живёт на VM), `bind: 0.0.0.0` (решение юзера — локалка);
- **CCU**: Aion_log TBL_GAME_WORLD_INFO (zone0: LIGHT/DARK/NPC_COUNT) + AionAccounts user_count (world/limit/auth/wait per server);
- **SQL-waits**: дельты dm_os_wait_stats (top8), blocked, RESOURCE_SEMAPHORE_QUERY_COMPILE глубина+ожидание — вкладка `/api/sql`;
- **act-слой**: план→safety→confirm→исполнение→audit (таблица actions): start (schtasks /run), stop (taskkill session-0 / kill_task для юзер-сессии), restart_pair; reject: locked/loading-window/probe-dead; dry_run=true по умолчанию (действия только планируются);
- алерты +2 SQL-правила: blocked>3 ×2 тика, compile-очередь>0 — root ночи 04–05.

⚠ Честно отложено на 1.5: watchdog-автопилот (ночной рестарт пары), реальное исполнение (operate+dry_run:false — по «го» юзера), async-ожидания маркеров в act-шагах.

## Пробы

Snapshot: `tasklist /fo csv /nh`, `netstat -ano -p tcp`, `quser`.
Метрики: `Get-Process -Name … | Select ProcessName,Id,Handles,MB | ConvertTo-Csv` + `Win32_OperatingSystem` (один вызов, read-only).
Логи: `Get-Content -LiteralPath '<путь>' -Tail N`.

## Структура

```
config.yaml            — топология + store/pprof/metrics/logs
main.go                — сборка: probe → store → tailer → metrics → alerts → web/pprof
internal/config/       — YAML + валидация
internal/core/         — state machine (чистые функции, тесты)
internal/probe/        — Prober+Runner: mock | ssh (read-only)
internal/logs/         — парсер правил + тейлеры (ssh|mock)
internal/metrics/      — collector (ssh|mock) + Holder
internal/store/        — SQLite WAL: events/proc_metrics/sys_mem/alerts
internal/alerts/       — движок правил
internal/web/          — API (GET-only) + embedded UI (вкладки, кнопки-замки)
```

## Дорожная карта (см. TRACK-A-PLAN.md)

- **0** ✅ скелет observe-only. **0.5** ✅ глаза.
- **1**: агент ~2 МБ в юзер-сессии VM (единственная инсталляция на прод, по «го») → режим operate, кнопки.
- **1.5**: watchdog-автопилот (ночной рестарт пары, эскалации).

# aion-op — оператор стека AION 7.7 (Phase 1: РЕАЛЬНОЕ УПРАВЛЕНИЕ стеком на проде)

Единый Go-бинарь. **Фаза 1 живёт на VM с 05.10, управляет стеком с 07.10** (`operate`, dry_run=false):
read-only пробы + лог-парсер + метрики + SQLite + алерты + CCU/SQL-waits + POST /api/action
(start/stop/restart/restart_pair) + группы fork (authdn/forkd) + kick-задачи.

## 🧭 OP-FIRST — единая точка управления стеком (правило для ВСЕХ агентов/чатов)

- Старт/стоп/рестарт/статус любого сервиса = **ТОЛЬКО через op**: ssh →
  `C:\Temp\op-act.ps1 -Action start|stop|restart|restart_pair -Id <svc> [-Confirm restart]`
  (helpers юзать ТОЛЬКО через `-File`, не инлайн-PS с $), статус = `C:\Temp\op-status.ps1`.
- PowerShell/schtasks напрямую = **последний рубеж** (op не помог) — и потом op чиним.
- `Start-Process` из ssh-сессии = ЗАПРЕЩЁН (умирает с сессией). op сам стартует `schtasks /run AionOp`.
- Песочница НЕ имеет TCP к VM кроме :22 — op-API дергать ТОЛЬКО через ssh на VM
  (Invoke-RestMethod на VM; «op висит» = ложный вывод таймаута с песочницы).
- Управляемые сервисы (config-vm.yaml): acc/logd/logsrv(locked)/ic/captcha/pa/authd(task=AionAuthOnly)/
  gate/gateorig(2109)/cache/npc/main/authdn(2117)/forkd(2116) + группа fork.
- **PA обязателен** (кнопка pa разблокирована: без PA = SYSTEM_ERROR(20)); старт-порядок PA ДО authd.
- Критерий мира = 8 коннектов на :2002 (`expected_conns: 8`, было 16 netstat-строк = FALSE ALERT).
Прод-деплой: `C:\aionop\` + задача AionOp — см. [DEPLOY.md](DEPLOY.md).
План: [ROADMAP.md](ROADMAP.md), постановка: [../PLAN.md](../PLAN.md).

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

## 📦 Артефакты

| Что | Где |
|---|---|
| Код/конфиг | этот каталог (internal/*, config.yaml — шаблон без секретов) |
| Роадмап/деплой | ROADMAP.md, DEPLOY.md |
| Доки эксплуатации | docs/ (authlog-*, config-inventory, direct-ports, chronology, session-cached-rootfix) |
| Прод | `C:\aionop\` (aionop-win.exe, config-vm.yaml, aionop.db) |
| Креды/доступы | VM `D:\SAION\creds\` (CREDS.md) |

## Дорожная карта (см. ROADMAP.md)

- **0** ✅ скелет observe-only. **0.5** ✅ глаза.
- **1** ✅ ЖИВОТ на VM (решение юзера 05.10: без агента — op на VM, `vm.mode: local`, `operate`, dry_run=false; start/stop/restart через schtasks/kick-задачи, кириллические пароли задач берутся из реестра Winlogon — в гит не сохраняются).
- **1.5** ⬜: watchdog-автопилот (ночной рестарт пары = тумблер юзера, эскалации), async-ожидания маркеров в act-шагах.

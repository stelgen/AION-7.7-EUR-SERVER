# 🧰 TECH-DEBT: стек nextgen — сквозной аудит по кросс-пульсу (10.10.2026)

> Статус: 🟡 OPEN. Собрано гейт-чатом 10.10 из пульса соседей за простой (authd R6-свитч,
> cache R1-prep, npc/main пульс, живой op aionstatus 12:57 VM). Дополняет
> [tech-debt-db-20261008.md](tech-debt-db-20261008.md) — там долг БД (~85%), здесь всё остальное.
> Правило WORKFLOW #4: закрыли → ✅+дата; устарело → удаляем, не тащим.

## 1. Правила листа

- Каждая строка: источник-факт (пульс/алерт/live-статус), владелец-компонент, оценка, статус.
- Статусы: ⬜ открыт · 🟡 в работе · ⏳ ждёт «го» юзера (прод-действие) · ✅ закрыт (дата).
- Владелец ведёт деталь в СВОЕМ README/ROADMAP; здесь — только сводка и приоритет.

## 2. Лист (по приоритету)

| # | Задача | Где (владелец) | Источник | Оценка | Статус |
|---|---|---|---|---|---|
| TD-1 | Сервис **authdprod** в config-vm.yaml (AionAuthdProd, 2110+2104) + рестарт op | aion-op (OP-1) | authd R6 09.10 | 15 мин | ⏳ «го» |
| TD-2 | **Kill-коллизия aion-authd.exe**: `stop authdn` убивает и ПРОД — kill по PID/переименование тени | aion-op (OP-2) | actions log 10.10 (taskkill /IM ×4) | полдня | ⬜ до фикса НЕ жать stop authdn |
| TD-3 | **expected_down** для ориг-сервисов (gateorig/authd) — шум алертов `down:*` на зелёном стеке | aion-op (OP-3) | aionstatus 12:57 | 2-4ч | ⬜ |
| TD-4 | Заголовок группы fork «R5 fork-stend…» → «Fork-откат (R6)» | aion-op (OP-4) | config line 194 | 5 мин | ⬜ |
| TD-5 | Async-ожидания маркеров (гонка старта гейта: первый /run после kick иногда не поднимает) | aion-op (OP-5) | деплой R6 09.10 | Phase 1.5 | ⬜ |
| TD-6 | **Gate T2-а/T4/T6**: TTL-сверка (опц.), стабильность (5 логинов суммарно/2 клиента параллельно — 2 инсталляции, критерий юзера), финализация+tag (T3 ✅ live 10.10: клиент 0x08 не шлёт; T5 ✅ 10.10, exe задеплоен f146a415) | aion-gate | README §T2/T3 10.10 | дни | ⬜ (T2-б/в ✅ R6, T3 ✅, T5 ✅) |
| TD-7 | **Authd R6-хвосты**: завершить наблюдение 24ч; pk1-эхо/IP-дворд A/B на живых логинах (мир теперь на нашем 2104); опц. ACS-клиент 2220 | aion-authd | ROADMAP 10.10 | 1-2 дня | 🟡 наблюдение идёт |
| TD-8 | **CacheD64 R1**: wire 2006 из готовых log/*.log (171 файл/356МБ); capture = fork-копия :2016 (go) / тест-мир LAN — **pktmon-2006 loopback-блокер доказан** | aion-cache | npc cross-pulse dd54a7d + d701395 | 1-2 дня | 🟡 R1-prep ✅ 10.10 |
| TD-9 | **proc_missing-каталог БД** (10 procs: GetItemCollection×5, LoadFameInfo, LoadReinventInfo, getItemAttributeDeltaListAll_20190919 +VendorDark/Light, DeleteItemByDate) — спам повторён live 12:30–12:44 | aion-main/БД (→ tech-debt-db TD1/TD2) | op-алерты 10.10 | 2-4 дня | 🟡 каталог ведётся |
| TD-10 | **Watch-lists мира**: too_slow Server64 (до 18с!), хендлы 856k (Δ −36/мин), NPCSvr 1086МБ/604k хендлов → утечка → ночной рестарт пары | aion-main / op Phase 1.5 | docs/app-architecture.md §7 + aionstatus | пассивно | 🟡 мониторится op |
| TD-11 | REF58-процы деплой (`ref58-logprocs-pending-20261005.sql`) + ship-приёмник logd/captcha | aion-logd | README §6 стека | 1 день | ⬜ |
| TD-12 | Телеметрия: rsyslog→Loki→Grafana на LAN + `ship.enabled: true` в прод-конфигах | сквозной (S2) | README §6 стека | полдня | ⬜ |
| TD-13 | Уборка probe-акков probetest1-14 (uid ~1007-1021, мусор в AionAccounts) | VM SQL | ROADMAP §2 #10 | 5 мин | ⏳ «го» |
| TD-14 | **Phase 1.5 watchdog**: событийный автопилот (ночной рестарт пары = тумблер юзера) | aion-op (OP-6) | ROADMAP op | 1-2 дня | ⬜ |
| TD-15 | Гигиена дат в доках: смешение 07/08/09/10.10 при одних коммитах — принять ЕДИНУЮ конвенцию (VM-время) и пройтись по хабам | процесс (WORKFLOW) | этот аудит | часы | ⬜ |
| TD-16 | aion-main R4.5+: свитч мира на тень :7778 → правка `worldPort` в config-prod.yaml гейта | aion-main (гейт = исполн.) | main R0-R4 пульс | деприор | ⬜ после MVP-мира |

## 3. Live-срез 12:57 VM 10.10 (источник фактов)

- **Зелёное**: gate 5480:2106 · authdprod 2110+2104 (жив, не в op-конфиге) · тень 2117 · forkd 2116 ·
  cache 2006/2007 · NPCSvr (1086МБ) · Server64 7777 · мир 8/8 conns :2002 · acc 2220 · logd(наш) 2051 ·
  IC 2005 · captcha(наш) 22206 · PA 10057 · SQL 1433.
- **Алерты**: `down:gateorig`, `down:authd` = EXPECTED (ориги остановлены, замены живы) → TD-3.
- **События**: proc_missing ×10 procs каждые ~5 мин (TD-9) · too_slow до 18.2с (TD-10) ·
  login_wait «Client(Acct: 1010) is not connected» 12:20 (разовый, мир-сторона).
- **Действия op за день**: рестарты gate (гонка старта, TD-5), stop/start authdn — taskkill /IM
  убивал оба процесса aion-authd.exe (TD-2 — источник факта).

## 4. Связанные доки

- БД-долг (~85% кита): [tech-debt-db-20261008.md](tech-debt-db-20261008.md) (TD1–TD6 БД).
- Известные баги кита: [fixes-known-issues.md](fixes-known-issues.md); реестр фиксов: [fixes-registry.md](fixes-registry.md).
- Чаркаунт/висяк выхода: [../nextgen/aion-authd/docs/techdebt-charcount-20261009.md](../nextgen/aion-authd/docs/techdebt-charcount-20261009.md), aion-authd/ROADMAP.
- Op-детали: [../nextgen/aion-op/ROADMAP.md §6](../nextgen/aion-op/ROADMAP.md).
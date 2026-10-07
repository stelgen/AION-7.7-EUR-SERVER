# 🚀 NEXTGEN — свой оператор-бинарь + поэтапная замена NC-сервисов

> Подраздел создан 05.10.2026. Полная постановка: [PLAN.md](PLAN.md).
> Локальный проект с артефактами: `~/STELGEN/projects/aion_rev_2026-10-05/` (бинари/PDB NC-контента в гит НЕ заливаются).

**Цель (постановка юзера):** один многопоточный Go-бинарь без утечек, который подключается к текущему стеку и обслуживает его: лаунчер (порядок старта, watchdog, парные рестарты), вкладки по сервисам (web-UI), бизнес+тех мониторинг. В будущем — максимум своего сервера, без трипанемы MSSQL/ODBC (DSN-диалоги/ACP/connStr-реестр — болезнь только NC-бинарей). Рассматривается Linux-хостинг оператора (KSM/zswap на Proxmox — потенциально −30–50% RAM).

- **Трек A** (недели): `aion-op` оператор — supervisor + watchdog + лог-тейлеры + метрики (SQL waits/CCU/RAM/хендлы) + алерты + вкладки. Прод не трогает (Phase 0 = SSH read-only).
- **Трек B** (месяцы): замены — CAPTCHA → AuthGateD → L2Authd → LogServer64 → AccountCache → ChannelChat → CacheD(RAM-кэш) → ICServer. Server64+NPCSvr+ScriptDLL64 — НЕ переписывать: Ghidra+PDB точечные патчи (метод #180).
  Статус: ✅ **aion-logd** (прод 2051), ✅ **aion-captcha** (прод 22206), ✅ **aion-gate** (прод 2106, РЕЛИЗ 08.10 —
  замена AuthGateD; README/архитектура: [aion-gate/README.md](aion-gate/README.md), [aion-gate/docs/architecture-aion-gate-20261007.md](aion-gate/docs/architecture-aion-gate-20261007.md)).
  **→ ТЕКУЩИЙ: L2Authd → свой authd** — ресёрч закрыт 07.10, шанс ~85%: [AUTHD-RESEARCH.md](AUTHD-RESEARCH.md),
  план [AUTHD-ROADMAP.md](AUTHD-ROADMAP.md) (R0-R6, реестр S1-S7), промпт [PROMPT-AUTHD.md](PROMPT-AUTHD.md); эталоны [authd-ref/](authd-ref/).
  **MVP-код готов 07.10** (R1-R4: wire 2110 + логика live-фактов + DB-слой, тесты зелёные, probe-e2e OK) —
  [aion-authd/](aion-authd/README.md): НЕ деплоен, прод не тронут; перед свитчем нужны R0-верификация
  (procs AionAccounts, роль 2104 Server64) и R5 fork-дифф.
  Ресёрч **AccountCacheServer** закрыт 07.10 (~92%): [ACCOUNTCACHE-RESEARCH.md](ACCOUNTCACHE-RESEARCH.md) + [accountcache-ref/](accountcache-ref/) (PDB 92МБ+map+101 procs+21 таблица; wire+dispatch сняты дизasmом: [dispatch-77.md](accountcache-ref/dispatch-77.md)).
  **Каркас R2 готов 07.10**: [aion-accache/](aion-accache/README.md) (Go: proto+dispatch+cache+db-интерфейс, тесты зелёные, БЕЗ capture) — план [ACCOUNTCACHE-ROADMAP.md](ACCOUNTCACHE-ROADMAP.md) (R1 capture-стенд :2220 → R3 SQLStore → R4 A/B → R5 свитч), промпт [PROMPT-ACCACHE.md](PROMPT-ACCACHE.md). Прод не тронут.
  Ресёрч **CacheD64 (мир-кэш, 2006)** закрыт 08.10 (~85%, R0 готов): [CACHE-RESEARCH.md](CACHE-RESEARCH.md) + [cached-ref/](cached-ref/) —
  PDB 106МБ+map сняты, словари RQ 382/RP 255/GQ 55/GP 53, класс-карта (DbToServer 303/ServerToDb 192/Admin 33+17/IC),
  DB-контракт 781/789 procs в нашей БД; публичный передний край = НОЛЬ; следующий шаг R1 = pktmon capture 2006.

**Джекпот проекта:** на VM лежат родные PDB-символы NC ко всем ключевым нативным бинарям (~1.1 ГБ; локально скачаны малые, MD5-манифест гигантов: [manifest-pdb-big.md](manifest-pdb-big.md), бинарей: [manifest-bin.md](manifest-bin.md)).

**Код Трека A (Phase 0, observe-only):** [aion-op/](aion-op/) — скелет оператора: YAML-топология, state machine, пробы mock/ssh (tasklist/netstat/quser), web-UI с вкладками; кнопки замком, управляющих роутов нет. Детальный план работ и карта кнопок: [TRACK-A-PLAN.md](TRACK-A-PLAN.md).

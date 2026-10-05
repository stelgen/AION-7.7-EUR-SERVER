# 🚀 NEXTGEN — свой оператор-бинарь + поэтапная замена NC-сервисов

> Подраздел создан 05.10.2026. Полная постановка: [PLAN.md](PLAN.md).
> Локальный проект с артефактами: `~/STELGEN/projects/aion_rev_2026-10-05/` (бинари/PDB NC-контента в гит НЕ заливаются).

**Цель (постановка юзера):** один многопоточный Go-бинарь без утечек, который подключается к текущему стеку и обслуживает его: лаунчер (порядок старта, watchdog, парные рестарты), вкладки по сервисам (web-UI), бизнес+тех мониторинг. В будущем — максимум своего сервера, без трипанемы MSSQL/ODBC (DSN-диалоги/ACP/connStr-реестр — болезнь только NC-бинарей). Рассматривается Linux-хостинг оператора (KSM/zswap на Proxmox — потенциально −30–50% RAM).

- **Трек A** (недели): `aion-op` оператор — supervisor + watchdog + лог-тейлеры + метрики (SQL waits/CCU/RAM/хендлы) + алерты + вкладки. Прод не трогает (Phase 0 = SSH read-only).
- **Трек B** (месяцы): замены — CAPTCHA → AuthGateD → L2Authd → LogServer64 → AccountCache → ChannelChat → CacheD(RAM-кэш) → ICServer. Server64+NPCSvr+ScriptDLL64 — НЕ переписывать: Ghidra+PDB точечные патчи (метод #180).

**Джекпот проекта:** на VM лежат родные PDB-символы NC ко всем ключевым нативным бинарям (~1.1 ГБ; локально скачаны малые, MD5-манифест гигантов: [manifest-pdb-big.md](manifest-pdb-big.md), бинарей: [manifest-bin.md](manifest-bin.md)).

**Код Трека A (Phase 0, observe-only):** [aion-op/](aion-op/) — скелет оператора: YAML-топология, state machine, пробы mock/ssh (tasklist/netstat/quser), web-UI с вкладками; кнопки замком, управляющих роутов нет. Детальный план работ и карта кнопок: [TRACK-A-PLAN.md](TRACK-A-PLAN.md).

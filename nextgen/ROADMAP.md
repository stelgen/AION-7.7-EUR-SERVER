# 🗺 ROADMAP — живой план проекта (обновляется в каждом чате, не терять контекст)

> **Последнее обновление: 10.10.2026.** Статус стека и дельты — в [README.md](README.md) §2 (главная таблица).
> Кратко: гейт ✅ релиз (f8912a9, полный юзер-флоу, юзер играет через fork-стенд); logd ✅ captcha ✅ в бою; op ✅ Phase 1 управляет стеком;
> authd ✅ R6 В БОЮ 09.10 (живой путь 2110+2104; fork 2116/тень 2117 = откат); accache 🟡 R2 каркас (R1 capture = следующий чат); CacheD64 🔬 R0 закрыт (R1 wire из log/*.log — pktmon-2006 loopback-блокер доказан); PA = ОБЯЗАТЕЛЕН (SYSTEM_ERROR 20 без него — старое «SKIP» исправлено везде).
> Следующие чаты: 1) accache R1 capture [aion-accache/PROMPT.md] 2) cache R1 wire из логов [aion-cache/PROMPT.md] 3) gate T2-а/T3/T4/T6 4) op Phase 1.5 + R6-конфиг (OP-1..OP-6).

## 1. СТАТУС КОМПОНЕНТОВ (что где — 10.10; свод с % — [README.md §2.5](README.md))

| Блок | Статус |
|---|---|
| Логгер aion-logd | ✅ штатный :2051 (exe MD5 `7aca9dca`, PID см. op), Л1–Л4 закрыты, 0 parse-ошибок; откат = `schtasks /run AionLog`; pending: REF58-процы (§2.3) + ship-приёмник (§2.2) |
| Капча aion-captcha | ✅ в бою :22206 (exe MD5 `5394aab1`), буфер 10000 за ~4с (ориг 6.4 мин), Server64.err чист; откат = retarget AionCAPTCHA на `C:\Temp\captcha.bat` |
| Гейт aion-gate | ✅ РЕЛИЗ на :2106 `mode: authgate` (коммит f8912a9, exe sha `7c4dcab`): e=65537, op=0x00 live-клиента, blob asm-форма 191Б, T1-фейлы (authdTimeoutSec=15, failClose=2, onlineTtl=0 ВЫКЛ, реестр текстов 1..22+45 live); полный юзер-флоу 1/1 и stelgen доказан; юзер играет через fork-стенд; откат exe = `.bak-63f6a47/9c85c12/9f2da98/e1dd475`, режим = `mode: fork` |
| Fork-стенд authd | ✅ ОТКАТ-РЕЗЕРВ (R6-свитч 09.10): живой путь = наш authd (2110+2104, задача AionAuthdProd); forkauthd :2116 + shadow :2117 остановлены КАК ПУТЬ, живы как откат (`C:\Temp\rollback-gate.ps1` + `D:\SAION\aion-authd\authd-rollback.cmd`); VERDICT-лог `D:\SAION\aion-authd\fork-authd.log` |
| Authd (наш) | ✅ **R6 В БОЮ 09.10**: живой путь (2110+2104), полный цикл юзера + мгновенный перелогин + pk1-эхо + квитанции; наблюдение 24ч |
| aion-accache | 🟡 R0 ✅ (PDB 92МБ+101 procs+21 табл), R0.5 ✅ (dispatch-таблица T1 0..39/T2 0..7, wire [len-2][cmd][0xEB][~cmd]), R2 ✅ (Go-каркас, тесты зелёные); R1 capture-стенд = следующий |
| CacheD64 ресёрч | 🔬 R0 ✅ 08.10 (PDB 106МБ/14281 publics, RPC-словари RQ382/RP255/GQ55/GP53, 781/789 procs, 356МБ готовых логов); кода нет; шанс ~85% |
| aion-op (Трек A) | ✅ Phase 1 в бою + **AGENT API** (07.10, S12): управляет стеком (start/stop/restart/restart_pair, группы fork: authdn/forkd), SQL/CCU-вкладки, алерты, kick-задачи (AionKickGate = /IM aion-gate.exe точно); expected_conns=8 (16 netstat-строк); канал агента = `:10200/api/agent/*` ([AGENT-SPEC.md](AGENT-SPEC.md)); Phase 1.5 НЕ начата; ⏳ R6-конфиг OP-1..OP-6 ([docs/tech-debt-stack-20261010.md](../docs/tech-debt-stack-20261010.md)) |
| Батники | ✅ `AION-START-ALL-v6.bat` (десктоп, всё в session 1) = канон; `C:\Temp\auth.bat = call start-all.bat` — НЕ трогать как кнопку (это старт ВСЕГО стека); изолированный L2Authd = `auth-only.bat`/AionAuthOnly; откат v5 рядом |
| Феномен «задачи сами Disabled» | ⚠ следить (смягчено pre-check в op + enable-all) |

## 2. ОТКРЫТЫЕ ПУНКТЫ (по приоритету)

| # | Пункт | Где | Оценка |
|---|---|---|---|
| 1 | **accache R1 capture**: копия ACS на :2221 (байтовая правка common.xml в КОПИИ каталога) + наш fork-proxy :2220→:2221 + логины юзера → payload-раскладки per-cmd, ACP-номера, T2-канал | aion-accache/PROMPT.md | 1–2 дня |
| 2 | **CacheD64 R1**: wire 2006 из готовых log/*.log (171 файл, 356МБ) + capture через fork-копию :2016 (go) / тест-мир LAN (pktmon-2006 = loopback-блокер доказан 10.10) → wire 2006 | aion-cache/RESEARCH.md §MVP | 1–2 дня |
| 3 | **authd R6-хвосты**: завершить наблюдение 24ч; pk1-эхо/IP-дворд A/B на живых логинах (мир уже на нашем 2104); опц. ACS-клиент 2220 | aion-authd/ROADMAP.md | 1–2 дня |
| 4 | **Гейт T2–T6**: T2-а TTL-сверка (опц. — флаг снимается миром 40/3 + [01] + TTL), T4 стабильность (5 логинов суммарно, 2 клиента параллельно — 2 инсталляции достаточно), T6 финализация+tag (T3 ✅ live 10.10 — kill-в-мире перелогин чистый, 0x08 клиент не шлёт; T5 ✅ 10.10 — exe f146a415 задеплоен) | aion-gate/README §T2/T3 | дни |
| 5 | Телеметрия: rsyslog→Loki→Grafana на LAN + `ship.enabled: true` в прод-конфигах | TELEMETRY-SPEC §2 | полдня |
| 6 | Деплой 2 REF58-проц (`scripts/sql/ref58-logprocs-pending-20261005.sql`) + маппинг metric1-4 → logdb UpdateMainStatus (методы готовы, вызов заглушен) | [aion-logd/SNAPSHOT.md](aion-logd/SNAPSHOT.md) | 1 день |
| 7 | op Phase 1.5: событийный watchdog (ночной рестарт пары = тумблер юзера), async-ожидания маркеров + **R6-конфиг**: сервис authdprod, expected-down, kill-коллизия aion-authd.exe (OP-1..OP-6) | aion-op/ROADMAP.md §6 | 1–2 дня |
| 8 | Ghidra-патчи: матчмейкер #108 (JZ→JNZ), манастоны #111 (перенос в копию #180) — только в MainServer_backup-копии | fixes-pending/ | дни, стенд |
| 9 | Watch-листы: хендлы Server64 (827k+228/мин), утечка NPCSvr (~600k блоков/сессия → ночной рестарт), RESOURCE_SEMAPHORE | docs/app-architecture.md §7 | пассивно |
| 10 | Уборка: тестовые probe-акки probetest1-14 в AionAccounts (мусор от probe, uid ~1007-1018) — удалить по «го» | VM SQL | 5 мин |
| 11 | ICServer/ChannelChat/Petition/ShopAgent/GM — папки-заготовки созданы (README+ROADMAP+PROMPT в каждой), к работе не запланированы | aion-ic/, aion-chat/, aion-petition/, aion-shopagent/, aion-gm/ | — |

## 3. ТРЕК B — КАРТА ПЕРЕПИСИ (08.10, ✅ нужен стеку / − некритичен / ❌ не нужен)

| # | Замена | Статус | Шанс | Примечание |
|---|---|---|---|---|
| ✅ 1 | LogServer64 → aion-logd | ✅ в бою | 100% | метод capture→PDB→Go→fork→свитч отработан |
| ✅ 2 | CAPTCHAImageServer → aion-captcha | ✅ в бою | 100% | свитч 05.10; промпт закрыт (АРХИВ) |
| ✅ 3 | AuthGateD → aion-gate | ✅ РЕЛИЗ | 100% | полный юзер-флоу живой; хвост T2-T6 |
| **4** | **L2Authd → aion-authd** | ✅ R6 в бою | 100% MVP | свитч 09.10; полный цикл юзера; откат-цепочка готова; тех-долг: charcount/висяк выхода (ниже среднего) |
| **5** | **AccountCacheServer → aion-accache** | 🟡 каркас | ~90% | dispatch+wire сняты дизasmом; R1 capture; PDB+procs в гите |
| **6** | **CacheD64 → aion-cache (будет)** | 🔬 R0 закрыт | ~85% | 8× больше ACS по RPC (~590 команд); MVP read-путь + write-транзит в SQL |
| 7 | ICServer → свой | ⬜ не тронут | ~50% | транзакционный хаб 3 сторон; PDB 104МБ; без него лупер IC — пока ориг |
| 8 | NPCSvr64 | 🔬 ДЕПРИОРИТ ~15% (R0 ✅, R1 ⏸) | ~85% эталонов ×7 | **перепись В ПЛАНЕ** (бескомпромиссно); пока мир на ориг — тактика = aion-binpatch |
| 9 | Server64/MainServer | 🟡 ДЕПРИОРИТ ~30% (R0–R4 ✅) | ~85% | **перепись В ПЛАНЕ** (последняя в свитче); пока Ghidra-патчи: #108/#111/#180 + обвязка |
| − 10 | Petition/ShopAgent/ChannelChat | − НЕ КРИТИЧНЫ | ~30% | exe нет в ките; ILSpy = ТЗ; луперы молча |
| − 11 | GMServer | − НЕ КРИТИЧЕН | — | GM = builder в SQL (user_data.builder); GMcmd.txt |
| − 12/13 | NPRelay/Ranking | − НЕ КРИТИЧНЫ | — | не биндят/конфига нет; задачи DISABLE |
| **PA** | PortalAuth | ✅ ОБЯЗАТЕЛЕН (ориг) | — | НЕ переписываем, НЕ скипаем: без него SYSTEM_ERROR(20); старт ДО authd; op-кнопка pa |

⚠️ Требование к каждой переписи: S1-S10 из [README.md §4](README.md) (шаблон README, TELEMETRY-SPEC, LOGGING-SPEC raw-first, FORK-SPEC, op-first, креды-политика, конфиги, деплой/откат, тесты, роадмап+промпт).

## 4. ПРАВИЛА ЭКСПЛУАТАЦИИ (не забыть)

- **КАНАЛ VM = Agent API (S12, основной с 07.10)**: `bash -c 'source nextgen/agent-cli.sh'` → `aionrun`/`aionrun_ps`/`aionput`/`aiongetb64`/`aionls`/`aionlog`/`aionact` (curl `http://192.168.0.125:10200/api/agent/*`, токен `X-Agent-Token` из `~/.aion-agent-token`). БЕЗ ssh-кавычек и PS-кавычек; каждый вызов в audit-лог op. Правило: новый шаг на VM = ps1-скрипт через `aionput` + `aionrun "powershell -File ..."` — НЕ инлайн. Спека: [AGENT-SPEC.md](AGENT-SPEC.md)
- **op-first**: старт/стоп/рестарт = `POST /api/action` через Agent API (или `aionact start <svc>`); helpers `C:\Temp\op-act.ps1`/`op-status.ps1` = фолбэк для человека (только `-File`); op-старт = ТОЛЬКО `schtasks /run AionOp` (Start-Process из ssh УМИРАЕТ с сессией). PowerShell напрямую — только если op не помог, потом чиним op. Оригинальные батники (start-all/auth.bat) = только как канон карты, не кнопки
- **SSH (аварийный/деплой-op)**: `ssh aion` (алиас, ключ dimini-agent, юзер кириллицей `Администратор`); дефолт-шелл = PowerShell (`&&` запрещён — только `;` или файл); scp с кириллицей = push-only, pull через Agent API `aiongetb64`
- Сессии: стек живёт в **session 1** (SYSTEM/session-0 = L2Authd умирает молча); API-команды op исполняются от SYSTEM (session 0) — юзер-сессионные действия только через /IT-задачи/act-слой op; TCP к VM: :22 (ssh) и :10200 (Agent API/UI, firewall-правило `aionop-agent-10200` = **Any с 08.10**, решение юзера — NAT-защита, токен = единственный рубеж), остальные порты закрыты
- Инцидент 08.10 «агент не дотянулся до VM» — закрыт: root = firewall /24; фиксы: RemoteAddress=Any + run.cmd → `op.log` (audit AGENT-строк с IP; диаг-пакет в AGENT-SPEC §Мульти-вантадж)
- Креды: единая папка `D:\SAION\creds\` (ssh, SQL sa, форумы, op) — читать оттуда; в гит/память/логи НЕ класть ([CREDS.md](CREDS.md))
- git: `--no-pager` ПЕРЕД подкомандой; sqlcmd глючит на больших XML — только SqlClient/`-y 0 -Y 0`; PS5 + SQL2022 через System.Data.SqlClient НЕ работает (только sqlcmd); TBL_GAME_* в схеме **aiongm_ur** (не dbo)
- Рестарты: пара NPC+MAIN только вместе (смерть Server64 каскадно убивает NPCSvr — graceful, leak-дампы в .err норма); окно NPC 10-15 мин — не дёргать; exe залочен живым процессом → `taskkill /F` ДО scp; после kill первый /run может словить bind-fail — bat ретраит ~35с, проверять баннер
- Конфиги: yaml на VM = UTF-8/ASCII, комментарии латиницей; правки байтово (python bytes, сначала encode потом write) + LEN-check; канарейка-баннер = конфиг прочитан; Windows-батники ASCII+CRLF; sed-анкер `$` не работает на CRLF
- Бекапы БД перед любым ALTER: `D:\_REF58\prod-backups\` + скрипт в гит + роллбак в скрипте
- Логи: raw-first ([LOGGING-SPEC.md](LOGGING-SPEC.md)); fork-логи арбитраж по позициям в логе (N-ONLY = гонка тени, не баг); parse-ошибки = ship-событие + .raw-дамп

## 5. ГДЕ ЧТО ЛЕЖИТ (контекст)

| Что | Где |
|---|---|
| Мастер-инвентарь приложений | docs/app-architecture.md (обновлён 05.10) |
| Хаб nextgen (стек-таблица + стандарты) | nextgen/README.md |
| Код переписей | nextgen/<svc>/ + прод `D:\SAION\<svc>\` (dev-наборы `D:\SAION\<svc>-dev\`) |
| Оператор | nextgen/aion-op/; прод `C:\aionop\` (Agent API + UI на 0.0.0.0:10200 — из песочницы напрямую; канал = [AGENT-SPEC.md](AGENT-SPEC.md)) |
| Компонентные доки | nextgen/<comp>/ (README/RESEARCH/ROADMAP/PROMPT/SNAPSHOT/docs — self-contained); общие доки в docs/ |
| Референс-сурсы | nextgen/{authd-ref,cached-ref,accountcache-ref}/README.md; эталоны: `STELGEN/projects/aion_server_2026-10-02/reference/` (Mobius 7.7, beyond-aion 4.8) |
| PDB/бинари | VM `D:\AION_LIVE_SERVER\`; локально `~/STELGEN/projects/aion_rev_2026-10-05/artifacts/`; манифесты nextgen/manifest-*.md |
| Capture-дампы | VM C:\Temp\*, C:\logd-capture\; локально ~/STELGEN/tmp/ |
| Процесс работы агентов | nextgen/WORKFLOW.md (запуск «WORKFLOW: <имя>», пульс, теорий-журнал, чистка) |
| Инструменты | tools/analysis/ (pdbpub.py, дизasmы, gen-скрипты); aion-gate/cmd/{probe,forkprobe} |
| Креды/доступы | VM `D:\SAION\creds\` — единственное место; политика nextgen/CREDS.md |
| Память | STELGEN/projects/aion_server_2026-10-02 (+fixes/, +aion-gate-fork-classic-20261006) — читать в начале каждого чата |

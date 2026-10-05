# 🗺 ROADMAP — живой план проекта (обновляется в каждом чате, не терять контекст)

> Последнее обновление: 05.10.2026, конец сессии «logd Л1–Л4 + подмена + ship e2e».
> Коммиты сессии: 1e7c9d7 (v6 батник) → fe2976b (Л1–Л4+ship+SPEC) → e6ede55 (подмена+rsyslog+дизasm) → см. git log.
> Подробности сессии: docs/session-20261005-logd-final.md. Протокол/статус логгера: nextgen/LOGD-STATUS-SNAPSHOT.md.

## 1. СТАТУС КОМПОНЕНТОВ (что где)

| Блок | Статус |
|---|---|
| Логгер aion-logd | ✅ **ЗАКРЫТ**: штатный на :2051 (PID 2704, exe MD5 `7aca9dca`), Л1 type-9, Л2 var-records (svc701 и пр. больше не теряются), Л3 InitializeCount@svc=3, Л4 retention 14д, ship-телеметрия по TELEMETRY-SPEC, откат = `schtasks /run AionLog` |
| Батники | ✅ `AION-START-ALL-v6.bat` на десктопе (наш логгер, всё в сессии 1); v5 рядом = откат |
| aion-op (Трек A) | ✅ Phase 1 (SQL-вкладка живая, bind 0.0.0.0, operate с кнопками рестартов, kick-задачи); фаза 1.5 — НЕ начата |
| Трек B | 🔄 порядок в §3; logd готов, следующий = **CAPTCHA** |
| Батники подмены логгера | процедура отработана 3 раза: `/end` → ЖДАТЬ смерти процесса (до 10с!) → copy → `/run` |

## 2. ОТКРЫТЫЕ ПУНКТЫ (по приоритету)

| # | Пункт | Где | Оценка |
|---|---|---|---|
| 1 | **Перепись CAPTCHAImageServer → свой** (155КБ, порт 22206, конфиг config.ini; PDB НЕТ) | промпт: nextgen/PROMPT-CAPTCHA.md | 1–2 дня |
| 2 | rsyslog → Loki → Grafana на LAN Linux + в прод-конфиге logd `ship.enabled: true` (секция готова, включить одной строкой) | nextgen/TELEMETRY-SPEC.md §2 | полдня |
| 3 | Деплой 2 REF58-проц (`scripts/sql/ref58-logprocs-pending-20261005.sql`: UpdateTotalMainStatus/InsertServerinfo) + маппинг metric1-4 → logdb UpdateMainStatus (методы готовы, вызов заглушен) | LOGD-STATUS-SNAPSHOT | 1 день |
| 4 | Хвост type-9 через Server64 PDB (284МБ): тик/floats/флаги после entries; MsgId-таблица уже найдена (0x644→4...0x648→9) | /tmp/logsrv.asm, /tmp/logpub.txt | опц. |
| 5 | aion-op Phase 1.5: событийный watchdog (реакция на факт смерти пары; ночной рестарт = тумблер юзера) | nextgen/TRACK-A-PLAN.md | 1–2 дня |
| 6 | Феномен «задачи планировщика сами становятся Disabled» — НЕ решён (смягчён pre-check в restarts + enable-all) | docs/session-20261005-cached-rootfix.md | следить |
| 7 | Watch-листы: хендлы Server64 (827k+228/мин), утечка NPCSvr Abyss (~600k блоков/сессия → ночной рестарт пары), RESOURCE_SEMAPHORE динамика | docs/app-architecture.md §7 | пассивно |
| 8 | Ghidra-патчи Server64 (#180): порог CheckIOThreadDeadlock, authorization-time (уйти от RunAsDate-зависимостей) | nextgen/PLAN.md §6.1 | стенд, потом |

## 3. ТРЕК B — ПОРЯДОК ПЕРЕПИСИ (прод всегда жив, замены переключаемые)

| # | Замена | Оценка | Шанс | Примечание |
|---|---|---|---|---|
| ✅ 1 | LogServer64 → aion-logd | готово | 100% | метод отработан: mirror-capture → PDB publics → Go → паралл. прогон → свитч с откатом |
| 2 | CAPTCHAImageServer → свой | 1–2 дня | высокий | промпт готов (PROMPT-CAPTCHA.md); ship копируем из logd как есть |
| 3 | .NET мелочь (Petition/ShopAgent/GMServer) | дни-недели | высокий | ILSpy-декомпил = готовое ТЗ; реально мёртвые — можно НЕ писать, просто не запускать |
| 4 | AuthGateD → свой гейт | 1–3 нед | высокий | PDB + дизasm готовы; RSA/сессии частично описаны (docs/auth-server-internals.md) |
| 5 | L2Authd → свой | 2–4 нед | средне-высокий | логика в SQL-процах |
| 6 | AccountCacheServer → свой кэш | 2–3 нед | средний | PDB 92МБ на VM |
| 7 | ChannelChat → свой | ~2 нед | средний | mxChat-ключи в common.xml |
| 8 | CacheD64 → RAM-кэш | 1–3 мес | средний | RPC 2006/2007 через PDB+MITM; убивает compile-stormы |
| 9 | ICServer → свой | 1–2 мес | средний | PDB 104МБ |
| 10 | Server64/NPCSvr/ScriptDLL | НЕ переписываем | — | Ghidra-точечные патчи (метод #180) + обвязка |

⚠️ ТРЕБОВАНИЕ к каждой переписи: TELEMETRY-SPEC (в сеть, не файлами; ship не критичный путь; self-статус; raw+ошибки; конфиг `ship.*` единый).

## 4. ПРАВИЛА ЭКСПЛУАТАЦИИ (не забыть)

- Прод-доступ: `ssh 'Администратор@192.168.0.125'` (дефолт-шелл = PowerShell; cmd через `cmd /c "..."`; `&` в PS ломает — только `;` или cmd /c)
- scp push работает, scp pull — нет («invalid user name»): файлы вниз через `cmd /c type` (текст) или PS base64
- git: `--no-pager` ПЕРЕД подкомандой; sqlcmd глючит на больших XML — только SqlClient ExecuteScalar
- TBL_GAME_* в схеме **aiongm_ur** (не dbo); OBJECT_DEFINITION пуст — тела через sp_helptext
- Рестарты пары NPC+MAIN только вместе; окно загрузки NPC 10-15 мин — не дёргать
- Windows-батники: ASCII+CRLF; кириллица в конфигах — только байтовая замена (python)
- Бекапы БД перед любым ALTER: `D:\_REF58\prod-backups\` + гит скрипта + роллбак в скрипте
- Пароли/conn-строки: только в конфигах на VM, в гит/память/логи НЕ сохранять

## 5. ГДЕ ЧТО ЛЕЖИТ (контекст)

| Что | Где |
|---|---|
| Мастер-инвентарь приложений | docs/app-architecture.md (обновлён 05.10) |
| Логгер: протокол/статус | nextgen/LOGD-STATUS-SNAPSHOT.md, LOGD-REWRITE-ANALYSIS.md |
| Стандарт телеметрии | nextgen/TELEMETRY-SPEC.md |
| Код logd | nextgen/aion-logd/ (+ README); прод: D:\SAION\aion-logd\ |
| Оператор aion-op | nextgen/aion-op/; прод: C:\aionop\ (UI 127.0.0.1:10200 через ssh -L) |
| Бинари+PDB | локально ~/STELGEN/projects/aion_rev_2026-10-05/artifacts/; гигантские PDB на VM |
| Дизasmы | /tmp/logsrv.asm (LogServer64), /tmp/logpub.txt; pdbpub.py — ~/STELGEN/tmp/ |
| Сессия 05.10 (этот чат) | docs/session-20261005-logd-final.md |
| Память | STELGEN/projects/aion_server_2026-10-02 (+fixes/) — читать в начале каждого чата |
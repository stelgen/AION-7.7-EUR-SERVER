# 🗺 ROADMAP — живой план проекта (обновляется в каждом чате, не терять контекст)

> Последнее обновление: 05.10.2026 (ночь-4): ДИЗASM 0x407d50 ЗАКРЫТ (docs/session-20261005-authgate-disasm407d50.md): fmt "cddbbbcccc" @0x42cf2c байт-в-байт; plaintext = [0x00][fc][V][mod128][gg16(нули при флаге 0)][key2 16B][S,B0,B1,B2] = 173; welcomeExtra4 = add eax,4 (НЕ поле); EncryptPrimary: roundup8→184, S включает old[k], dword0 не трогается, csum=finalS@dword46(offset 184), ECB 192 → wire 194 ✓; key1=key формулы bit-exact, ctx#1=BF(key1) asm; key2 генерится 0x4075d0 и передаётся ВНУТРИ welcome [153:169] (как классика SM_INIT); 42B echo = EncryptSecondary([sid][28×0]). ⚠ ПАРАДОКС: capture dec(key1)[0]≠0x00 ни в одном offset/направлении при железной asm-модели → sid-гипотеза прошлой сессии = подгон; их Go-билдер (c←PlainByte 0x23) заведомо не совпадает с оригиналом. РЕШАЮЩИЙ ТЕСТ: Go-билдер по asm-модели → живой клиент. AuthGateD ~70%. (ночь-4+, ГО СДАНО): Go приведён к asm-семантике — welcome[0]=0x00 (PlainByte убран), DecryptSecondary исправлен по 0x417ad0 (XOR (len-8)/4 dword, csum@k, возврат len-8), добавлен EncryptSecondary @0x417a80, 34b-путь = handleAuthGG (ответ EncryptSecondary([sid][28×0]), RSA только в логине), encryptToClient удалён (позиция csum была неверной); тесты зелёные -count=1 (в т.ч. capture-структура echo [P][Q][Q][Q][P]). ДАЛЕЕ: win cross-build → живой клиент. РЕСЁРЧ ЧУЖОГО ОПЫТА ЗАКРЫТ (docs/authgate-research-20261005.md + tools/analysis/diag6_sminit_probe.py): welcome НЕ классический SM_INIT (константы c621/197635/2097152/3FCE09ED отсутствуют), GG-зона нулей в ECB-dec не обязана быть видна (скрамбл) — противоречие §7 снято; plaintext[0:8] = per-run сид (первый diff между сессиями @ байт 8); inv_mod == AC-Login encryptModulus (цепочка скрамбла модуля подтверждена опенсорсом); diag7/diag8 (док §7): bit0-тождество cumsum отсутствует для ВСЕХ 45 пар dec и csum≠сумма любых окон dec ⇒ скрамбл-цепочка из capture напрямую не видна, решает только дизasm. ⚠ БЛОКЕР СВИТЧА прежний: дизasm 0x407d50 welcome-билдер (чек-лист вопросов в доке ресёрча §6) → byte-exact vs capture → фейк-клиент → win-build → свитч (го дано заранее). (ночь-2: скелет ГОТОВ — proto/authdclient/server/config/ship/main, e2e зелёный, wire 2110 1-в-1, rsapricrt закрыт, session-док docs/session-20261005-authgate-skeleton.md)
> Коммиты CAPTCHA-сессии: 3bfe87d (recon) → 297b716 (протокол) → 179a808 (код) → финал см. git log.
> Доки сессии: docs/captcha-recon-20261005.md, docs/captcha-protocol-20261005.md, docs/session-20261005-captcha.md,
> статус: nextgen/CAPTCHA-STATUS-SNAPSHOT.md. Прошлая сессия (логгер): docs/session-20261005-logd-final.md.

## 1. СТАТУС КОМПОНЕНТОВ (что где)

| Блок | Статус |
|---|---|
| Логгер aion-logd | ✅ **ЗАКРЫТ**: штатный на :2051 (PID 2704, exe MD5 `7aca9dca`), Л1 type-9, Л2 var-records (svc701 и пр. больше не теряются), Л3 InitializeCount@svc=3, Л4 retention 14д, ship-телеметрия по TELEMETRY-SPEC, откат = `schtasks /run AionLog` |
| Капча aion-captcha | ✅ **ЗАКРЫТА (05.10, в бою на :22206)**: PID 5572, задача AionCAPTCHA → D:\SAION\aion-captcha\run.cmd, exe MD5 `5394aab1`, буфер 10000 наливается за ~4с (оригинал 6.4 мин), Server64.err чист; откат = `schtasks /change /tn AionCAPTCHA /tr "C:\Temp\captcha.bat"` + `/run` (оригинал не тронут) |
| Батники | ✅ `AION-START-ALL-v6.bat` на десктопе (наш логгер, всё в сессии 1); v5 рядом = откат; CAPTCHA стартует своей задачей (не в v6) |
| aion-op (Трек A) | ✅ Phase 1 (SQL-вкладка живая, bind 0.0.0.0, operate с кнопками рестартов, kick-задачи); фаза 1.5 — НЕ начата |
| Трек B | ✅ **AuthGateD СВИТЧНУТ В ПРОД (06.10 ~04:00)**: aion-gate.exe на :2106 (PID см. tasklist), оригинал потушен, откат = C:\Temp\gate.bat.orig-AUTHGATED → gate.bat + schtasks /run AionGate; дизasm 0x407d50 закрыт (док ночь-4), asm-модель self-test'ом подтверждена; лог D:\SAION\aion-gate\gate-prod.log; ОСТАЛОСЬ: живой клиент (решающий тест парадокса capture) |
| Батники подмены логгера | процедура отработана 3 раза: `/end` → ЖДАТЬ смерти процесса (до 10с!) → copy → `/run` |

## 2. ОТКРЫТЫЕ ПУНКТЫ (по приоритету)

| # | Пункт | Где | Оценка |
|---|---|---|---|
| ✅ 1 | aion-op config: captcha-строка → `aion-captcha.exe` + рестарт AionOp — СДЕЛАНО 05.10 (config-vm.yaml, /api/status captcha=RUNNING) | nextgen/aion-op/config.yaml | — |
| 2 | rsyslog → Loki → Grafana на LAN Linux + в прод-конфиге logd/captcha `ship.enabled: true` (секции готовы) | nextgen/TELEMETRY-SPEC.md §2 | полдня |
| 3 | Деплой 2 REF58-проц (`scripts/sql/ref58-logprocs-pending-20261005.sql`: UpdateTotalMainStatus/InsertServerinfo) + маппинг metric1-4 → logdb UpdateMainStatus (методы готовы, вызов заглушен) | LOGD-STATUS-SNAPSHOT | 1 день |
| 4 | Хвост type-9 через Server64 PDB (284МБ): тик/floats/флаги после entries; MsgId-таблица уже найдена (0x644→4...0x648→9) | /tmp/logsrv.asm, /tmp/logpub.txt | опц. |
| 5 | aion-op Phase 1.5: событийный watchdog (реакция на факт смерти пары; ночной рестарт = тумблер юзера) | nextgen/TRACK-A-PLAN.md | 1–2 дня |
| 6 | Феномен «задачи планировщика сами становятся Disabled» — НЕ решён (смягчён pre-check в restarts + enable-all; следить и за AionCAPTCHA) | docs/session-20261005-cached-rootfix.md | следить |
| 7 | Watch-листы: хендлы Server64 (827k+228/мин), утечка NPCSvr Abyss (~600k блоков/сессия → ночной рестарт пары), RESOURCE_SEMAPHORE динамика | docs/app-architecture.md §7 | пассивно |
| 8 | Ghidra-патчи Server64 (#180): порог CheckIOThreadDeadlock, authorization-time (уйти от RunAsDate-зависимостей) | nextgen/PLAN.md §6.1 | стенд, потом |

## 3. ТРЕК B — ПОРЯДОК ПЕРЕПИСИ (прод всегда жив, замены переключаемые)

| # | Замена | Оценка | Шанс | Примечание |
|---|---|---|---|---|
| ✅ 1 | LogServer64 → aion-logd | готово | 100% | метод отработан: mirror-capture → PDB publics → Go → паралл. прогон → свитч с откатом |
| ✅ 2 | CAPTCHAImageServer → aion-captcha | готово | 100% | 05.10: capture 101/102/1001/1002 → Go → свитч (метод отработан 2-й раз); промпт nextgen/PROMPT-CAPTCHA.md — ЗАКРЫТ |
### Карта переписи ВСЕГО стека (06.10, сортировка по лёгкости; ✅ нужен стеку / − некритичен / ❌ не нужен)

| # | Компонент | Критичен? | Оценка | Шанс | Вердикт |
|---|---|---|---|---|---|
| ✅ 1 | LogServer64 → aion-logd | ✅ | готово | 100% | в бою 05.10; метод capture→PDB→Go→свитч отработан |
| ✅ 2 | CAPTCHAImageServer → aion-captcha | ✅ | готово | 100% | в бою 05.10 |
| ✅ 3 | AuthGateD → aion-gate | ✅ | готово (тяжёлая: RSA/asm) | 100% | в бою 06.10; хвост = решающий живой логин (forkprobe) |
| 4 | L2Authd → свой authd | ✅ НУЖЕН | 2–4 нед | ~70% | ЛУЧШИЙ следующий кандидат: wire 2110 уже раскрыт aion-gate (authdclient 1-в-1), DB-процы ap_*/web_* инвентаризованы; убирает хрупкость L2Authd |
| 5 | AccountCacheServer → свой кэш | ✅ НУЖЕН (2220: authd/Server64 ходят) | 2–3 нед | ~60% | PDB 92МБ на VM; ориг стабилен — низкий приоритет |
| 6 | ICServer → свой | ✅ НУЖЕН для старта (без него лупер «Can't connect to Interchange» у Server64+CacheD) | 1–2 мес | ~50% | PDB 104МБ; транзакционный хаб 3 сторон; пока не трогать |
| 7 | CacheD64 → свой RAM-кэш | ✅ НУЖЕН | 1–3 мес | ~40% | RPC 2006/2007 через PDB+MITM; кэш всего мира+Strings DB |
| 8 | NPCSvr64 | ✅ КРИТИЧЕН | НЕ переписываем | ~5% | симуляция мира (World::MoveNPC, abyss-цикл, 15 ГБ); только watchdog/ночной рестарт (есть) |
| 9 | Server64/MainServer | ✅ КРИТИЧЕН (ядро игры) | НЕ переписываем | ~0% | только Ghidra-патчи: матчмейкер #108 (JZ→JNZ), манастоны #111, date #180 + обвязка |
| − 10 | Petition/ShopAgent/ChannelChat | − НЕ КРИТИЧНЫ | дни-недели | ~30% | ILSpy = готовое ТЗ; мёртвые — просто не запускать; луперы лечит Ghidra-silence |
| − 11 | GMServer | − НЕ КРИТИЧЕН | — | — | GM = builder в SQL; gm.exe = шифровальщик connstr; веб-БД REF58L если захочется |
| − 12 | NPRelayServer | − НЕ КРИТИЧЕН | — | — | мёртвый (не биндит портов), задачи DISABLE |
| − 13 | RankingServer | − НЕ КРИТИЧЕН | — | — | .NET веб-рейтинг; config.xml в ките нет |
| ❌ 14 | PAServer (PortalAuth) | ❌ НЕ НУЖЕН СОВСЕМ | — | — | **СКИП НАВСЕГДА** (docs/pa-research-20261006.md): релей payStat веб-портала, auth-path его не трогает; если однажды понадобится — python/docker эмулятор из треда L2 1205208 |

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
| Код капчи | nextgen/aion-captcha/ (+ README, CAPTCHA-STATUS-SNAPSHOT.md); прод: D:\SAION\aion-captcha\ |
| CAPTCHA: протокол/доки | docs/captcha-recon-20261005.md, docs/captcha-protocol-20261005.md, docs/session-20261005-captcha.md |
| Оператор aion-op | nextgen/aion-op/; прод: C:\aionop\ (UI 127.0.0.1:10200 через ssh -L) |
| Бинари+PDB | локально ~/STELGEN/projects/aion_rev_2026-10-05/artifacts/; гигантские PDB на VM |
| Дизasmы | /tmp/logsrv.asm (LogServer64), /tmp/logpub.txt; pdbpub.py — ~/STELGEN/tmp/ |
| Ресёрч SM_INIT/классика LS | docs/authgate-research-20261005.md; прогон: tools/analysis/diag6_sminit_probe.py |
| Дизasm 0x407d50/0x417a20/ключи | docs/session-20261005-authgate-disasm407d50.md (ночь-4) |
| Сессия 05.10 (этот чат) | docs/session-20261005-logd-final.md |
| Память | STELGEN/projects/aion_server_2026-10-02 (+fixes/) — читать в начале каждого чата |
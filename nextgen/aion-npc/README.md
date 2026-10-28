# aion-npc — перепись NPCSvr64 (мир-симуляция) — ДЕПРИОРИТЕТ, полный сервер бескомпромиссно

> 🔬 **~15% · ДЕПРИОРИТЕТ, НО В ПЛАНЕ** (политика юзера: переписываем ВСЕ бескомпромиссно — низкий приоритет ≠ отмена): R0 ✅ 08.10 (ресёрч+эталоны ×7 клонированы), R1 ⏸ приостановлен юзером до прогресса соседей; старт = после authd/accache/cache/ic закрытия. Тактический промежуточный слой — [../aion-binpatch/](../aion-binpatch/) (Ghidra-патчи ориг-бинаря).
> Запуск чата: `WORKFLOW: npc` ([../WORKFLOW.md](../WORKFLOW.md)). Стандарты S1–S12: [../README.md](../README.md) §4.

## 🧭 Граф кода (S13)

- Код компонента навигируется AST-графом `graphify`, НЕ чтением всего пакета: `query`/`explain`/`path`/`god-nodes` — правила и команды: [../WORKFLOW.md §6.1](../WORKFLOW.md).
- ПЕРЕД правкой символа — `graphify affected <имя>` (кто зависит = что сломается); ПОСЛЕ правки — `graphify update .` в этой папке; `graphify-out/` = кэш (в гит не кладётся).
- Граф = навигатор, не арбитр: семантика протокола = канон этого README + live capture.


## 📊 Статус и фазы

| Фаза | Статус |
|---|---|
| R-сурсы: чужие киты (2.7 PTS распакован на VM `D:\SAION\downloads\rz\unpacked\2.7\` с NPCSvr64), PDB 129МБ (publics 23501), дизasm-заготовки, Java-эталоны собраны ([RESEARCH.md](RESEARCH.md): Mobius7.7/beyond-aion4.8/AL7.8/AG7.8 — 4 клона в `../../reference/`) | ✅ частично (эталоны есть, R0-словари не извлечены) |
| R0: разведка | ✅ 08.10: каналы live + ScriptDLL-тип (C++ классы) + PDB-реестры (4630 класса) + эталонный словарь AI (111 типов) + строк-дифф 2.7/7.7 — [docs/r0-vm-inventory-20261008.md](docs/r0-vm-inventory-20261008.md) |
| R1: wire capture (pktmon — рестарт дорогой!) + карта 2002-протокола NPC↔Server64 | ⬜ |
| R2–R3: Go `nextgen/aion-npc` MVP: мир-цикл абстракция + NPC-спавны из XML/ScriptDLL + интеграция с aion-cache/aion-main | ⬜ |
| R4 A/B → R5 свитч (пара NPC+MAIN!) → R6 наблюдение | ⬜ |

## 📟 Канон (известное сейчас)

| Факт | Источник |
|---|---|
| NPCSvr64 грузится 10–15 мин, RAM ~15 ГБ при загрузке → **падает до ~1 ГБ после спавна** (норма — выгрузка загрузочных структур); маркер готовности «NPC Server Started» в .err | live |
| Утечка Abyss ~600k блоков/сессия → ночной рестарт пары NPC+MAIN (op умеет restart_pair); смерть Server64 каскадно убивает NPCSvr (graceful, leak-дампы 13МБ в .err) | live |
| Пара NPC+MAIN = ЕДИНАЯ единица управления (рестарт только парой; окно загрузки = рестарты заблокированы) | op |
| abyss-цикл 60с (в логах CacheD «abyss-цикл 60с» — R6-критерий cache-свитча) | RESEARCH cache |
| **live 08.10**: NPCSvr64 ↔ CacheD по **2006** (общий пул); CacheD слушает 2006/2007/**2009**, клиентов на 2009 НЕТ; 8/8 conns :2002 = NPCSvr→Server64; Server64 слушает :2012 (⏳); CacheD→IC по 2305 | netstat live ([r0-vm-inventory](docs/r0-vm-inventory-20261008.md)) |
| **wire 2002 = 148 хендлеров из PDB Server64** (cros-пульс aion-main R3.7): ServerToNPCServer (86: SendCreateMonster(2)/SendDespawn/SendMagic_Summon(Trap/Servant)/SendDoorState/SendWeather/SendTeleport/EncodeAbyssInfo/EncodeCreateDynamicWorld/Quest*/Duel/ValidMemberList...), NPCServerToServer (62: DecodeMove2(PointFloat)/DecodeAttack/DecodeGiveSkill/DecodeLoot/DecodeAbyssBossDie/DecodePet*/Quest*/Fly...); семантика в мангл-сигнатурах (PointFloat/EmotionType/KillerInfo/QuestShareInfoMultiple) — реестр: [docs/npcsvr-2002-handlers-raw.txt](docs/npcsvr-2002-handlers-raw.txt) | PDB Server64 publics (74164) |
| ScriptDLL64 = **C++ классы скриптов NPC** (V/U/I-интерфейсы: VIScriptDLL, VIAIScriptNpc::UpdateNpcPatterns), NPCSvr = ядро (спавн/таймеры/движение/Abyss); PDB NPCSvr64 с символами (4630 класса/7051 пар — [docs/](docs/npcsvr64-mangled-classes.txt)) | PDB live 08.10 |
| Кросс-версионный эталон: 2.7-кит NPCSvr64.exe (28МБ Server64/19.9МБ CacheD64/NPCSvr64) на VM | ic-RESEARCH §3 |

## 🚧 Блокеры

- Самый тяжёлый компонент трека B: симуляция мира + ScriptDLL-совместимость. Стартовать только после закрытия authd/accache/cache/ic.
- Рестарты дорогие (10–15 мин) — capture только pktmon.

## ⏭️ Следующий шаг

`WORKFLOW: npc` → R1 ⏸️ (приостановлен юзером до прогресса соседей; каркас wire готов — [r1-док](docs/r1-capture-2002-20261008.md)); рабочий трек = R2 дизasm dispatch по сигнатурам (блокеров нет) + словари из 7 клонов ([RESEARCH §6](RESEARCH.md) — GiGatR00n 4.7.5 = 578 handlers/471 AI-типов, Nexus27 = 139/133). Промпт: [PROMPT.md](PROMPT.md).

## 📦 Артефакты

| Что | Где |
|---|---|
| Промпт | [PROMPT.md](PROMPT.md) |
| Ресёрч Java-эталонов | [RESEARCH.md](RESEARCH.md) — карта мимикрии ai2/spawnengine→наш мир; клоны вне гита |
| Словарь AI-типов (R0) | [docs/etalon-ai-dict-20261008.md](docs/etalon-ai-dict-20261008.md) + [docs/ai-dict-all-20261008.json](docs/ai-dict-all-20261008.json) — 111 типов/143 handler'а (7.7), 0 «XML без handler'а»; think event-driven; walker_id/walker_index |
| R0 VM-инвентарь | [docs/r0-vm-inventory-20261008.md](docs/r0-vm-inventory-20261008.md) — каналы live, MD5 бинарей, строки ObjectClass::Set/AI_name, PDB-карта подсистем; PDB-реестры: [npcsvr64-mangled-classes.txt](docs/npcsvr64-mangled-classes.txt), [npcsvr64-class-method-pairs.txt](docs/npcsvr64-class-method-pairs.txt); бинари: `~/STELGEN/tmp/r0/` |
| PDB 129МБ | локально `~/STELGEN/projects/aion_rev_2026-10-05/artifacts/pdb-big/` (+ дизasm-заготовки npcs.asm в tmp) |
| 2.7-кит (NPCSvr64 старой версии) | VM `D:\SAION\downloads\rz\unpacked\2.7\` |
| Ориг | VM `D:\AION_LIVE_SERVER\` (задача AionNPC; пары с AionMain) |
| Креды/доступы | VM `D:\SAION\creds\` |

## 📜 Логи

Стандарт S3: raw-first io-дампы + fork C>/O>/N> ([../LOGGING-SPEC.md](../LOGGING-SPEC.md)); ориг-логи NPCSvr (.err) тейлерит op — наша перепись пишет .err-совместимый формат или переносит тейлеры.

## 📊 Сосед узнал (08.10, чат aion-main R3.7)
- **CM_MOVE структура live**: payload@0 = **OID u32** (capture `0x443E29B5`), далее X/Y f32 — пригодится твоему NPC-миру.
- **OID игрока персоно-агностичен в Server64-раскладках** (13 прод-payload'ов, тест-защищено): NC привязывает игрока соединением — твой мир-стейт может делать так же.
- **2002 = Shared-канон**: PDB Server64 классы `ServerToNPCServer(86)/NPCServerToServer(62)` — тот же паттерн, что ServerToDb(2006): твои хендлеры = зеркальные к cached-ref картам.
- Server64-тень aion-main готова принять 2002-коннекты на R4.1 (fork-стенд) — синхронизируем wire-факты по мере твоего R1.

## 📊 Сосед узнал (23.10, чат cached R4.6.2/R5 #4): NPCSvr-канал 2006 разобран

- **NPCSvr = ВТОРОЙ клиент 2006** (не Server64): маркеры те же 0xAE/0xAF; различение по первому кадру (Server64 шлёт RQ_VERSION 215, NPCSvr — cmd=170-поток). Твой фингерпринт готов.
- **NPCSvr-флоу к CacheD** (Т27 + R4.7.1 capture): cmd=170 ×N UTF-16 `tl_dp_id:`/`tl_abyss_id:` строки; cmd=200 `[SAVE]v01/ctrl`; cmd=179/224/106/299/115 fire-and-forget БЕЗ ответов (канон).
- **W30-бёрст разобран и реплеится** (R4.6.2): C277→O187 ×150 (1:1), C62→пачка 45 (40×O18), 179/224 fire-and-forget — 228 O-кадров байт-в-байт (`npcsrv_w30.go`); цепочки NPCSvr-init в тест-эталонах `nextgen/aion-cached/testdata/npcsrv_*.bin` (201→28 SAME байт-в-байт).
- **R5 свитч #4 (23.10)**: при нашем кеше на 2006 NPCSvr НЕ дошёл до «NPC Server Started» — гипотеза: NPCSvr валидирует ответы кеша до сетевой фазы (или падает на SQL/файлах). Твой R1 (рестарт пары + pktmon) снимет и NPCSvr-старт-путь — он общий с cached-мир-init.
- **2009 npcDb = 0 пакетов при старте мира** (pktmon 23.10) — канал спит; слепая зона НЕ там, приоритет ниже ожидавшегося.
- Мир сейчас 0/8 (пара остановлена) — твой R1 capture возможен при следующем подъёме пары (по «го»).

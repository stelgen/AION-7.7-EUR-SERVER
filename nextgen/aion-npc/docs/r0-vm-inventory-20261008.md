# 🛰 R0 aion-npc — инвентарь VM/бинарей/каналов + PDB-реестры (08.10.2026)

> Канал: Agent API (S12, `aion-agent.sh`). Бинари локально: `~/STELGEN/tmp/r0/` (вне гита). PDB-реестры — в этом доке-каталоге (гит). Выполнено: netstat-инвентарь live, MD5-инвентарь, скачивание бинарей, strings-поиск AI-типов, экстракт PDB.

## 1. Каналы live (netstat 08.10, VM 192.168.0.125)

| Процесс | PID | LISTEN | Established (наружу) |
|---|---|---|---|
| **Server64** | 3580 | **2002**, **2012**(!), 7777 | →2006 (CacheD), →2220 (ACS), →22206 (CAPTCHA), →2005 ×9 (IC), →2104 (L2Authd), →2051 (log) |
| **NPCSvr64** | 6568 (start 05:06:44, WS 1094МБ) | — | **→2002 ×8** (Server64), **→2006** (CacheD), →2051 (log) |
| **CacheD64** | 6640 | **2006, 2007, 2009** | →1433 ×~100 (MSSQL), **→2305** (IC), клиенты: Server64+NPCSvr64 по 2006 |

📌 **Канон-факты (live)**:
1. **8/8 conns на :2002 (критерий «мир собран» op) = 8 соединений NPCSvr64→Server64** — это NPC-канал, НЕ игроки.
2. **2009-гипотеза отхлопнута**: CacheD слушает 2009, но NPCSvr64 подключается к CacheD по **2006** (общий пул, как Server64); клиентов на 2009 в бою НЕТ. Ранее «NPCSvr ↔ CacheD :2009» из доков — ❌ исправлено.
3. **Новый порт Server64 :2012** (LISTEN, клиент не виден) — назначение неизвестно, ⏳.
4. CacheD→IC подтверждён по **2305** (не только 2005).
5. NPCSvr64 не слушает TCP (только клиент).

## 2. Бинари (MD5, сверены при скачивании)

| Файл | Размер | Дата | MD5 |
|---|---|---|---|
| NPCSvr64.exe (7.7 ориг, `D:\AION_LIVE_SERVER\NPCServer\`) | 24 991 744 | 2020-06-09 | ce6bee5ddc65aa46c1f37011669694ee |
| ScriptDLL64.dll (7.7 NPCServer = MainServer, одинаковый) | 87 731 200 | 2020-06-09 | e7016eaeb4fab3936123c0efac006075 |
| Server64.exe (7.7, патченый 2024?) | 45 499 392 | 2024-10-31 | c515730263c2330c003b5d2d7ae1aca8 |
| CacheD64.exe (7.7) | 22 526 464 | 2020-06-09 | 15e213947931fbd0bf346c952ff236fe |
| NPCSvr64.exe (2.7-кит) | 22 369 280 | 2011-12-22 | 03534cb4e92ce6d492c2b4de2242ffc8 |
| ScriptDLL64.dll (2.7-кит, один на Main+NPC) | 15 636 480 | 2011-12-22 | da27309c7288d589bff0b02d71c1d945 |

Локальные копии: `~/STELGEN/tmp/r0/{npcs77.exe,npcs27.exe,scriptdll77.dll,scriptdll27.dll}`. Локальный PDB: `~/STELGEN/projects/aion_rev_2026-10-05/artifacts/pdb-big/AION_LIVE_SERVER/NPCServer/NPCSvr64.pdb` (129МБ, Jun 2020).

## 3. Строки бинарей (эталонный словарь применили — docs/etalon-ai-dict-20261008.md)

- **NPCSvr64.exe (7.7)**: пул UTF-16LE `ObjectClass::Set(), <атрибут>` — парсер XML-шаблонов читает: `AI_name`, `Quest_AI_name`, `PET_AI_name`, `idle_name` (+ ошибки `Undefined AI_Name %s, %s`, `Undefined Pet_AI_Name`, `Undefined Quest_AI_Name`, `Undefined idle_name`), статы `bound_radius/sensory_angle/attack_delay/critical/magical_critical/npc_type/hpgauge_level/altitude/...` — **те же поля, что npc_templates.xml Java-эталонов!**
- Эталонные AI-имена, найденные как слова в NPCSvr64: `aggressive, dummy, flag, general, portal, resurrect, skillarea, trap` (8/111; `artifact/book/buffer/blessed` — ложные попадания item-строк). AI-строки НЕ одним пулом: рассеяны по скрипт-контенту.
- **ScriptDLL77.dll**: алфавитный строк-пул атрибутов: `ai_name, ai_type, ai_pattern(!), summon_class/summon_count/summoned, effect4_reserved*, advancement_rate*, sub_weather_*, protect_item*` — ScriptDLL = пул дескрипторов мира.
- **2.7 vs 7.7**: ядро строк-парсера идентично (`ObjectClass::Set(), aggressive` в обоих; 2.7 без skillarea).

## 4. PDB NPCSvr64 — МАНГЛИРОВАННЫЕ СИМВОЛЫ ЕСТЬ (экстракт в этом каталоге)

Файлы: `npcsvr64-mangled-classes.txt` (4630 классов), `npcsvr64-class-method-pairs.txt` (7051 class::method). Экстрактор: strings-regex по PDB (см. /tmp/pdb_extract.py в сессии).

**Карта подсистем NPCSvr64 (префиксы V/U/I — ScriptDLL-интерфейсные маркеры):**
| Подсистема | Символы |
|---|---|
| Спавн-система | VNPCMaker, VSpawnGroupMaker, VSpawnGroupListMaker, VSpawnPoint, VSpawnTimer(+Data), VPointListSpawnArea, VOverseasNpcMakerMgr, VEventSpawnParam, UDynamicSpawnParam, USpawnData, VIDespawnEvent(+Imp), VISpawnResultEvent, VDespawnableFieldObj, VNpcReturnSpawnPointState |
| **ScriptDLL-интерфейс** | VIScriptDLL, VIScriptMain, VNpcScriptMgr, VIAIScriptNpc (+UpdateNpcPatterns/UpdateHandlerVTable), UAIScriptNpcClassData, UNpcAIScriptData, VIQuestScriptNpc, VIOneQuestScriptNpc, Viless_npcscript, UHouseScript |
| Таймеры | IEventTimer::TimerExpired/UpdateTime/UpdateEvent, CIOObject::TimerExpired(+WithParam), VINpcIdleTimerEvent(+Imp), VIBattleTimerEvent, VCQuestTimerMgr, UQuestTimerData, AITimeConditionMgr::TimerExpired, TryRepeatHelper::TimerExpired |
| **Abyss (доказано в NPCSvr!)** | класс Abyss (+UpdateAbyssInfo), **AbyssMgr::UpdateAbyssPvPStatus/UpdateServerAbyssInfo/UpdateServerAbyssStrengthInfo**, AbyssLevelGroup::Load/FindData/FindGroupById/Parse*, AbyssZoneInfo, AbyssOwnerRaceInfo, AbyssLevelStrengthRatio, AbyssNpcMakerStorage, g_abyssZoneInfoDb; путь исходника: `D:\_build\src\SERVER\NPCServer\x64\Release_VS2013\` |
| ObjectClass (XML-шаблоны) | ObjectClass::Set(...), GetAIScript, GetQuestScript, DumpAIDebug/SetDumpAIDebug, ChaseFlying, MoveSpeedNormalRun/CombatRun/Walk, GetInvisibleDetectLevel... |
| NPC-движение/AI-состояния | NPC::UpdateGotoWayPointState, NPC::UpdatePathFindAlgorithType, NPCGiantOrd::UpdateAnimation/AttackMode/AttackState, IdleActionMgr::UpdateMoveActionEndTime |

📌 **Архитектурный вывод**: ScriptDLL64 = **C++ классы скриптов NPC** (не XML-скрипты), NPCSvr64 дёргает их через V/U/I-интерфейсы; в NPCSvr64 — ядро мира: спавны, таймеры (idle/battle), движение (WayPoint/PathFind), Abyss (Mgr+Level+Zone+OwnerRace). Прямой аналог Java-эталонов: ai2-engine+spawnengine+world (NPCSvr) vs ai2-handlers (ScriptDLL).

## 5. Следующий шаг R0→R1

- Дизasm точки входа ScriptDLL (VIScriptDLL / VIAIScriptNpc::UpdateNpcPatterns) в Ghidra — карта Script-API.
- pktmon capture 2002 (8 NPC-коннектов) + 2006 при рестарте пары (по «го»).
- Server64 :2012 — кто подключается (нет клиентов в бою; вероятно сиделка/2.7-легаси).
- AITimeConditionMgr/IdleActionMgr — кандидаты «периодики» (abyss-цикл 60с: AbyssMgr::Update* — дизasm R2).
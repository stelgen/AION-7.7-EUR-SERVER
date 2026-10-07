# 🔬 RESEARCH aion-npc — Java-эталоны для мир-симуляции NPCSvr64 (08.10.2026)

> Политика [WORKFLOW §5.2](../WORKFLOW.md) (СУРСЫ-ПРИОРИТЕТ, мимикрия-first): перед дизasm-ом собраны и **склонированы локально** все публичные Java-эталоны Aion-эмуляторов. NC-сурсов (PTS NPCSvr64/Server64) в публичном мире НЕТ — перепроверено ранее (cached-ref, ic-RESEARCH). Клоны лежат в `../../reference/` (вне гита, диск-only).

## 1. Реестр эталонов (все склонированы 08.10, --depth 1)

| Эталон | Версия | Локация (`../../reference/`) | Лицензия | Статус | Размер |
|---|---|---|---|---|---|
| Mobius_AionEmu | **7.7** (Java 25, Ant, один проект) | `Mobius_AionEmu/` | GPL-3.0 | ✅ живой (60 коммитов; база = последний коммит AionGermany 7.7; клиент 7.7 Internet Archive) | 331М |
| aion-server (beyond-aion) | **4.8** (Maven, fix-first) | `aion-server/` | GPL-3.0 | ✅ живой — эталон КАЧЕСТВА: months-without-restart, фиксы geo/leak/AI-events | 464М |
| AionLightning (ZON3DEV) | **7.8** (+ветки 1.9/4.6/5.8) | `AionLightning/` (ветка 7.8.0) | GPL-3.0 | ✅ живой (Todo: пакеты/статы/minions/cubics) | 1.1G |
| aion-germany (AionGermany) | **7.8** + `AL-Game-5.8` | `aion-germany/` | GPL-3.0 | ⚠️ «Development Ended» (662 коммита; прародитель Mobius) | 981М |

**Генеалогия**: AionEmu (2009) → Aion Lightning (1.9→4.6→4.7.5) → beyond-aion 4.8 (fix-first форк AL) → AionGermany (5.8→7.5→7.8, ended) → Mobius 7.7 (= AG 7.7, вычищен) → ZON3DEV AL 7.8 (живой форк наследия AG). RZ 06.2026: Nexus Connect портировал falke-2.7 (AL-основу) на JDK25+Maven — репо за RZ-логином; Mantios предупреждает о сломанном core ⇒ кросс-версионный дифф 2.7 ведём по НАШЕМУ 2.7-киту (VM) + AL-веткам 1.9/4.6/5.8, не по Nexus.

## 2. Карта мимикрии: подсистемы Java-эталонов → наша мир-симуляция (Go)

| Java-подсистема | Объём | Состав | Маппинг в aion-npc (Go) |
|---|---|---|---|
| `ai2/` (AL/AG/Mobius) / `ai/` (beyond-aion) | 53 / 39 файлов | AI2Engine, NpcAI2, AIState/AISubState, AttackIntention; handlers: Activate/Aggro/Attack/Creature/Died/Follow/Freeze/Move/Returning/Shout/Spawn/Talk/Target/Think (+AL7.8 `SimpleAbyssGuardHandler`!) | наш `ai/`: конечный автомат NPC (spawned→attacking→returning→dying), agro-логика, think-циклы |
| `spawnengine/` | 15 файлов | SpawnEngine, VisibleObjectSpawner, TemporarySpawnEngine, WalkerGroup/WalkerFormations (патрули), ClusteredNpc, StaticDoor/StaticObject/Conquest | спавны из XML (данные клиента), патрули, временные спавны |
| `world/` | пакеты | контейнеры видимости, map regions | наш мир-контейнер |
| `taskmanager/` | пакеты | периодические таски | abyss-цикл 60с и прочие периоды |
| `geoEngine/` | пакеты | коллизии из client-data | movement-валидация NPC |
| `dataholders/` | пакеты | npc_templates, spawns, AI-типы из XML | наши спавн-данные |

⚠️ **Архитектурная разница**: у NC мир разнесён — Server64 (мир/креатуры) ↔ NPCSvr64 (NPC-симуляция) по каналу 2002 (+:2009 npcDb-гипотеза); Java-эталоны держат ВСЁ в одном GameServer. Мимикрия = **ПОДСИСТЕМНАЯ** (ai/spawn/task/geo), не целым процессом. Протоколы клиент↔GS у эталонов (CM_/SM_) — НЕ наш внутренний 2002.

## 3. Открытия для соседей (кросс-пульс сделан)

- **aion-main**: Abyss-логика у эмуляторов живёт в NPC-AI (`SimpleAbyssGuardHandler` AL7.8) ⇒ поддерживает теорию «abyss-цикл 60с живёт в NPCSvr, не Server64».
- **aion-binpatch**: дизasm dispatch NPCSvr64 (R2) сверяем с ai2-машиной эталонов — имена `NpcAI2/AI2Actions/AggroEventHandler` ищем в PDB publics (129МБ, 23501 publics).
- **aion-cache**: прямого касания нет; 2009/npcDb-гипотеза не тронута.

## 4. Недоступно / кто докачивает

- Nexus Connect 2.7 (JDK25): репо за RZ-логином — некритично (см. §1).
- Юзер/VM: клиент 7.7 (Internet Archive) + geo v7.3 (MEGA) для Mobius; AL-ветки 1.9/4.6/5.8 (`git clone -b X.9.0`) при надобности кросс-диффа.

## 5. В R0 из этого ресёрча

- Извлечь эталонные словари: AI-типы ↔ npc_templates.xml ↔ spawn_map.xml, think-интервалы, WalkerGroup-модель → заготовка дизasm-dispatch карты NPCSvr64.
- beyond-aion `ai/` = упрощённая/очищенная версия той же машины (+HpPhases, очередь скиллов) — второй взгляд на ту же модель.
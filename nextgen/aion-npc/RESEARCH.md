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

## 5. Результат R0 (08.10): эталонный словарь AI — ГОТОВ

Словарь собран по всем 4 клонам (+AG 5.8): [docs/etalon-ai-dict-20261008.md](docs/etalon-ai-dict-20261008.md) (+JSON + экстрактор tools/). Ключевое: 111 AI-типов ↔ 143 handler-класса (7.7), 0 пробелов; AL 7.8 ≡ AG 7.8; think event-driven (не в данных); walker_id/walker_index в спавнах; `simple_abyssguard` в 4.8 = 859 NPC, в 7.7 — движковый. Диффы: 5.8-only 463 (свёрнуты), 4.8-only 367, 7.7-only 5.
## 6. 🌊 Артефакт-ресёрч волна 2 (08.10, политика СУРСЫ-ПРИОРИТЕТ: NPC-реализации любых версий)

**Скачано (reference/, вне гита):**
| Эталон | Версия | AI-handler'ов | AI-типов XML | NPC-шаблонов | Примечание |
|---|---|---|---|---|---|
| GiGatR00n Rework (`gigatr00n-475/`, AC-*) | 4.7.5 | **578** | **471** | 59231 | САМАЯ богатая AI-база; AionCore-линейка |
| NexusConnect (`nexus27/`, Maven) | 2.7 | 139 | 133 | 29703 | JDK25-модернизация falke-2.7; кросс-версия к нашему 2.7-киту |

**Карта сходства (проверено эмпирически)**: модель AI-мира ЕДИНА от 2.7 до 7.8 — `ai2/` + `spawnengine/` + `data/scripts/system/handlers/ai/` + `@AIName` + npc_templates `ai=` — консервативное ядро при мажорных разницах контента (133→471→432→111 типов). ⇒ NPCSvr64 обязан иметь то же трио: реестр AI-имен (уже нашли: AI_name + Undefined AI_Name) + event-driven движок (ObjectClass::Set-пул) + спавн-таблицы (VSpawn*). Сходство ожидаемо высокое — подтверждено.

**Не дотянулся (юзеру):**
- **Encom 7.5 (Vieka-сборка, «web and source files»)**: GDrive `https://drive.google.com/file/d/1t9tykxq6-j8L7k0xD1tguiXADfbtyNtQ/` (perm denied из песочницы) или MEGA из треда RZ 1196933. Отдельная Java-ветка 7.5 (Encom leak, тест-файлы: миньоны частично, базы багги). Кросс-проверка против 7.7 бесценна.
- **Encom-сурс Robson26** (иногда правит сам) и **SVN Voidstar** (7.5–7.9) — доступ по PM на RZ.
- **Aion-Extreme 2.1/2.5** (первая полная EMU, 2010, прародитель AL) — RZ-аттачи f587 тред 759356; у юзера RZ-логин.
- savior — не публичен (pada8801 #104: постил в L2-тред ранее).

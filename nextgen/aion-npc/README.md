# aion-npc — перепись NPCSvr64 (мир-симуляция) — ДЕПРИОРИТЕТ, полный сервер бескомпромиссно

> ⬜ **НЕ НАЧАТ, ДЕПРИОРИТЕТ** (политика юзера: переписываем ВСЕ компоненты бескомпромиссно — низкий приоритет ≠ отмена; старт = после authd/accache/cache/ic закрытия). Тактический промежуточный слой — [../aion-binpatch/](../aion-binpatch/) (Ghidra-патчи ориг-бинаря).
> Запуск чата: `WORKFLOW: npc` ([../WORKFLOW.md](../WORKFLOW.md)). Стандарты S1–S12: [../README.md](../README.md) §4.

## 📊 Статус и фазы

| Фаза | Статус |
|---|---|
| R-сурсы: чужие киты (2.7 PTS распакован на VM `D:\SAION\downloads\rz\unpacked\2.7\` с NPCSvr64), PDB 129МБ (publics 23501), дизasm-заготовки, Java-эталоны собраны ([RESEARCH.md](RESEARCH.md): Mobius7.7/beyond-aion4.8/AL7.8/AG7.8 — 4 клона в `../../reference/`) | ✅ частично (эталоны есть, R0-словари не извлечены) |
| R0: разведка (ScriptDLL64-скрипты, каналы: Server64 2002, CacheD npcDb-канал :2009, лог 2051, World::MoveNPC/abyss-цикл 60с) | 🟡 частично: ✅ эталонный словарь AI-типов (111 типов ↔ классы ↔ частоты по 5 деревьям: [docs/etalon-ai-dict-20261008.md](docs/etalon-ai-dict-20261008.md), экстрактор [tools/](tools/extract_ai_dict.py)); ⬜ каналы/ScriptDLL64/2.7-дифф |
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
| 2009-канал CacheD64 = NPC-DB гипотеза (аналог L2 npcDbHandlers — карта каналов в cached-ref) | cached-ref |
| ScriptDLL64 = скрипты NPC (DLL) — часть мира; перепись включает совместимый лоадер | PLAN |
| Кросс-версионный эталон: 2.7-кит NPCSvr64.exe (28МБ Server64/19.9МБ CacheD64/NPCSvr64) на VM | ic-RESEARCH §3 |

## 🚧 Блокеры

- Самый тяжёлый компонент трека B: симуляция мира + ScriptDLL-совместимость. Стартовать только после закрытия authd/accache/cache/ic.
- Рестарты дорогие (10–15 мин) — capture только pktmon.

## ⏭️ Следующий шаг

`WORKFLOW: npc` → R0: инвентарь каналов (netstat 2002/:2009), ScriptDLL64-инвентарь, 2.7-кит дифф → ROADMAP-детализация. Промпт: [PROMPT.md](PROMPT.md).

## 📦 Артефакты

| Что | Где |
|---|---|
| Промпт | [PROMPT.md](PROMPT.md) |
| Ресёрч Java-эталонов | [RESEARCH.md](RESEARCH.md) — карта мимикрии ai2/spawnengine→наш мир; клоны вне гита |
| Словарь AI-типов (R0) | [docs/etalon-ai-dict-20261008.md](docs/etalon-ai-dict-20261008.md) + [docs/ai-dict-all-20261008.json](docs/ai-dict-all-20261008.json) — 111 типов/143 handler'а (7.7), 0 «XML без handler'а»; think event-driven (AttackManager→think()); walker_id/walker_index в спавнах |
| PDB 129МБ | локально `~/STELGEN/projects/aion_rev_2026-10-05/artifacts/pdb-big/` (+ дизasm-заготовки npcs.asm в tmp) |
| 2.7-кит (NPCSvr64 старой версии) | VM `D:\SAION\downloads\rz\unpacked\2.7\` |
| Ориг | VM `D:\AION_LIVE_SERVER\` (задача AionNPC; пары с AionMain) |
| Креды/доступы | VM `D:\SAION\creds\` |

## 📜 Логи

Стандарт S3: raw-first io-дампы + fork C>/O>/N> ([../LOGGING-SPEC.md](../LOGGING-SPEC.md)); ориг-логи NPCSvr (.err) тейлерит op — наша перепись пишет .err-совместимый формат или переносит тейлеры.
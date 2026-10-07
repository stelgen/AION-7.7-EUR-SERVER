# 🛰 ROADMAP aion-cache — CacheD64 → свой RAM-кэш (Go)

> Обновлено 08.10. Полный ресёрч: [../CACHE-RESEARCH.md](RESEARCH.md). Запуск чата: `WORKFLOW: cache`.
> Метод переписи отработан 4 раза (logd/captcha/gate/accache-каркас); здесь объём 8× ACS.

## Фазы

| Фаза | Что | Результат | Статус | Оценка |
|---|---|---|---|---|
| R0 | Ресёрч: PDB/словари/DB-контракт/логи/веб-передний край (НОЛЬ публичного) | CACHE-RESEARCH.md + cached-ref/ | ✅ 08.10 | — |
| R1 | pktmon filter port 2006 на VM (без прокси — рестарт Server64 дорогой!) + разбор готовых log/*.log (RPC-строки с параметрами) | wire-фрейм 2006 подтверждён (гипотеза: [u16 self-len][op][payload][2Б csum], L2-эволюция) | ⬜ | 1–2 дня |
| R2 | Сверка dispatch-таблиц + семантика payload'ов (метод accache: ctor-таблица + мангл-сигнатуры). **10.10: нумерация ВСЕХ 8 протоколов уже снята из log/*.profile** → [../cached-ref/profile-opcode-map.md](../cached-ref/profile-opcode-map.md) — R2 = раскладки параметров, не поиск таблиц | аннотированная dispatch-таблица → cached-ref/ | 🔶 нумерация ✅ / семантика ⬜ | 1–2 дня |
| R3 | Go `nextgen/aion-cache`: proto + RAM MapStore (user/item/guild/vendor/…) + DB-слой {call aion_* 781} + Admin-канал (GQ/GP) + Log-клиент + IC-клиент; **MVP = read-путь (char login/item load) + write-транзит SQL** | каркас, тесты зелёные | ⬜ | 1–2 нед |
| R4 | A/B: pktmon-сверка нашего vs ориг (байт-в-байт по наблюдаемым RPC) | VERDICT=SAME | ⬜ | 1–2 дня |
| R5 | Свитч по «го»: common.xml cacheServerPort → наш; откат одной правкой | в бою | ⬜ | 1 день |
| R6 | Наблюдение 24ч (abyss-цикл 60с, SQL-дифф, логины) | финал+память | ⬜ | 1 день |

**Суммарно MVP: 2–4 недели. Шанс ~85%.**

## 🧪 Журнал теорий (фиксация на момент; ✅/❌ по проверке)

| Дата | Теория | Проверка | Статус |
|---|---|---|---|
| 08.10 | Wire 2006 = эволюция L2 CacheD C1: `[u16 self-len LE][op][payload][2Б csum]` + rolling-XOR ключ | R1 pktmon + L2-референс | ⏳ |
| 08.10 | 2009 = NPC-DB канал (NPCSvr64 клиент), аналог L2 npcDbHandlers | netstat live 08.10: NPCSvr ходит в CacheD по 2006 (общий пул с Server64), на 2009 клиентов НЕТ | ❌ → леджер |
| 10.10 | 2009-канал не профилируется DBProfiler'ом (в .profile нет его секции) — служебный/спящий канал; клиент неизвестен | дизasm CreateListener-цепочки (R2) | ⏳ |
| 10.10 | Словарь опкодов NC append-only (5.8 ⊂ 7.7, 0 удалений/переименований; +31 RQ/+19 RP/+1 GP) → dispatch = superset, нумерация стабильна между версиями | дифф strings 5.8 vs 7.7 → [../cached-ref/diff-dict-58-77.md](../cached-ref/diff-dict-58-77.md) | ✅ канон |
| 08.10 | RAM-модели CUser/CItem/CPledge/CTransaction/CWarehouse (L2 C1) структурно соответствуют нашим доменам | R2 дизasm | ⏳ |

## Леджер решений (исправленное удалено из канона)

- **2009 ≠ NPC-DB/NPCSvr64** (10.10): думали «третий листенер = npcDb-канал NPCSvr по аналогии L2 C1» → live-netstat 08.10 (чат aion-npc): NPCSvr64 коннектится к CacheD по 2006, 2009 работает без единого клиента; в .profile (DBProfiler) секции для 2009-протокола нет. Клиент 2009 — ⏳ (кандидаты: logd-класс LogClientSocket? админ-инструменты?). Доказательство: netstat + отсутствие секции в профайлере.

## Порядок в R3 (MVP-политика юзера)

1. Read-путь: RQ_CHARACTER_LIST → RP_CHARACTER_LIST + пачка RP_LOAD_* (WAREHOUSE/ITEMS/CLIENT_SETTINGS/SKILL/PETS/… — кластер ~10 пакетов на логин, подтверждён профилем нагрузки 02–07.10).
2. Write-транзит: мутации → SQL (781 proc) — транзитом байт-в-байт, без семантики. Топ: RQ_U_UPDATE_COUNT/COUNT_RELATIVE/U_SAVE/ADD_SKILL/UPDATE_QUEST + фоновый abyss (RQ_UPDATE_ABYSS_INFO/RQ_SET_SERVER_INFO/RQ_ALERT_MSG = 90%+ объёма).
3. Admin/IC/Log каналы — после read/write.
4. 2009-канал — после подтверждения роли.

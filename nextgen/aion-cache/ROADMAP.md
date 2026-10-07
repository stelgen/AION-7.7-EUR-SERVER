# 🛰 ROADMAP aion-cache — CacheD64 → свой RAM-кэш (Go)

> Обновлено 08.10. Полный ресёрч: [../CACHE-RESEARCH.md](../CACHE-RESEARCH.md). Запуск чата: `WORKFLOW: cache`.
> Метод переписи отработан 4 раза (logd/captcha/gate/accache-каркас); здесь объём 8× ACS.

## Фазы

| Фаза | Что | Результат | Статус | Оценка |
|---|---|---|---|---|
| R0 | Ресёрч: PDB/словари/DB-контракт/логи/веб-передний край (НОЛЬ публичного) | CACHE-RESEARCH.md + cached-ref/ | ✅ 08.10 | — |
| R1 | pktmon filter port 2006 на VM (без прокси — рестарт Server64 дорогой!) + разбор готовых log/*.log (RPC-строки с параметрами) | wire-фрейм 2006 подтверждён (гипотеза: [u16 self-len][op][payload][2Б csum], L2-эволюция) | ⬜ | 1–2 дня |
| R2 | Дизasm dispatch по opcode-словарям (метод accache: ctor-таблица, параллельные массивы имён) | аннотированная dispatch-таблица → cached-ref/ | ⬜ | 2–4 дня |
| R3 | Go `nextgen/aion-cache`: proto + RAM MapStore (user/item/guild/vendor/…) + DB-слой {call aion_* 781} + Admin-канал (GQ/GP) + Log-клиент + IC-клиент; **MVP = read-путь (char login/item load) + write-транзит SQL** | каркас, тесты зелёные | ⬜ | 1–2 нед |
| R4 | A/B: pktmon-сверка нашего vs ориг (байт-в-байт по наблюдаемым RPC) | VERDICT=SAME | ⬜ | 1–2 дня |
| R5 | Свитч по «го»: common.xml cacheServerPort → наш; откат одной правкой | в бою | ⬜ | 1 день |
| R6 | Наблюдение 24ч (abyss-цикл 60с, SQL-дифф, логины) | финал+память | ⬜ | 1 день |

**Суммарно MVP: 2–4 недели. Шанс ~85%.**

## 🧪 Журнал теорий (фиксация на момент; ✅/❌ по проверке)

| Дата | Теория | Проверка | Статус |
|---|---|---|---|
| 08.10 | Wire 2006 = эволюция L2 CacheD C1: `[u16 self-len LE][op][payload][2Б csum]` + rolling-XOR ключ | R1 pktmon + L2-референс | ⏳ |
| 08.10 | 2009 = NPC-DB канал (NPCSvr64 клиент), аналог L2 npcDbHandlers | netstat при старте NPC + дизasm | ⏳ |
| 08.10 | RAM-модели CUser/CItem/CPledge/CTransaction/CWarehouse (L2 C1) структурно соответствуют нашим доменам | R2 дизasm | ⏳ |

## Леджер решений (исправленное удалено из канона)

- (пусто)

## Порядок в R3 (MVP-политика юзера)

1. Read-путь: FIRST_LOAD/CHAR_LOGIN (каша чара/предметов при входе) — самое частое.
2. Write-транзит: мутации → SQL (781 proc) — транзитом байт-в-байт, без семантики.
3. Admin/IC/Log каналы — после read/write.
4. 2009-канал — после подтверждения роли.

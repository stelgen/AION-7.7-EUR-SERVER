# aion-cache — перепись CacheD64 (RAM-кэш мира, :2006/2007/2009) — код НЕ начат

> 🔬 **Ресёрч R0 ЗАКРЫТ 08.10, кода НЕТ.** Самый большой компонент трека B: ~590 RPC-команд (8× ACS). Оценка MVP 2–4 нед, шанс ~85%.
> Ресёрч: [../CACHE-RESEARCH.md](RESEARCH.md) · референсы: [../cached-ref/](../cached-ref/README.md) · запуск чата: `WORKFLOW: cache` ([../WORKFLOW.md](../WORKFLOW.md)).
> Прод НЕ тронут: ориг CacheD64 жив; рестарт Server64 дорогой — capture только pktmon.

## 📊 Статус и фазы

| Фаза | Статус |
|---|---|
| R0 ресёрч (PDB 106МБ, словари, DB-контракт, 356МБ логов) | ✅ 08.10 |
| R1 wire 2006 (pktmon + разбор log/*.log) | ⬜ СЛЕДУЮЩИЙ |
| R2 дизasm dispatch по словарю | ⬜ |
| R3 Go MVP (read-путь логина чара + write-транзит SQL) | ⬜ |
| R4 A/B pktmon-сверка | ⬜ |
| R5 свитч (common.xml serverPort) по «го» | ⬜ |
| R6 наблюдение (abyss-цикл 60с) | ⬜ |

**Общий прогресс ~15%** (ресёрч = фундамент, кода 0).

## 📟 Канон (что уже известно — НЕ переснимай заново)

| Факт | Источник |
|---|---|
| CacheD64.exe = «AION CacheServer64 77.20.0604.15625» (2020-06), 22.5МБ, MD5 `15e21394` | exe |
| PDB 106МБ ПОЛНАЯ символика (14281 publics) + .map 10МБ — рядом с exe на VM; локально `aion_rev/artifacts/pdb-big/CacheD64/` | R0 |
| Роли: RAM-кэш мира + единственный ODBC-писатель в `_AionWorldNew114_rc`; мир НЕ пишет в SQL напрямую — всё через CacheD RPC + ~781 aion_* procs | R0 |
| Каналы: 2006 (мир, единственный TCP-клиент = Server64) / 2007 interactive / 2009 третий листенер (NPC-DB гипотеза) → IC 2305, лог 2051 | netstat+strings |
| Словари: RQ_ 382 / RP_ 255 (мир), GQ_ 55 / GP_ 53 (builder/GM), ACQ_ 39 / ACP_ 28 (встроенный ACS-клиент) | strings |
| L2 CacheD C1 MasterToma (cached-ref/l2-c1-cached) = единственный публичный reversed CacheD: карта каналов serverHandlers↔2006 / adminHandlers↔2007 / npcDbHandlers↔2009; L2 wire `[u16 self-len][op][payload][2Б csum]` + rolling-XOR DummyCrypt | cached-ref |
| DB-контракт: exe ссылается 789 procs, в прод-БД есть 781, 140 старых версионных отсутствуют (мультиверсионный dispatch) | R0 |
| МВП-стратегия: MVP read-путь (char login/item load) + write-транзит в SQL; остальное итерациями | RESEARCH |

## 🚧 Блокеры R1

1. Wire-фрейминг 2006 не подтверждён (гипотеза L2-эволюции — проверить pktmon).
2. Семантика RAM-мутаций и роль 2009 — дизasm/capture.
3. IC-подпротокол (клиент 2305) — после aion-ic ресёрча.

## ⏭️ Следующий шаг

`WORKFLOW: cache` → R1: `pktmon filter port 2006` на VM + разбор готовых log/*.log (171 файл, RPC-строки с параметрами — материал уже локален). Промпт: [PROMPT.md](PROMPT.md).

## 📦 Артефакты

| Что | Где |
|---|---|
| Ресёрч-док | ../CACHE-RESEARCH.md |
| RPC-карты/словари/procs-списки/strings | ../cached-ref/ (README-индекс) |
| PDB/бинари/логи 356МБ | локально `aion_rev/artifacts/pdb-big/CacheD64/`; VM `D:\AION_LIVE_SERVER\CacheServer\` |
| Креды/доступы | VM `D:\SAION\creds\` (CREDS.md) |

## 📜 Логи

📊 сосед узнал (08.10, от aion-npc R0 netstat live): CacheD64 слушает **2006/2007/2009**; NPCSvr64 подключается к CacheD по **2006** (общий пул, как Server64) — 2009 в бою БЕЗ клиентов (npcDb-гипотеза отхлопнута, см. aion-npc/ROADMAP леджер); CacheD→IC по **2305**; ~100 коннектов к MSSQL 1433.

Стандарт S3 ([../LOGGING-SPEC.md](../LOGGING-SPEC.md)): raw-first — при реализации io-дампы + fork-лог C>/O>/N>; ориг-логи CacheD (log/*.log) = эталон материала R1, трогать только read-only.

## 📊 Сосед узнал (08.10, чат aion-main R0/RES)
- PDB Server64 (74164 publics): `ServerToDb`/`DbToServer`/`ServerToDb_Update` имена 1-в-1 с RPC-картами cached-ref — канал 2006 является симметричным с серверной стороны ядра (маппинг RQ/RP продолжается из обеих сторон).

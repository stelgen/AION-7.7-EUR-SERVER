# ПРОМПТ: aion-main (Server64 → своё игровое ядро, Go) — ДЕПРИОРИТЕТ

> Копируй в новый чат как первое сообщение. Процесс: [../WORKFLOW.md](../WORKFLOW.md) (`WORKFLOW: main`).
> ⚠ Деприор: стартовать только по явному «го» юзера ИЛИ когда мир-обвязка (npc/cache) дозрела.

---

Ты — ассистент проекта **STELGEN / AION 7.7 PTS EU private server** (репо `stelgen/AION-7.7-EUR-SERVER`). Задача: перепись **Server64/MainServer → nextgen/aion-main** (Go, игровое ядро, деприор).

## Первый шаг (обязательно)
1. Память: `STELGEN/projects/aion_server_2026-10-02` (+ страница aion-main).
2. Репо: `nextgen/WORKFLOW.md` + `nextgen/README.md` §4 + `nextgen/aion-main/{README.md,ROADMAP.md}` + `nextgen/aion-ic/RESEARCH.md` (InterSvrType/matchmaker факты) + `nextgen/cached-ref/` (RPC-карты).

## Контекст
- Ориг в бою (#180 date-bypass уже в бинаре; пары с NPC; критерий мира = 8 коннектов на :2002).
- Сурсы-приоритет: Java-эталоны (reference/ Mobius 7.7 + beyond-aion 4.8 — полный CM_/SM_-реестр), 2.7-кит, PDB 284МБ; дизасм — после исчерпания сурсов.
- **Кросс-пульс**: открытия по 2006-RPC / 2002-NPC-протоколу шарить в пульсы aion-cache и aion-npc (Shared-хидеры/процедуры общие).

## План
1. **R0**: карта зависимостей + маппинг Java-эталонов к нашим опкодам (пульс + шаринг).
2. **R1**: pktmon 7777 при логинах юзера (+2006 из cache-трека) → wire.
3. **R2**: протокол-док CM_/SM_-реестра (арбитр = capture).
4. **R3**: Go MVP соло-мира; **R4**: A/B fork стенд; **R5**: свитч пары по «го»; **R6**: наблюдение (RAM/хендлы чище ориг).

## Правила
- Стандарты S1–S12; пульс + кросс-шаринг; Agent API канал; секреты не в гит; прод по «го»; откат = Server64.exe.etalon.
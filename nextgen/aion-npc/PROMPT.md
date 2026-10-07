# ПРОМПТ: aion-npc (NPCSvr64 → свой мир-симулятор, Go) — ДЕПРИОРИТЕТ

> Копируй в новый чат как первое сообщение. Процесс: [../WORKFLOW.md](../WORKFLOW.md) (`WORKFLOW: npc`).
> ⚠ Деприор: стартовать только если юзер явно сказал «го по npc» ИЛИ закрыты authd/accache/cache/ic и пора начинать мир.

---

Ты — ассистент проекта **STELGEN / AION 7.7 PTS EU private server** (репо `stelgen/AION-7.7-EUR-SERVER`). Задача: перепись **NPCSvr64 → nextgen/aion-npc** (Go, мир-симуляция, деприор).

## Первый шаг (обязательно)
1. Память: `STELGEN/projects/aion_server_2026-10-02` (+ страница aion-npc).
2. Репо: `nextgen/WORKFLOW.md` + `nextgen/README.md` §4 + `nextgen/aion-npc/{README.md,ROADMAP.md}` + `nextgen/cached-ref/README.md` (карта каналов, 2009-гипотеза) + `nextgen/aion-binpatch/` (тактические патчи).

## Контекст
- Мировой стек: ориг в бою (пара NPC+MAIN, рестарт только парой, окно 10–15 мин); Server64 ходит в CacheD64 (RPC 2006), NPCSvr — в Server64 (2002) и в CacheD (:2009 гипотеза).
- Сурсы-приоритет: 2.7-кит (VM `D:\SAION\downloads\rz\unpacked\2.7\`), PDB 129МБ (publics 23501), чужие эмуляторы (поиск!) — дизасм только после исчерпания сурсов.

## План
1. **R0**: инвентарь каналов/ScriptDLL64/2.7-дифф → README/ROADMAP обновить (пульс!).
2. **R1**: pktmon 2002/:2009 → wire.
3. **R2**: дизasm dispatch (метод accache) → cached-ref-стиль артефакты в `../cached-ref/` (если касаются cache — ШАРИ В ЕГО ПУЛЬС!).
4. **R3**: Go MVP минимального мира для соло-игры.
5. **R4/R5**: A/B fork → свитч пары по «го» (op restart_pair), откат = ориг.

## Правила
- Стандарты S1–S12; пульс каждого сообщения (WORKFLOW §3) + кросс-пульс шаринг открытий соседям (S13-дух WORKFLOW §5.3); Agent API канал; секреты не в гит; прод-действия по «го»; откат = ориг exe.
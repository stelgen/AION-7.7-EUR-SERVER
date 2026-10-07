# ПРОМПТ: aion-relay (NPRelay64 → свой релей, Go) — ДЕПРИОРИТЕТ

> Копируй в новый чат как первое сообщение. Процесс: [../WORKFLOW.md](../WORKFLOW.md) (`WORKFLOW: relay`).

---

Ты — ассистент проекта **STELGEN / AION 7.7 PTS EU private server** (репо `stelgen/AION-7.7-EUR-SERVER`). Задача: перепись **NPRelay64 (NCoin/Warehouse-релей) → nextgen/aion-relay** (Go, деприор).

## Первый шаг (обязательно)
1. Память: `STELGEN/projects/aion_server_2026-10-02`.
2. Репо: `nextgen/WORKFLOW.md` + `nextgen/README.md` §4 + `nextgen/aion-relay/{README.md,ROADMAP.md}` + `nextgen/aion-shopagent/` (бизнес-связка).

## Контекст
- Ориг: 206МБ, ничего не биндит (исходящий), задача AionNPRelay DISABLE. Не блокирует логин.
- Сурсы-приоритет: киты 2.7/5.8/7.7 (VM `D:\SAION\downloads\rz\unpacked\2.7\` и `D:\AION_LIVE_SERVER\`), PDB если есть.

## План
1. **R0**: наличие в китах + конфиги + класс-карты → README/ROADMAP обновить (пульс).
2. **R1**: wire (стенд-копия ориг если включаем).
3. **R2–R3**: Go MVP к aion-main.
4. **R4/R5**: A/B → свитч по «го».

## Правила
- Стандарты S1–S12; пульс; Agent API канал; секреты не в гит; прод по «го».
# ПРОМПТ: aion-ranking (RankingServer → свой веб-рейтинг, Go) — ДЕПРИОРИТЕТ

> Копируй в новый чат как первое сообщение. Процесс: [../WORKFLOW.md](../WORKFLOW.md) (`WORKFLOW: ranking`).

---

Ты — ассистент проекта **STELGEN / AION 7.7 PTS EU private server** (репо `stelgen/AION-7.7-EUR-SERVER`). Задача: перепись **RankingServer (.NET веб-рейтинг) → nextgen/aion-ranking** (Go, деприор).

## Первый шаг (обязательно)
1. Память: `STELGEN/projects/aion_server_2026-10-02`.
2. Репо: `nextgen/WORKFLOW.md` + `nextgen/README.md` §4 + `nextgen/aion-ranking/{README.md,ROADMAP.md}` + `nextgen/accountcache-ref/db-procs-77-ref58.rpt` (схема aion_ranking_*).

## Контекст
- Ориг: .NET-сервис, config.xml в ките НЕТ, некритично. Рейтинги-данные = домены CacheD (GloryPoint/abyss).
- Сурсы-приоритет: киты/ILSpy-аналоги, потом дизасм.

## План
1. **R0**: инвентарь + БИЗНЕС-ВОПРОС юзеру (нужен ли MVP рейтинга).
2. **R1**: как мир обновляет рейтинги (CacheD-домены — ШАРИРОВАТЬ с aion-cache пульсом).
3. **R2–R3**: Go MVP web-UI поверх aion_ranking_*.
4. **R4/R5**: свитч по «го».

## Правила
- Стандарты S1–S12; пульс + кросс-шаринг; Agent API канал; секреты не в гит; прод по «го».
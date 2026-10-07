# ПРОМПТ: aion-shopagent (ShopAgent/NCoin → свой, Go) — низший приоритет, бизнес-выбор юзера

> Копируй в новый чат как первое сообщение. Процесс: [../WORKFLOW.md](../WORKFLOW.md) (`WORKFLOW: shopagent`).

---

Ты — ассистент проекта **STELGEN / AION 7.7 PTS EU private server** (репо `stelgen/AION-7.7-EUR-SERVER`). Задача: перепись **ShopAgent (:10100, cash-shop/NCoin) → nextgen/aion-shopagent** (Go, низший приоритет).

## Первый шаг (обязательно)
1. Память: `STELGEN/projects/aion_server_2026-10-02`.
2. Репо: `nextgen/WORKFLOW.md` + `nextgen/README.md` §4 + `nextgen/aion-shopagent/{README.md,ROADMAP.md}` + `fixes-pending/loops-shopagent-channelchat-petition/`.

## Контекст
- exe НЕТ; NPRelay (NCoin-релей) скипнут — связка проверяется R0.
- Выдача предметов в user_item отработана (методика админ-выдач с бекапом/rollback).

## План
1. **R0**: конфиг-инвентарь (10100-строки), cash-shop БД-схема, .err-лупер → README/ROADMAP обновить.
2. **R0.5**: БИЗНЕС-ВОПРОС юзеру: нужен ли магазин и какой MVP-сценарий (каталог+покупка?).
3. **R1**: протокол-реконструкция (ILSpy-ТЗ).
4. **R2**: Go MVP по выбранному сценарию → R4 клиент-тест → R5 включение по «го».

## Правила
- Стандарты S1–S10; пульс (WORKFLOW §3); креды из `D:\SAION\creds`; секреты не в гит; тесты зелёные до пуша; прод-действия по «го».

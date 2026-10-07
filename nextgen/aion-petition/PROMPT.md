# ПРОМПТ: aion-petition (Petition → свой, Go) — низший приоритет

> Копируй в новый чат как первое сообщение. Процесс: [../WORKFLOW.md](../WORKFLOW.md) (`WORKFLOW: petition`).

---

Ты — ассистент проекта **STELGEN / AION 7.7 PTS EU private server** (репо `stelgen/AION-7.7-EUR-SERVER`). Задача: перепись **Petition (:2107) → nextgen/aion-petition** (Go, низший приоритет).

## Первый шаг (обязательно)
1. Память: `STELGEN/projects/aion_server_2026-10-02`.
2. Репо: `nextgen/WORKFLOW.md` + `nextgen/README.md` §4 + `nextgen/aion-petition/{README.md,ROADMAP.md}` + `fixes-pending/loops-shopagent-channelchat-petition/`.

## Контекст
- exe НЕТ; БД **PetitionDB/BkPetitionDB существует** (restore есть) — стартуем с неё.
- Гипотеза тишины конфигом: `disablePetitionFrom/To 0..24` (проверить по Server64.pdb).

## План
1. **R0**: sqlcmd-инвентарь PetitionDB (таблицы+procs sp_helptext; PS5+SqlClient к SQL2022 не работает — только sqlcmd), конфиг-поля Server64, .err-лупер → README/ROADMAP обновить.
2. **R1**: протокол-реконструкция (ILSpy-ТЗ аналогов + клиентский флоу); теорий-журнал.
3. **R2**: Go MVP: петиции CRUD в БД + минимальный клиентский протокол.
4. **R4/R5**: клиент-тест → включение по «го»; откат = конфиг.

## Правила
- Стандарты S1–S10; пульс (WORKFLOW §3); креды из `D:\SAION\creds`; секреты не в гит; тесты зелёные до пуша; прод-действия по «го».

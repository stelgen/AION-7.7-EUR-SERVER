# aion-petition — перепись PetitionD (:2107, .NET) — НЕ НАЧАТ

> ⬜ **НЕ НАЧАТ, низший приоритет.** exe в ките НЕТ (лупер безвреден); БД PetitionDB/BkPetitionDB в SQL СУЩЕСТВУЮТ (restore есть) — схема/тела достаются sqlcmd'ом.
> Запуск: `WORKFLOW: petition`. Стандарты: [../README.md](../README.md) §4.

## 📊 Статус и фазы

| Фаза | Статус |
|---|---|
| R0: БД PetitionDB/BkPetitionDB (таблицы+процы sp_helptext), конфиг-строки Server64 (petitionFrom/To, disablePetitionFrom/To 0..24 — гипотеза), .err-лупер | ⬜ |
| R1: протокол (клиентская петиция + Server64-релей) — реконструкция (exe нет) | ⬜ |
| R2: Go `nextgen/aion-petition` MVP (создание/список петиций + запись в БД) | ⬜ |
| R4/R5: клиент-тест → включение по «го» | ⬜ |

## 📟 Канон

| Факт | Источник |
|---|---|
| Порт 2107, exe НЕТ, лупер event-driven | fixes-pending/loops |
| БД PetitionDB/BkPetitionDB в ките есть (DSN + restore) | root README §restore |
| Гипотеза тишины: `disablePetitionFrom/To 0..24` в конфиге Server64 (семантику проверить по Server64.pdb) | fixes-pending/loops |

## 🚧 Блокеры

- exe нет → арбитр = клиент/Server64-ожидания + схема БД.

## ⏭️ Следующий шаг

`WORKFLOW: petition` → R0: sqlcmd-инвентарь PetitionDB (таблицы/процы) + конфиг-поля Server64 → ROADMAP-детализация.

## 📦 Артефакты

| Что | Где |
|---|---|
| Промпт | [PROMPT.md](PROMPT.md) |
| Лупер-контекст | ../fixes-pending/loops-shopagent-channelchat-petition/ |
| Креды | VM `D:\SAION\creds\` |

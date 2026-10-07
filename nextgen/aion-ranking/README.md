# aion-ranking — перепись RankingServer (.NET веб-рейтинг) — ДЕПРИОРИТЕТ

> ⬜ **НЕ НАЧАТ, ДЕПРИОРИТЕТ** (фулл-сервер бескомпромиссно: низкий приоритет ≠ отмена). RankingServer = .NET-сервис веб-рейтинга; config.xml в ките НЕТ — реконструкция.
> Запуск чата: `WORKFLOW: ranking` ([../WORKFLOW.md](../WORKFLOW.md)). Стандарты S1–S12: [../README.md](../README.md) §4.

## 📊 Статус и фазы

| Фаза | Статус |
|---|---|
| R-сурсы: киты (есть ли Ranking в 4.6/5.8/7.7; ILSpy-декомпил аналогичных .NET-сервисов) | ⬜ |
| R0: разведка (что ждёт мир от рейтингов; БД aion_ranking_* уже есть в ACS-схеме REF58!) | ⬜ |
| R1–R3: Go `nextgen/aion-ranking` MVP (web-рейтинг поверх aion_ranking_*) | ⬜ |
| R4/R5: A/B → свитч | ⬜ |

## 📟 Канон

| Факт | Источник |
|---|---|
| RankingServer.exe = .NET-сервис; config.xml в ките НЕТ; некритично (не блокирует) | root README |
| БД `aion_ranking_*` существует в схеме AionAccountCacheD (тела proc сняты в accountcache-ref/db-procs-77-ref58.rpt) | accountcache-ref |
| abyss-рейтинг = домен CacheD (UserGloryPointDB/GloryPoint) — данные идут через CacheD | cached-ref класс-карта |

## 🚧 Блокеры

- config.xml нет → реконструкция по ILSpy-аналогам; бизнес-нужность определяет юзер (веб-страница рейтинга для соло-игры — сомнительна).

## ⏭️ Следующий шаг

`WORKFLOW: ranking` → R0: ILSpy-инвентарь + aion_ranking_*-схема → вопрос юзеру (нужен ли MVP). Промпт: [PROMPT.md](PROMPT.md).

## 📦 Артефакты

| Что | Где |
|---|---|
| Промпт | [PROMPT.md](PROMPT.md) |
| Ориг | VM `D:\AION_LIVE_SERVER\` (RankingServer.exe — наличие проверить R0) |
| Креды/доступы | VM `D:\SAION\creds\` |

## 📜 Логи

Стандарт S3/S2; web-сервис = HTTP-логи raw-first.
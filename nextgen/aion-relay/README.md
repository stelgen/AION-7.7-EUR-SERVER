# aion-relay — перепись NPRelay64 (NCoin/Warehouse-релей) — ДЕПРИОРИТЕТ

> ⬜ **НЕ НАЧАТ, ДЕПРИОРИТЕТ** (фулл-сервер бескомпромиссно: низкий приоритет ≠ отмена). NPRelay — исходящий релей к MainServer: ничего не биндит, без него логин не блокирует.
> Запуск чата: `WORKFLOW: relay` ([../WORKFLOW.md](../WORKFLOW.md)). Стандарты S1–S12: [../README.md](../README.md) §4.

## 📊 Статус и фазы

| Фаза | Статус |
|---|---|
| R-сурсы: чужие киты (есть ли NPRelay в 2.7/5.8/7.7 китах — проверить; в 4.6 есть рабочий связка config) | ⬜ |
| R0: разведка (задача AionNPRelay DISABLE; 206МБ бинарь; NPRelayToMain/Ncoin-пакеты из класс-карт cached-ref) | ⬜ |
| R1–R3: Go `nextgen/aion-relay` MVP (NCoin/Warehouse-релей) | ⬜ |
| R4/R5: A/B → свитч | ⬜ |

## 📟 Канон

| Факт | Источник |
|---|---|
| NPRelay64.exe 206МБ, ничего не биндит (исходящий), рабочая связка config.xml (countryCode=2, dataCenter=1) + common.xml | live-тест 03.10 |
| Роли: NPRelayToMain/Ncoin (NCoin/Warehouse-релей к MainServer); НЕ блокирует логин | fixes-pending loops |
| Задача AionNPRelay создана и DISABLE | live |
| Связка с ShopAgent (10100) проверяется при shopagent-треке | aion-shopagent README |

## 🚧 Блокеры

- Бизнес-нужность: NCoin/склад — связка с магазином (aion-shopagent, тоже деприор). Старт = после решения юзера по магазину.

## ⏭️ Следующий шаг

`WORKFLOW: relay` → R0: наличие NPRelay в китах (2.7/5.8/7.7 — на VM), конфиги, класс-карты → ROADMAP-детализация. Промпт: [PROMPT.md](PROMPT.md).

## 📦 Артефакты

| Что | Где |
|---|---|
| Промпт | [PROMPT.md](PROMPT.md) |
| Ориг | VM `D:\AION_LIVE_SERVER\` (NPRelay64.exe, задача AionNPRelay DISABLE) |
| Креды/доступы | VM `D:\SAION\creds\` |

## 📜 Логи

Стандарт S3/S2; ориг молчит (не биндит) — наша перепись будет первая с внятным логом (raw-first).
## 📊 Сосед узнал (10.10, чат aion-cache R1-prep)
- CacheD64 — УЧАСТНИК NPRelay-протокола: в его DBProfiler-таблицах есть секции **NPRelay2Server (55 ops)** / **Server2NPRelay (52 ops)** — протокол relay живёт и в кэше; нумерация: cached-ref/profile-opcode-map.md. Для нашей переписи NPRelay = готовый словарь опкодов обеих сторон.

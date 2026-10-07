# aion-ic — перепись ICServer (Interchange/Channel, :2005/:2305) — НЕ НАЧАТ

> ⬜ **НЕ НАЧАТ.** Низший приоритет: ориг жив, без него лупер «Can't connect to Interchange» у Server64+CacheD (event-driven, безвреден). PDB 104МБ на VM (manifest-pdb-big).
> Запуск чата: `WORKFLOW: ic` ([../WORKFLOW.md](../WORKFLOW.md)). Стандарты S1–S10: [../README.md](../README.md) §4.

## 📊 Статус и фазы

| Фаза | Статус |
|---|---|
| R0 разведка (exe+PDB+конфиги+клиенты 2005/2305, что просит Server64 при старте) | ⬜ СЛЕДУЮЩИЙ |
| R1 wire capture (fork-proxy или pktmon — по цене рестарта) | ⬜ |
| R2 Go каркас `nextgen/aion-ic` | ⬜ |
| R4 A/B → R5 свитч (конфиг-порты) → R6 наблюдение | ⬜ |

## 📟 Канон (известное сейчас)

| Факт | Источник |
|---|---|
| Порты: **2005** (MainServer/Server64-сторона), **2305** (CacheD64 = клиент IC, исходящее) | конфиги + cached-ref класс-карта |
| Функции из cached-ref класс-карты: ICToCache 7 / CacheToIC 7 / ICClient 9 — ImportItem/ExportItemResult (меж-серверный перенос предметов), ReqGuildBasisInfo, ReqRankInfo, VersionAck, **ShutdownOtherServer** | pdb-publics CacheD64 |
| Без IC: Server64 и CacheD64 луперят «Can't connect to Interchange», старт стека не блокируют | live |
| Транзакционный хаб 3 сторон (план PLAN.md): import/export предметов + гильдии + рейтинги | PLAN §4 |
| PDB 104МБ на VM — дизasm по методу pdbpub.py | manifest-pdb-big.md |

## 🚧 Блокеры

- Ничего не снято: R0 даст exe/PDB/конфиги; семантика транзакций — с capture.

## ⏭️ Следующий шаг

`WORKFLOW: ic` → R0: ssh → скачать ICServer.exe+PDB+конфиги (read-only), netstat 2005/2305, конфиг-инвентарь → создать `nextgen/ic-ref/` (по шаблону) → обновить README. Промпт: [PROMPT.md](PROMPT.md) (черновик-скелет).

## 📦 Артефакты

| Что | Где |
|---|---|
| Промпт-черновик | [PROMPT.md](PROMPT.md) |
| PDB (будущее) | VM `D:\AION_LIVE_SERVER\ICServer\` (скачать в ic-ref) |
| Креды/доступы | VM `D:\SAION\creds\` |

## 📜 Логи

Стандарт S3: raw-first io-дампы + fork C>/O>/N> ([../LOGGING-SPEC.md](../LOGGING-SPEC.md)), ship-события по [../TELEMETRY-SPEC.md](../TELEMETRY-SPEC.md).

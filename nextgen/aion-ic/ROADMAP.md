# 🛰 ROADMAP aion-ic — ICServer → свой Interchange (Go)

> Обновлено 08.10 (создан). Запуск чата: `WORKFLOW: ic`. Черновик — уточняется R0.

## Фазы

| Фаза | Что | Статус | Оценка |
|---|---|---|---|
| R0 | Разведка: exe+PDB (104МБ)+конфиги+DSN; netstat 2005/2305 клиенты; что реально требует Server64/CacheD при старте; ic-ref/ по шаблону | ⬜ | 1 день |
| R1 | Wire capture: fork-proxy (IC дешёв в рестарте — прокси ок) или pktmon; ImportItem/ExportItem-транзакции на живом переносе | ⬜ | 1–2 дня |
| R2 | Дизasm dispatch (метод accache/cached-ref) | ⬜ | 2–3 дня |
| R3 | Go `nextgen/aion-ic`: proto + транзакции (ImportItem/ExportItemResult/ReqGuildBasisInfo/ReqRankInfo/VersionAck/ShutdownOtherServer) + store | ⬜ | 1–2 нед |
| R4 | A/B fork (VERDICT=SAME) | ⬜ | 1–2 дня |
| R5 | Свитч по «го» (порты в конфигах Server64/CacheD), откат одной правкой | ⬜ | 1 день |
| R6 | Наблюдение 24ч | ⬜ | 1 день |

## 🧪 Журнал теорий

| Дата | Теория | Проверка | Статус |
|---|---|---|---|
| 08.10 | IC — лёгкий транзакционный релей без SQL-писателя (бывает «хаб») | R0: есть ли у IC собственный DSN | ⏳ |
| 08.10 | ShutdownOtherServer = единственный канал управления IC-кластером (важно для op) | дизasm + capture | ⏳ |

## Леджер решений

- (пусто)

## Правила

- Ориг IC не рестартить без «го»; capture через fork-proxy (рестарт IC дешевле мира, но всё равно «го»).
- МВП: handshake/VersionAck + ImportItem/ExportItem = минимум для выживания стека; RankInfo/GuildBasis — следом.

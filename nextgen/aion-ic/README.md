# aion-ic — перепись ICServer (Interchange/Channel, :2005/:2305) — РЕСЁРЧ ЗАКРЫТ, R0 ждёт

> 🔬 **~10% · веб-ресёрч закрыт 07.10, R0 ждёт** ([RESEARCH.md](RESEARCH.md)): публичных сурсов/эмуляторов IC **нет** (GitHub 0 репо, Java-эмулиаторы без IC); протокольная семантика получена из открытого конфига AKllX. Бинари чужих китов на VM. Ориг жив, без него лупер «Can't connect to Interchange» у Server64+CacheD (event-driven, безвреден). PDB 104МБ на VM (manifest-pdb-big).
> Запуск чата: `WORKFLOW: ic` ([../WORKFLOW.md](../WORKFLOW.md)). Стандарты S1–S10: [../README.md](../README.md) §4.

## 📊 Статус и фазы

| Фаза | Статус |
|---|---|
| R-внешние источники (RZ/mmo-dev/GitHub/киты) | ✅ 07.10 ([RESEARCH.md](RESEARCH.md)) |
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
| `InterSvrType`: **1**=live, **2**=matchmaker/beginner (Server64/config.xml) | AKllX пост #26 (RZ 1197955) |
| `ICServerAddr`/`ICServerPort`(2005)/`ICServerId` — в common.xml Server64, блок помечен `3.0.1221` | AKllX пост #26 |
| IC распределяет слоты миров: numBeginnerServer/numIdentifiedServer/numGAb1Server (+gabyssGroupIdType) | ICServer.common 7.7 + AKllX |
| Matchmaker = отдельный стек (Main+Npc+Cache+Log), порты не пересекаются; `aion_event=true` = вход только через MM | AKllX пост #26 |
| `matchmaker_type 1→0` в matchmaker.xml — фикс арена-старта 4.6 PTS | RZ 1250659 |
| ICServer опционален для старта стека (можно не запускать) | RZ 1250659 (Mr. House) |
| Серийная защита 4.6/5.8/7.7 одинакова (Themida-nop) | marisa-chan (RZ 1234115) |
| Interchange появился между 2.7 и 3.0.1221 — 2.7 кит (2011) без IC | скачанный кит 2.7 |

## 🚧 Блокеры

- Ничего не снято: R0 даст exe/PDB/конфиги; семантика транзакций — с capture.

## ⏭️ Следующий шаг

`WORKFLOW: ic` → R0: ssh → скачать ICServer.exe+PDB+конфиги (read-only), netstat 2005/2305, конфиг-инвентарь → создать `nextgen/ic-ref/` (по шаблону) → обновить README. Промпт: [PROMPT.md](PROMPT.md) (черновик-скелет).

## 📦 Артефакты

| Что | Где |
|---|---|
| Веб-ресёрч (внешние источники) | [RESEARCH.md](RESEARCH.md) |
| Промпт-черновик | [PROMPT.md](PROMPT.md) |
| PDB (будущее) | VM `D:\AION_LIVE_SERVER\ICServer\` (скачать в ic-ref) |
| Креды/доступы | VM `D:\SAION\creds\` |
| RZ cookies | VM `rz-cookies.txt` + локально `/tmp/rz.txt` (обновлены 07.10) |
| Чужие киты на VM | `D:\SAION\downloads\rz\` (2.7 кит, Server64_byAKllX, 4.6 DB, 58Server, 5.8static_data) |

## 📜 Логи

Стандарт S3: raw-first io-дампы + fork C>/O>/N> ([../LOGGING-SPEC.md](../LOGGING-SPEC.md)), ship-события по [../TELEMETRY-SPEC.md](../TELEMETRY-SPEC.md).

## 📊 Сосед узнал (08.10, чат aion-main R0/RES)
- PDB Server64 publics (74164) содержат классы `MatchingMgr`, `World_BattleGround`, `World_IDArenaTournament`, `World_IDARENAPvP` — **matchmaker-миры существуют в бинаре ядра** = подтверждение теории InterSvrType=2 из бинаря.
- `ServerToIC` (44 публича) — RPC-семейство IC в Server64, имя совпадает с паттерном ServerToDb/ServerToNPCServer (Shared-код NC).

## 📊 Сосед узнал (10.10, чат aion-cache R1-prep)
- IC-канал со стороны CacheD64 снят в нумерации: **IC2DB 11 ops / DB2IC 9 ops** (cached-ref/profile-opcode-map.md) — донор для IC-подпротокола, плюс ICClient-классы в publics CacheD (14281).

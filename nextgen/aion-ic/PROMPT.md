# ПРОМПТ-ЧЕРНОВИК: ICServer → свой (Трек B #7, низший приоритет)

> 📡 **Канал VM (S12):** Agent API — `curl http://192.168.0.125:10200/api/agent/*`, токен `X-Agent-Token` (на VM `D:\SAION\creds\CREDS.md`, в песочнице `~/.aion-agent-token`), обёртка `nextgen/agent-cli.sh`. Новый шаг на VM = ps1 через `aionput`+`aionrun "powershell -File"`. SSH (алиас `aion`) — ТОЛЬКО деплой самого op. Спека: ../AGENT-SPEC.md

> Статус: ⬜ НЕ ТРОНУТ. Без IC лупер «Can't connect to Interchange» у Server64+CacheD (безвредно, стект старт живёт). PDB 104МБ на VM (см. manifest-pdb-big.md). Кодим по остаточному принципу — после accache/cache/authd.
> Копируй в новый чат как первое сообщение.

---

Ты — ассистент проекта **STELGEN / AION 7.7 PTS EU private server** (репо `stelgen/AION-7.7-EUR-SERVER`). Задача: оценить и затем переписать **ICServer (2005/2305) → свой IC** (Interchange/Channel — транзакционный хаб 3 сторон: Server64, CacheD64, ImportItem/ExportItem/Rank/GuildBasis).

## Первый шаг (обязательно)
1. Память: `STELGEN/projects/aion_server_2026-10-02`.
2. Репо: `nextgen/ROADMAP.md` §4 + `nextgen/README.md` §4 (стандарты S1–S10) + `docs/server-internals.md` (роль IC).
3. Скачай с VM (read-only, по ssh) ICServer.exe + его PDB/конфиги → манифест MD5 → publics через `tools/analysis/pdbpub.py` → `nextgen/ic-ref/` (создать по шаблону README-TEMPLATE.md).

## План (MVP, шаблонный — уточнить после разведки)
1. **R0**: разведка (exe/PDB/конфиги/клиенты netstat 2005/2305; что реально просит Server64 при старте).
2. **R1**: wire capture (fork-proxy или pktmon — по цене рестарта).
3. **R2**: Go `nextgen/aion-ic`: proto + transactions (ImportItem/ExportItemResult/ReqGuildBasisInfo/ReqRankInfo/VersionAck/ShutdownOtherServer — из cached-ref класс-карты).
4. **R4**: fork A/B → **R5** свитч по «го» (common.xml/конфиги), откат одной командой.

## Правила
- Стандарты S1–S10 из nextgen/README.md §4; TELEMETRY-SPEC; LOGGING-SPEC raw-first; FORK-SPEC; op-first; креды из D:\SAION\creds; секреты не в гит; тесты зелёные до пуша.
- Прод-действия — только по «го»; ориг IC не рестартить.

# Session 07.10.2026 — aion-authd MVP (R1-R4, код готов, НЕ деплоен)

> Промпт: nextgen/PROMPT-AUTHD.md. Режим: «реализация, прод не ломать, просто писать,
> не деплой — фулл цикл разработки MVP». Прод-инвариант: НИ ОДНОГО касания VM
> (192.168.0.125), L2Authd/гейт/мир не тронуты.

## Что сделано

`nextgen/aion-authd/` (Go, module aion-authd; deps: yaml.v3 + go-mssqldb):

- **R1 wire-фундамент**: `internal/wire` — фрейминг 2110 1-в-1 с гейтом
  (`aion-gate/internal/authdclient`): `[00][sid][ip-LE]`/`[01][sid]`/`[02][sid][len
  самоинклюзивный][blob]` ←; → `[03][V]` greeting при accept, `[01][sid]` negative-ack,
  `[02][id][len][type][payload]`, лимит 0x1ffb. Golden-тесты.
- **R2 каркас**: `internal/server` (accept/multi-gate/диспетчер op), `internal/config`
  (yaml, дефолты = live), `internal/ship` (копия из aion-logd как есть, App=`aion-authd
  2110`), `main.go` (banner+ship+sweeper).
- **R3 логика** (`internal/logic`): порт C1 CAccount на live-фактах: логин-blob 191Б
  strict (86Б → тишина — probe-факт), decbuf user14+pwd16+otp4, TrimSpace+ToLower,
  автосоздание (пароль НЕ проверяется — live), блоки (flags+block_msg → фейл 22),
  не-ASCII → фейл 2 (единственный live-фейл), relogin онлайн-акка → ТИШИНА (live) или
  fail7 (опция), online-флаг TTL 300с (live 2-6 мин), `[01]` флаг НЕ снимает (live).
  Payload'ы: type=3 (52Б: accId/token/16×0/2000/unk1/20×0), type=1 (4Б mid → wire 18Б),
  type=4 (31Б: [010101][IP][7777][f40101 01][00000002][010001][7×0]), type=7 (15Б:
  pk1/pk2 Rnd + serverID). Коды фейлов = Mobius AionAuthResponse.
- **R4 DB-слой** (`internal/store`): interface Store; MapStore (тесты/стенд, seed live-uid
  «1»→7, «stelgen»→1010); SQLStore (go-mssqldb, дефолтные SQL = тела C1-проц ap_GStat/
  ap_SLog/block_msg по схеме ReleaseAuthDBSchema.sql; имена переопределяются конфигом —
  сверка с реальным AionAccounts на R0 sp_helptext). ConnStr — секрет, env/конфиг на VM.
- **Доки**: `docs/authd-wire-20261007.md` (полный wire + open questions), README.

## Верификация

- `go vet ./...` + `go test -count=1 ./...` — 5 пакетов ЗЕЛЁНЫЕ.
- **probe-e2e**: собран `nextgen/aion-gate/cmd/probe` (ЖИВОЙ эталон, им же проверялся
  ориг L2Authd) против `aion-authd :2110`: greeting `[03] 0xc621` ✓, login-blob 191Б →
  `type=3 payload(52)` ✓, вердикт «healthy» ✓. Payload hex сходится с раскладкой:
  `01000000 4253f713 [16×0] d0070000 0b9fc6a0 [20×0]`.

## Готчи сессии

- FillDefaults: bool-поля из yaml не отличить от нуля — `StrictAsmBlob`/`AutoCreate`
  дефолтятся кодом (strict=true всегда; autoCreate=true, для mssql — явно из db.autoCreate).
- Gate шлёт IP в `[00]` как u32 LE от BE-значения → на проводе байты IP реверснуты;
  парсить `LE → октеты BE значения`.
- Тест: после silence-проверки с коротким read deadline — ОБЯЗАТЕЛЬНО сбросить deadline
  перед ожиданием ответа (i/o timeout на «молчание» съедает следующий ответ).
- PacketFrame: len-поле = len(blob)+2, blob ≤ MaxBody валиден целиком; ReplyPkt: body =
  1+payload ≤ MaxBody.

## Открытые позиции (не блокер MVP, блокер R5/R6)

1. **2104 (Server64/world)** — не реализован; роль/протокол верифицировать дизasmом
   L2Authd (pdbpub.py малый PDB локально) на R0. БЕЗ него свитч невозможен.
2. Точные unk-dword'ы type=3 payload (0xa0c69f0b; хвост 74b из форк-дампа `1a6bc068` на
   pt[56:60] против probe-payload 52Б — противоречие снят R5-диффом).
3. CM_UPDATE_SESSION (0x08) — ответ authd не известен (T3), лог+тишина.
4. Реальные имена procs/таблиц AionAccounts (R0 sp_helptext vs C1-схема).
5. GM-порт 2108 / QMAS 10062 / AccountCache 2220 — вне MVP (нужно ли — R0).

## Следующий шаг (следующий чат)

R5: fork-proxy на 2110 (наш гейт → fork → ориг + КОПИЯ в наш authd, diff O-vs-N на
каждый фрейм — метод fork-трека гейта) + R0-разведка VM read-only (config.txt, procs,
netstat 2104, PDB). НЕ ДЕПЛОИТЬ, НЕ СВИТЧИТЬ — прод живёт на оригинале.

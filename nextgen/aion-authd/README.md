# aion-authd — замена L2Authd.exe для AION 7.7 EU (Go, трек B #4)

**MVP 07.10.2026** — код R1-R4 полный, тесты зелёные, probe-e2e против живого эталона OK.
**ПРОД НЕ ТРОНУТ и не деплоен** — свитч только по «го» и только после R0/R5 (см. §Статус).

Цель: замена NC `L2Authd.exe` (1,198,592 Б): **2110** (serverExPort — наш aion-gate),
конфиг `etc\config.txt`, БД `AionAccounts` (L2Conn.dsn). Порты 2104/2108/10062/2220 — вне MVP.

## Статус фаз (ROADMAP.md)

| Фаза | Статус |
|---|---|
| R0 разведка VM (procs AionAccounts sp_helptext, роль 2104) | 🟡 procs ✅ 09.10: 31 proc + тела сняты → authd-ref/procs-aionaccounts-77.rpt (логин = ap_GPwdWithFlag→ap_AutoReg, serverlist = ap_GetServers); 2104 ✅ 09.10: протокол-корень снят живым packet-логом authd (heartbeat 60с + type0-релей логина + uid-эхо; корпус authd-ref/logs-2104/) — дизasm не нужен |
| R1 wire-фундамент | ✅ `docs/authd-wire-20261007.md` + golden-тесты |
| R2 каркас (framing/listener/config/ship) | ✅ (+мир-канал 2104: internal/world 09.10) |
| R3 логика (логин/автосоздание/online-TTL/фейлы) | ✅ live-факты 06-07.10 + каноны fork (type=3/4/7/fail) |
| R4 DB-слой | ✅ mem + mssql (C1-схема); 09.10 реальные procs AionAccounts сняты — SQLStore переводится на вызов ap_* procs |
| R5 fork-стенд на проде | 🟡 **ЖИВОЙ с 07.10 и ОСТАВЛЕН юзером** (forkauthd 2116 → ориг 2110 + копия → shadow 2117): каноны сняты, shadow структурно паритетен; осталась очередь арбитража O-vs-N по (sid,type) — N-ONLY = гонка тени |
| R6 свитч живого пути на наш + наблюдение 24ч | ⬜ после R0 (2104/procs/mssql) по «го» |

## Собрать/запустить

```bash
cd nextgen/aion-authd
go vet ./... && go test ./...
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o aion-authd.exe .   # для VM
go build -o aion-authd-linux . && ./aion-authd-linux -config config.yaml            # стенд
```

Конфиг: `config.example.yaml` (все дефолты = live-факты). Секрет (`db.connStr`) — ТОЛЬКО
на VM (`AUTHD_CONNSTR` env или config), в гит не попадает.

## Smoke (локально, без прода)

```bash
./aion-authd-linux -config /dev/null &
/tmp/probe 127.0.0.1:2110 probeacc 192.168.0.253
# → A>G [03] sid=50721 (приветствие)
# → A>G [02] type=3 payload(52) → ВЕРДИКТ: authd ОТВЕТИЛ type=3 (healthy)
```

`/tmp/probe` = `nextgen/aion-gate/cmd/probe` — ЖИВОЙ эталон, этим же инструментом
проверялся оригинальный L2Authd на VM. probe-логины лочат акк на onlineTTL — тестовые
имена держать пулом (прецедент: probeacc, probeacc2…).

## Тесты (что золотое)

- `wire`: CltConnect с LE-инверсией IP, самоинклюзивный len, greeting `03 21 c6 00 00`,
  лимиты 0x1ffb, negative-ack.
- `logic`: автосоздание+normalize, 52Б-payload (uid/token/2000/unk1), фейлы
  (не-ASCII=2, blocked=22, db=1), relogin-тишина и fail7, TTL-sweep, type=4 (31Б,
  IP/порт), type=7 (15Б, pk Rnd).
- `server` e2e (TCP): полный флоу фейк-гейта greeting→login→type3→serverlist→type4→
  play→type7, unknown-session → [01][sid], тишины (strict 86Б, unknown-op, relogin).
- `store`: map-стор (seed live-uid), ErrNotFound, blocks.
- `ship`: копия из aion-logd (App-имя = aion-authd 2110).

## R5 fork-стенд (07.10, ЖИВОЙ на VM — юзер решил ОСТАВИТЬ как есть)

Топология: `aion-gate (2106, authPort=2116) → forkauthd (:2116) → ориг L2Authd (:2110, живой путь
юзера) + копия всех фреймов → aion-authd shadow (:2117, mem-store)`. Ответы shadow НЕ идут юзеру;
лог `D:\SAION\aion-authd\fork-authd.log`: `C>/O>/N>` + VERDICT=SAME/DIFF по ключу (frame,sid,type).
Управление — через aion-op (группа fork: authdn/forkd); задачи SYSTEM AionAuthdShadow/AionForkAuthd;
`fork.cmd` пишет в fork-run.log (редирект `>> ... 2>&1` — паники видно).

Каноны сняты fork'ом НА ЮЗЕР-ФРЕЙМАХ (структурный паритет тени): greeting `[03][0xc621]` SAME;
type=3 = 52Б `[accId][token Rnd][8×0][2000][unk1 Rnd][28×0]`; type=4 = 26Б `[04]+[010101][ip][port 7777]…`;
type=7 = 9Б `[07]+[pk1][pk2][serverID]`; fail type=1 = 1Б кода + `[01][sid]` после.
Ориг отвечает SYSTEM_ERROR(20) на login пока мир не собран (8 conns на :2002) и пока жив PA.

## 📦 Артефакты

| Что | Где |
|---|---|
| Код/конфиг-пример | этот каталог (internal/{wire,logic,server,store,ship,config}; проверка через `../aion-gate/cmd/probe`) |
| Доки wire/сессии | docs/authd-wire-20261007.md, docs/session-20261007-authd-mvp.md, docs/auth-server-internals.md (общий референс авторизации) |
| Ресёрч/план/промпт | RESEARCH.md, ROADMAP.md, PROMPT.md |
| Референсы-сурсы | ../authd-ref/ (README-индекс) |
| Прод: exe+конфиги+логи | VM `D:\SAION\aion-authd\` (aion-authd.exe, forkauthd.exe, config-shadow.yaml, shadow.cmd, fork.cmd, *.log) |
| Креды/доступы | VM `D:\SAION\creds\` (CREDS.md) |

## 📜 Логи

Стандарт S3: fork-лог `D:\SAION\aion-authd\fork-authd.log` (C>/O>/N> + VERDICT), тень пишет shadow.log; run.cmd редирект `>> ... 2>&1` (паники видно); ship по [../TELEMETRY-SPEC.md](../TELEMETRY-SPEC.md).

## ⚠ Перед R6 (свитч) — блокеры

1. **Порт 2104 НЕ реализован** — Server64 (мир) ходит в L2Authd: протокол-корень уже снят
   живым packet-логом authd (09.10, дизasm не нужен — см. docs/authd-2104-recon-20261009.md +
   authd-ref/logs-2104/): heartbeat 60с (type 2/5), релей логина type 0, uid-эхо 13–44.
   ✅ 09.10 MVP реализован (`internal/world`: greeting `[03][authVersion][1]` + heartbeat 60с +
   type0-релей логина по живому корпусу; вкл. `gsPort`, дефолт 0 = выкл; квитанции `gsAcks` — T2 после fork-диффа 2104).
2. **procs AionAccounts** — ✅ 09.10 сняты (authd-ref/procs-aionaccounts-77.rpt), SQLStore → на ap_* procs.
3. **mssql-стор в shadow** — ✅ 09.10: SQLStore дефолты = РЕАЛЬНЫЕ procs AionAccounts
   (ap_GPwdWithFlag+ap_GStat/ap_AutoReg/ap_GetRestriction; тела в authd-ref/procs-aionaccounts-77.rpt);
   живую сверку на VM (shadow+connStr) — на R6-стендe.
4. **Арбитраж fork-лога** — ✅ 09.10: FIFO-очередь пар по ключу ОТВЕТА (sid,type) в forkauthd:
   пара открывается первым пришедшим (O или N), вердикт только при полной паре — гонки N-ONLY
   исключены; нзапрошенные = SINGLE; дроп зависших >60с (ARBITRATION-DROP). Race-тест в main_test.
5. Дисциплина: L2Authd хрупкий; probe-логины лочат акки (TTL 2–6 мин, тестовый пул
   probeacc*); PA жив ДО authd; секреты не в гит.

## 📊 Сосед узнал (08.10, чат aion-main R0/RES)
- Утёкшие сурсы Encom 7.5–7.7 (`reference/encom-leak-7577/`) содержат `network/loginserver/clientpackets/*` = **Java-эталон GS↔LS протокола 2104** (CM_GS_AUTH_RESPONSE, CM_ACCOUNT_RECONNECT_KEY и др.) — прямой материал для R6-блокера 2104-канала.
- Крипта/флоу 7777 Server64: GG(GameGuard)+Blowfish+AES из PDB publics.

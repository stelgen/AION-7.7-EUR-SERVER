# aion-authd — замена L2Authd.exe для AION 7.7 EU (Go, трек B #5)

**MVP 07.10.2026** — код R1-R4 полный, тесты зелёные, probe-e2e против живого эталона OK.
**ПРОД НЕ ТРОНУТ и не деплоен** — свитч только по «го» и только после R0/R5 (см. §Статус).

Цель: замена NC `L2Authd.exe` (1,198,592 Б): **2110** (serverExPort — наш aion-gate),
конфиг `etc\config.txt`, БД `AionAccounts` (L2Conn.dsn). Порты 2104/2108/10062/2220 — вне MVP.

## Статус фаз (AUTHD-ROADMAP)

| Фаза | Статус |
|---|---|
| R0 разведка VM (config.txt, dsn, порты, procs sp_helptext, PDB) | ⬜ нужно для R4-mssql/R5 |
| R1 wire-фундамент | ✅ `docs/authd-wire-20261007.md` + golden-тесты |
| R2 каркас (framing/listener/config/ship) | ✅ |
| R3 логика (логин/автосоздание/online-TTL/фейлы) | ✅ (live-факты 06-07.10) |
| R4 DB-слой | ✅ mem + mssql (C1-схема; реальные procs — сверка на R0) |
| R5 fork-proxy A/B на 2110 (копия фреймов, diff O-vs-N) | ⬜ СЛЕДУЮЩИЙ ШАГ |
| R6 свитч 2110 по «го» + наблюдение 24ч | ⬜ |

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

## R5 fork-стенд (07.10, ЖИВОЙ на VM)

Топология: `aion-gate (2106) → forkauthd (:2116) → ориг L2Authd (:2110, живой путь юзера)
+ копия всех фреймов → aion-authd shadow (:2117)`. Ответы shadow НЕ идут юзеру; лог
`D:\SAION\aion-authd\fork-authd.log`: `C>/O>/N>` + VERDICT=SAME/DIFF по ключу (frame,sid,type).
Управление — через aion-op (группа fork: authdn/forkd, кнопки start/stop/restart).
Первые дифф-факты: greeting SAME; ориг-фейл = payload 1Б кода + [01][sid] после; ориг
отвечает SYSTEM_ERROR(20) на login пока мир не собран (8/16 conns).

## ⚠ Перед R6 (свитч) — блокеры

1. **Порт 2104 НЕ реализован** — Server64 (мир) ходит в L2Authd: его канал нужно
   дизasm-верифицировать (R0) и реализовать, иначе свитч уронит мир.
2. **procs AionAccounts** — сверить sp_helptext с дефолтами C1-схемы.
3. **fork-proxy A/B** — diff O-vs-N до 100% паритета (метод трека гейта).
4. Дисциплина: L2Authd хрупкий; probe-логины лочат акки; секреты не в гит.

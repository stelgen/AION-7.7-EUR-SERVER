# aion-authd — замена L2Authd.exe для AION 7.7 EU (Go, трек B #4)

> ✅ **В БОЮ — R6 свитч выполнен 09.10 по «го» юзера.** Живой путь = наш authd:
> `клиент → aion-gate (2106, authPort=2110) → наш aion-authd (:2110)` + `Server64 → наш :2104`.
> Полный цикл юзера подтверждён (логин→мир→выход→мгновенный перелогин). exe MD5: authd `8c651f1e`
> (prod) — см. §Деплой; живёт `D:\SAION\aion-authd\`, задача **AionAuthdProd** (SYSTEM, onstart).

## 📊 Статус и фазы
| Фаза | Статус |
|---|---|
| R0 разведка VM (procs AionAccounts, роль 2104, L2Conn.dsn) | ✅ 09.10: 31 proc + тела (`../authd-ref/procs-aionaccounts-77.rpt`); 2104 = GS-канал (Encom-семантика + собственный packet-лог L2Authd) |
| R1 wire-фундамент 2110 | ✅ `docs/authd-wire-20261007.md` + golden-тесты |
| R2 каркас (framing/listener/config/ship) | ✅ (+ мир-канал 2104: `internal/world`, 09.10) |
| R3 логика (логин/автосоздание/online-флаг/фейлы) | ✅ live-факты + каноны fork |
| R4 DB-слой | ✅ **mssql-стор на РЕАЛЬНЫХ ap_* procs** (ретаргет 09.10); mem = fallback/тесты |
| R5 fork-стенд A/B | ✅ 07.10 (2116→ориг+копия→тень 2117); арбитраж O-vs-N FIFO-парами (верки SAME/DIFF/WAIT, N-ONLY-гонки исключены) |
| R6 свитч живого пути + наблюдение 24ч | ✅ **СВИТЧ 09.10**; полный цикл юзера ✓ (вкл. новый-аккаунт test1=uid 1021, мгновенный перелогин); наблюдение идёт |

Не работает: QMAS 10062 / GM 2108 (вне MVP, не используются стеком), char-count на сервер-селекте (тех-долг, ниже среднего).

## 📟 Канон протокола (проверено живыми корпусами, «НЕ трогать»)

**2110 (gate↔authd)** — см. `docs/authd-wire-20261007.md`: greeting `[03][V=0x0000c621]`,
`[00][sid][ip]`, `[01][sid]`, `[02][sid][len=body+2][op][blob]`, ответы type=3/4/7.

**2104 (мир↔authd, `internal/world`)** — фрейм `[u16 X LE][type][payload]`, **X = body+2 = total**
(та же самоинклюзивная формула, что 2110; C1 packetSizeType=3; подтверждено живым pong'ом мира):
| Type | Напр. | Смысл | Payload (канон) |
|---|---|---|---|
| 3 | A→W | greeting при accept | `792b3978 01000000 00` (9Б! хвостовой 00 обязателен) |
| 2 | A→W | heartbeat (60с) | пусто |
| 5 | W→A | world status | `[users u16][limit u16]` (f401=500, 0100f401=users 1) |
| 0 | A→W | relay PLAY (на CM_PLAY!) | 107Б: `[uid][name 20Б][2000][13×0][«0000000\0»][ip-dword реверс][zeros6][ff×24][zeros4][50c2366b×2][zeros12]` |
| 0 | W→A | ack | `[uid][N u32]` → N = pk1 для type=7 (эхо, канон 2/4/8/11/14/15/16) |
| 38/27/35/24/25/9/3/39/40 | W→A | события (вход чара/выход) | uid-базовые; **на каждое — квитанция** (см. acks) |
| 40/3 | W→A | logout | → снять онлайн-флаг (R6) |

Квитанции (acks, пайлоады 1-в-1 из корпуса 09:33): `38/39/40→44 (uid+63000000 00000000)`,
`27→19 (uid+0000)`, `35→31 (uid+11×00)`, `24→13 (uid+01010001)`, `25→16 (uid+0000)`, `3→14 (uid+01)`.
Без квитанций мир ретранслирует type=35 каждые 1-3с (висяк входа в мир).

**Создание аккаунта** — правит БД, не authd: `ap_GPwdWithFlag` → (нет акка + ASCII) → `ap_AutoReg`
(user_account pay_stat=1 / ssn / user_auth pwd=0×16 flag=3 / user_info kind=99). Пароль не
проверяется (и у ориг); PA не участвует. Тест test1 12:46: uid 1021 выдан ✓.

## 🧱 Структура / сборка
```
nextgen/aion-authd/
├── main.go                  # сборка: 2110-листенер + world-канал (gsPort) + колбэки
├── internal/{wire,logic,server,world,store,ship,config}
├── cmd/forkauthd/           # R5 fork-прокси (арбитраж O-vs-N)
├── config.example.yaml      # все дефолты = live-факты (секретов нет)
└── docs/ + ../authd-ref/    # wire/сессии/ресёрч/корпуса
```
`go vet ./... && go test ./...` (7 пакетов, зелёные); win:
`GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o aion-authd.exe .`

## 🧪 Тесты и верификация
- Golden: greeting/ping/relay 2104 (1-в-1 с корпусами 04.10+09.10), wire 2110, логика (автосоздание, фейлы 2/22/1, TTL), e2e 2110 (TCP), mssql-стор (RC-контракты).
- Живое: probe-e2e против ориг-эталона; fork-арбитраж O-vs-N (SAME по type=4, type=3 = структурный паритет); R6 живой путь: полный цикл юзера ✓✓, Server64 принял greeting 9Б + pong.

## 📦 Артефакты
| Что | Гит | VM |
|---|---|---|
| код/конфиги | этот каталог | `D:\SAION\aion-authd\` (aion-authd.exe `8c651f1e`/`b954de48`+, forkauthd `3bd24ff3`, config-prod-authd.yaml, config-shadow.yaml, authd-prod.log, shadow.log, fork-authd.log) |
| корпуса wire | `../authd-ref/logs-2104/` | `D:\AION_LIVE_SERVER\AuthD\etc\log\*.packet` (текстовый wire-дамп!) |
| procs БД | `../authd-ref/procs-aionaccounts-77.rpt` | — |
| doc wire/каноны | `docs/authd-wire-20261007.md`, `docs/authd-2104-recon-20261009.md` | — |

## 🗂 Сурсы-эталоны
S1 live-логи gate/authd (главный актив) · S2 C1-декомпил `../authd-ref/L2Auth-chaospaladin/` (WorldSrvSocket = фрейминг 2104) · S3 БД-схема `l2-c1-mastertoma/DBScript/` · S4 L2Authd.pdb (4513 publics, в гите) · S5 Encom Java-LS (GS↔LS семантика, `reference/encom-leak-7577/`) · S6 Packet Samurai Login_4.0.x.xml (клиентский LS-протокол, SM_LOGIN_OK/SERVER_LIST/PLAY_OK) · S7 aion-logd ship.

## 🚢 Деплой и откат
- Прод: `D:\SAION\aion-authd\authd-prod.cmd` → задача **AionAuthdProd** (2104/2110); тень — shadow.cmd → AionAuthdShadow (2117); fork — fork.cmd → AionForkAuthd (2116).
- **ОТКАТ** (одна цепочка): `D:\SAION\aion-authd\authd-rollback.cmd` (taskkill наш + schtasks AionAuthOnly = ориг L2Authd) → `C:\Temp\rollback-gate.ps1` (gate authPort 2110→2116) → `aionact restart gate restart`. Бекап гейт-конфига: `config-prod.yaml.bak-0910-preR6`.
- Канон-готчи: exe сменять только kill→copy→start; задача гейта иногда не поднимает с первого /run (повторить).

## 📜 Логи
`authd-prod.log` (prod), `shadow.log` (тень), `fork-authd.log` (арбитраж; сейчас вне цепочки) — все `>> file 2>&1` (паники видны). RAW-диаг мира: `gsRawLog: true` → «world RAW >/<» hex-строки. ship = TELEMETRY-SPEC (выкл. на проде).

## 🚧 Тех-бэклог (приоритеты)
1. **[ниже среднего] Висяк выхода из игры** (~10 мин, иногда краш клиента) — сервер-стор чистая (мир отпускает за 5с), клиент-локально; ревизия по триггеру.
2. **[ниже среднего] charcount на сервер-селекте** — 2 эксперимента, вердикты в `docs/techdebt-charcount-20261009.md`; НЕ наша регрессия (у ориг тоже пусто).
3. **[T2] pk1-эхо / 2104-квитанции** — live-верификация done; сверка байт-в-байт с ориг возможна только на A/B-стенде.
4. **[T2] IP-дворд релея** — конвенция 04.10↔09.10 неоднозначна (реверс vs direct) — сверить last_ip в БД.
5. **[T3] Наш authd ↔ ACS 2220** — не реализовано (мир ходит в ACS сам; authd-клиент ACS = фича-запрос).

## ⏭️ Следующий шаг
Наблюдение 24ч завершить → дельта ROADMAP/README. Следующие компоненты по приоритетам стека:
accache R1 capture (`../aion-accache/PROMPT.md`), gate T2-T6 (`../aion-gate/`), `WORKFLOW: <имя>`.

## 📡 Кросс-пульс соседям (10.10, гейт-чат)

- **op**: добавить сервис `authdprod` (задача AionAuthdProd, контроль портов 2110+2104) — сейчас
  живой authd в op НЕ контролируется (видна только тень authdn :2117); ⚠ `stop authdn` =
  `taskkill /F /IM aion-authd.exe` бьёт И ПРОД (общее имя exe у prod и тени) — разделить по
  kill_task/PID или переименовать тень. Детали: [../aion-op/ROADMAP.md](../aion-op/ROADMAP.md) §6.
- **gate**: T2-б/T2-в закрыты этим R6 ([../aion-gate/README.md](../aion-gate/README.md) §T2/T3);
  откат-цепочка на месте (rollback-gate.ps1 + authd-rollback.cmd).
- **Статус live**: op-метрики 12:57 VM 10.10 — оба процесса aion-authd.exe живы (prod 2110+2104 +
  тень 2117); алерты `down:authd` (ориг AionAuthOnly) = EXPECTED.
- README приведён к [README-TEMPLATE.md](../README-TEMPLATE.md) (S1) — этот коммит.

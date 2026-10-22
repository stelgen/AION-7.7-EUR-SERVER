# aion-accache — перепись AccountCacheServer 7.7 (порт 2220)

🟢 **~85% · R4 offline ЗАКРЫТ 22.10: replay 16/16 (100%)** — 4 XFAIL-пуша реализованы и посажены на golden (TestR4PushGoldens + TestR4ReplayCapture): ACP_13 (0x0D, [ts] константа 0x640831AE, пуш сразу после VERSION-ответа), ACP_15 (0x0F, [charRef][0x000A07EA][3] при CHAR_LOGIN), ACP_20 (0x14, 56Б: [charRef][..][lastLogin][..][1][..][flag][maxLevel] — lastLogin из global_user_data.last_login_time, maxLevel=MAX(user_level) аккаунта), CONFIRM_LUNA_REWARD (0x1C, [accId][charId=charRef&0xFFFF][u16 0]) — corpus capture-20261022 (254КБ, 2 чара × 3 логина + logout-кластер). Остаток: R3 SQLStore (go-mssqldb + UserInfo из global_user_data), live A/B, R5 свитч.
Статус трека: [ROADMAP.md](ROADMAP.md). Факты: RESEARCH.md (§6 сурс-свип, §9-11).

## Канон (доказано дизasmом, не догадки)

- **Wire**: `[u16 N LE][u16 cmd LE][u8 0xEB][u16 ~cmd LE][payload]`, N = ПОЛНАЯ длина кадра вкл. len-поле (тело = N-2; «lenMinus2» = артефакт R1, исправлено дизasmом OnRead в R2.5); лимит 0x2000; cmd ≥ 0x6C reject.
- **Dispatch**: `handler = table[cmd]` (ctor VA 0x140078B40), cmd 22 = НЕ ЗАНЯТ.
- **T1** (Server64→ACS): cmds 0..39 (см. dispatch/dispatch.go) — VERSION, FIRST_LOAD_ACCOUNT_INFO (4),
  CHAR_*-цикл 15..21, LUNA 26..29+36, TRIAL 7/8, CUSTOM 10..13, PLAYTIME_POLLS 32..35, MONSTER_CORE 38/39.
- **T2** (второй канал, cmds 0..7): MoveChar×2, PromotionCoolTime×2, GEN_TEST×2 — семантика канала = R1.
- **БД**: `AionAccountCacheD` (21 таблица, 101 proc, тела в accountcache-ref/db-procs-77-ref58.rpt).
- **Клиент 2220 на проде = Server64** (authd подключается лениво).
- **ACP-ответы**: тот же фрейм (`PutCmd_ACP`), номера ответных cmd = та же enum ACQ с маркером 0xEC (пары 1↔1/4↔4/5↔5/26↔26; «второй формат» и «S2C ≠ ACQ» = артефакты офф-бай-2, R2.5 10.10).

## Структура

```
main.go                 — баннер, ship, serve (SQLStore = R3)
internal/proto          — фрейм Build/Validate/Reader (state-машина как OnRead)
internal/dispatch       — таблицы T1/T2 (имена ACQ_*)
internal/cache          — RAM-хранилище (hidden_fatigue, packs; UpdateFatigue)
internal/db             — Executor + {call}-обёртки ключевых procs (SQLStore R3)
internal/server         — accept + dispatch + каркасные хендлеры (version/firstload/synctest/fatigue)
internal/ship           — телеметрия (копия стандарта, "aion-accache 2220")
config.example.yaml
```

## Тесты

`go vet ./... && go test ./...` — зелёные (proto golden/roundtrip/bad-marker/inv/split;
server e2e: FIRST_LOAD → 5×int-ответ, unknown → дроп, bad-marker → close;
wire-regression W2/W3: proc-имена + arg-раскладки + off-by-2 + тишина).

R4 replay (TestR4ReplayCapture в internal/server/replay_r4_test.go): автономный
реплей golden-корпуса capture-20261007 — все C2S-кадры из frames.txt шлются в
живой сервер (fakeDB + RAM-срез global_user_data) в одно соединение, ответы
байт-в-байт сравниваются с S2C-последовательностью capture.
**R4-финал 22.10: 16/16 (100%) байт-в-байт.** Golden-пуши дополнительно
закреплены тестом TestR4PushGoldens (internal/payload) — обе сессии, оба чара.
Расхождение = XFAIL-отчёт — тест НЕ падает, покрытие фиксируется (см. ROADMAP R4).

## Что осталось

| Фаза | Что |
|---|---|
| R1 ✅ | capture-стенд accmirror :2220→:2221 + раскладки (10.10, 22.10 ×2 corpus) |
| R3 | SQLStore (go-mssqldb) + полный хендлер-набор по proc-телам + UserInfo из global_user_data |
| R4 | live A/B fork-прогон с оригом (offline 16/16 ✅ 22.10); опц. дозахват T2-канала |
| R5 | свитч по «го» (откат = retarget задачи AionAcc) |

## 📦 Артефакты

| Что | Где |
|---|---|
| Код/конфиг-пример | этот каталог (internal/{proto,dispatch,cache,db,server,ship}) |
| Ресёрч/план/промпт | RESEARCH.md, ROADMAP.md, PROMPT.md |
| Референсы (dispatch/procs/конфиги) | ../accountcache-ref/ (README-индекс) |
| PDB/бинари ориг | VM `D:\AION_LIVE_SERVER\AccountCacheServer\`; локально `~/STELGEN/projects/aion_rev_2026-10-05/artifacts/pdb-big/AccountCacheServer/` |
| Креды/доступы | VM `D:\SAION\creds\` (CREDS.md) |

## 📜 Логи

Стандарт S3 (raw-first, io-дампы, parse.err+raw) — см. [../LOGGING-SPEC.md](../LOGGING-SPEC.md); ship-маркировка «aion-accache 2220».

## Готчи

- Event ship: поля Ev/Svc/Msg (не Time/Text) — фейл компиляции при копировании из старых апок.
- ACP-ответы в каркасе = эхо-cmd (TODO R1) — НЕ деплоить без R4-диффа.
- `sh`/braces в шелле песочницы; yaml править байтово (правила ROADMAP §4).

## 🔗 09.10 cross-pulse (authd R6): наш authd в ACS пока НЕ ходит

- Ориг L2Authd имел ленивый клиент к ACS :2220 (AccountContainer); наш authd-клиент ACS не
  реализован (T3, см. authd ROADMAP тех-бэклог) — мир сам ходит в ACS 2220, логины работают.
  При фичах «сервер-инфо из ACS» (char-count и т.п.) — вспомнить этот канал.

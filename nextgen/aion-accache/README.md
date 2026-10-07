# aion-accache — перепись AccountCacheServer 7.7 (порт 2220)

Каркас R2 готов (07.10.2026): фрейминг + dispatch + RAM-кэш + db-интерфейс — **без capture**.
Статус трека: nextgen/ACCOUNTCACHE-ROADMAP.md. Факты: RESEARCH.md (§9-10).

## Канон (доказано дизasmом, не догадки)

- **Wire**: `[u16 lenMinus2 LE][u16 cmd LE][u8 0xEB][u16 ~cmd LE][payload]`; лимит 0x2000; cmd ≥ 0x6C reject.
- **Dispatch**: `handler = table[cmd]` (ctor VA 0x140078B40), cmd 22 = НЕ ЗАНЯТ.
- **T1** (Server64→ACS): cmds 0..39 (см. dispatch/dispatch.go) — VERSION, FIRST_LOAD_ACCOUNT_INFO (4),
  CHAR_*-цикл 15..21, LUNA 26..29+36, TRIAL 7/8, CUSTOM 10..13, PLAYTIME_POLLS 32..35, MONSTER_CORE 38/39.
- **T2** (второй канал, cmds 0..7): MoveChar×2, PromotionCoolTime×2, GEN_TEST×2 — семантика канала = R1.
- **БД**: `AionAccountCacheD` (21 таблица, 101 proc, тела в accountcache-ref/db-procs-77-ref58.rpt).
- **Клиент 2220 на проде = Server64** (authd подключается лениво).
- **ACP-ответы**: тот же фрейм (`PutCmd_ACP`), номера ответных cmd TBD (indirect vtable — R1 capture).

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
server e2e: FIRST_LOAD → 5×int-ответ, unknown → дроп, bad-marker → close).

## Что осталось

| Фаза | Что |
|---|---|
| R1 | fork-proxy :2220 (невидим) + capture при логинах юзера → payload-раскладки + ACP-номера + семантика T2 |
| R3 | SQLStore (go-mssqldb) + полный хендлер-набор по proc-телам |
| R4 | A/B fork-прогон, diff байт-в-байт с оригом |
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

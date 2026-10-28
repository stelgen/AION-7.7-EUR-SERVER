# aion-accache — перепись AccountCacheServer 7.7 (порт 2220)

✅ **R5 СВИТЧ ✅ 22.10: ПРОД = aion-accache :2220** (задача AionAccacheProd, ориг AccountCacheServer отведён; контрольный логин чист: 0 db.err, ll/lo = сек-в-сек штампов НАШИХ записей — Т16/Т17 ✅, trade=70 сохранён — Т23 ✅) · R4 live A/B ✅ (VERDICT байт-в-байт, 3 раунда) · R4 offline ✅ replay 16/16 (100%) · R3 SQLStore ✅ (EXEC @pN) — 4 XFAIL-пуша реализованы и посажены на golden (TestR4PushGoldens + TestR4ReplayCapture): ACP_13 (0x0D, [ts] константа 0x640831AE, пуш сразу после VERSION-ответа), ACP_15 (0x0F, [charRef][0x000A07EA][3] при CHAR_LOGIN), ACP_20 (0x14, 56Б: [charRef][..][lastLogin][..][1][..][flag][maxLevel] — lastLogin из global_user_data.last_login_time, maxLevel=MAX(user_level) аккаунта), CONFIRM_LUNA_REWARD (0x1C, [accId][charId=charRef&0xFFFF][u16 0]) — corpus capture-20261022 (254КБ, 2 чара × 3 логина + logout-кластер). Остаток: R3 SQLStore (go-mssqldb + UserInfo из global_user_data), live A/B, R5 свитч.
Статус трека: [ROADMAP.md](ROADMAP.md). Факты: RESEARCH.md (§6 сурс-свип, §9-11).

## 🧭 Граф кода (S13)

- Код компонента навигируется AST-графом `graphify`, НЕ чтением всего пакета: `query`/`explain`/`path`/`god-nodes` — правила и команды: [../WORKFLOW.md §6.1](../WORKFLOW.md).
- ПЕРЕД правкой символа — `graphify affected <имя>` (кто зависит = что сломается); ПОСЛЕ правки — `graphify update .` в этой папке; `graphify-out/` = кэш (в гит не кладётся).
- Граф = навигатор, не арбитр: семантика протокола = канон этого README + live capture.


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
server e2e: FIRST_LOAD → 5×int-ответ, unknown → дроп, bad-marker → close,
VERSION→[ответ+пуш ACP_13], CHAR_LOGIN → [16+15+14+1c] байт-в-байт, login/logout →
SetLoginUser/SetLogoutUser_20120703 арг-чек; wire-regression W2/W3;
db: GetUserInfo SELECT-форма/пустой-резалт, SetLogin/SetLogout арг-чек).
⚠ известный баг фейков: copy(dest,row) без deref = нулевые колонки — честный Scan
(как replayRows/capRows).

R4 replay (TestR4ReplayCapture в internal/server/replay_r4_test.go): автономный
реплей golden-корпуса capture-20261007 — все C2S-кадры из frames.txt шлются в
живой сервер (fakeDB + RAM-срез global_user_data) в одно соединение, ответы
байт-в-байт сравниваются с S2C-последовательностью capture.
**R4-финал 22.10: 16/16 (100%) байт-в-байт.** Golden-пуши дополнительно
закреплены тестом TestR4PushGoldens (internal/payload) — обе сессии, оба чара.
Расхождение = XFAIL-отчёт — тест НЕ падает, покрытие фиксируется (см. ROADMAP R4).

## 🧾 VERDICT live A/B (22.10, accmirror v1.3 + тень :2222 read-only)

| cmd | Вердикт | Примечание |
|---|---|---|
| VERSION (1) + ACP_13 (0x0D) | ✅ SAME (3 раунда) | ACP_13 = [ts] константа 0x640831AE после VERSION-ответа (1:1:1) |
| FIRST_LOAD (4) | ✅ SAME | ts = hidden_fatigue_updatetime из БД; point=1 clamp (Т4/Т6 live) |
| LOAD_BM_PACK (5) | ✅ SAME | count=0 |
| S2C 0x15 (ответ на ASK_JUMPING 0x1F) | ✅ SAME | канон переехал с «пуша после BM» (Т2 опровергнута live 5/5) |
| trial-ответ (S2C 0x06 на LOAD_TRIAL) | ✅ SAME | sell=917/trade=70 из БД; C2S UPDATE_TRIAL несёт те же поля (Т3 live) |
| UPDATE_LUNA (27) | ✅ SAME | префикс = [acc&0xFF][00][01][00] (F2/FF — Т20) |
| LOAD_LUNA (26) | ✅ SAME | ts = limit_play_reset_time (Т1 live) |
| CHAR_LOGIN (16) + ACP_15 (0x0F) + CONFIRM_LUNA (0x1C) | ✅ SAME | кластер 15→20→28; ACP_15 dword3: 3 (существующие) vs 22 (новый чар) = Т19 ⏳ |
| ACP_20 (0x14) | 🟡 СТРУКТУРНЫЙ | [13]=maxLevel SAME, [12]=flag SAME; [5]=lastLogin: значения различаются возрастом entry (ориг заморожен с 06:53, наш с создания entry) — механизм Т10-М2 совпадает, арбитр = дизasm |
| CHAR_LEVEL_CHANGED / UHF / LOGOUT / JUMPING / TRANSFORM (C2S) | ✅ тишина | совпадает с ориг |
| Записи в БД | ⏸ read_only | Exec глушится (Т15 SQL-дифф = следующий этап по «го»); Т17 ll/lo подтверждена телом proc + живой БД (ll=штамп логина, lo=штамп логаута) |

Инфраструктура A/B: accmirror v1.3 (front :2220 → copy :2221, shadow-tap с авто-реконнектом, кадры тени = S> в общем логе), тень aion-accache :2222 (задача AionAccShadow, db.read_only=true, aionop_ro гранты: EXEC ×4 + SELECT global_user_data). Откат: bak-v11 exe/run.cmd; тень независима.

## 🚢 R5-свитч (22.10) — сделано/откат

- Прод: `D:\SAION\aion-accache\aion-accache.exe` (v4 `82779ab6`) + `config-prod.yaml` (:2220, write-SQL, server_id 1); задачи `AionAccacheProd` (+`AionKickAccache`); Server64 переподключился сам (~8с), VERSION-хендшейк ✓.
- **Откат одной командой:** `schtasks /end /tn AionAccacheProd; schtasks /run /tn AionAccMirror` (стенд v1.3+копия-2221 целы).
- Стенд-наследие: accmirror v1.3 + копия-2221 остановлены как путь (живы для отката); тень :2222 остановлена (accstop).
- Write-урок (Т23): первый прод-логин поймал затёртый trade 70→0 → канон cmd 8 исправлен, БД восстановлена UPDATE-ом, повторный логин подтвердил.

## Что осталось

| Фаза | Что |
|---|---|
| R1 ✅ | capture-стенд accmirror :2220→:2221 + раскладки (10.10, 22.10 ×2 corpus) |
| R3 ✅ | SQLStore-каркас был с R2; R3 22.10: db.GetUserInfo (SELECT last_login_time + MAX(user_level) FROM global_user_data — read-proc в REF58 НЕТ), db.SetLoginUser/SetLogoutUser_20120703 (тела REF58; 20121120/20121206 = Т15 ⏳), config db.server_id (дефолт 1); тесты db (формы/аргументы) + e2e login/logout SQL |
| R4 | live A/B fork-прогон с оригом (offline 16/16 ✅ 22.10; SQL-дифф логин/логаут + UserInfo-раскладки Т10/Т15/Т16); опц. дозахват T2-канала |
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

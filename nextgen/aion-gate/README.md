# aion-gate — замена AuthGateD для AION 7.7 EU (Go)

> 📡 **Канал VM (S12):** Agent API — `curl http://192.168.0.125:10200/api/agent/*`, токен `X-Agent-Token` (на VM `D:\SAION\creds\CREDS.md`, в песочнице `~/.aion-agent-token`), обёртка `nextgen/agent-cli.sh`. Новый шаг на VM = ps1 через `aionput`+`aionrun "powershell -File"`. SSH (алиас `aion`) — ТОЛЬКО деплой самого op. Спека: ../AGENT-SPEC.md

✅ **РЕЛИЗ · 100% · 08.10.2026** (T1-фейлы + таймауты + live-тексты ошибок + тест-крутилка); хвосты T2-а/T3/T4/T6 = хардинг (см. TD-6). Прод: `192.168.0.125:2106`,
`mode: authgate`, полный живой флоу доказан (`1/1` → accId 7; `stelgen` → accId 1010):
`welcome 194B → AUTH_GG 42b → CM_LOGIN → blob cbdb 191Б → authd type=3 → [03]74Б →
[05] relay → [04]42Б → [02] relay → [07]26Б → мир 7777`.

Версия релиза: коммит **f8912a9** (exe sha256 `7c4dcab…`). Откаты exe: `aion-gate.exe.bak-63f6a47`,
`.bak-9c85c12`, `.bak-9f2da98`, `.bak-e1dd475` (последовательность дня 08.10).
Архитектура/выжимка решений: `docs/architecture-aion-gate-20261007.md` (артефакты/креды/логи — §Эксплуатация ниже; реестр доступов: VM `D:\SAION\creds\` CREDS.md).

Текущая прод-топология (R6-свитч 09.10, актуально): наш гейт 2106 (`authPort: 2110`) →
**наш aion-authd** (:2110, задача AionAuthdProd = живой путь R6) + мир-канал `Server64 → наш :2104`.
Fork-цепочка 2116 (`forkauthd`→ориг 2110 + shadow :2117) остановлена КАК ПУТЬ и живёт только
как откат: `C:\Temp\rollback-gate.ps1` (authPort 2110→2116) + `D:\SAION\aion-authd\authd-rollback.cmd`
(бекап гейт-конфига: `config-prod.yaml.bak-0910-preR6`). Методология fork-A/B: [../FORK-SPEC.md](../FORK-SPEC.md).

## ✅ Канонические факты протокола (все live-доказаны, НЕ трогать)

| Факт | Значение |
|---|---|
| Welcome | 194B = `[u16 0xC2][ECB(key1) 192]`; pt 173+4: `[0]=0x00, [1:5]=fc=sid, [5:9]=V (authd [03]), [9:137]=ScrambleModulusServer(N), [137:153]=GG-нули, [153:169]=key2, [168:172]=65650072, [172]=0x00` |
| Крипта | Blowfish канон; key1 = LUT(0x04bd) static `6b60cb5b…`; EncryptPrimary (cumsum, csum@n', dword0 не трогается); EncryptSecondary/DecryptSecondary @0x417a80/0d |
| RSA | RSA-1024, **e=65537** (доказано оракулом m^65537 mod N == ct), пул ключей, серверный скрамбл swap→xorL→dword→xorU |
| CM_AUTH_GG | клиент 34b = EncryptSecondary([sid][20Б]); ответ 42b = EncryptSecondary([0b][sid][27×0]) |
| CM_LOGIN | **op=0x00 у живого EU-клиента** (эталон Mobius = 0x0B — НЕ для leak-клиента!); pt = [op][ct 128×k][tail ≤64]; loginex k≥2: user=combined[78:142], pwd=combined[206:238] (реально на [220:230]), otp=[238:242]=0 |
| Blob в authd | **asm-форма 191Б**: `[00][decbuf34=user14+pwd16+otp4][dword=pt[148:152]][tail=pt[152:] (152Б)]` — ровно как оригинал; authd игнорит содержимое dword/tail, но требует asm-размер (86Б → тишина!) |
| Релей authd (2110) | `[00][sid][IP-be]` / `[01][sid]` / `[02][sid][len][blob]`; назад `[03][sid]` (V=0xc621), `[02][id][len][type][payload]`; **клиентский опкод = тип authd** (prepend): type=3→`[03]+pad64`=74Б, type=4→42b (pt 32), type=7→26b (pt 16) |
| State-машина | CONNECTED{0x07→authgg, 0x08→UPDATE_SESSION relay} → AUTHED_GG{0x00‖0x0B→login} → AUTHED_LOGIN{0x05,0x02→relay}; unknown = лог, НЕ cc45 |
| Username | `TrimSpace+ToLower` перед decbuf (К-5) |
| LoginFail | cc/SM_LOGIN_FAIL = messageId AionAuthResponse: 0,1,2,4,5,6,7,8,10,11,12,15,16,18,21,22 (22=BAN_IP «заблокирован» — live); 45 = authgate-спец |

## 🛡️ T1 — фейлы вместо тишины (08.10; ИСТОЧНИК = сорс эталона, не догадки)

Эталон: `reference/Mobius_AionEmu` (Aion-Germany 7.7 → Aion-Lightning; клон в
`STELGEN/projects/aion_server_2026-10-02/reference/`). Формы и коды взяты из сорса:

| Пакет | Форма (plaintext) | Wire |
|---|---|---|
| `SM_LOGIN_FAIL.java` (super 0x01) | `[01][D messageId]` (5Б) | **18Б** = EncryptSecondary(8Б) — ровно live-форма 06.10 11:08 |
| `SM_PLAY_FAIL.java` (super 0x06) | `[06][D messageId]` (5Б) | **18Б** |
| `SM_UPDATE_SESSION.java` (super 0x0c) | `[D accountId][D loginOk][C 0x00]` (9Б) | 26Б |

Реестр кодов = `AionAuthResponse.java` (messageId): 0 AUTHED(внутр.), 1 SYSTEM_ERROR,
2 INVALID_PASSWORD, 4 FAILED_ACCOUNT_INFO, 5 FAILED_SOCIAL_NUMBER, 6 NO_GS_REGISTERED,
**7 ALREADY_LOGGED_IN**, **8 SERVER_DOWN**, 10 NO_SUCH_ACCOUNT, 11 DISCONNECTED, 12 AGE_LIMIT,
15 SERVER_FULL, 16 GM_ONLY, 18 TIME_EXPIRED, 21 ALREADY_USED_IP, **22 BAN_IP** («заблокирован» — live);
45 = authgate-спец (гейт не готов). Полный список 0–22 — `internal/server/authfail.go`.

Триггеры (конфиг; **`authdTimeoutSec=15` — ЕДИНЫЙ бизнес-таймаут тишины authd** для
логина И play, `failCloseSec=2` = после фейла сессия рвётся через 2с
(«рве сессии с ошибкой» — юзер 08.10), дефолт-коды: login=1, play=8, online=7):

- **(а) таймаут authd (login)** — после релея blob нет type=3 за `authdTimeoutSec` →
  `SM_LOGIN_FAIL(loginFailCode=1)` + close через `failCloseSec`. Клиент НЕ висит по 1-2 мин
  (жёлтый фикс UX 08.10). Молчание authd = его онлайн-флаг/флейк — НЕ блокер гейта.
- **(в) таймаут play** — после релея `[05]`/`[02]` нет type=4/7 за тот же
  `authdTimeoutSec` → `SM_PLAY_FAIL(playFailCode=8)` + close. Эталон `CM_PLAY`: GS offline →
  SERVER_DOWN без close; SERVER_FULL=15 — альтернатива конфигом.
- **(б) гейт-лок relogin — ВЫКЛ** (решение юзера 08.10: «пусть логинятся как могут»).
  Фича осталась: `onlineTtlSec: N>0` → login-ok по этому username моложе N сек →
  НЕМЕДЛЕННЫЙ `SM_LOGIN_FAIL(7)`; blob всё равно релеится (эталон `AccountController.login`:
  kick + ALREADY_LOGGED_IN(7), следующая попытка проходит). Дефолт 0 = не используем.

Семантика close: эталон в дефолт-ветках делает `close(packet,false)` (фейл + закрыть),
но RSA-fail в CM_LOGIN и SERVER_DOWN в CM_PLAY — БЕЗ close. Наше поведение: close через
`failCloseSec` (2с — фрейм уже в TCP-буфере, клиент успевает показать текст).

**Живые тексты клиента 7.7 EU (прогон юзера 08.10, полный цикл 1→22+45)** — захардкожены
в `authfail.go` (`authFailText`), логируются на КАЖДУЮ выдачу: `SM_*_FAIL -> КЛИЕНТ:
messageId=N (NAME) текст="..." sid= ip=` + ship-событие `text`:

| ID | Текст клиента | ID | Текст клиента |
|---|---|---|---|
| 1 | Ошибка авторизации. Пожалуйста, попробуйте зайти в игру позже. | 12 | Ваш возраст не соответствует возрастному цензу игры. |
| 2/3 | Неверный логин или пароль. | 13 | Под вашим аккаунтом зашли с другого компьютера. |
| 4 | Невозможно найти информацию об аккаунте. | 14 | Вы уже в игре. |
| 5 | Отсутствуют паспортные данные | 15 | Сервер переполнен. |
| 6 | Ни один игровой сервер не был авторизован на сервере авторизации. | 16 | На сервере ведутся работы. Повторите попытку позже. |
| 7 | Вы уже залогинись. | 17 | Смените пароль и повторите попытку входа. |
| 8 | Выбранный сервер временно недоступен. Подключение невозможно. | 18 | Соединение невозможно. Время подписки закончилось… |
| 9 | Введенная информация при входе в игру не соответсвует указанной ранее. | 19 | На аккаунте не осталось оплаченного времени. |
| 10 | Отсутсвует информация о входе в игру. | 20 | Системная ошибка. Пожалуйста, обратитесь в системную поддержку пользователей. |
| 11 | Соединение было прервано обращением на главную страницу plaync. | 21 | Данный IP уже используется. |
|  |  | 22 | Ваш аккаунт заблокирован. |
|  |  | 45 | Вы сможете запустить AION только после авторизации на главной странице сайта. |

**Тест-крутилка ошибок** (`loginTestFail`/`playTestFail`: -1=CYCLE 1→22+45, >0=фикс; blob
не релеится, акк не лочится) — фича в коде/конфиге ОСТАВЛЕНА, на проде НЕ используется
(в config-prod.yaml строка закомментирована; прогон 08.10 = 24 логина, весь реестр
1..22+45 подтверждён живым клиентом).

Тесты: `TestAuthFailFrames` (wire 18 + roundtrip), `TestAuthFailTexts`, `TestLoginTimeoutFail`,
`TestPlayTimeoutFail`, `TestReloginOnlineCache`, `TestAuthFailCancel` (`authfail_test.go`).
Логи: `RAW G>C SM_LOGIN_FAIL messageId=…` + ship-события `login.onlinefail`/`login.timeout`/`play.timeout`.

## ⏭️ T2/T3 — статусы хвостов (актуализация 10.10 после R6-свитча authd)

- **T2-б (relogin онлайн-акка) — ЗАКРЫТО 09.10 (authd R6)**: «наш authd молчит (flag TTL 2-6 мин)»
  УСТАРЕЛО — удалено из канона. R6-ревизия authd: `clearOnlineOnDisconnect: true` — [01] от гейта
  снимает флаг; мир 40/3 = основной путь; TTL 5 мин = страховка. Полный юзер-цикл
  логин→выход→мгновенный перелогин подтверждён live. Остаток (ниже среднего): silent возможен
  только при крэше клиента В МИРЕ до сигнала 40/3 — тогда TTL 5 мин.
- **T2-в (GS-logout снимает ли флаг) — ЗАКРЫТО 09.10 live**: Server64 → наш authd (2104) шлёт
  type=40/3 → флаг снят (users 1→0 в type=5), смена персонажа работает.
- **T2-а (TTL флага probe-циклом 30с) — открыт, приоритет низкий**: три пути снятия флага уже
  покрывают UX; probe-сверка TTL 5 мин = опционально. Гейт-лок `onlineTtlSec` остаётся 0 (ВЫКЛ).
- **T2-г (health-check probe)**: живые инструменты есть — `cmd/probe` / `cmd/forkprobe` (ниже).
- **T3 (CM_UPDATE_SESSION 0x08)**: relay реализован (type=0x08 prepend). Сорс-контракт:
  валидный reconnectKey → SM_UPDATE_SESSION (26Б), иначе `closeNow()`. Сценарий: уйти в мир →
  убить клиент → перезайти; теперь проверка на ЖИВОМ пути (наш authd R6). Форма поймана живьём 06.10.
- **T4 стабильность** — частично покрыт R6-циклами юзера (многократные логины/выходы без пауз);
  формальный прогон (5 логинов, 2 клиента параллельно) — осталось.
- **T5 паритет 42b-фолбэка — ЗАКРЫТО 10.10**: фолбэк 0x05 строил ad-hoc pt 39Б → wire 50 ≠ ориг;
  теперь `build26ReplyPt(0x05)` = `padLoginOK(4, buildServerListPayload(), charCount)` — 26Б канон-payload
  (fork-дамп ориг 07.10: `010101 c0a8007d 611e 6×0 f4010101 00000002 010001`, ip/порт/id из конфига) →
  pt 27/28Б → **wire 42 = байт-паритет с живым релеем** (forkprobe 10.10 10:23 live:
  `pt(32)=04010101 c0a8007d 611e…010001`, VERDICT SAME). Тест `TestFallbackServerListParity`.
  ⚠ Новый exe НЕ деплоен (фолбэк в проде не стреляет — authd жив); задеплоить при следующем
  «го»/поводе вместе с прочими правками.
- **T6 финализация+tag** — осталось (после T3/T4).

## 🔧 Режимы (`mode` в config.yaml)

- **authgate** — наш флоу 7.7 (ПРОД).
- **fork** — прозрачный прокси `клиент ↔ наш гейт ↔ ориг AuthGateD (forkOrigAddr:Port)`,
  байт-в-байт + shadow-сверка (FORK-строки в логе: скрамбл-модель, SAME/DIFF ответов).
  Использовался для сверки raw-байт: welcome/authgg = SAME vs ориг.
- **classic** — beyond-aion 4.8 флоу (standalone, SM_INIT-210).

## 🚀 Деплой

```bash
cd nextgen/aion-gate && go vet ./... && go test ./... \
  && GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o aion-gate.exe .
scp aion-gate.exe 'Администратор@192.168.0.125:D:/SAION/aion-gate/aion-gate.exe'
ssh Администратор@192.168.0.125 "cmd /c 'schtasks /end /tn AionGate & taskkill /F /IM aion-gate.exe'"
ssh Администратор@192.168.0.125 "cmd /c 'schtasks /run /tn AionGate'"
```

⚠️ Готчи: exe залочен живым процессом (kill до scp); после kill первый `/run` может
словить bind-fail — bat ретраит ~35с, проверять баннер; кириллический scp — только push
(pull через `ssh … powershell Get-Content`); L2Authd живёт ТОЛЬКО в session 1 (десктоп).

Верификация после деплоя: баннер `mode=authgate rsa_exponent=65537` + `RAW AUTHD A>G [03] 21c60000`
в `D:\SAION\aion-gate\gate-prod.log`; клиентский smoke — `cmd/forkprobe` (welcome-разбор +
AUTH_GG вердикт) или решающий логин.

## 🧪 Тесты и инструменты

- `go test ./...` — golden welcome/echo/фреймы + Фаза-1 (loginex-asm-blob, state-переходы,
  k=1 две раскладки 4.8/7.7, normalize, playOk Rnd, SplitLogin-op).
- `cmd/probe` — прямой клиент authd: `probe.exe 127.0.0.1:2110 stelgen 192.168.0.253 [sid] [tail47]`
  → type=3 = healthy (tail47 = blob-форма гейта 86Б, дефолт = asm 191Б).
- `cmd/forkprobe` — синтетический клиент 2106 (welcome→AUTH_GG→26b) с raw-вердиктами.

## ⚙️ Ключи конфига (config-prod.yaml на VM)

`serverPort:2106, authAddr/AuthPort:127.0.0.1:2110, mode:authgate, rsaExponent:65537,
authdTimeoutSec:15 (ЕДИНЫЙ бизнес-таймаут), onlineTtlSec:0 (гейт-лок ВЫКЛ, опция),
loginFailCode:1, playFailCode:8, loginFailOnline:7, failCloseSec:2 (T1-фейлы, 🛡️ выше);
loginTestFail/playTestFail = тест-крутилка (не юзаем, закомментирована),
loginDecbufLen:34, forkOrigAddr/Port:127.0.0.1:2109, worldIP:192.168.0.125, worldPort:7777,
welcomeTestCC:0, welcomeWaitAuthdMs:2000, serverID:1, smAuthGgWire:42 (live-форма;
50 = эталон Mobius для A/B), dumpPacket, blockIPsFile, tryInterval/Count/BlockInterval`.

## 📚 Доки

- `docs/fork-classic-deploy-20261006.md` — леджер поправок дня (№1–13: эталон ≠ live,
  asm-blob, session-1 стек) + runbook после ребута ВМ.
- `docs/mobius-77-flow-review-20261006.md` — сверка с эталоном Mobius (К-1..К-6, все закрыты;
  К-1 REVISED, К-3 live-поправка op=0x00).
- `docs/beyond-aion-48-…`, `docs/rsa-hunt-…` — АРХИВ (история гипотез, не актуальны).
- `reference/` (вне репо: `STELGEN/projects/aion_server_2026-10-02/reference/`) — клоны эталонов:
  `Mobius_AionEmu` (7.7, главный) и `beyond-aion/aion-server@4.8`. ПРАВИЛО: ответы по протоколу
  фейлов/сессии брать из сорса эталона (`loginserver/network/aion/...`), НЕ гадать; live-дамп — арбитр.
## 🔬 Cross-pulse 09.10 (из authd-чата): сервер-селект «Персонажи» = пусто

- 74Б login-ok = **SM_LOGIN_OK** (Packet Samurai Login_4.0: accountId/loginOk/?/?/1002/126282164/garbage47) —
  НЕ serverlist; байт [63] клиент игнорирует (эксперимент, вердикт №1).
- SM_SERVER_LIST = type=4 (42Б): 26Б payload = [listSize][lastServer][id][ip][port D][age][pvp][cur h][max h]
  [online][bits d][brackets][countsSize h=1][autoConnect] — **обрыв, count-байтов нет (и у ориг)** →
  колонка «Персонажи» всегда пустая. У НАШЕГО гейта флаг `serverListCharCount` (padLoginOK, теперь и в фолбэке T5): >0 добавляет
  count-байт в хвост type=4 (wire 42Б не меняется); **сейчас 0 = байт-паритет с ориг** — две попытки
  заполнения вердикта не дали, приоритет ниже среднего; гипотезы и история:
  ../aion-authd/docs/techdebt-charcount-20261009.md.

## 🔗 09.10 R6-финал (authd-трек): гейт свитчен на наш authd

- `authPort: 2110` — гейт ходит в **наш aion-authd** (задача AionAuthdProd, живой путь R6);
  fork-цепочка 2116 остановлена как путь (forkauthd жив, откат = rollback-gate.ps1 + authd-rollback.cmd).
- Сервер-селект: наш type=4 = SM_SERVER_LIST (26Б) — колонка «Персонажи» пустая у ВСЕХ (и у ориг):
  см. charcount-секцию выше (тех-долг ниже среднего).
- Статус сервера для клиента: ip/port для клиента берутся из `AionAccounts.dbo.server` (ip=192.168.0.125).

## 📊 Кросс-пульс 10.10 (гейт-чат, из пульса соседей за простой)

- **authd R6 online-логика (12:40 09.10, коммиты fc27473/46c0be4)**: `clearOnlineOnDisconnect: true` —
  [01] от гейта теперь снимает онлайн-флаг → крэш клиента больше НЕ лочит акк; мир 40/3 = основной
  путь; TTL 5 мин = страховка. T2-б/T2-в закрыты (см. §T2/T3 выше).
- **Прод-статус (op aionstatus, 12:57 VM 10.10)**: aion-gate RUNNING PID 5480 :2106, мир 8/8 conns
  :2002; алерты `down:gateorig` / `down:authd` = EXPECTED (ориги остановлены — замены живы).
  Ложных тревог по гейту нет; юзер-циклы логин/выход/логин в наблюдении R6 — без пауз.
- ⚠ **Риск от op (не наш баг)**: `stop authdn` в op бьёт `taskkill /F /IM aion-authd.exe` —
  УБИВАЕТ и ПРОД authd (общее имя exe у prod и тени) → гейт останется без 2110. НЕ жать stop
  authdn до разделения в op-конфиге (OP-1..OP-5: [aion-op/ROADMAP.md](../aion-op/ROADMAP.md) §6,
  [docs/tech-debt-stack-20261010.md](../../docs/tech-debt-stack-20261010.md)).
- **aion-main (сосед, R4.5+)**: свитч мира на Go-тень :7778 потребует правки `worldPort` в
  config-prod.yaml гейта (сейчас 7777) — деприор, только после полного MVP-мира (гейт = их свитч-точка).
- **CacheD64 (сосед, R1-prep)**: гейт не касается 2006; спам `proc_missing aion_GetItemCollection*/...`
  в op-алертах = мир-сторона (БД ~85%, см. docs/tech-debt-db-20261008.md), не гейт.

# aion-gate — АРХИТЕКТУРА И ВЫЖИМКА РЕШЕНИЙ (08.10.2026, релиз f8912a9, exe 7c4dcab)

> Полная карта «как работает и почему так решено». Источник истины по факту = live-дамп/форк-дамп;
> по формам/кодам фейлов = сорс эталона `reference/Mobius_AionEmu` (клон: STELGEN/projects/aion_server_2026-10-02/reference/).
> История ошибочных догадок (лиджер №1–16): docs/fork-classic-deploy-20261006.md.

## 1. Топология и поток

```
AION клиент 7.7 EU ──2106──▶ aion-gate (Go, mode:authgate) ──2110──▶ L2Authd (NC-бинарь, session 1)
                                  │ welcome 194B (key1 static)          │ authd wire [00]/[01]/[02]/[03]
                                  ◀── релей authd-ответов ──────────────┘
успех: type=3 → [03]74Б login-ok → [05] relay → [04]42Б server-info → [02] relay → [07]26Б play-ok → мир 7777 (Server64)
```

- PA/PortalAuth НЕ участвует в НАШЕМ пути гейта, но ЖИВОЙ В СТЕКЕ ОБЯЗАТЕЛЕН (07.10: без PA ориг
  отклоняет логин SYSTEM_ERROR(20) — authd сам держит акки через L2Conn.dsn→AionAccounts,
  авторегистрация по ASCII-логину, пароль НЕ проверяется; не-ASCII логин = 18Б LoginFail).
- Гейт = замена AuthGateD. fork-proxy/mode:fork — инструмент A/B с оригиналом, НЕ прод.

## 2. Крипта и фрейминг (live-доказано, НЕ ТРОГАТЬ)

- Blowfish **LittleEndian** dword-порядок (L2-вариант) — big-endian был корневой багой эпохи 06.10.
- key1 = LUT-генератор (seed 0x04bd) static `6b60cb5b82ce90b1cc2b6c556c6c6c6c`; key2 = LUT[r]×4 per-conn (r=rand8).
- Frame: `[u16 LE total][ECB]`; welcome 194 = `[0x00][0xC2 00]`? — фактически `[u16 0xC2][ECB 192]`;
  pt 173+4: `[0]=0x00, [1:5]=fc=sid, [5:9]=V (authd [03], live 0xc621), [9:137]=ScrambleModulusServer(N),
  [137:153]=GG-нули, [153:169]=key2, [168:172]=65650072, [172]=0x00`.
- EncryptPrimary: roundup8 → dword-скрамбл cumsum (dword0 не трогается) → csum dword → ECB(+8).
- EncryptSecondary/DecryptSecondary @0x417a80/0d: csum=XOR dword'ов сразу после данных, pad после csum.
- **Серверный скрамбл модуля** swap→xorL→dword→xorU (= EncryptedRSAKeyPair.encryptModulus гита;
  клиентский unscramble — точная инверсия; перепутанные направления = «decbuf мусор» эпоха).
- RSA-1024, **e=65537 (F4)** — доказано оракулом m^65537 mod N == ct; e=17 (Pub.key[1]) = МЁРТВАЯ
  гипотеза; rsaExponent конфиг; пул ключей, ключ per-session.
- CM_LOGIN (op **0x00** у live EU-клиента; 0x0B = эталон Mobius — принимаем ОБЕ): pt=[op][ct 128×k][tail≤64];
  RSA nopadding; loginex k≥2: user=combined[78:142] (после DecryptSecondary), pwd на [220:230] (authd
  пароль игнорит), otp=[238:242]=0; username → TrimSpace+ToLower (автосоздание чувствительно к регистру).
- Blob в authd "cbdb" — asm-форма **191Б** `[00][decbuf34=user14+pwd16+otp4][dword=pt[148:152]][tail=pt[152:]]`
  ровно как оригинал; authd игнорит содержимое dword/tail, но требует asm-РАЗМЕР (86Б → тишина на существующих акках).
- Релей authd→клиент: **клиентский опкод = тип от authd** (prepend); type=3→`[03]+pad64`=74Б, type=4→42Б (pt 32), type=7→26Б (pt 16); type=1 → pt 5Б = SM_LOGIN_FAIL-форма (18Б) — авторетрансляция.

## 3. State-машина и диспетчер

`CONNECTED{0x07→authgg, 0x08→UPDATE_SESSION relay} → AUTHED_GG{0x00|0x0B→login} → AUTHED_LOGIN{0x05,0x02→relay}`;
unknown = лог+raw-relay (НЕ cc45, НЕ рвать). По эталону AionPacketHandlerFactory (Mobius); длины 32/24/≥184 —
fallback-эвристика (leak-клиент шлёт 312/314b). CM_AUTH_GG: клиент 34b = EncryptSecondary([sid][20Б]),
ответ 42b = EncryptSecondary([0b][sid][27×0]) (live; 50-форма эталона = опция smAuthGgWire:50).

## 4. Фейлы (T1) — формы, коды, тексты, триггеры

- Формы из сорса: SM_LOGIN_FAIL = `[01][D messageId]` (5Б) → wire 18; SM_PLAY_FAIL = `[06][D messageId]`;
  SM_UPDATE_SESSION = `[0c][D accountId][D loginOk][C 0]` → 26Б.
- Реестр messageId 0–22 (+45 authgate-спец) — AionAuthResponse.java; **live-тексты клиента 7.7 EU
  захардкожены** (authFailText) и логируются на КАЖДУЮ выдачу:
  `SM_*_FAIL -> КЛИЕНТ: messageId=N (NAME) текст="..." sid ip (failClose=2s)` + ship-событие text.
  Ключевые: 1 «Ошибка авторизации…», 2/3 «Неверный логин или пароль.», 7 «Вы уже залогинись.»,
  8 «Выбранный сервер временно недоступен…», 22 «Ваш аккаунт заблокирован.», 45 «…только после
  авторизации на главной странице сайта». Полная таблица — README §🛡️.
- Триггеры: **authdTimeoutSec=15 единый** (нет type=3 → LOGIN_FAIL(1); нет type=4/7 → PLAY_FAIL(8)) —
  юзер: «рвать сессии секунд за 15 с подходящей ошибкой»; failCloseSec=2 (close после фейла; фрейм
  уже в TCP-буфере). Гейт-лок relogin `onlineTtlSec` — **ВЫКЛ (0)** по решению юзера («пусть логинятся
  как могут»); при N>0 — немедленный LOGIN_FAIL(7) + relay blob (kick-семантика эталона).
- Логин в тест-крутилке (loginTestFail) НЕ релеится → authd не видит → акк НЕ лочится.

## 5. Почему так: решения vs отвергнутые гипотезы (краткая выжимка)

| Решение | Обоснование |
|---|---|
| opcode-first welcome, LE-Blowfish, DecryptSecondary-до-RSA | live-дамп 06.10 (третья корневая бага за день) |
| e=65537, серверный скрамбл | оракул m^65537==ct на живом логине; клиентский unscramble = инверс |
| op 0x00 принимаем наряду с 0x0B | live-клиент шлёт 0x00 (форк-дамп); эталон Mobius = 0x0B |
| asm-blob 191Б (не 86Б) | probe: stelgen 191Б → type=3, 86Б → тишина |
| эталон ≠ live: арбитр = форк-дамп | README canon: «эталон Mobius ≠ live leak-клиент» |
| фейлы из сорса, не догадки | юзер-правило 08.10; формы совпали с live-18Б 1-в-1 |
| welcome 194 variant-0 только | эпоха probe-вариантов завершена (клиент принял variant-0) |
| authd молчит на relogin онлайн-акка | authd-инхерент (флаг TTL ~2-6 мин, [01] не снимает) → гейт отвечает 1 через 15с |

## 6. Эксплуатация

- **Деплой**: `go vet ./... && go test ./... && GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o aion-gate.exe .`
  → taskkill → scp `D:\SAION\aion-gate\aion-gate.exe` → `schtasks /run /tn AionGate` → баннер.
- **Канарейка баннера**: `rsa_exponent=65537` = конфиг прочитан; `rsa_exponent=17` = конфиг ПУСТ/не читан
  (инцидент 08.10: WriteAllBytes 0 байт после python open('w')+encode → гейт на дефолтах) — немедленный откат.
- **Правка config-prod.yaml**: только байтово (b64→WriteAllBytes→LEN-check; PS через scp+File, cmd-кавычки
  через ssh ломаются; файл UTF-8 — python только utf-8, комменты латиницей; НИКОГДА open('w') в одном
  выражении с encode). Бекапы: exe.bak-<commit>; пулы b64 в C:\Temp.
- **Стек**: L2Authd/весь стек живут ТОЛЬКО в session 1 (SYSTEM/session-0 = тихая смерть); после ребута
  ВМ — AION-START-ALL-v6.bat → свап гейта; taskkill /F /IM всегда (schtasks /end процессы не убивает);
  exe залочен живым процессом — kill до scp; рестарт L2Authd: /end+/run AionAuthIT → wait 2104 → /end+/run AionGate.
- **Откат**: exe.bak-63f6a47 / 9c85c12 / 9f2da98 / e1dd475 на месте; крутилка: раскомментировать
  `loginTestFail: -1` + рестарт.
- **Инструменты**: `cmd/probe` (authd health: `probe.exe 127.0.0.1:2110 [user] [ip] [sid] [tail47]` — type=3
  healthy; probe-логины ЛОЧАТ акк на TTL authd); `cmd/forkprobe` (синтет-клиент 2106: welcome-разбор + AUTH_GG verdict).
- **Ship-телеметрия** (TELEMETRY-SPEC): syslog/http/file, не критичный путь; события login/onlinefail/
  timeout/play.testfail с полем text.

## 7. Открытые задачи (по ТЗ)

- T2-а: точный TTL онлайн-флага authd (probe-цикл 30с; флаг снимает ли GS-logout).
- T3: CM_UPDATE_SESSION (0x08) живьём (уйти в мир → kill клиент → перезайти; контракт сорса:
  валид reconnectKey → SM_UPDATE_SESSION 26Б, иначе closeNow; relay реализован).
- T4: стабильность (5 логинов подряд, 2 клиента параллельно, рестарт гейта без рестарта L2Authd при authReconnectInterval=30).
- T5: ✅ ЗАКРЫТО 10.10 — фолбэк 0x05 = `padLoginOK(4, buildServerListPayload()[26Б канон fork-дампа ориг], charCount)`
  → pt 27/28Б → wire 42 (был ad-hoc pt 39Б → wire 50); live-паритет подтверждён forkprobe 10.10
  (релей type=4 = те же байты, SAME); тест `TestFallbackServerListParity`; exe НЕ деплоен (фолбэк в проде не стреляет).
- T6: git tag gate-7.7-final после T2–T4 (T5 ✅ 10.10).
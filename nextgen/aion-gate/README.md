# aion-gate — Go-замена AuthGateD (AION 7.7 PTS EU)

Гейт-логин: клиент :2106 → гейт → authd :2110. Протокол снят дизasm'ом AuthGateD,
сверен с гитом beyond-aion 4.8 и **полностью расшифрован на живом клиенте 06.10.2026**.
Референс-доки: `docs/` (beyond-aion-48-protocol-vs-gate, fork-classic-deploy, rsa-hunt).

## 🏆 ИТОГ ПРОТОКОЛА — ВСЁ РАСКРЫТО (живые доказательства 06.10)

### Крипта (доказано байт-в-байт на проде)
- Фрейм: `[u16 LE total][BF-ECB]` (total включает len-филд). Blowfish **little-endian**,
  канонические P/S. Static key1 = `6b60cb5b82ce90b1cc2b6c556c6c6c6c` (= LUT[0x04bd]).
- `EncryptPrimary` (первый пакет, key1): cumsum-цепочка от **dword[0]** (dw0 не трогается),
  `new[k]=old[k]^S`, финальный S → `[n:n+4]`, `[n+4:n+8]` нули. `DecryptPrimary` — инверс.
- Сессия: key2 = LUT-ключ (client знает из welcome), фреймы `[data][chk][pad]`, chk = XOR dword.
- **Скрамбл модуля**: сервер шлёт `ScrambleModulusServer` (гит-порядок swap→xorL→dword→xorU);
  клиент снимает своим unscramble (`ScrambleModulus`, порядок инверсный). **ДОКАЗАНО НА ЖИВОМ
  ОРИГЕ дважды**: `ServerScramble(unscramble(wire)) == wire`, N_orig = 1024 бита.
  Отправлять клиентский unscramble-порядок или RAW-модуль = гарантированный мусор (не инволюции).

### Welcome (SM_INIT, 194B) — 7.7 EU форма, live-принята
`pt(177→192 ECB)` = `[00][sid D][V D][mod128 ServerScramble][gg 16×0][key2 16][65 65 00 72][00]+4×0`.
`V` = authd `[03]`-payload (живой: **0x0000c621**). `EncryptPrimary(key1)` → wire 194.
Client echo: `authgg [07][sid][27×0][...PRO-резидент]` → наш ответ `[0b][sid][27×0]` =
**байт-в-байт SAME с оригом** (fork-сравнение).

### RSA / CM_LOGIN — ГЛАВНАЯ РАЗГАДКА
- **e = 65537 (F4), ДОКАЗАНО** оффлайн-оракулом `m^65537 mod N_orig == ct` на живом ct юзера
  (m^17 опровергнут). Гит beyond-aion прав, «e=17 из Pub.key» — артефакт.
- 7.7 клиент шлёт **loginex всегда**: `pt = [op][ct1 128][ct2 128][tail ВАРИАТИВНЫЙ 45-55Б]`
  (tail: sid LE, нули, `0x20`, `68ffdab3e2fda892`, `2d9cc7baa87e0d49`, `00000000` — структура
  константна, длины плавают). Фикс-хвост 55Б гита НЕ работает.
- Раскладка (combined = m1||m2): **user = combined[78:142]** (блок1[78:128]+блок2[0:14]) —
  подтверждено HIT'ом; pwd = combined[206:238] — во 2-м блоке есть иные данные (ct2 не сводится
  к простым формам), **authd пароль НЕ проверяет** — не блокирует. decbuf для authd =
  `BuildLoginDecbuf`: user14+pwd16+otp4 = 34Б (asm arg3=0x22).
- Правило резки: `SplitLogin: k = (len(pt)-1)/128, rem ≤ 64` (55-константа убита).

### Релей authd (wire 2110) — контракт живой
- gate→authd: `[00][sid][IP]` CltConnect | `[01][sid]` Disconnect | `[02][sid][len][blob]`;
  authd→gate: `[01][id]` | `[03][sid]` | `[02][id][len][type][payload]`.
- **Клиентский ОПКОД = ТИП authd** — гейт ОБЯЗАН prepend байт типа к payload:
  type=3 → `[03]+payload+пад до 64Б` = wire 74 (login-ok); type=4 → `[04]` = wire 42 (server-info);
  type=7 → `[07]` = wire 26 (play-ok). Без байта типа клиент молча игнорит ответ.
- `[05]`/`[02]` от клиента РЕЛЕЯТСЯ в authd (эмуляция 42b/26b — только фолбэк без authd).
- blob login = `cbdb`: `[00][decbuf34][dw][tail клиента]` — authd принял (type=3, accountId).
- LIVE-ПРУФ 19:12: наш blob → authd type=3 (accountId=7, токены) — вся цепочка работает.

### ТРИ КОРНЯ старого «decbuf мусор» (все убиты, не повторять!)
1. Скрамбл: шла клиентская unscramble-форма вместо серверной (см. выше).
2. `handleLogin`/`handle26`/default-релей получали **RAW ECB(key2) без DecryptSecondary**
   (RSA глушил ещё зашифрованные байты; handleAuthGG расшифровывал — потому authgg «жил»).
3. Relay без байта-типа (клиент получал оп 0x07 вместо `[03]`).

### Реестр кодов LoginFail/PlayFail = messageId AionAuthResponse (P2-7, эталон Mobius 7.7)
cc-канал и SM_LOGIN_FAIL/SM_PLAY_FAIL шлют один и тот же D messageId; 45 = authgate-спец.

| Код | Константа | Смысл |
|----|-----------|-------|
| 0 | AUTHED | успех |
| 1 | SYSTEM_ERROR | системная ошибка |
| 2 | INVALID_PASSWORD | неверный пароль |
| 4 | FAILED_ACCOUNT_INFO | ошибка данных аккаунта |
| 5 | FAILED_SOCIAL_NUMBER | ошибка соц. номера |
| 6 | NO_GS_REGISTERED | нет зарегистрированного GS |
| 7 | ALREADY_LOGGED_IN | уже в игре |
| 8 | SERVER_DOWN | сервер недоступен |
| 10 | NO_SUCH_ACCOUNT | нет такого аккаунта |
| 11 | DISCONNECTED | разрыв |
| 12 | AGE_LIMIT | возрастное ограничение |
| 15 | SERVER_FULL | сервер полон |
| 16 | GM_ONLY | только GM |
| 18 | TIME_EXPIRED | время истекло |
| 21 | ALREADY_USED_IP | IP уже используется |
| 22 | BAN_IP | аккаунт заблокирован (**live-подтверждено** 06.10: клиент показывает «заблокирован») |

## РЕЖИМЫ (mode в config.yaml)
- **authgate** — живой 7.7 EU флоу (ПРОД СЕЙЧАС). welcome 194/EncryptPrimary/key1,
  диспетчер по длинам (32=authgg, 24=26b-пинги, ≥184=login), релей authd.
- **classic** — beyond-aion 4.8 флоу: SM_INIT pt192 (`0x0000c621`, magic `0x3FCE09ED`) →
  `EncryptGitInit` (**Java-пад `length += 8 - length%8` при кратности 8 добавляет ЕЩЁ 8**:
  192→208→wire 210; для 7.7 pt=177→192 — совпадает с EncryptPrimary кроме сида Rnd vs dword[0]);
  диспетчер (op,state) CONNECTED/AUTHED_GG/AUTHED_LOGIN; SM_AUTH_GG 32Б/гит-37Б(wire 50,
  ggXorTail); SM_LOGIN_OK/SERVER_LIST/PLAY_OK/LOGIN_FAIL. Standalone, без authd.
- **fork** — прозрачный прокси клиент↔ориг (forkOrigAddr:Port): welcome ориг расшифровывается
  → sid/V/модуль/key2 в лог; фреймы обеих сторон расшифровываются key2 ориг; shadow-ответы
  нашего движка vs ориг (SAME/DIFF); LOGIN → полный дамп ct+N_orig. authd НЕ подключаем.
  **Готча: ReadFrame возвращает ECB без len-филда — клиенту форвардить WriteFrame(payload).**

## ЗАПУСК НА VM (192.168.0.125) — ПОРЯДОК ПОСЛЕ РЕБУТА
⚠ **L2Authd и весь стек живут ТОЛЬКО в SESSION 1** (desktop). SYSTEM/session-0 запуск
L2Authd = мгновенный тихий выход (проверено дважды). Батники стартуют с рабочего стола
или через /IT-задачи.
1. **`AION-START-ALL-v6.bat` с рабочего стола** (SQL→Cache 2220→logd 2051→IC 2005→
   CAPTCHA 22206→PA 10057→**L2Authd 2104**→AuthGateD→CacheD 2006→NPC→Server64 7777;
   мир собирается 10-25 мин, 16 NPC-коннектов). Задача `AionAuthIT` переопределена на
   `C:\Temp\auth.bat` → `call C:\Temp\start-all.bat` (копия v6) — запуск в session 1.
2. Свап гейта на 2106 (v6-бат поднимает ориг на 2106!):
   `taskkill /F /IM AuthGateD.exe & taskkill /F /IM fork-proxy.exe & schtasks /run /tn AionGate`
   Задачи: `AionGate` = наш aion-gate (2106), `AionGateOrig` = ориг (2109, откат),
   `AionAuth` = **DISABLED** (SYSTEM-инстанс, не запускать!), `AionForkProxy` = **DISABLED**
   (старый внешний форк-прокси — перехватывал 2106 после ребута!).
3. Проверка: `netstat 2104/2106/2109/7777` + баннер `mode=authgate rsa_exponent=65537`
   в `D:\SAION\aion-gate\gate-prod.log` + `RAW AUTHD A>G [03] 21c60000`.

## ДЕПЛОЙ ГЕЙТА
```bash
cd nextgen/aion-gate && go vet ./... && go test ./... \
  && GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o aion-gate.exe .
scp aion-gate.exe deploy/prod/config-prod.yaml 'Администратор@192.168.0.125:D:/SAION/aion-gate/'
ssh Администратор@192.168.0.125 "cmd /c 'schtasks /end /tn AionGate & taskkill /F /IM aion-gate.exe'"
ssh Администратор@192.168.0.125 "cmd /c 'schtasks /run /tn AionGate'"
```
- ⚠ `schtasks /end` НЕ убивает процессы задачи — всегда добавляй `taskkill /F /IM`.
- ⚠ exe заблокирован пока процесс жив — scp только после taskkill.
- ⚠ scp с кириллическим логином работает в форме `cd … && scp file 'Администратор@VM:D:/…'`.
- Откат на форк: `config-prod.yaml → mode: fork` + рестарт задачи.

## ДИАГНОСТИКА
- `cmd/probe` — health-check authd напрямую: `probe.exe 127.0.0.1:2110 [user] [ip] [sid]`
  (шлёт [00]+login-blob, ждёт type=3; **обязательно шлёт [01]-дисконнект** — иначе течёт сессия).
- fork-режим: строки `FORK ...` в gate-prod.log (welcome-разбор, N_orig, ct-дамп, SAME/DIFF).
- Лог тянуть: `ssh … "cmd /c 'type D:\SAION\aion-gate\gate-prod.log'" > local`
  (кириллица в PS-выводе мажется — hex читаем; grep по таймстампу).
- authd флейкит на повторные логины в живую сессию → рестарт L2Authd (session 1!).

## ТЕСТЫ
`go test ./...` — blowfish-векторы, static/key2 LUT, EncryptPrimary known-answer,
welcome-194, DecryptPrimary roundtrip, скрамбл-инверс (200/200) + не-инволюция,
welcome-210 (classic, Java-пад), SM_AUTH_GG формы, CM_LOGIN golden (не-loginex/loginex/
304Б-динамический хвост), BuildLoginDecbuf, checksum-layouts, RSA e∈{17,65537},
E2E (welcome→authgg→login→relay→serverlist), fork-passthrough, authdclient wire.

## АРТЕФАКТЫ/ИСТОРИЯ
- `docs/beyond-aion-48-protocol-vs-gate-20261006.md` — карта протокола vs гит 4.8 (§7 = спека фиксов).
- `docs/fork-classic-deploy-20261006.md` — деплой fork/classic + топология.
- `docs/rsa-hunt-20261006.md` — история охоты (мертвые теории НЕ возвращать: Pub.key,
  m=[login][md5], «e=17 фикс решает»).
- Креды тестовые: authd авто-создаёт аккаунты, пароль не проверяет (1/1 = аккаунт «1»).
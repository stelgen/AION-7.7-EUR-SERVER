# beyond-aion/aion-server (4.8, HEAD 6c6eee6) vs aion-gate — карта протокола + root-cause decbuf-мусора

Дата: 06.10.2026. Источник: `/tmp/ba48` (git clone, branch 4.8), login-server `com.aionemu.*`.
Гейт: `nextgen/aion-gate` (internal/proto, internal/server). Код НЕ менялся — только анализ.

---

## 0. ВЕРДИКТ (3 строки)

1. **Протокол один и тот же.** Эмулятор 4.8 реализует ту же схему, что снята дизasm'ом с AuthGateD:
   фрейм `[u16 LE][BF-ECB]`, opcode-first, RSA-1024 nopadding, скрамбл модуля, session-key в welcome,
   опкоды `0x05→0x04`, `0x02→0x07` — всё совпадает с нашим живым маппингом.
2. **Root-cause «decbuf = мусор» НАЙДЕН В ГИТЕ:** наш `ScrambleModulus` — это КЛИЕНТСКИЙ unscramble
   (порядок xorU→dword→xorL→swap). Сервер обязан отправлять модуль в СЕРВЕРНОМ порядке
   swap→xorL→dword→xorU (т.е. `EncryptedRSAKeyPair.encryptModulus` из гита — математическая инверс
   нашей функции). Гейт шлёт «анскрамбленный» модуль → клиент анскрамбит ЕЩЁ РАЗ → `N_client ≠ N_our`
   → RSA-decbuf = случайщина. Закрывает гипотезу №2 из `docs/rsa-hunt-20261006.md`.
3. **e в гите = 65537 (F4)**, а не 17. Наше «e=17» из клиента под вопросом → сделать конфигом и
   откалибровать живым тестом (креды известны из fork.log).

---

## 1. Фрейминг и крипто (совпадения ✓)

| Элемент | beyond-aion 4.8 | Наш гейт | Статус |
|---|---|---|---|
| Фрейм | `[u16 LE total][ECB]`, total включает len-филд (`AConnection`+`LoginConnection.encrypt`) | `WriteFrame/ReadFrame` | ✓ 1-в-1 |
| Blowfish | ECB, канонические P/S (`BlowfishCipher`), dword LE | `blowfish.go` + `blowfish_const.go` (LE-фикс) | ✓ |
| Чексумма сессии | `CryptEngine.verifyChecksum/appendChecksum`: XOR всех dword == 0, последний dword = XOR предыдущих | `DecryptSecondary/EncryptSecondary` | ✓ семантика та же |
| Позиция чексуммы | `[data][pad][chk]` (pad перед chk) | `[data][chk][pad]` (chk сразу за roundup8) | ⚠ разные байты, оба валидны при pad=0 (клиент-верификатор один: XOR==0) |
| Static key (init) | `CryptEngine`: `6b 60 cb 5b 82 ce 90 b1 cc 2b 6c 55 6c 6c 6c 6c` | `StaticKeyHex = "6b60cb5b82ce90b1cc2b6c556c6c6c6c"` в keys.go; но фактически шлём `GenerateInitialKey(0x04bd)` | ⚠ сверить TestStaticKey: если 0x04bd ≠ 6b60… — у нас свой LUT-статик (live-подтверждён), оставить наш |
| Смена ключа | static → session key доставлен В SM_INIT (`LoginConnection.java:323-326 updateKey`) | key2 в welcome, `BF2` per-session | ✓ тот же слот [153:169] |

## 2. SM_INIT / welcome (op 0x00) — слоты

Гит `SM_INIT.java` plaintext = **192B**:

```
[0]      = 0x00 opcode
[1:5]    = sessionId (D)
[5:9]    = 0x0000c621 protocol revision (D)
[9:137]  = encryptedModulus 128B (EncryptedRSAKeyPair.encryptModulus)
[137:153]= 16×0
[153:169]= session Blowfish key 16B
[169:176]= 7×0
[176]    = test server id (C, 0)
[177:181]= test server ip (D, 0)
[181:183]= test server port (H, 0)
[183]    = flag (C, 0)
[184:188]= 0x3FCE09ED (D)
[188:192]= 0 (D)
```

Шифрование первого пакета (`CryptEngine.encrypt`, `updatedKey=false`): `len+4(key)+4(chk)` → pad до 8
(192+8=208) → `encXORPass`: cumsum-цепочка с **рандом-сидом** (dword[0] не трогается, chain от dword[1],
`ecx += edx; edx ^= ecx`), финальный `ecx` пишется в `[200:204]`, `[204:208]` = chk-нули → BF(static) → wire **210**.

Наш welcome (`welcome.go`, live-verified) plaintext = **177B**:

```
[0]=00, [1:5]=fc(sid), [5:9]=V(authd [03]), [9:137]=модуль (variant4=RAW),
[137:153]=GG-нули, [153:169]=key2, [169:173]=65 65 00 72, [173]=00, +4×0
EncryptPrimary: chain с сидом dword[0], roundup 184, csum@184, [188:192]=0 → BF(key1) → wire 194
```

**Совпадают слоты:** [0], [1:5], [9:137], [137:153], [153:169]. **Расходятся:** dword[5:9] (rev vs V —
наш V=authd live-принят; classic-режиму дать конфиг `welcomeDword1: authd|c621`), хвост, и формат
первого пакета (random-seed+stored vs dword0-seed). Для 7.7 EU клиента рабочая форма — НАША (194,
EncryptPrimary, key1). Classic-режим (4.8-клиент) должен использовать форму гита (210, encXORPass,
static 6b60…).

## 3. СКРАМБЛ — главный фикс

Гит `EncryptedRSAKeyPair.encryptModulus` (сервер):
1. swap `m[0..4) ↔ m[0x4d..0x51)`
2. `m[i] ^= m[0x40+i]` для i<0x40 (lower ^= upper)
3. `m[0x0d+i] ^= m[0x34+i]` для i<4
4. `m[0x40+i] ^= m[i]` для i<0x40 (upper ^= lower, обновлённый)

Наш `ScrambleModulus` (rsa256.go): xorU → dword → xorL → swap — **это клиентский unscramble**
(инверс гитовского порядка; каждая операция инволюция, обратный порядок = инверс композиции —
математически строго). Живой roundtrip из rsa-hunt это подтверждает: наша функция снимает скрамбл
оригинала.

⇒ Гейт сейчас применяет к raw-N КЛИЕНТСКИЙ алгоритм, клиент применяет его же ещё раз → `unscramble(unscramble(N)) ≠ N`.
Variant 4 (RAW) тоже ломается: `unscramble(raw N) ≠ N`. Обе ветки дают мусорный decbuf — ровно как в логах.

> Верифицировано численно (саб-агент, 500/500 случайных 128B): `F_user(F_git(N)) == N` (инверс точный);
> `F_git(F_git(N)) ≠ N` и `F_user(F_user(N)) ≠ N` — обе функции НЕ инволюции ⇒ отправка
> `ScrambleModulus(N)` или RAW-модуля гарантированно даёт клиенту чужой N.

**Фикс:** серверный скрамбл = порядок гита (swap→xorL→dword→xorU). Достаточно развернуть порядок шагов
в `ScrambleModulus` (или добавить `ScrambleModulusServer` = 4 шага в обратном порядке).

## 4. RSA / CM_LOGIN (op 0x00, state AUTHED_GG)

Гит `KeyGen`: RSA-1024, **e = RSAKeyGenParameterSpec.F4 = 65537**, пул 10 ключей.
Расшифровка: `RSA/ECB/nopadding`, по 128B-чанкам (`CM_LOGIN.decryptLoginData`).

`CM_LOGIN` plaintext: `readB(remaining-55)` = RSA-чанки, затем хвост 55B:
`[sid D][16×0][7B: 20 00 00 00 00 00 01][16B: 9D DA 47 A7 21 C0 A6 A5 4B B7 5E E3 CE C9 26 AA][D 0][D unk][D 0]`.

Раскладка расшифрованного чанка (в документации гита, с примерами байтов):
- без `-loginex` (1 чанк): username **@94:108 (14B)**, password **@108:124 (16B)**, otp **@124:128** (LE i32, `FFFFFFFF` если не используется); ведущие нули — заполнение
- c `-loginex` (>1 чанк): username @78 (64B) в чанке 1, password @78 (32B) в чанке 2, otp следом; чанки сдвигаются влево
- charset Cp1252; `isLoginEx = len(encryptedLoginData) > 128`

Наш `handleLogin`: `data[:128]` → `DecryptBlock` → decbuf(32/128 DIAG) + dword148 + tail → relay «cbdb» в authd.
После фикса скрамбла decbuf станет реальным plaintext — разбор по раскладке гита, в authd отдавать
`[login][password]` (не мусор). Валидный чанк = ASCII username@94, ASCII pwd@108, otp=FFFFFFFF —
это ГОТОВЫЙ ОРАКУЛ для выбора e.

**e: 17 vs 65537.** Гит работает на 4.8-ритейле с F4. Наш «e=17» (Pub.key[1]) мог быть ошибкой чтения.
С фиксом скрамбла: расшифровать живой ct (creds известны: stelgen/1 и т.д.) с d(17) и d(65537) —
какой даст валидную раскладку, тот и есть. Сделать `rsaExponent` конфигом (default 17, switch 65537).

## 5. State machine и опкоды (гит = ритейл KOR 8.2.22, комментарий в AionPacketHandlerFactory)

Клиент (AQ_*): 0=LOGIN, 1=SERVER_LIST, **2=ABOUT_TO_PLAY**, 3=LOGOUT, 4=LOGIN_MD5, **5=SERVER_LIST_EX**,
6=SCCHECK, **7=GAMEGUARD**, 8=UPDATE_SESSION_REQ, 9=WEBSESSION_LOGIN, 10=OTPCHECK, 11=EXTERNAL_TOKEN_LOGIN,
12=AUX_AUTH_ACK, 16=IOVATION_CHECK, 18=LOGIN_TOKEN.
Сервер (AC_*): 0=PROTOCOL_VER, **1=LOGIN_FAIL**, 2=BLOCKED_ACCOUNT, **3=LOGIN_OK**, **4=SEND_SERVER_LIST**,
5=SEND_SERVER_FAIL, **6=PLAY_FAIL**, **7=PLAY_OK**, 8=ACCOUNT_KICKED, 9=BLOCKED_ACCOUNT_WITH_MSG, 10=SCCHECK_REQ,
**11=GAMEGUARD**, 12=UPDATE_SESSION_ACK, 13=OTPCHECK_REQ…

Переходы: CONNECTED{0x07→CM_AUTH_GG, 0x08→CM_UPDATE_SESSION} → AUTHED_GG{0x00→CM_LOGIN} →
AUTHED_LOGIN{0x05→CM_SERVER_LIST, 0x02→CM_PLAY}.

**Наш гейт УЖЕ совпадает по опкодам живьём:** 26b op 0x05 → отвечаем [04] (AC_SEND_SERVER_LIST),
op 0x02 → отвечаем [07] (AC_PLAY_OK); AUTH_GG=0x07, SM_AUTH_GG=0x0b, cc=[01]=AC_LOGIN_FAIL (plain).
Но диспетч у нас по ДЛИНЕ, а не по (op,state) — classic-режиму нужен opcode-диспетчер.

## 6. Пакеты, которых у нас нет / отличаются

- **SM_AUTH_GG (0x0b)** гит: 40B pt `[sid][0×4][0xCD5000][0][0x0b<<24][sid^0xCD5000][3×0]` (wire 50);
  наш live-вариант 32B `[0b][sid][27×0]` (wire 42) — подтверждён 7.7 клиентом. Classic-режиму — форма гита (конфиг-флаг).
- **SM_LOGIN_OK (0x03)**: `[accountId][loginOk][0][0][0x000003ea][0×7][0x13×0]`. У нас не шлётся (всё в authd-relay).
- **CM_SERVER_LIST (0x05)**: `[accountId][loginOk][C=7][6B][D][D]` → проверка `checkLogin(accountId, loginOk)`.
- **SM_SERVER_LIST (0x04)** полный формат: `C(count), C(lastServer)`, на сервер: `[C id][B ip4][H port][H 0][C age][C pvp][H cur][H max][C online][C type][C hide][H 0][C brackets]`, затем `H(maxIdWithChars+1), C(1 автолинк), charCount[] , 13×0`.
  Наш 32b-ответ [04][01 01 01][ip][port][…] — упрощение; classic-режим строит полный формат (lastServer + char counts из authd).
- **CM_PLAY (0x02)**: `[accountId][loginOk][servId C][6B][Q random]` → `SM_PLAY_OK (0x07): [playOk1 D][playOk2 D][servId C][14×0]`.
- **SessionKey**: `accountId = Account.getId()`, `loginOk=Rnd, playOk1=Rnd, playOk2=Rnd`; `checkLogin(accountId, loginOk)`;
  playOk1/2 передаются в GS (CM_ACCOUNT_RECONNECT_KEY). Наш хардкод `[07][1][1010][1]` заменить
  на реальные ключи + прокинуть их в authd/GS.

## 7. ПРОМПТ НА ИСПРАВЛЕНИЕ (вставить агенту как есть)

```text
Проект: nextgen/aion-gate (Go). Референс протокола: beyond-aion/aion-server ветка 4.8
(login-server, com.aionemu.loginserver) — клонировать shallow, НЕ копировать Java, реализовать в Go
по снятой раскладке. Цель: режим "classic" (beyond-aion-совместимый флоу) + РАЗВОРОТ скрамбла.
Мой код не эмулирует ничего — собираем/парсим пакеты по известной раскладке.

ПРИОРИТЕТ-1 (root-cause decbuf=мусор): в internal/proto/rsa256.go ScrambleModulus сейчас =
клиентский unscramble (порядок: xorU -> dword(0x0d^0x34) -> xorL -> swap). Сервер должен слать
модуль в СЕРВЕРНОМ порядке (инверс): (1) swap m[0..4)<->m[0x4d..0x51); (2) m[i]^=m[0x40+i] i<0x40;
(3) m[0x0d+i]^=m[0x34+i] i<4; (4) m[0x40+i]^=m[i] i<0x40 — это EncryptedRSAKeyPair.encryptModulus
из гита. Добавь ScrambleModulusServer с этим порядком (старую функцию НЕ удалять — она клиентская,
пригодится для калибровки/тестов), в welcome.go используй серверную (учти: plain() скрамблирует БЕЗУСЛОВНО, BuildWelcomeVariant — при variant!=4). Не трогай variant-механику:
variant 0/1/2/3/5-9 остаются, но теперь variant 0 = ScrambleModulusServer, а RAW-вариант (4) пометь
в логе как "математически сломан (unscramble(raw)!=N)".

ПРИОРИТЕТ-2 (RSA e): в internal/config/config.go добавь rsaExponent int (yaml rsaExponent,
default 17). В rsa256.go GenerateRSAKey()/RSAKeyFromHex() используй cfg-экспоненту (d=e^-1 mod φ,
gcd(e,φ)=1; e=65537 валиден). Лог при старте: "rsa: exponent=%d".

ПРИОРИТЕТ-3 (CM_LOGIN по гиту): в internal/server/server.go handleLogin парси plaintext
(DecryptSecondary key2 уже даёт pt, op=pt[0]=0x00): RSA-чанки кратно 128B с конца (число чанков
k=len_ct/128), хвост 55B: [sid D][16x0][7B: 20 00 00 00 00 00 01][16B: 9D DA 47 A7 21 C0 A6 A5
4B B7 5E E3 CE C9 26 AA][D 0][D unk][D 0]. Расшифровка каждого чанка: RSA nopadding (твоё
DecryptBlock, но вернуть ПОЛНЫЙ m как 128B BE — убери FillBytes(32)). Разбор: isLoginEx = k>1;
не-loginex: username=ASCII(Cp1252) чанк[94:108] до нуля (макс 14), password=чанк[108:124] до нуля
(макс 16), otp=LE i32 чанк[124:128] (FFFFFFFF = нет); loginex: username=чанк1[78:142] (64),
password=чанк2[78:110] (32), otp=чанк2[110:114]. Валидатор: username printable, otp==0xFFFFFFFF —
если НЕ валидно: лог "login decode FAIL (exp=%d, chunks=%d)" и relay в authd как раньше (не рвать).
В blob для authd ("cbdb") теперь клади РЕАЛЬНЫЕ [login][password]: согласуй с authd-контрактом
(если authd ждёт decbuf-32 — собери 32B = username zero-padded 14 + password zero-padded 16 + otp 2?
— возьми из rsa-hunt контракта; если authd умеет 128B — отрайтай полное m).

ПРИОРИТЕТ-4 (classic-режим, флаг mode: authgate|classic): в config.go поле Mode string (yaml mode,
default "authgate"). В classic:
  a) welcome строится по SM_INIT гита (plaintext 192B, см. раскладку в
     docs/beyond-aion-48-protocol-vs-gate-20261006.md §2): [0]=00, [1:5]=sid, [5:9]=0x0000c621,
     [9:137]=ScrambleModulusServer(N), [137:153]=16x0, [153:169]=sessionKey16 (random или key2-LUT),
     [169:176]=7x0, [176]=0, [177:181]=0, [181:183]=0, [183]=0, [184:188]=0x3FCE09ED, [188:192]=0;
     шифрование первого пакета: len+8, encXORPass (cumsum chain с рандом-сидом Rnd, dword[0] не
     трогается, финальный ecx в [len-8:len-4], chk-нули [len-4:len]) -> BF(static 6b60cb5b82ce90b1cc2b6c556c6c6c6c)
     -> wire 210. В authgate-режиме welcome НЕ ТРОГАТЬ (текущий 194/EncryptPrimary/key1 живой).
  b) диспетчер по (op,state) вместо длины: CONNECTED: 0x07=AUTH_GG, 0x08=UPDATE_SESSION;
     AUTHED_GG: 0x00=LOGIN; AUTHED_LOGIN: 0x05=SERVER_LIST, 0x02=PLAY. Прочее — relay в authd.
  c) SM_AUTH_GG форма гита (40B: [sid][0x4][0xCD5000][0][0x0b<<24][sid^0xCD5000][3x0], wire 50)
     под флагом ggXorTail (default false = наш живой 32B).
  d) SM_LOGIN_OK (0x03): [accountId][loginOk][0][0][0x3EA][0x7][19x0]; sessionKey = 4 случайных
     u32 (accountId, loginOk, playOk1, playOk2) в Session struct; на LOGIN: валидация, SM_LOGIN_OK,
     state=AUTHED_LOGIN.
  e) CM_SERVER_LIST (0x05): checkLogin(accountId, loginOk) иначе LOGIN_FAIL; SM_SERVER_LIST (0x04)
     полный формат: C(count) C(lastServer), на сервер [C id][ip4][H port][H0][C age][C pvp][H cur]
     [H max][C online][C type][C hide][H0][C brackets], H(maxId+1), C(1), charCounts, 13x0.
     Источник данных: cfg (worldIP/port) + lastServer/charCounts из authd если есть (иначе 0).
     f) CM_PLAY (0x02): checkLogin -> SM_PLAY_OK [playOk1][playOk2][servId][14x0]; playOk1/2
     передать в authd (если контракта нет — лог WARN и фиксированные значения как сейчас).
  g) LOGIN_FAIL (0x01): [D responseId] зашифрованный key2 (cc-plaintext оставить для authgate-режима).

ТЕСТЫ (internal/proto/proto_test.go, internal/server/server_test.go, ничего не удалять):
  1. TestScrambleModulusServerInverse: for 200 случайных N: UnscrambleClient(ServerScramble(N)) == N
     (клиентский unscramble = текущая ScrambleModulus).
  2. TestScrambleNotInvolution: ServerScramble(ServerScramble(N)) != N и ScrambleModulus(ScrambleModulus(N)) != N.
  3. TestClassicWelcome210: длина 210, dword[5:9]==0x0000c621, расшифровка static-BF + обратный
     encXORPass даёт исходный plaintext.
  4. TestCMLoginParse: golden из доки гита: user "abcdefghijklmn" pw "abcdefghijklmnop" otp -1 ->
     username/pwd/otp; loginex-вариант (2 чанка) -> то же.
  5. TestSessionXorChecksum эквивалентность: EncryptSecondary(pt) расшифровывается DecryptSecondary
     и паддинг-вариант [data][pad][chk] тоже проходит (XOR==0).
  6. e=65537: TestRSAPoolRoundtrip параметризовать экспонентой.

ЖЁСТКИЕ ОГРАНИЧЕНИЯ: не менять формат authgate-welcome (194, EncryptPrimary, key1, variant-механика),
не менять authdclient-контракты кроме расширения blob, ключи/сиды не хардкодить из памяти (только из
конфига), ничего не коммитить. После сборки: прогон go vet + все тесты. Отчёт: список изменённых
файлов + лог первого живого логина (decbuf раскладка, e=17/65537 результат).
```

## 8. Что НЕ трогать (живые факты 7.7 EU)

- welcome 194B / EncryptPrimary (cumsum от dword[0]) / key1 = GenerateInitialKey(0x04bd) — live-принят клиентом;
- SM_AUTH_GG 32B `[0b][sid][27×0]` — live-принят;
- cc-plaintext `[01][code]` (22 = «аккаунт заблокирован») — live-показан;
- Blowfish LE-блоки, канон P/S — live-подтверждено и совпадает с гитом;
- relay authd 74b/42b/26b/18b — работает;
- V=authd[03] в dword[5:9] — live-принят (classic-режим отдельно шлёт 0x0000c621).

## 9. Следующий живой тест после фикса (1 прогон решает 3 вопроса)

1. welcomeForceVariant=0 (теперь с ScrambleModulusServer), rsaExponent=17 → логин юзера:
   decbuf валидная раскладка? Да → e=17 подтверждён, authd получает creds.
2. Если нет → rsaExponent=65537, повторить. Да → e=65537 (как в гите).
3. Если оба нет → клиентская раскладка/padding отличается → путь rsa-hunt №2 (live-дамп,
   калибровка unscramble по известному m) — но с СЕРВЕРНЫМ скрамблом в welcome шанс #1/#2 высокий:
   гит-математика и «мусор при любом варианте» сходятся именно на нём.
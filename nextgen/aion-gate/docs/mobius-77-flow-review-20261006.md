# Mobius_AionEmu 7.7 vs aion-gate — полное ревью флоу (Фаза 1)

Дата: 06.10.2026. Эталон: `MobiusDevelopment/Mobius_AionEmu` (Aion-Germany 7.7 → Aion-Lightning код, JDK 25).
Клон: `/tmp/m77` (shallow). Наш код: `nextgen/aion-gate`. Анализ — без правок кода.

## 0. ВЕРДИКТ

Протокол ЭТОТ ЖЕ, наш флоу структурно совпадает с эталоном по всем опкодам/шифрам.
Наш гейт опережает эталон в главном: у эталона НЕТ loginex, нет fork/authd-relay,
нетasm-точной крипты welcome 194 (эталон = классическая SM_INIT 176/192).
Наших косяков найдено **6** (1 критичный, 2 структурных, 3 корректности) — все точечные.

## 1. Сверка «флоу целиком» (эталон → наш)

| Шаг | Эталон 7.7 (Aion-Lightning) | Наш гейт | Вердикт |
|---|---|---|---|
| Welcome | SM_INIT: op+D sid+D 0xc621+mod128+B16+bf16+D 197635+D 2097152 (pt 176) | welcome 194 live-принят (sid,V=0xc621,mod,key2,65650072) | ✓ расхождение форм = авторитарно: 7.7 EU leak ≠ AL-классика. НЕ ТРОГАТЬ |
| Blowfish | LE dword, канон P/S (BlowfishCipher) | LE (фикс 06.10), канон | ✓ 1-в-1 |
| Static key | `6b60cb5b82ce90b1cc2b6c556c6c6c6c` | тот же (LUT 0x04bd) | ✓ |
| 1й пакет шифр | length+=4+4+pad8 → encXORPass(Rnd seed) → BF → updateKey | EncryptPrimary (asm-точный) / EncryptGitInit (classic) | ✓ |
| Чексумма | verifyChecksum: XOR dword==0, chk=dword[len-4:] | DecryptSecondary/EncryptSecondary (asm 0x417a80/0d) | ✓ |
| RSA | RSA-1024, **e=F4=65537**, пул 10, `/ECB/nopadding` | rsaExponent=65537 (доказано оракулом), пул | ✓ |
| Скрамбл модуля | swap→xorL→dword→xorU (EncryptedRSAKeyPair) | ScrambleModulusServer (та же форма) | ✓ |
| State machine | CONNECTED→AUTHED_GG→AUTHED_LOGIN | **НЕТ** (диспетчер по длине) | ❌ К-2 |
| CM_AUTH_GG (0x07, CONNECTED) | D sid + B27 → SM_AUTH_GG | handleAuthGG len==32 | ✓ |
| SM_AUTH_GG (0x0b) | D sid + B35 (pt 39 → wire 50) | [0b][sid][27x0] (pt 32 → wire 42) | ⚠ наш live-принят; эталон-форма опцией |
| CM_LOGIN | op=**0x0B** (AUTHED_GG): D(4Б junk) + B128 RSA | SplitLogin: op+ct k×128+tail≤64 (op не проверяется) | ⚠ К-3: лог op, диспетч по state |
| CM_LOGIN m-раскладка k=1 | user **@64(32)**, pwd **@96(32)**, otp LE @124 | user@94(14), pwd@108(16) (4.8) | ⚠ К-4: добавить 3-ю гипотезу |
| CM_LOGIN loginex k≥2 | эталон НЕ УМЕЕТ (читает 128, остальное игнор) | combined[78:142]/[206:238] — **доказано оракулом** | ✓ наш вариант единственный валидный |
| Trim/login | user.trim().toLowerCase() | только isPrintableASCII | ⚠ К-5: trim+lower перед authd |
| SM_LOGIN_OK (0x03) | D acct, D loginOk, D0, D0, D 1002, D 126282165, B47 | authd payload 64 → 74b (relay) | ✓ relay; для эмуляции — форма эталона |
| SM_LOGIN_FAIL (0x01) / SM_PLAY_FAIL (0x06) | **D messageId** (AionAuthResponse) | 18b LoginFail = EncryptSecondary(8Б) | ✓ совпало |
| Коды ошибок | 0,1,2,4,5,6,7,8,10,11,12,15,16,18,21,22 (22=BAN_IP) | cc22 «заблокирован» live ✓ | ✓ cc-код == messageId; 45 = authgate-спец |
| CM_SERVER_LIST (0x05) | D acct, D loginOk, D x → полный SM_SERVER_LIST (+charCounts) | relay в authd (ориг так же) + 42b-эмуляция фолбэк | ✓ |
| CM_PLAY (0x02) | D acct, D loginOk, C servId → SM_PLAY_OK | relay / 26b [07][pk1][pk2][sid][14x0] | ✓ форма = эталон; хардкод 1/1010 → Rnd |
| SessionKey | accountId, loginOk=Rnd, playOk1=Rnd, playOk2=Rnd, checkLogin | relay от authd | ⚠ К-6 в эмуляции |
| CM_UPDATE_SESSION (0x08, CONNECTED) | D acct, D loginOk, D reconnectKey → authReconnectingAccount | НЕ ОБРАБОТАН (попадёт в «прочее») | ❌ К-2 fix |
| SM_UPDATE_SESSION (0x0c) | D acct, D loginOk, C sysmsg | нет | опция |
| SM_INIT classic-хвост | pt 176 (короткий) | classic по 4.8 (pt 192) | опц. флаг |

## 2. НАЙДЕННЫЕ КОСЯКИ (по убыванию важности)

### К-1 (КРИТИЧНО, может быть решающим для authd): relay-хвост логина при loginex
`handleLogin` (authgate): `dword148 = pt[148:152]`, `tail = pt[152:]` — жёстко от офсета 148,
как для 186b-фрейма (k=1). Но live 7.7 = loginex (pt 304: ct=pt[1:257]) → pt[148:152] и
pt[152:] попадают ВНУТРЬ ШИФРТЕКСТА → blob «cbdb» несёт мусор вместо dword/tail.
Фикс: при k≥2 брать dword и tail из SplitLogin-хвоста (tail=47Б: [sid][нули][0x20]
[68ffdab3e2fda892][2d9cc7baa87e0d49][0]); dword — сверить с live (у 186b pt[148:152] =
tail[19:23] = 0x00200000-зона — сверить живым логом), при неоднозначности relay tail как есть.

### К-2 (структура): нет state-машины и opcode-диспетчера
Эталон: CONNECTED{0x07,0x08}, AUTHED_GG{0x0B}, AUTHED_LOGIN{0x05,0x02}, прочее=unknownPacket(log).
Наш len-диспетчер не знает CM_UPDATE_SESSION (0x08 — релогин-флоу!), чужие опкоды = cc45/relay.
Фикс: state в Session (CONNECTED→AUTHED_GG по AUTH_GG-эхо→AUTHED_LOGIN по login OK),
диспетчер (state,op); длины оставить как fallback-эвристику (leak-клиент шлёт 312/314b).

### К-3 (структура): CM_LOGIN op не проверяется
В 7.7 эталоне op=0x0B. Наш SplitLogin берёт pt[1:] безусловно. Логировать pt[0],
диспетчить по op в AUTHED_GG (0x0B → login), неизвестное → лог+relay raw.

### К-4 (корректность): 3-я гипотеза раскладки k=1
Эталон 7.7: user=m[64:96](32), pwd=m[96:128](32), otp=m[124:128] LE. У нас только 4.8-форма
(@94/14, @108/16). Если придёт k=1 не-loginex — валидатор может зафейлить.
Фикс: в DecodeLoginPlain k=1 пробовать 4.8-раскладку → при не-printable пробовать 7.7-раскладку.

### К-5 (корректность): trim+lowercase username
Эталон: `new String(...).trim().toLowerCase()`. Мы отдаём authd сырую строку. Фикс:
strings.TrimSpace + ToLower перед BuildLoginDecbuf (authd авто-создание по ASCII-логину —
регистр влияет на имя аккаунта: «StelGeN» vs «stelgen»!).

### К-6 (корректность, эмуляция): SessionKey/26b-ответы захардкожены
build26ReplyPt: [07][1][1010][1] фиксированные. В relay-контракте значения приходят от authd
✓; но эмуляция (без authd) должна генерить Rnd и вести checkLogin как эталон. Приоритет низкий
(эмуляция = фолбэк).

## 3. ЧТО НЕ ТРОГАТЬ (живые факты перевешивают эталон)
- welcome 194 / EncryptPrimary / key1 LUT — live-принят 7.7 EU клиентом (у эталона другой SM_INIT).
- SM_AUTH_GG 42b — live-принят (эталонная 50b-форма = опция, не замена).
- e=65537 / ScrambleModulusServer / DecryptSecondary-до-SplitLogin / relay prepend-типа — доказаны.
- Классический SM_INIT (pt 176 vs 4.8-192) — только как конфиг-флаг classic-режима.

## 4. ПЛАН ЗАКРЫТИЯ ФЛОУ (Фаза 1, промпт в ./PROMPT-AUTHGATE-FLOW.md)

| # | Правка | Файл | Приоритет |
|---|---|---|---|
| 1 | relay-хвост/dword из SplitLogin-tail при k≥2 | server.go handleLogin | P0 |
| 2 | state-машина + (state,op)-диспетчер + CM_UPDATE_SESSION | server.go, Session | P0 |
| 3 | лог/диспетч op CM_LOGIN (0x0B) | server.go, login.go | P1 |
| 4 | DecodeLoginPlain: 3-я гипотеза k=1 (@64/@96 7.7) | proto/login.go | P1 |
| 5 | trim+lower username | server.go | P1 |
| 6 | SessionKey Rnd в эмуляции + checkLogin | server.go build26ReplyPt | P2 |
| 7 | реестр кодов AionAuthResponse в доку (cc=messageId) | docs | P2 |
| 8 | конфиг-флаг SM_AUTH_GG-форма эталона (wire 50) | classic/config | P3 |

После правок: тесты (§7 beyond-aion дока) + 4 новых (loginex-tail, state-переходы,
k=1-гипотеза-7.7, trim/lower) → cross-build → прод: свап 2106 на новый exe → решающий логин.

## 5. Артефакты эталона (пути в клоне /tmp/m77)
- java/com/aionemu/loginserver/network/aion/LoginConnection.java — state-машина, decrypt/encrypt
- .../network/ncrypt/{CryptEngine,BlowfishCipher,KeyGen,EncryptedRSAKeyPair}.java — крипта
- .../network/aion/clientpackets/CM_{AUTH_GG,LOGIN,SERVER_LIST,PLAY,UPDATE_SESSION}.java
- .../network/aion/serverpackets/SM_{INIT,AUTH_GG,LOGIN_OK,LOGIN_FAIL,SERVER_LIST,PLAY_OK,PLAY_FAIL,UPDATE_SESSION}.java
- .../network/factories/AionPacketHandlerFactory.java — диспетчер (state,op)
- .../network/aion/{SessionKey,AionAuthResponse}.java — ключи и коды ошибок

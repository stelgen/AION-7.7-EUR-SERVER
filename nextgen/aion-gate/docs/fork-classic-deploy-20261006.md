# aion-gate: fork/classic + root-cause fix скрамбла — деплой 06.10 (день)

Док-спека: `docs/beyond-aion-48-protocol-vs-gate-20261006.md` (§7 — промпт на исправление).
Исходники-референс (проверено fetch'ем): AC-Login 4.7.5 `SM_AUTH_GG.java`, `SM_INIT.java`,
`LoginConnection.java`, `CryptEngine.java` (caltus/aioncore-v4-7-5).

## Что сделано (все пункты §7)

1. **П1 root-cause**: `ScrambleModulusServer` (порядок гита swap→xorL→dword→xorU) —
   именно её теперь шлём в welcome (вариант 0); старая `ScrambleModulus` = КЛИЕНТСКИЙ
   unscramble, оставлена для анскрамбла чужих модулей/тестов. RAW-вариант (4) помечен
   в логе как сломанный.
   ✅ **ДОКАЗАНО НА ПРОДЕ**: fork-режим расшифровал welcome оригинала, анскрамблил модуль
   (N_orig = 1024 бита, top=0xbd — валидный RSA-1024) и `ScrambleModulusServer(N_orig) ==
   wire` — оригинал шлёт ровно гит-форму.
2. **П2**: `rsaExponent` в конфиге (17 | 65537), генерация ключей с e из конфига,
   `rsa: exponent=` в логе при старте.
3. **П3**: `handleLogin` по раскладке гита: `[op][ct 128×k][tail 55]`, полный m 128B BE,
   user@94:108 / pwd@108:124 / otp LE@124:128 (loginex: чанки склеиваются, user@78:142,
   pwd@206:238, otp@238:242); валидатор (printable + otp=FFFFFFFF) = оракул выбора e;
   blob "cbdb" теперь с РЕАЛЬНЫМИ кредами (decbuf = m[128-loginDecbufLen:], дефолт 34 =
   asm arg3=0x22); FAIL → legacy-релей (не рвать).
4. **П4 classic** (`mode: classic`): SM_INIT гита — pt 192B ([5:9]=0x0000c621, модуль
   сервер-скрамбл, sessionKey@153:169, magic 0x3FCE09ED@184) → `EncryptGitInit`
   (encXORPass с рандом-сидом; **Java-пад: `length += 8 - length%8` — при кратности 8
   добавляет ещё 8** → 192+4+4=200 → 208 → wire 210; сверено с CryptEngine.java) →
   диспетчер по (op,state) CONNECTED/AUTHED_GG/AUTHED_LOGIN; SM_AUTH_GG (32Б живая /
   37Б гит → wire 50, флаг ggXorTail); SM_LOGIN_OK / SM_SERVER_LIST (полный формат) /
   SM_PLAY_OK / LOGIN_FAIL; sessionKey = 4 случайных u32.
5. **FORK-режим** (`mode: fork`, требование юзера — «сравнить ответ оригинала и наш
   отдельным режимом в конфиге»): прозрачный релей клиент↔оригинал (forkOrigAddr:Port),
   всё байт-в-байт; welcome оригинала расшифровывается static key1 → sid/V/модуль/key2
   в лог; каждый фрейм расшифровывается key2 оригинала; на AUTH_GG/26b строится НАШ
   shadow-ответ и сравнивается (SAME/DIFF); на LOGIN дампятся ct-чанки (полный hex) +
   N_orig → **оффлайн-калибровка e**: m^17 / m^65537 mod N_orig == ct по известным кредам.
   authd НЕ подключается (нет фантомных сессий).
6. Тесты §7: инверс-скрамбл (200/200), не-инволюция, welcome-210 (включая Java-пад),
   CM_LOGIN golden (обе формы), checksum-layouts, RSA e∈{17,65537}, fork-passthrough
   (нашёл и починил баг: welcome клиенту шёл без len-префикса). `go vet` + `go test` зелёные.

## Прод-топология (после деплоя 06.10 ~14:48 VM-времени)

| Порт | Что | Задача |
|---|---|---|
| 2104/2110 | L2Authd | AionAuth |
| **2106** | **aion-gate mode=fork** (PID 1644) | AionGate |
| 2109 | AuthGateD оригинал (PID 3868) | AionGateOrig |

- Клиент ходит на `192.168.0.125:2106` как обычно → попадает на оригинал через наш форк.
- Логи: `D:\SAION\aion-gate\gate-prod.log` — строки `FORK ...` (welcome-разбор,
  N_orig, ct логина, shadow-сравнения).
- Бэкапы на VM: `aion-gate.exe.bak-forkpre`, `config-prod.yaml.bak-forkpre`.
- Откат на «наш самостоятельный гейт»: в config-prod.yaml `mode: authgate` + `/end /run AionGate`.
- Офф: `schtasks /end /tn AionGateOrig` (после калибровки).

## Следующий шаг (решающий прогон)

1. Юзер логинится через 2106 (форк) → в логе `FORK C>O LOGIN ct[i]` + `FORK N_orig`.
2. По кредам юзера строим m (раскладка §4), считаем m^17 и m^65537 mod N_orig —
   совпадение с ct фиксирует e И раскладку.
3. `rsaExponent` в конфиг + `mode: authgate` → рестарт → живой логин уже на НАШ гейт:
   decbuf = user14+pwd16+otp4 → authd (авто-создание) → 74b serverlist → мир.
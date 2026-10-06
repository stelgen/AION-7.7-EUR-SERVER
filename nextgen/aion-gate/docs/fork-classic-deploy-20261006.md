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
## ⚠ ПОПРАВКИ ПО ИТОГАМ ДНЯ (06.10, вечер) — ошибки ранних формулировок здесь исправлены

### Версия деплоя
- Прод-экзешник на VM (D:\SAION\aion-gate\aion-gate.exe) = сборка кода коммита `f2d5ba6`
  (после него менялись только README/docs/probe). Бинари в гит НЕ кладутся (*.exe в
  .gitignore) — версия деплоя = коммит.
- Конфиг прода: mode: authgate, rsaExponent: 65537, loginDecbufLen: 34,
  welcomeWaitAuthdMs: 2000, welcomeForceVariant: 0 (СЕРВЕРНЫЙ скрамбл).

### ПОПРАВЛЕННЫЕ НЕВЕРНЫЕ ПОЛОЖЕНИЯ/ДОГАДКИ (ledger — не возвращаться к ним)
1. «ScrambleModulus = серверный скрамбл (инверс клиентского unscramble)» (commit 3b78928) —
   НЕВЕРНО: это и есть клиентский unscramble; сервер обязан слать ScrambleModulusServer
   (гит-порядок swap→xorL→dword→xorU). ДОКАЗАНО: ServerScramble(unscramble(wire)) == wire
   на живом ориг (дважды).
2. «e=17 (Pub.key[1])» — ОПРОВЕРГНУТО: **e = 65537 (F4)**, доказано оракулом
   m^65537 mod N_orig == ct на живом ct юзера (креды 1/1). Переборы e по wrong-раскладке
   из rsa-hunt были невалидны (ct резали не из того места).
3. «хвост CM_LOGIN = фикс 55Б» (гит 4.8) — НЕВЕРНО для 7.7: хвост ВАРИАТИВНЫЙ (47/55Б,
   структура константна: sid LE, нули, 0x20, 68ffdab3e2fda892, 2d9cc7baa87e0d49). Правило:
   k=(len(pt)-1)/128, rem≤64.
4. «RSA-чанки = data[:128] из RAW фрейма» — ГРУБАЯ ОШИБКА старого кода: фрейм зашифрован
   key2, нужен DecryptSecondary ДО SplitLogin/RSA. Это корень №2 «decbuf мусор»
   (authgg работал, т.к. там DecryptSecondary был).
5. «authd-ответы = EncryptSecondary(payload)» — НЕПОЛНО: клиентский опкод = ТИП authd,
   гейт обязан prepend байт типа (type=3 → [03]+пад до 64Б = wire 74 login-ok). Корень №3.
6. «26b-эмуляция [05]→42b/[02]→26b — паритет с ориг» — НЕВЕРНО как основной путь: ориг
   РЕЛЕИТ [05]/[02] в authd (ответы-типы 4/7 идут от authd). Эмуляция = фолбэк без authd.
   Плюс кейс длины был 26 (wire) вместо 24 (ECB) — никогда не матчился.
7. «m2 = pwd@78 + otp FFFFFFFF@110» — НЕ ПОДТВЕРДИЛОСЬ: во 2-м блоке есть иные данные
   (ct2 не сводится к простым формам). Не блокирует — authd пароль не проверяет.
8. «L2Authd хрупкий из-за detached-процессов / schtasks /end не убивает» — корневая
   причина другая: **стек живёт только в SESSION 1**; SYSTEM/session-0 запуск L2Authd =
   мгновенный тихий выход. taskkill после /end всё равно нужен (процессы задачи переживают /end).
9. «2106 занят самим L2Authd (второй листенер)» (старый стенд-док) — в актуальной топологии
   2106 = гейт; после ребута ВМ порт перехватывал старый fork-proxy.exe (AionForkProxy,
   теперь DISABLED) — проверять, кто держит порт (tasklist /fi "PID eq N").
10. «authd ПАРОЛЬ НЕ ПРОВЕРЯЕТ + авто-создание» — ПОДТВЕРЖДЕНО ещё раз живьём (probe:
    fresh-аккаунты создавались, type=3 приходил).
11. «V в welcome = 0x0000c621» — ПОДТВЕРЖДЕНО: наш authd шлёт [03]=21c60000 — тот же V,
    что у ориг (не конфликт с classic-ревизией 0xc621 в SM_INIT — совпадение значений).
12. «CM_LOGIN op = 0x0B (эталон Mobius 7.7)» — НЕВЕРНО для живого EU-клиента: форк-дамп
    07.10 00:43 (креды 1/1) показал pt(304) с op=**0x00** (как 4.8); ориг релеил как есть.
    P1-3 в форме «вход только по 0x0B» заворачивал живой логин в raw-relay → authd молчал
    (ожидал cbdb-blob) → клиент отвалился через 16с (решающий логин на свитче 00:49).
    Фикс: диспетчер AUTHED_GG принимает op 0x00 || 0x0B. Мораль: эталон = Mobius-исходники,
    live-клиент = leak-бинарь — расходятся и в оп Harding-кодах; арбитр — форк-дамп.
13. «К-1 dword/tail решающ для authd» — ОПРОВЕРГНУТО живьём: authd принял blob даже со
    старым мусорным dword из ct-зоны + 152Б хвостом (19:12 06.10 → type=3 login-ok acc=7).
    Authd читает только decbuf (user); dword/tail игнорит. К-1 фикс оставлен (гигиена логов).

### RUNBOOK ПОСЛЕ РЕБУТА ВМ (доказанный порядок)
1. Десктоп ВМ → `AION-START-ALL-v6.bat` (или /IT-задача `AionAuthIT` → C:\Temp\auth.bat →
   `call C:\Temp\start-all.bat`). СТРОГО session 1 — SYSTEM/session-0 L2Authd умирает молча.
2. Свап гейта: `taskkill /F /IM AuthGateD.exe & taskkill /F /IM fork-proxy.exe &
   schtasks /run /tn AionGate` (AionGateOrig = ориг на 2109 для отката).
3. Проверка: 2104/2106/2109/7777 + баннер `mode=authgate rsa_exponent=65537` +
   `RAW AUTHD A>G [03] 21c60000` в gate-prod.log.
4. Health-check authd: `probe.exe 127.0.0.1:2110 probeuser 192.168.0.253` → ждём type=3.
5. Если authd флейкит на повторный логин (тишина) — рестарт L2Authd (session 1).

# AuthGateD: ресёрч чужого опыта (SM_INIT / классический LS-протокол) — 05.10.2026 ночь-3

> Цель: снять блокер byte-exact welcome (0x407d50) чужими находками — «может уже разгадано».
> Метод: web-ресёрч + diag6-проба констант по capture. Вердикт: прямой перенос НЕ работает,
> но получены структурный априор, снятие одного противоречия §7 и независимое подтверждение
> цепочки scrambleModulus.

## 1. Источники

| # | Что | Где |
|---|---|---|
| 1 | Aion Classic RE: методология (декомпил→capture→byte-exact), 3 нумерации опкодов, клиент отбрасывает 127.0.0.1 | forum.ragezone.com/threads/1267025/ |
| 2 | SM_INIT (aioncore-v4-7-5 = Aion-Lightning 4.7.5): эталонный welcome-билдер, opcode 0x00 | github.com/caltus/aioncore-v4-7-5 .../serverpackets/SM_INIT.java |
| 3 | beyond-aion 4.8 SM_INIT: вторая независимая реализация | github.com/beyond-aion/aion-server (branch 4.8) |
| 4 | Тред CM_LOGIN: порядок blowfish→RSA-priv, XOR-цепочка encryptModulus (AC-Login), логин/пароль половинами RSA-блока | forum.ragezone.com/threads/1211174/ |

## 2. Эталонный welcome классики (SM_INIT, opcode 0x00)

aioncore 4.7.5 [2]:
```
writeD(sessionId);            // dd — session id
writeD(0x0000c621);           // dd — protocol revision (константа сборки)
writeB(publicRsaKey);         // b  — SCRAMBLED RSA-модуль (0x80; докстринг: 0x90 = 0x80 RSA + 0x10 at 0x00)
writeD(0) x4;                 // dddd unk
writeB(blowfishKey);          // s  — blowfish key ОТКРЫТЫМ текстом внутри welcome
writeD(197635);               // dd unk (0x00030403)
writeD(2097152);              // dd unk (0x00200000)
```
beyond-aion 4.8 [3]: скелет тот же (dd sessionId → dd 0xc621 → b RSA-128), далее:
`b[16] нулей → blowfishKey 16b → b[7] нулей → C0/D0/H0 (test-server ip/port) → C0 → D 0x3FCE09ED → D 0`.
Итого ≈191 байт plaintext. Различия деталей между версиями ⇒ скелет константен, спейсеры плавают.

CM_LOGIN-флоу [4]: клиент шифрует креды blowfish-ключом из welcome → внутри RSA-блок → RSA-priv
→ логин/пароль = половины блока с \x00-strip. RSA-модуль перед отправкой скрамблится
`encryptModulus` (AC-Login): swap[i]↔[0x4d+i] → [i]^=[0x40+i] → [0x0d+i]^=[0x34+i] → [0x40+i]^=[i].

## 3. Совпадение с нашим кодом (важно!)

`unscramble_welcome.py::inv_mod` — ПОБАЙТОВО точная инверсия этой самой AC-Login `encryptModulus`
(реверс порядка шагов). Т.е. наша гипотеза «RSA-модуль в welcome скрамблится классической XOR-цепочкой»
подтверждена независимой опенсорс-реализацией. В asm гейта свой аналог найден: `scrambleModulus @0x417c50`.

## 4. diag6-проба (tools/analysis/diag6_sminit_probe.py, прогон 05.10 ~22:40)

Вход: 82 welcome (proxylog + handshake-capture-20261003), ECB-dec(key1=6b60cb5b…).

| Тест | Результат |
|---|---|
| Константы SM_INIT (0xc621, 197635, 2097152, 0x3FCE09ED) на любых смещениях | **НЕ найдены** (ни в одном из 82) |
| Нулевые прогоны ≥8 байт в dec (scrambled) | **Нет ни одного** |
| Групп по block0=(d0,d1) | 49 (82 plaintext, все уникальны) |
| Timestamp-корреляция d0/d1 (unix / GetTickCount) | фон, сигнала нет (доли совпадений ≈ вероятности для случайных dword) |
| Первый различающийся байт между соединениями одного run | **байт 8** (plaintext[0:8] per-run константа) |

## 5. Выводы

1. **Наш welcome ≠ байт-в-байт SM_INIT.** Гейт 7.7-эпохи — свой формат (RSA-256 + LUT-ключи + GG-зона +
   key2). Прямое заимствование layout'а классики невозможно. Но структурный априор остаётся:
   «RSA-модуль + 16b блок + 16b ключ в открытом виде внутри welcome» классика подтверждает.
2. **Снято противоречие §7 про GG-зону:** нулевая зона НЕ ОБЯЗАНА быть видна в ECB-dec (это SCRAMBLED
   plaintext — скрамбл обязан её разрушить, кроме data[0]). Искать её надо в old-кандидатах
   unscramble, а не в dec. Отчёт «GG-зона нулей не найдена ни на одном смещении» — ожидаемый результат,
   не контрдоказательство.
3. **plaintext[0:8] — per-run значение, не статическое поле.** Первый diff между сессиями одного
   запуска = байт 8. Значит «plaintext[0]=0x23 динамический» — свойство per-run сида/заголовка,
   а не опкода. При дизasmе 0x407d50 целевой вопрос: откуда берутся первые 8 байт
   (per-process init? GG seed? boot tick?) — они же не скрамблятся (data[0] нетронут).
4. **Противоречие y0==a0 (§7)** скорее всего артефакт сравнения: dword0 после ECB-dec = old[0]
   при любой корректной модели (data[0] не скрамблится); «plaintext[0]» — это LE-байт dword0,
   динамический per-run. Перепроверить, что сравнивались dword-величины, а не байт-vs-dword.
5. Методология [1] подтверждена: коды из декомпила → один реальный capture = ground truth →
   byte-for-byte gate → loopback только регрессия. Ровно наш план; менять нечего.

## 6. Следующий шаг (не меняется, уточнён чек-листом)

Дизasm `0x407d50` (welcome-билдер). Чек-лист вопросов:
- точный порядок варов Assemble (fmt «cddbbbcccc» гипотеза) и где welcomeExtra4(+4);
- что попадает в EncryptPrimary (offset-заголовок? len-байты блобов в plaintext?);
- источник первых 8 байт plaintext (per-run);
- len-байты блобов: есть ли в plaintext байты 128/16/16 (якоря для unscramble-DFS —
  резать перебор по ним: dword2 byte1=0x80, dword17/19 byte1=0x10 при layout из unscramble_welcome.py);
- порядок ECB↔скрамбл и куда пишется csum (dec[184:188]).

После byte-exact: фейк-клиент с реальным plaintext LoginEx → win-сборка → деплой по плану §6
(D:\SAION\aion-gate, AionGate retarget, aion-op config, откат C:\Temp\gate.bat). «го» дано заранее.
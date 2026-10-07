> ⚠ АРХИВ — задача ЗАКРЫТА (компонент в бою/релизе). Документ сохранён для истории. Открытые промпты: PROMPT-ACCACHE.md, PROMPT-AUTHD.md; промпты компонентов: <компонент>/PROMPT.md (aion-cache, aion-ic, aion-chat, aion-petition, aion-shopagent, aion-gm, aion-binpatch; шаблон: README-TEMPLATE.md).

# AION AuthGateD — ПРОДОЛЖЕНИЕ (RSA-decbuf блокер)

ЧИТАТЬ ПЕРВЫМ: ./rsa-hunt-20261006.md — полное состояние, все теории (включая мёртвые), артефакты, косяки.

ГДЕ МЫ: наш Go-гейт на 2106 (le8b, fixed key 653fa8e1/0771db79 с p,q сохранёнными в pair-pq.txt, force=0). Весь флоу живого клиента работает ДО authd: welcome ✓ AUTH_GG ✓ LOGIN ✓ relay [02] уходит — НО decbuf (RSA-расшифровка) = 128Б мусора → authd молчит → клиент «думает».

ЧТО ИЗВЕСТНО: authd ПАРОЛЬ НЕ ПРОВЕРЯЕТ (любой ASCII-логин авто-создаёт акк; фейл только не-ASCII логин → 18b LoginFail). Plaintext логина ≠ [login16][md5-16] ни в одной кодировке (6144 гипотезы с правильным ct=128Б — пусто). Вероятно PKCS-пэддинг с рандомом (оракулы слепы в принципе) или клиентский unscramble ≠ классика.

ЗАДАЧА: ЛАЙВ-дамп AuthGateD ≤2 сек после login: sid= → живые буферы rsapricrt → ИСТИННЫЙ plaintext m + n/e/d/p,q сессии → раскладка → калибровка клиентского unscramble оракулом m^17 mod U(zone)==ct → фикс → decbuf 32Б → authd → 74b → мир.

ПЛАН:
1. Поднять fork-топологию: fork-v2.ps1 / fork-final.ps1 / fork-restore.ps1 на VM (2106=fork→2109=ориг, наш→2116; L2Authd проверить 2104/2110, морг → /end+/run AionAuth → wait 2104 → /run AionGate)
2. Запустить СИНХРОННЫЙ watcher (НАДЁЖНО): ssh ... "powershell -NoProfile -ExecutionPolicy Bypass -File C:\Temp\watch-dump.ps1" — он сам дождётся login: sid= в gate-prod.log ОРИГА-сессии (внимание: watcher читает gate-prod.log НАШЕГО гейта — в fork-режиме наш лог пойдёт пустым! Пересмотреть: вести fork.log-монитор ИЛИ дампить по появлению 314b в fork.log) → дамп C:\Temp\authgate-live.dmp мгновенно
3. Тянуть дамп: enc-dmp.ps1 → type → локальный base64-декод; НЕ certutil через cmd-цепочки
4. Скан живых буферов: m-plaintext все mpn-формы (raw/reversed/dword-swap), 64Б-делители p,q (тест: A=m^17−ct при известных creds юзера — creds он называет в чате!), decbuf-32, n/e/d mp-структуры
5. ИЗ m_true: калибровка клиентского unscramble (скрипт-скелет sweep в этом чате: 24 перестановки × cum × оффсеты × BE/LE × e) → ScrambleModulus фикс → le9 → decbuf 32Б
6. Relay decbuf → authd авто-создание → 74b → 26b/42b эмуляция → мир

АЛЬТЕРНАТИВЫ если дамп пустой снова:
- cdb-брейк rsapricrt @0x417b60 на AuthGateD-2109 (входной буфер ct + выходной m видны в момент; Debugging Tools: SDK silent fail — попробовать /IT-задачу, winget, или procdump -ma)
- Дизasm клиентского RSA-контекста (aion.bin у юзера; клиент шифрует m^E mod N_client — найти E и N_client)
- Запускать наш гейт с НЕскрамбленным welcome (variant=4) И корректным decbuf-обработчиком: если клиент шифрует против raw-zone — ct^d17 mod N_our не сработает, НО ct^17 mod zone_int == m можно проверить напрямую оракулом!

ГОТЧИ: ct в оракулах = ПЕРВЫЕ 128Б ФРЕЙМА (не весь 312Б! — косяк этой сессии, потерял 10К гипотез); N.BitLen() может быть 1023 (фикс <1000 стоит); PS через scp+File; watcher синхронный (Start-Process detached падает); PS-лог гейта UTF8-ломается — править байтово Latin-1; firewall: новые порты VM закрыты.

ФАЙЛЫ: pair-pq.txt (p,q,λ,d пары 653fa8e1), fork5/6.log (ct всех логинов юзера + creds названы в чате), authgate.dmp (11:44 — буферы мертвы), scanN2.py/scanAll.py (numpy+gmpy2 сканы), fixed-pub-corrected.der.hex. Все в ~/STELGEN/tmp/aion-vm/ и на VM C:\Temp\.

ГОТЧА СЕКРЕТОВ: creds юзера в гит/память НЕ класть. Наша fixed RSA-пара — сгенерена нами, не секрет.
> ⚠ АРХИВ — задача ЗАКРЫТА (компонент в бою/релизе). Документ сохранён для истории. Открытые промпты: PROMPT-ACCACHE.md, PROMPT-AUTHD.md, PROMPT-CACHED.md, PROMPT-ICSERVER.md (шаблон: README-TEMPLATE.md).

# Промпт для нового чата: ПЕРЕПИСЬ AuthGateD (Трек B, шаг 3)

> Скопируй текст ниже в новый чат как первое сообщение.

---

Ты — ассистент проекта **STELGEN / AION 7.7 PTS EU private server**. Продолжаем Трек B: поэтапная замена NC-бинарей на свои (Go). Готовы и в бою: **aion-logd** (логгер, :2051) и **aion-captcha** (капча, :22206) — метод capture → PDB → Go → паралл. прогон → свитч с откатом отработан дважды. Теперь по плану **AuthGateD** (гейт, :2106) — внешняя поверхность авторизации, самый ценный кандидат (ROADMAP §3 #4, оценка 1–3 нед).

## Первый шаг (обязательно, до любых действий)
1. Прочитай память по пути `STELGEN/projects/aion_server_2026-10-02` (хронология, готчи, схема прод-стека; особенно записи про auth-разведку 03–04.10: патчи p1–p5, хендшейк, коды World→Auth).
2. Прочитай в репо `~/STELGEN/projects/aion_server_2026-10-02/AION-7.7-EUR-SERVER/`:
   - `nextgen/ROADMAP.md` — живой план и правила эксплуатации (§4 ОБЯЗАТЕЛЬНО);
   - `nextgen/TELEMETRY-SPEC.md` — ОБЯЗАТЕЛЬНЫЙ стандарт телеметрии;
   - `docs/auth-server-internals.md` — главный референс: схема авторизации, хендшейк гейта (§2), дизасм проверки сессии (§3), все процы/таблицы auth-БД;
   - `nextgen/CAPTCHA-STATUS-SNAPSHOT.md` + `docs/session-20261005-captcha.md` — метод-референс последней замены;
   - `docs/app-architecture.md` — инвентарь (строка AuthGateD), `docs/errors.md` гл. 14–19 (Session mismatch, brute-блок, console-API);
   - `nextgen/aion-captcha/` — пример готового кода (структура internal/*, ship, config, тесты).

## Цель
Своя замена **AuthGateD.exe** (native C++, GUI-приложение; клиентский порт **2106**, к authd ходит по **2110**; конфиг `AuthGateD\etc\config.txt` — `serverPort = 2106` С ПРОБЕЛАМИ и 48704 не-ASCII корейских байт комментариев — править ТОЛЬКО байтовой заменой python). Гейт принимает клиента, выдаёт welcome с RSA-модулем, валидирует sessionId, парсит LoginEx, форвардит в authd, отдаёт serverlist. PDB **ЕСТЬ** — `~/STELGEN/projects/aion_rev_2026-10-05/artifacts/pdb-small/AION_LIVE_SERVER/AuthGateD/` (+ `AuthGateD.pdb`).

## Уже известные факты (не переоткрывать — проверить capture'ом)
**Активы:** `tools/analysis/` = `AuthGateD_original.exe` (0601, эталон) + `AuthGateD.map` (символы!) + `AuthGateD_disasm.asm` (62k строк objdump); парсер publics `pdbpub.py` (`~/STELGEN/tmp/` или tools/analysis).

**Хендшейк (docs/auth-server-internals.md §2, hex-эталоны в C:\Temp\proxylog.txt от 03.10):** фрейминг 2 байта LE = длина: welcome 194 (0xC2) — scramble RSA-1024 модуль + Blowfish-ключ сессии (ВСЕ байты меняются по сессиям, открытых маркеров aioncore НЕТ); клиент 34 (0x22) — RSA-обмен (32b payload); сервер 42 (0x2A) — Blowfish ECB 8-байтные блоки, ключ сессионный; логин 186 (0xBA) зашифрован. Рабочий клиент = ru-Innova с флагами `-loginex -pwd16 -ip:<IP> -port:2106`: логин 314b с MD5-хешем (пароль в БД = binary(16) = MD5), auth-ответ 74b, serverlist 26b, select → close 2106 → мир 7777.

**Проверка сессии (VA, raw = VA − 0x400000):** `0x4079d0`: m_iSessionId [ecx+0xfc] vs [esp+4]; WARN UTF-16 `Session id mismatched.(%s)` строка @0x42ce38, "(%s)" @0x42c7e4; 7 call-сайтов = 7 обработчиков: `0x4063d4/0x406678/0x4067d7/0x406bc9/0x406d39/0x406e68/0x4070e7`; парсер логина `0x407ac0` (после RecvLogin `0x4063e1: call 0x407ac0`); генератор id: писатели [esi+0xfc] @0x4041b8 и @0x4076f3 (call 0x408070). Западные клиенты шлют sessionId=0 ВСЕГДА (портал-сессия) → mismatch и разрыв — ЭТО ОЖИДАЕМОЕ поведение стека; STL ios_base::badbit дампы в `PrtcGetAuthQuery` при портал-попытках = известная безвредность.

**УРОК ПАТЧЕЙ p1–p5 (критично):** брутальные always-true патчи валидации УБИВАЛИ authd-контур (authd умирал за 96–118с) или гейт тихо EXITED в парсере логина. Вывод: authd — хрупкий партнёр; наш гейт должен говорить с authd по wire 1-в-1 и НЕ слать ему ничего нового; фейлы клиентов обрабатываем сами, не дёргая authd.

**Wire gate↔authd:** Auth→Gate тип 4 = push serverlist (IP мира из `AionAccounts..server` + байт#23 = region); Gate→Auth = форвард select/login; коды World→Auth `f2030000+NN` — authd-сторона (см. authlog-analysis). Точный формат закрыть capture'ом 2110 + дизasmом.

**Brute-protect:** задержки 20/60/120 после фейлов — воспроизвести.

**Эксплуатация:** гейт живёт ТОЛЬКО в интерактивной юзер-сессии (Console 1); от SYSTEM/ssh — мгновенный exit -1. Задача `AionGate` (/IT) → обёртка `C:\Temp\gate.bat` v3.1 (taskkill в начале + waitport 2106). Порядок старта: SQL → AccountCache → **authd** → гейт (authd первым!). Наружу торчит только 2106 (firewall-правило Aion7.7 Client-Login).

## Дисциплина (не нарушать)
- **Прод трогать ТОЛЬКО после явного «го» юзера.** Всё до того — read-only анализ + локальная разработка + фейк-клиенты.
- Каждая замена = переключаемая: бекапы, откат одной командой; бекапы БД перед любым SQL ALTER.
- Перепись обязана соответствовать `nextgen/TELEMETRY-SPEC.md`: телеметрия В СЕТЬ (syslog/HTTP), НЕ срать файлами, ship не критичный путь, self-статус, raw+ошибки наружу. Пакет `internal/ship` копируй из `nextgen/aion-logd/internal/ship` как есть. Конфиг-ключи `ship.*` идентичны logd'у.
- Свои апки живут в `D:\SAION\<имя>\` (layout как у aion-logd/aion-captcha: exe + config.yaml + run.cmd).
- Секреты/ключи — только в конфиг на VM, в гит/память не сохранять.
- Готчи доступа к VM (192.168.0.125, `ssh 'Администратор@192.168.0.125'`): дефолт-шелл PowerShell (cmd через `cmd /c "..."`, `&` в PS запрещён); scp push работает, pull — нет (вниз через `cmd /c type` или PS base64); кириллица/корейщина в конфигах — только байтовая замена python'ом; git: `--no-pager` перед подкомандой; бинари win — кросс-сборка Go (`~/STELGEN/go-dist/go/bin`); подмена: `schtasks /end` → ЖДАТЬ смерти процесса до 10с → copy → `/run`.
- После свитча поправить aion-op config (`display`+`exe` → `aion-gate.exe`, иначе proc_missing-алерт) + рестарт AionOp — паттерн CAPTCHA-свитча.

## План работ (шаги, каждый = коммит + пуш + дельта в память)
1. **Разведка (read-only)**: инвентарь `D:\AION_LIVE_SERVER\AuthGateD\` (exe MD5, config.txt байтово, логи/dump'ы); netstat: кто на 2106/2110, процесс-владелец; задача AionGate + gate.bat; строки aion-op config (gate-строки); сверить PDB-md5 с бинарем на проде.
2. **Live capture (по «го», окно)**: реанимировать python-прокси `scripts/proxy/aionproxy.py` (2106→2107, hex-дамп) для клиентской стороны; capture 2110 (gate↔authd) pktmon-ом или зеркалом; юзер логинится (1–3 сессии: happy-path + фейл-пароль + портал-попытка для фейла). Реставрация портов по бэкап-процедуре 04.10 (`.bak-2106` и т.п.). Итог: эталонные hex-файлы в `nextgen/aion-gate/testdata/`.
3. **Реверс + протокол-док**: `pdbpub.py` на AuthGateD.pdb → publics; закрыть в дизasmе: scramble welcome, RSA-обмен (32b), производная Blowfish-ключа, формат LoginEx-парсера (0x407ac0, поля 314b-логина), wire 2110, генератор session-id, brute-таймеры. Оформить `docs/authgate-protocol-<дата>.md` (коммит).
4. **Реализация** `nextgen/aion-gate/` (Go, структура как у aion-captcha): `internal/proto` (framing 2b LE, welcome-генератор со scramble + RSA-1024 keypair при старте, Blowfish ECB, LoginEx-пакеты), `internal/server` (сессии, session-id генератор+проверка, brute 20/60/120, состояния), `internal/authdclient` (wire 2110 1-в-1), `internal/ship`, `internal/config` (yaml — зеркало полей config.txt + ship.*). Тесты: фейк-клиент (handshake → LoginEx happy + sessionId=0-фейл + фейл-пароль → brute), фейк-authd (wire-фикстуры из capture), byte-в-byte на живых fixture.
5. **Параллельный прогон**: наш гейт на свободном порту (напр. 21055) + фейк-клиент e2e + фейк-authd; сравнение поведения с эталонными capture.
6. **Свитч (по «го»)**: `schtasks /change /tn AionGate /tr "D:\SAION\aion-gate\run.cmd"` + `/run`; верификация: 2106 LISTENING, 2110 ESTABLISHED к authd, юзер логинится e2e (serverlist → выбор → Server64 7777), authd-логи чисты после N сессий; aion-op config → aion-gate.exe + рестарт AionOp. Откат одной командой: retarget на `C:\Temp\gate.bat` + `/run`.
7. **Финал**: ROADMAP.md обновить (статус #4 → готово; следующий = L2Authd или .NET-мелочь), `nextgen/AUTHGATE-STATUS-SNAPSHOT.md`, session-док в docs/, дельта в память.

## Критерии успеха
- Юзер логинится ru-клиентом (`-loginex -pwd16`) через НАШ гейт: welcome → LoginEx → serverlist → выбор → вход в мир.
- **authd жив и здоров после всех сессий** (ни одного лишнего пакета от нас).
- Фейлы обрабатываются красиво: неверный пароль → отказ, sessionId=0/портал → разрыв без краша и без спама в authd.
- Телеметрия по SPEC; откат одной командой проверен.

Начни с шага 1 (разведка read-only) и дай план уточнений после осмотра. Ничего на проде не меняй без «го».

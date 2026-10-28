# 🧾 СТАТУС-СНИМОК — aion-captcha (замена CAPTCHAImageServer.exe) на 22.10.2026

> Прод: **наш aion-captcha в бою на :22206, ФИНАЛ 100%** (PID см. op, задача AionCAPTCHA → D:\SAION\aion-captcha\run.cmd,
> откат = `schtasks /change /tn AionCAPTCHA /tr "C:\Temp\captcha.bat"` + `schtasks /run AionCAPTCHA`).
> Протокол capture-верифицирован: docs/captcha-protocol-20261005.md.

## 🏁 ФИНАЛ 22.10.2026 — SHIP В ЕДИНЫЙ СТЕК ПО СЕТИ (правило №2 закрыто)

- **Мост телеметрии**: reverse-SSH-туннель песочница→VM (`ssh -R 127.0.0.1:2514`),
  супервизор с авто-реконнектом на Linux (`~/STELGEN/tmp/ship-tunnel-supervisor.sh`, retry 10с),
  приёмник rsyslog в песочнице (imtcp 127.0.0.1:2514 → `/var/log/aion-ship.log`, конфиг `/etc/rsyslog.d/30-aion-ship.conf`).
- **Верификация блокера 3×** (22.10, финальная): TCP VM→192.168.0.2 (514/15140) = CONNECT-FAIL,
  UDP 514/15141 = SENT OK но не доходит → входящий в NAT-контейнер закрыт на ВСЕХ портах.
  Обход = туннель (канал песочница→VM уже был рабочим).
- **ship вкл.**: `config.yaml` (22.10): `ship.enabled: true`, `syslog { net: tcp, host: 127.0.0.1:2514 }`
  (комменты латиницей, байтово 861→695Б, LEN-check 695=695).
- **APP-фикс**: ship.go хардкод `aion-logd 2051` убран — Cfg.App/Cfg.ProcID (RFC5424 APP-NAME/PROCID,
  спека §3); main.go задаёт `aion-captcha`/`22206`; дефолт = basename argv0; тесты зелёные
  (new TestSyslogAppDefault). Новый exe MD5 `6dbc18255db7c13bf5fb66f1572f42dc`
  (bak на VM: aion-captcha-bak-86f6e311.exe).
- **LIVE-верификация**: после рестарта в /var/log/aion-ship.log: `ev=start` (`aion-captcha started`,
  listen :22206) + поток `ev=captcha` (налив буфера, seq 801→4401, ms 0-2) с шапкой
  `AION-77-PTS-EU aion-captcha[22206]` — телеметрия идёт по сети, ЕДИНЫЙ СТАК.
  Финальный self 19:17: `ok_syslog=102, err_syslog=0, dropped=0` (uptime 300с), буфер seq→9901+ (норм-тишина).
- file-фолбэк: **OFF 22.10** после self-верификации (`ok_syslog=102, err=0, dropped=0, queue=0`, uptime 300с) — ноль мусора на диске, правило №1.
- Откат ship: `ship.enabled: false` в config.yaml + рестарт AionCAPTCHA (одна правка).

## 📡 SHIP TELEMETRY ВКЛЮЧЁН (file-фолбэк) — 21.10.2026

- Новый exe: MD5 `86f6e31161d7b37eacc5125de418f9fc` (код без изменений — только config.yaml
  с ship-секцией; пересборка для идентичности тулчейна).
- ⚠️ Сетевой блокер: **UDP VM→192.168.0.2:514 дропается** (тест 21.10: VM `UDP SENT OK`,
  listener на Linux жив на 0.0.0.0:514, пакет не дошёл — NAT/фаервол). Syslog-синк ГОТОВ,
  но `ship.enabled: false` до открытия порта 514/UDP VM→Linux.
- ✅ Фолбэк по правилу №1 TELEMETRY-SPEC (явно в конфиге): `ship.file.enabled: true`
  → `D:\SAION\aion-captcha\logs\ship\YYYY-MM-DD.ndjson`.
- Деплой верифицирован 21.10: процесс PID 2328 LISTENING :22206, ship-файл создан,
  первое событие `{"ev":"start","msg":"aion-captcha started",...}` — это первый ship-выброс.
- Статус остаётся ~95%: file-фолбэк ≠ единый стак по сети (правило №2), до открытия порта.

## ✅ СВИТЧ ВЫПОЛНЕН И ВЕРИФИЦИРОВАН (13:50-13:55 VM 05.10)

- exe MD5 `5394aab1d956bce4e1300f835c2d6ee0` (win, 7.5МБ, из коммита 179a808)
- D:\SAION\aion-captcha\: aion-captcha.exe + config.yaml (verbose=true, ship disabled) + run.cmd
- Задача AionCAPTCHA пере-таргечена: C:\Temp\captcha.bat (оригинал, сохранён) → D:\SAION\aion-captcha\run.cmd
- Смерть оригинала: schtasks /end не убил process (start-детач) → одноразовая SYSTEM-задача taskkill (паттерн AionCapKill)
- Server64 (7348) переподключился сам за ~5с, налил буфер 10000 капч (~27с у нашего рендера против 6.4 мин у GDI+),
  затем тишина — ИДЕНТИЧНО поведению с оригиналом
- Server64.err после свитча: НОЛЬ captcha-ошибок (последнее = CaptchaServer Close при свитче);
  хвост — только известные мёртвые ChannelChat/petition/shopagent
- captcha.log: conn.up от 127.0.0.1:56861, captcha-события seq=1..9901 (rate 1/100), медленных нет
- Бекап: оригинал НЕ тронут (exe/config.ini/xml в D:\AION_LIVE_SERVER\CAPTCHAImageServer\ на месте);
  C:\Temp\captcha.bat (оригинальный лаунчер) сохранён

## ПОВЕДЕНИЕ В ПРОДЕ (captured 05.10)

- После reconnect Server64 → наш сервер: handshake 101/102 (эхо lang), затем ровно 10000 запросов
  1001→1002 (буфер captchaBufferSize=10000 из common.xml), затем тишина до следующего reconnect
- При логине юзера капча выдаётся Server64 из буфера — на :22206 в этот момент ТИШИНА (норма)
- Формат ответа: [seq][u32 0][u32 0][u16 0][u16 2176][u16 0x0C02][DDS header 128 + DXT1 2048][текст 12Б][u32 0]
- Валидация ответа юзера — в Server64 по тексту из 1002 (замена не участвует)
- Копейки: u16@18 (3074) и u32@6 — константы, верифицированы на 10000 пакетов

## РЕАЛИЗАЦИЯ (nextgen/aion-captcha)

- internal/proto: фрейминг/типы/parse/build; fixture-тесты байт-в-байт на живых пакетах
- internal/render: битмап-цифры 5×7 (scale+rotate+bold), кривые Безье, точки шума, палитра,
  DXT1-компрессор (c0>c1 гарантия, bbox-расширение), DDS header программно = fixture
- internal/server: accept/handshake/цикл, ship-телеметрия по TELEMETRY-SPEC (ship из logd как есть)
- конфиг: server.render = зеркало default.xml/en.xml оригинала; ship.* = формат logd
- тесты: proto (fixture), render (размеры/скорость/header), server e2e (handshake+3+налив 100)

## ПРОИЗВОДИТЕЛЬНОСТЬ

| Метрика | Оригинал (GDI+ x86 2009) | Наш (Go 2026) |
|---|---|---|
| Генерация 1 капчи | ~38мс (26/с) | ~0.4мс (2415/с) |
| Налив буфера 10000 | ~6.4 мин | ~4.2с |
| Логи | 290КБ/час (спам) | ~40 строк/налив (1/100) |

## ОТКАТ (одной командой + рестарт)

```cmd
schtasks /end /tn AionCAPTCHA
taskkill /F /IM aion-captcha.exe
schtasks /change /tn AionCAPTCHA /tr "C:\Temp\captcha.bat"
schtasks /run /tn AionCAPTCHA
```
(Server64 переподключится к оригиналу сам и заново нальёт буфер; С временно зафиксирован taskkill —
«file in use» не возникает: наш exe запущен от run.cmd, kill по IMAGENAME работает)

## АВТОСТАРТ

- Задача AionCAPTCHA (ServiceAccount, триггер Boot) → run.cmd → exe в foreground
- AION-START-ALL-v6.bat: шаг капчи не трогал (капча стартует своей задачей на буте; v6 шаг [2/12] пропускает if LISTENING)
- После бута Server64 подключается сам и наливает буфер

## АРТЕФАКТЫ

- Репо: aion-captcha/ (код+тесты+fixture), docs/captcha-recon-20261005.md, docs/captcha-protocol-20261005.md
- VM: D:\SAION\aion-captcha\ (exe+config.yaml+run.cmd+captcha.log), C:\Temp\captcha.bat (ориг. лаунчер), C:\Temp\capcap\conn-1.hex (capture 77МБ — можно удалить)
- Локально: ~/STELGEN/tmp/aion-vm/captcha-capture/ (conn-1.hex 77МБ, dds-header.hex, full-1002-sample.txt),
  ~/STELGEN/tmp/aion-vm/capmirror/main.go (capture-прокси), fake_captcha_client.py
- Коммиты: 3bfe87d (recon) → 297b716 (протокол) → 179a808 (код) → см. git log

## ЧТО СЛЕДИТЬ (пассивно)

- После рестартов пары Server64: Server64.err — не должно быть «Can't connect to Captcha server»
  (если появилось — наш exe умер; schtasks /query AionCAPTCHA + captcha.log)
- captcha.log: медленные (>50мс) события — сейчас нулевые
- Феномен «задачи сами становятся Disabled» — следить и за AionCAPTCHA

## 🖱 GUI-паритет — полная функциональная миграция (правило юзера 28.10)

> У ориг-приложения в GUI есть кнопки/опции — каждая должна получить функциональный аналог в нашей замене (функционал ориг НЕ теряем; само GUI-окно не нужно — наши сервисы headless, что ок и даже плюс для session-0). Политика: [../WORKFLOW.md §5](../WORKFLOW.md).

- ⬜ R-GUI-1: инвентаризация GUI ориг (noVNC-скриншоты окна/меню; семантика каждой кнопки/опции — PDB publics: dialog/WinProc-хендлеры) → таблица «кнопка → что делает → где аналог у нас» (сюда или в RESEARCH.md).
- ⬜ R-GUI-2: функциональный аналог каждой кнопки (op-кнопка / команда op-API / конфиг / SQL-действие) + тест; непокрытое = блокер.


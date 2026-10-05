# 🧾 СТАТУС-СНИМОК — aion-captcha (замена CAPTCHAImageServer.exe) на 05.10.2026

> Прод: **наш aion-captcha в бою на :22206** (PID 5572, задача AionCAPTCHA → D:\SAION\aion-captcha\run.cmd,
> откат = `schtasks /change /tn AionCAPTCHA /tr "C:\Temp\captcha.bat"` + `schtasks /run AionCAPTCHA`).
> Протокол capture-верифицирован: docs/captcha-protocol-20261005.md.

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

- Репо: nextgen/aion-captcha/ (код+тесты+fixture), docs/captcha-recon-20261005.md, docs/captcha-protocol-20261005.md
- VM: D:\SAION\aion-captcha\ (exe+config.yaml+run.cmd+captcha.log), C:\Temp\captcha.bat (ориг. лаунчер), C:\Temp\capcap\conn-1.hex (capture 77МБ — можно удалить)
- Локально: ~/STELGEN/tmp/aion-vm/captcha-capture/ (conn-1.hex 77МБ, dds-header.hex, full-1002-sample.txt),
  ~/STELGEN/tmp/aion-vm/capmirror/main.go (capture-прокси), fake_captcha_client.py
- Коммиты: 3bfe87d (recon) → 297b716 (протокол) → 179a808 (код) → см. git log

## ЧТО СЛЕДИТЬ (пассивно)

- После рестартов пары Server64: Server64.err — не должно быть «Can't connect to Captcha server»
  (если появилось — наш exe умер; schtasks /query AionCAPTCHA + captcha.log)
- captcha.log: медленные (>50мс) события — сейчас нулевые
- Феномен «задачи сами становятся Disabled» — следить и за AionCAPTCHA

# 🔍 CAPTCHAImageServer — разведка (шаг 1, read-only) 05.10.2026

> Все выводы read-only, прод не тронут. Следующий шаг = live capture по «го».

## 1. Каталог `D:\AION_LIVE_SERVER\CAPTCHAImageServer\`

| Файл | Что |
|---|---|
| `CAPTCHAImageServer.exe` 155,648 | MD5 `b216c07935d228dc623d047fa6c79215` (локальная копия в artifacts — идентична байт-в-байт). Сборка **Sep 15 2009 13:04:29**, версия **1.0.0.2**, PDB-путь в бинаре: `d:\CruiseControlBuild\Captcha\bin\CAPTCHAImageServer.pdb` (PDB у нас НЕТ) |
| `config.ini` 1195 (наш фикс 02.10) + `config.ini.bak-43330` | SrcIp=0.0.0.0, SrcPort=22206, SessionCount=1024, KeepAliveTimeout=0; ConcurrentThreadCount=4, WorkerThreadCount=8; LanguageCode=en (ko/th в конфиге, xml есть); [Log] Console=1 File=1 FileNamePeriod=60 (ротация 60 мин — отсюда часовые логи). Комменты cp949 — править байтово |
| `default.xml` 2094 | **ГЕНЕРАТОР КАРТИНКИ**: mime=`dds/dxt1` (!), 128×32, фон rgb 0-30, font_size=32 bold, rotate −8..8°, палитра 10 цветов (розовый/оранжевый/жёлтый/зелёный/красный/мисти + белый×6), 2 кривые по 3 точки (width 1.7), 200 точек шума 2.3×2.3 |
| `en.xml` | **ГЕНЕРАТОР ТЕКСТА**: 6 символов из `1234567890`, codepage 1200 (UTF-16), шрифты Arial/Century/Courier New/Lucida Console/MS Gothic/MS Mincho/MS PGothic/MS PMincho, include default.xml |
| `ko.xml` 6010 / `th.xml` 2132 | аналоги для ko/th (не активны: LanguageCode=en) |
| `CAPTCHAImageServer_*.log` ×60+ | часовые логи ~290КБ: спам `session = 1/1024\trequest = X\treply = Y` (~2 строки/с; X/Y = времена последнего request/reply, мс-порядок, 0 при простое); события: `ConnectionHandler/DisconnectionHandler 127.0.0.1:<порт>`, `DataHandler ... packetLength = %u, languageCodeCount = %u`; старт: Build/threads/languageCode=en/xml/SrcIp/SrcPort/SessionCount/KeepAliveTimeout |
| `*.dsn` ×6 | мусор общего пакета (aiongm/aionworld_new/L2Conn/PetitionDB) — **бинарь их не использует** (SQL-строк в exe нет, ODBC32 = мёртвый импорт) |

## 2. Кто подключается к :22206

**Единственный клиент = Server64 (PID 7348, session 1)** — постоянная сессия `127.0.0.1:54012 → 127.0.0.1:22206` (CAPTCHA PID 6968, session 0, SYSTEM-задача AionCAPTCHA Ready). **authd/AuthGateD/NPCSvr/CacheD — НЕ связаны с капчей** (grep по бинарям и конфигам = 0).

Server64: при смерти капчи ретраит сам (`Can't connect to Captcha server at 127.0.0.1:22206` в err, ~ каждые 12с), после рестарта капчи переподключается за ~9с (доказано 08:03-08:04 рестартом через aion-op). ⇒ **замена безопасна, мир не зависит**.

## 3. Server64-сторона (реверс строк, без capture)

- Модули: `CaptchaClient.cpp/CaptchaSocket.cpp/CaptchaAttr.cpp`; поток «Captcha thread».
- `CaptchaSocket`: OnCreate/OnRead/SendIOBuffer/OnClose; логи `Captcha server send wrong packet type`, `bad packet size %d`, `Captcha REPLY ERROR: (%d) %s`.
- Валидация в Server64: `[CAPTCHA] Answer is arrived wrong time`, `[CAPTCHA] Answer count is different` (правильный текст Server64 знает).
- Мир-пакеты: `C_CAPTCHA` (ответ юзера), `S_CAPTCHA` (вопрос юзеру); `User::SendCaptchaQuestion`, `ReceiveCaptchaAnswerPacket`.
- CacheD-RPC: `RQ_CAPTCHA/RP_CAPTCHA` (+ БД: `RequestCaptchaInfo send to db error`); builder-канал: `GQ_CAPTCHA/GP_CAPTCHA`, команды `turn_captcha/ask_captcha/clear_captcha` («captcha enabled/disabled by builder command»).
- Конфиг в `MainServer\common.xml` L37-40: `useCaptcha=true`, `captchaBufferSize=10000`, `captchaServerAddr=127.0.0.1`, `captchaServerPort=22206`.
- Капча НЕ на каждый логин: активность request/reply только в окнах логинов 05.10 (00:30, 01:31, 08:03+) — механизм `captchaBoostRate`/бот-детект.

## 4. Протокол-строки капчи (utf16 в exe)

`CAPTCHA_LANGUAGE_CHECK_REPLY`, `CAPTCHA_REPLY`, `unsupported language code`, форматы `image/jpeg | dds/dxt1 | dds/dxt3 | dds/dxt5`, `invalid/unsupported code page`, `text conversion failure`, `packetLength`, `languageCodeCount`. Рисование GDI+, компрессия DXT1/DXT3/DXT5 (константы `DXT1/DXT3/DXT5/DDS ` в .rdata). Классы: `CAPTCHAImageServer/CAPTCHAImageManager/CAPTCHAImageSession/KeepAliveManager/ISession/IKeepAlive/captcha_image_c`.

## 5. Что ещё нужно (шаг 2 — capture по «го»)

Wire-формат: handshake (что шлёт Server64 при OnCreate / LANGUAGE_CHECK?), формат запроса генерации (текст от Server64 или капча генерит сама?), формат CAPTCHA_REPLY (текст+картинка?). DXT1 128×32 = ровно 2048 байт данных — вероятно, payload.

Схема capture (как logd): копия капчи на `:22207` (конфиг байтово) + наш mirror-прокси на `:22206` (rx/tx → hex) → юзер делает логины/провоцирует капчу → реставрация (kill прокси + `/run AionCAPTCHA`). Риски: мир жив (retreat доказан).

## 6. Интеграция при замене

- aion-op: задача `AionCAPTCHA` — пере-таргетить на `D:\SAION\aion-captcha\run.cmd` (паттерн AionLogCap); метрика порта 22206 остаётся.
- Замена = D:\SAION\aion-captcha\ (exe + config.yaml + run.cmd), автостарт юзер-сессией (шаг в AION-START-ALL-v7.bat или замена шага капчи в v6).
- ship из nextgen/aion-logd/internal/ship как есть; TELEMETRY-SPEC.

## 7. Артефакты разведки

- Локально: строки exe `/tmp/cap-ascii.txt`, `/tmp/cap-u16.txt`; Server64 `/tmp/s64-u16.txt`, `/tmp/s64-ascii.txt`; gate/auth строки `/tmp/gate-*.txt`, `/tmp/auth-*.txt`
- VM: `C:\Temp\captcha_recon1..5.ps1`, `captcha_q1..2.sql` (read-only, можно удалить)

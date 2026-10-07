# aion-captcha — замена CAPTCHAImageServer.exe (Трек B #2 — ✅ В БОЮ с 05.10)

> Протокол capture-верифицирован 05.10.2026: `docs/captcha-protocol-20261005.md`.
> Телеметрия по `nextgen/TELEMETRY-SPEC.md` (ship скопирован из aion-logd как есть).
> ✅ Прод: `D:\SAION\aion-captcha\` (exe MD5 `5394aab1` + config.yaml + run.cmd), задача AionCAPTCHA
> → run.cmd, PID 5572 на :22206, буфер 10000 за ~4с, Server64.err чист. Откат:
> `schtasks /change /tn AionCAPTCHA /tr "C:\Temp\captcha.bat"` + `/run AionCAPTCHA`.

## Протокол (TCP :22206)

Фрейм `[u16 LE totalLen][u16 LE type][payload]`, seq — u16 счётчик пер-тип с 1.

| Тип | Напр. | total | Payload |
|---|---|---|---|
| 101 LANGUAGE_CHECK | C→S | 8 | [seq][langCode u16 0x656e = «en» BE-порядок] |
| 102 LANG_CHECK_REPLY | S→C | 8 | эхо seq+langCode |
| 1001 CAPTCHA_REQUEST | C→S | 18 | [seq][u32 0][u16 0][codepage u32=1200][langCode u16] |
| 1002 CAPTCHA_REPLY | S→C | 2212 | [seq][u32 0][u32 0][u16 0][u16 imageSize=2176][u16 0x0C02][DDS blob 2176][текст 6 цифр UTF-16LE 12Б][u32 0] |

Server64 после reconnect наливает буфер ровно captchaBufferSize=10000 запросов (temp ~26/с у
GDI+-оригинала), потом тишина; при логине юзера капча выдаётся из буфера. Наш рендер ~2400/с —
налив занимает ~4с.

## Сборка

Go 1.22+ (тулчейн `~/STELGEN/go-dist/go/bin`):
```bash
go test ./...
go build -o aion-captcha-linux .            # linux
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-s -w' -o aion-captcha.exe .
```

## Тесты

- `proto` — байт-в-байт на fixture живых пакетов (testdata/*.hex из capture 05.10).
- `render` — размеры, DXT1 2048Б, DDS header fixture, скорость (<30мс/картинку).
- `server e2e` — фейк-клиент: handshake + 3 запроса + налив 100 (seq/текст/размеры).
- фейк-клиент для VM/локали: `~/STELGEN/tmp/aion-vm/fake_captcha_client.py` (host port n).

## Конфиг

`config.yaml` — зеркало полей оригинала (default.xml/en.xml): размеры 128×32, 6 цифр
«1234567890», rotate ±8°, 2 кривые, 200 точек, палитра 10 цветов + `ship.*` секция
(идентична logd; enabled=false по умолчанию — прод без приёмника не меняется).

## События ship

start/stop (uptime, счётчики), conn.up/conn.down (remote), `captcha` (каждый 100-й запрос +
медленные >50мс: seq/text/ms), parse.err (raw капится), self (каждые 300с).

## 📦 Артефакты

| Что | Где |
|---|---|
| Код/конфиг/fixture | этот каталог (internal/{proto,render,server,ship}, testdata/) |
| Статус/ресёрч/промпт-архив | SNAPSHOT.md, PROMPT-ARCHIVE.md, docs/ (recon/protocol/session) |
| Прод | `D:\SAION\aion-captcha\` (exe+config.yaml+run.cmd) |
| Креды/доступы | VM `D:\SAION\creds\` (CREDS.md) |

## Деплой (по «го»)

1. `D:\SAION\aion-captcha\`: aion-captcha.exe + config.yaml + run.cmd.
2. Пере-таргет задачи `AionCAPTCHA` на run.cmd (паттерн AionLogCap); откат = taskkill aion-captcha
   + вернуть /tr на exe оригинала + `schtasks /run AionCAPTCHA`.
3. Верификация: netstat :22206 LISTENING + ESTABLISHED от Server64; Server64.err — нет
   «Can't connect to Captcha server»; наш ship — captcha-события.

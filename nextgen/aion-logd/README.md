# aion-logd — замена LogServer64 (В БОЮ на проде, Л1–Л4 закрыты)

Один Go-исходник → `GOOS=windows/linux`. Анализ протокола: [../LOGD-REWRITE-ANALYSIS.md](RESEARCH.md).
Статус-снимок: [../LOGD-STATUS-SNAPSHOT.md](SNAPSHOT.md). Стандарт телеметрии: [../TELEMETRY-SPEC.md](../TELEMETRY-SPEC.md).

## Статус: ПРОД (замена штатная) + Л1–Л4 (05.10.2026)

- ✅ Протокол 100%, DB-слой (freedisk/serverstatus/InitializeCount), боевая замена на :2051 (см. снимок);
- ✅ **Л1** `internal/textlog`: type-9 парсер (онлайн-сессии `[key][wchar NUL]` + SYSTEMTIME-хвост
  = последние 16 байт; fixture'ы — реальные capture) → per-service .err/.log + ship-события;
- ✅ **Л2** logdb: методы UpdateMainStatus/UpdateTotalMainStatus (позиционные {call}) ГОТОВЫ,
  вызов ЗАГЛУШЕН до деплоя REF58-проц (на проде их НЕТ — оригинал тоже не писал зоны, 2812) + маппинга metric1-4;
- ✅ **Л3** InitializeCount на ServerStarted svc=3 (старт мира), config `server.init_svc`;
- ✅ **Л4** retention: writer.Sweep чистит base_dir старше `server.retention_days` (старт + 30 мин);
- ✅ **ship** `internal/ship`: syslog RFC5424 udp/tcp + HTTP ndjson + локальный фолбэк (только явно);
  неблокирующе, не падает без приёмника, self-статус ev=self, raw+ошибки наружу — по TELEMETRY-SPEC;
- ✅ Тесты: proto/records(317)/logdb/textlog(живые fixture)/ship(UDP/TCP/file/недоступный) — все зелёные.

## Запуск (dev)

```bash
go build -o aion-logd . && ./aion-logd -config config.yaml   # :2051
GOOS=windows GOARCH=amd64 go build -o aion-logd.exe .        # кросс-сборка
```

## Остаток дорожной карты

1. Подмена exe на проде новым (staged: `C:\Temp\logd-staged\aion-logd.exe`, MD5 89254d1b):
   `schtasks /end AionLogCap` → копировать в `D:\SAION\aion-logd\` → `schtasks /run AionLogCap` (откат — старый exe там же в bak);
2. Л2-добой: деплой 2 REF58-проц (`scripts/sql/ref58-logprocs-pending-20261005.sql`) + маппинг metric1-4 → mainstatus;
3. Приёмник на Linux (rsyslog) → включить `ship.enabled` в прод-конфиге;
4. Опц.: дизasm хвоста type-9 (tick/coords/flags) — в raw уже всё есть.

## ПРОД-ДЕПЛОЙ ФИНАЛ (05.10.2026, решение юзера: наш логгер НАВСЕГДА)

| Что | Путь |
|---|---|
| Бинарь | `D:\SAION\aion-logd\aion-logd.exe` (конфиг `config.yaml`, запуск `run.cmd` → задача AionLogCap) |
| Логи/дампы | `D:\SAION\aion-logd\logs\` (status CSV per-day svc301/302/309, io-дамп rx/tx, capture.raw, badstatus.raw — всё с таймстампами) |
| Dev-набор | `D:\SAION\aion-logd-dev\` (src/bin/scripts/artifacts + README-BUILD.txt) — всё для финальной компиляции |
| Откат на оригинал | `taskkill /F /IM aion-logd.exe` + `schtasks /run /tn AionLog` (ориг LogServer64; common.xml на :2051 — возвращён) |
| Секреты | SQL-пароль только в `config.yaml` на VM (в гит НЕ попадает); полный реестр доступов: `D:\SAION\creds\` (CREDS.md) |
| Ресёрч/план/промпт | RESEARCH.md (анализ+протокол), SNAPSHOT.md (статус) — в этой папке |

`D:\SAION\` = папка наших переписанных апок (следующие — туда же).

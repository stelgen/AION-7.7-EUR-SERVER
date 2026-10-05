# aion-logd — замена LogServer64 (skeleton)

Один Go-исходник → `GOOS=windows/linux`. Анализ протокола: [../LOGD-REWRITE-ANALYSIS.md](../LOGD-REWRITE-ANALYSIS.md).

## Статус: SKELETON

- ✅ Фрейминг `[u16 LE len][type 0..4][0xBB][~type][payload]`, ограничение 0x2000 (как в `GetCmd_LP`);
- ✅ Handshake: клиентский `Version(builderNumber)` (проверка ≥10003) → ответ `Version(наш builder, min 10003)` + `VersionAndTime([02 BB FD][qword FILETIME])` — по мотивам `LogServerSocket::OnCreate`/`SendVersionAndTime` (байтовая разметка сверена с дизasm'ом);
- ✅ Неизвестные payload (кандидат — тип 4, лог-данные) → **не пишутся в боевые файлы**, падают в `logs/payload.raw/*.t4.hex` до живого capture;
- ✅ Писатель per-service дневных файлов (`YYYY-MM-DD.err/.log` — формат оригинала), DB-проц (`aion_BulkInsertWide`, таймеры) — **каркас, включается после capture**;
- ✅ Тесты: proto roundtrip/marker/двух-пакетная склейка + фейк-клиент e2e (handshake, разрезание пакета по read'ам).

## Запуск (dev)

```bash
go build -o aion-logd . && ./aion-logd -config config.yaml   # :2051
GOOS=windows GOARCH=amd64 go build -o aion-logd.exe .        # кросс-сборка
```

## Дорожная карта (см. LOGD-REWRITE-ANALYSIS.md)

1. Живой capture: параллельный прогон logd-прокси на :2052 (зеркало трафика 2051 без вмешательства) → точный payload лог-батча.
2. Парсер батча (LogSvcType + SYSTEMTIME + wide text, DecodeBotLog-эквивалент) → запись .err как оригинал.
3. DB-слой (go-mssqldb): 3 таймер-проц + `aion_BulkInsertWide`/`aion_SetInserted`.
4. Параллельный прогон → diff файлов с оригиналом → свитч (откат = вернуть задачу AionLog).

## ПРОД-ДЕПЛОЙ ФИНАЛ (05.10.2026, решение юзера: наш логгер НАВСЕГДА)

| Что | Путь |
|---|---|
| Бинарь | `D:\SAION\aion-logd\aion-logd.exe` (конфиг `config.yaml`, запуск `run.cmd` → задача AionLogCap) |
| Логи/дампы | `D:\SAION\aion-logd\logs\` (status CSV per-day svc301/302/309, io-дамп rx/tx, capture.raw, badstatus.raw — всё с таймстампами) |
| Dev-набор | `D:\SAION\aion-logd-dev\` (src/bin/scripts/artifacts + README-BUILD.txt) — всё для финальной компиляции |
| Откат на оригинал | `taskkill /F /IM aion-logd.exe` + `schtasks /run /tn AionLog` (ориг LogServer64; common.xml на :2051 — возвращён) |
| Секреты | SQL-пароль только в `config.yaml` на VM (в гит НЕ попадает) |

`D:\SAION\` = папка наших переписанных апок (следующие — туда же).

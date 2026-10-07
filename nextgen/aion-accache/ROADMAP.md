# ROADMAP aion-accache — AccountCacheServer (Go) — живой план

Обновлено: 07.10.2026. Факты/ресёрч: RESEARCH.md; артефакты: accountcache-ref/.
Общие правила эксплуатации: ../ROADMAP.md §4 (ssh-PowerShell готчи, «го» на прод-действия,
бекапы, секреты не в гит, TELEMETRY-SPEC).

## Прогресс

| Фаза | Статус | Что |
|---|---|---|
| R0 | ✅ | VM-разведка: PDB 92МБ+map+exe скачаны, тела 101 procs+21 таблица, клиент 2220=Server64 (07.10) |
| R0.5 | ✅ | Dispatch-таблица дизasmом: wire-фрейм [len-2][cmd][0xEB][~cmd], T1 cmds 0-39, T2 0-7, cmd 22 не занят (07.10) |
| R2 | ✅ | Каркас nextgen/aion-accache: proto+dispatch+cache+db-интерфейс+server+ship, тесты зелёные (07.10) |
| R1 | ⏳ | fork-proxy :2220 (невидим, паттерн authd R5) + capture при логинах юзера → payload-раскладки каждого cmd + ACP-номера ответов + семантика T2-канала |
| R3 | ⏳ | SQLStore (go-mssqldb) + полный хендлер-набор по телам procs (заглушки по мере наблюдения) |
| R4 | ⏳ | A/B fork-прогон: копия трафика в наш, ориг НЕ трогать, байт-в-байт дифф |
| R5 | ⏳ | Свитч по «го» (задача AionAcc retarget на D:\SAION\aion-accache\run.cmd; откат = retarget обратно) |

## R1-детали (следующий чат)

1. fork-proxy Go (паттерн forkauthd): front :2220 → ориг-порт... ⚠ прод-ACS слушает 2220 НАПРЯМУЮ;
   схема = копия ACS на :2221 (байтовая правка common.xml serverPort, паттерн captcha-mirror) +
   наш proxy на :2220 → forward :2221 + копия hex в лог. Ориг НЕ трогать, юзер ничего не замечает.
   Альтернатива (если Server64 капризен к смене порта) — pktmon/netsh trace на 2220.
2. Capture-сценарий: юзер логинится ×N (Server64 сам наливает чар-флоу: FIRST_LOAD/CHAR_LOGIN/SAVE_CUSTOM/LUNA).
3. Разбор: payload-раскладка per-cmd (арбитр = PDB-сигнатуры Decode*), ACP-номера (vtable SendIOBuffer),
   T2-канал (кто второй клиент: возможен отдельный сокет того же Server64).
4. Промпт: PROMPT.md.

## Критерии готовности R4

- Все наблюдаемые cmds парсятся без ошибок (нет unhandled/parse.err в логах)
- FIRST_LOAD/CHAR_LOGIN/LOGOUT ответы байт-в-байт == ориг (fork-дифф)
- БД-запись: account_data/hidden_fatigue апдейты == ориг (SQL-дифф по времени)

## Откаты

- R5-свитч: schtasks /change /tn AionAcc /tr обратно (C:\Temp\acc.bat) + /run
- Бекапы: ориг-каталог D:\AION_LIVE_SERVER\AccountCacheServer НЕ трогается

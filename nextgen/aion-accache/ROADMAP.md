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
| R1 | ✅ | **capture + wire-канон 10.10**: accmirror v1.1 (:2220→:2221) + копия ACS-2221; 2 логина; дизasm OnRead/GetCmd_ACQ → **len=total** (офф-бай-2 в v1.0/каркасе исправлен), маркеры EB/EC, 0 bad frames на всём capture (RESEARCH §11). Стенд ОСТАВЛЕН работать (дозахват) |
| R2.5 | ✅ | **Раскладки 10.10**: internal/payload (Version/FirstLoad/BM/Luna/CharLogin/Logout/Fatigue с UTF-16 stamp) на живых golden-кадрах, тесты зелёные; tools/accparse.py v2; frames.txt в capture-артефактах. Хвосты в R3: роли полей Fatigue/Trial (PDB I,H,I,I,I), push 13/15/20/21, T2-канал |
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

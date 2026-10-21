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
| R4 | ⏳ | **частично**: replay-корпус готов (TestR4ReplayCapture — автономный прогон capture C2S → fakeDB-сервер → байт-дифф с S2C, XFAIL-отчёты, % покрытия), фулл A/B fork-прогон с оригом — дальше |
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

## 🧪 Журнал теорий (R4-хвосты, 21.10)

| # | Гипотеза | Статус |
|---|---|---|
| Т1 | LOAD_LUNA reply ts = limit_play_reset_time (account_data) | ✅ ПРОД-ФАКТ: acc=1010 колонка = 0x6AC5DFE0 == capture ts байт-в-байт. Реализовано |
| Т2 | ACP_21 (S2C 0x15) = push после LOAD_BM_PACK [acc][02][01][8×0] | ✅ capture ×2, байты идентичны. Реализовано (позиция async) |
| Т3 | S2C 0x06 = ответ на LOAD_TRIAL (НЕ UPDATE_BM_PACK!): payload содержит sell=917/trade=70 из trial_account_data acc=1010 (прод-факт) | ✅ Реализовано: [acc][X echo][reset][sell][CONST0][trade][decompose][gather][extract][pad]; dword3=CONST0 (роль TBD, единственный кадр) |
| Т4 | FIRST_LOAD при пустой строке account_data → point=1 | ⚠ эмпирика golden acc=7 (строки нет, ответ point=1); альтернатива — строка существовала и удалена после capture. Реализовано как capture-канон |
| Т5 | FIRST_LOAD u8@12/u16@17 = производные limit_play_accum_time | ⚠ гипотеза (все golden = 0; byte(limit_play_reset_time)=0xE0 исключён). Реализовано accum-based |
| Т6 | UPDATE_HIDDEN_FATIGUE point НЕ зеркалится ACS в account_data | ⚠ golden#2 point=1 несмотря на UPDATE point=0; prod сейчас point=0 (пост-capture активность). Fake replay не мутирует point |
| Т7 | ACP_13 (S2C 0x0d, 4Б ts 0x640831AE=2023-03-05) | ❌ значения НЕТ ни в одной таблице prod-БД; отправлен 1 раз за capture; триггер неизвестен. НЕ реализовано (риск для A/B) |
| Т8 | Logout-кластер S2C 0x0f+0x14+0x1c = отложенный flush CHAR_LOGOUT (пришёл через 13 мин, вместе с reply на следующий FIRST_LOAD) | ⚠ данные частично необъяснимы: 0x14 содержит level=80 (=global_user_data.user_level прод ✓) и ts 0x6AC7E7B9 (источник не найден); 0x0f содержит 0x000A07EA (657386с?) без источника. НЕ реализовано до живого capture |
| Т9 | CHAR_LOGIN reply приходит РАНЬШЕ replies на предыдущий батч (UPDATE_LUNA/LOAD_LUNA после CHAR_LOGIN) | ✅ ACS многопоточен (numberOfDBThreads=10) — порядок replies недетерминирован; выравнивание по cmd |

## R4 итог 21.10

**12/16 golden S2C (75%) байт-в-байт.** Реализованы: ts-фикс LOAD_LUNA (Т1), ACP_21 push (Т2), trial-ответ S2C 0x06 (Т3), point=1 дефолт (Т4), accum-based X/Y (Т5), stateful replayExec (updatetime мутится Exec'ом, trial-row 917/70). Остаток = 4 кадра с непроверенными данными (Т7/Т8) — эмитить до живого capture опасно для Server64.

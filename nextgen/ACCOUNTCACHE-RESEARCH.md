# AccountCacheServer — ресёрч (07.10.2026)

> Контекст: переписываем стек AION 7.7 PTS на Go. Сделано: aion-captcha ✅ (прод), aion-logd ✅ (прод),
> aion-gate ✅ (прод), aion-authd MVP ✅ (R1-R4, не деплоен). Следующий кандидат = AccountCacheServer (2220).
> Этот док = всё, что накопано до нас + наша добыча из 5.8-кита.

## 1. Что это такое

**AccountCacheServer (ACS)** — NCSoft-стиль CacheD для аккаунтов: RAM-кэш + ЕДИНСТВЕННЫЙ писатель
в БД `AionAccountCacheD`. Все "аккаунт-уровневые" данные вне мира живут тут и в ней:

- core account data (`aion_Get/SetAccountData`) — грузится при первом логине акка (`ACQ_FIRST_LOAD_ACCOUNT_INFO`)
- login/logout/create/delete юзеров (`aion_SetLoginUser_20121206` и т.п.)
- кастомизация чаров (shape/appearance): `ACQ_LOAD/SAVE_CAHR_CUSTOM` (опечатка CAHR — фирменная, NCSoft)
- BM packs, Board BM, hidden fatigue, login events, promotion cooltime
- Luna (валюта): load/update/reward/key
- trial accounts, jumping characters, playtime polls, user ranking (GM)
- перенос чаров между серверами (`ACQ_MOVE_CHAR_REQUEST_FROM_ORI_SVR`)

⚠ НЕ путать: **CacheServer (CacheD64, порт 2006)** = кэш мира (персонажи-в-мире, items).
AccountCache = аккаунты и чар-мета. Клиенты разные.

## 2. Топология (кто с кем)

```
L2Authd (AuthD) ──TCP 2220──► AccountCacheServer ──ODBC──► [AionAccountCacheD]
                                   ▲ (22.02 и др. каналы — проверить capture)
Server64/мир (char login/logout/level/custom/luna/GM-поиск)
GMServer (GM_* procs, GQ/GP)
```

- authd: `AuthD/etc/config.txt` → `accountCachedPort = 2220` (в нашем ките подтверждено)
- сам ACS: `_SERVER/AccountCacheServer/common.xml` → `<serverPort>2220</serverPort>` (в ките 7.7 = 2220)
- БД: SQL Login окно при ПЕРВОМ старте (File DSN `AccountCacheD.dsn` / `aion_accoutdb`),
  после чего connStr сохраняется в реестр (`LoadConnStrFromReg`/`SaveConnStrToReg`, ветка NC Soft)
- в common.xml есть `mailServer` — dev-mail алерты (SMTP `aionserver@ncsoft.net`), сам канал 2051 LogServer закомментирован в 5.8 конфиге

Порядок старта в гайдах (4.6/7.7): ACS = **сервис №1** (раньше AuthServer). Ждать в логе
`* server ready on port %d`.

## 3. Тех. профиль бинаря (5.8-кит, AccountCacheServer.exe)

| Параметр | Значение |
|---|---|
| Размер | 20 528 640 байт (20.5 МБ) |
| MD5 | `4f1dd275db1470a59b62fd4f3b8459b4` |
| Тип | PE32+ x64, GUI, 6 секций |
| Дата сборки | 29.05.2020 (5.8-эпоха) |
| PDB-путь | `D:\_build\out\server\AccountCacheServer.pdb` — **PDB публично НЕ выложен** |
| БД-доступ | ODBC32.dll (прямой), DSN-имя дефолт `aion_accoutdb` |
| connStr | реестр (`SaveConnStrToReg`), окно SQL Login при первом старте |
| Прочее | perfmon.ini счётчики, deadlock-dumper (`Deadlock or severe lag detected`, LockChecker, TimerQueue dump в AIONErr.txt), UseFullDump |
| Артефакт в ките | рядом лежит **RegisterAccount.exe** (CLI создания аккаунтов) — есть в 4.6-ките |

⚠ Exe содержит и общую мир-кодовую базу NCsoft (mail/item/guild/GP_*) — 20 МБ это shared-кода,
не пугаться. AccountCache-специфика = ACQ/ACP + aion_* procs.

## 4. RPC-протокол (главная находка)

Полный словарь из strings exe (UTF-16): **ACQ_* = запросы к ACS, ACP_* = ответы/пуши ACS.**
Имена = enum-опкоды (есть `ACQ_MAX`/`ACP_MAX` → дискретная таблица диспетча). 71 команда.
Файл: `nextgen/accountcache-ref/rpc-opcodes-58.txt`. Ключевые:

- Handshake: `ACQ_VERSION_PACKET` → `ACP_VERSION_RESPONE`
- Логин-флоу: `ACQ_FIRST_LOAD_ACCOUNT_INFO` → `ACP_FIRST_LOAD_ACCOUNT_INFO` (+ `ACP_REQUEST_USER_INFO`)
- События мира: `ACQ_CHAR_LOGIN/LOGOUT/CREATED/DELETE/DELETE_COMPLETED/INFO_REFRESH/LEVEL_CHANGED`
- Чар-кастом: `ACQ_LOAD_CAHR_CUSTOM` / `ACQ_SAVE_CAHR_CUSTOM` / `..._BY_ITEM`
- Luna: `ACQ_CONFIRM_LUNA_REWARD / DECREASE_LUNA_KEY / LOAD_LUNA / UPDATE_LUNA / RESET_LUNA_REWARD`
- BM/fatigue/events: `ACQ_UPDATE_BM_PACK`, `ACQ_LOAD_BM_PACK_LIST`, `ACQ_UPDATE_BOARD_BM_STATE`,
  `ACQ_UPDATE_HIDDEN_FATIGUE`, `ACQ_UPDATE/DELETE_LOGIN_EVENT_RECORD`
- Trial/jump/move: `ACQ_LOAD/UPDATE_TRIAL_ACCOUNT_DATA`, `ACQ_ASK_CAN_MAKE_JUMPING_CHARACTER` →
  `ACP_CAN_MAKE_JUMPING_CHARACTER`, `ACQ_MOVE_CHAR_REQUEST_FROM_ORI_SVR` → `ACP_MOVE_CHAR_REQUEST_FROM_ACC`,
  `ACP_REQUEST_MOVE_SVR_USER`
- Тесты/служебные: `ACQ_TEST_INSERT/SELECT`, `ACQ_GEN_TEST_PACKET1-3`, `ACQ_SYNC_PACKET_TEST`, `ACQ_LOG_INFO`
- Результаты: `ACP_SELECT_RESULT`, `ACP_INSERT_RESULT`

Формат фрейма strings не даёт (нужен capture или дизasm). Ожидание: NCSoft-RPC как на 2110 —
[u16 len][u16 opcode][payload], опкод = номер из enum ACQ_*.

## 5. БД AionAccountCacheD (эталон 5.8)

Полный список процедур: `nextgen/accountcache-ref/db-procs-58.txt` (UTF-16 → открыть в utf-16).
Сигнатуры-версии: `_20120703 / _20121206 / _20160303 / _20170428` — миграции по релизам.
Ядро: `aion_Get/SetAccountData(_20151117/_20170428)`, `aion_SetCreateUser_20160303`,
`aion_SetLogin/LogoutUser_20121206`, `aion_SetUserInfo_20160303`,
`aion_PutUserDataFrom(All)Server`, `aion_SyncUserDataFromServer`, `aion_GetOldestCreateDate`,
`aion_GetServerReplaceList`, `ap_getservers`. Тела процедур в инете НЕ выложены (проверено);
на VM эталон в SQL (REF58) — снять `sp_helptext` в R0.

## 6. Что прошли до нас (публичный передний край)

- **НИКТО** публично не реверсил и не переписывал ACS. Сурцов нет, PDB нет. Проверено:
  ragezone (треды 1197401 AION 4.6 retail files, 1211744 AION 7.7pts EU — это наш кит),
  mmo-dev, github (`AccountCacheServer`, `AccountCacheD` — только шум/чужие проекты).
- В ragezone 1197401/1211744 ACS упоминается только в гайдах запуска: старт №1, DSN
  `AccountCacheD.dsn`/`aion_accountdb`, жаловались на битые stored procedures в ДБ.
- mmo-dev тред 22808 («AuthD classic off», x64 classic 162-287, `Auth.7z` 609КБ) — только authd,
  файла под логином; ACS там нет. Ссылка для юзера: https://mmo-dev.info/threads/22808 (нужна регистрация).
- L2-аналог (MasterToma C1 `CacheD`, 692 cpp/h, локально `~/STELGEN/tmp/authd-research/artifacts/l2_c1/`)
  — архитектурный референс семейства, НЕ протокольный паритет.
- AION-эмуляторы (beyond-aion, Mobius, AionLightning) — Java/GS-стек, ACS не реализуют.

## 7. Артефакты (в гит: nextgen/accountcache-ref/)

- `config.xml` — полный конфиг 5.8 (serverPort 2220, mailServer, DSN-примеры, numberOfDBThreads=10)
- `AccountCacheServer.common` / `.config` — снятые с нашей VM 7.7 (country=7, serverTitle «AION ACS»)
- `rpc-opcodes-58.txt` — 71 RPC ACQ/ACP
- `db-procs-58.txt` — procs эталонной БД 5.8
- `perfmon.ini`, `AIONErr.txt` (deadlock-дамп от 06.09.2024)
- exe 5.8 локально (НЕ в гит): `~/STELGEN/tmp/aion-vm/accache58/AccountCacheServer/AccountCacheServer.exe`

## 8. План MVP (aion-accache, по аналогии aion-authd)

- **R0 VM-разведка (read-only)**: MD5 exe 7.7 (`D:\AION_LIVE_SERVER\AccountCacheServer\`) vs 5.8
  `4f1dd275...` (если совпадёт — у нас полный бинарь), PDB рядом?, netstat 2220 (кто клиенты),
  `sp_helptext` procs REF58_AionAccountCacheD (тела!), таблицы БД (sys.tables), реестр connStr,
  конфиг common.xml 7.7, лог-файлы сервиса, `RegisterAccount.exe` в папке?
- **R1 wire**: fork-proxy на 2220 (паттерн authd R5) — capture при логинах юзера. Выяснить:
  фрейм-формат, opcode-нумерацию, version-handshake, payload'ы FIRST_LOAD/CHAR_LOGIN.
- **R2 Go-каркас**: nextgen/aion-accache — фрейминг + dispatch по opcode + RAM-кэш.
- **R3 DB-слой**: SQLServer через процы (реальные из R0; недостающие — заглушки по наблюдению).
- **R4 A/B fork-proxy**: наш параллельно, ориг НЕ трогать, diff.
- **R5 свитч**: только после паритета, откат = retarget задачи.

**Оценка шансов: ~80-85%.** Блокеров нет: capture-методика отработана (captcha/logd/gate/authd),
протокольный словарь уже снят, клиент один (authd) + редкий трафик мира. Главная неизвестность —
wire-формат фрейма (закрывается R1) и тела procs (закрывается R0 sp_helptext).

**Что НЕ трогаем**: прод-ACS живой, порт 2220 занят им, fork-proxy только по «го».

## 9. R0-РАЗВЕДКА ВЫПОЛНЕНА (07.10.2026, read-only, SSH администратор@192.168.0.125 ключ dimini-agent)

**ГЛАВНАЯ НАХОДКА: ПОЛНЫЙ PDB НА VM.** `D:\AION_LIVE_SERVER\AccountCacheServer\`:

| Файл | MD5 | Примечание |
|---|---|---|
| AccountCacheServer.exe (20 587 520) | `33ada1f84d0bce0f86018924a2dc3f93` | 7.7-бинар, сборка 09.06.2020, НЕ равен 5.8 (`4f1dd275`) — другой билд |
| **AccountCacheServer.pdb (92 467 200)** | `17481fc5c4445251e20c25ae94550299` | ПОЛНЫЕ СИМВОЛЫ! (= manifest-pdb-big) |
| AccountCacheServer.map (4 835 616) | `1286ea3423735dfe14d3a8c84072b5ab` | секции + symbols, Timestamp 5ed4dff5 (01.06.2020) |
| AIONErr.txt | — | deadlock-дамп 04.10.2026 21:20 (TimerQueue) |
| dsn-набор | — | aion_accoutdb/aion_accountdb/L2Conn/aiongm (в AC папке — коннекты к ВСЕМ базам стека!) |
| config.xml (77Б), common.xml (621Б) | — | сняты в ref |

Артефакты локально (НЕ в гит): `aion_rev_2026-10-05/artifacts/pdb-big/AccountCacheServer/` (exe+pdb+map+**pdb-publics-8035.txt**).

**Сеть (netstat 2220, живой прод):** единственный клиент = **Server64 (PID)**, ESTABLISHED 127.0.0.1→2220. Authd НЕ держит постоянный коннект (подключается лениво/по требованию — уточнить при capture). Аккаунт-мета (char login/logout/custom/luna) льётся из МИРА, не только из authd.

**БД REF58_AionAccountCacheD (5.8-эталон) — ТЕЛА ПРОЦЕДУР СНЯТЫ** (`accountcache-ref/db-procs-77-ref58.rpt`, 101 proc + 21 таблица, 91КБ): account_data (hidden_fatigue_point/updatetime/npckill, limit_play_reset/accum), account_fatigue, account_luna(+_reward), account_pack, aion_ranking_info/season_status, aion_server_data, aion_serverlist, aion_user_ranking_*(season_history/servermove), cosmetic_data, global_user_data, jumping_character_config, server_operation, trial_account_data, user_board_bm(+dice/game), user_login_event_data(+daily/other/renewal), user_monster_core, user_promotion_cooltime, user_transform. Тела — тривиальные SELECT/INSERT/UPDATE, все параметры сняты (GetAccountData_20170428(accountId) → hidden_fatigue поля; GetAccountPackList → pack_type/expire_date...).

**RPC 7.7 = 74 команды** (`rpc-opcodes-77.txt`), +3 vs 5.8: `ACQ_MONSTER_CORE_UPDATE_VALUE`, `ACQ_MONSTER_CORE_UPGRADE`, `ACQ_TRANSFORM_OPERATION`.

**PDB publics (8035, pdbpub.py)** — RPC-кодеки с ПОЛНОЙ параметрикой: классы `ServerToAccountCached` (Decode*: запросы мира→ACS) и `AccountCachedToServer` (Encode*: ответы) — DecodeFirstLoadAccountInfo/Version/CharLogin(SpecialSvrTypeEnum,userId wchar)/CharDelete/RefreshUserInfo(AccGlobalCharInfo)/UpdateLoginEvent/UpdateHiddenFatigue/LoadLuna/ConfirmLunaReward(LunaParam)/DecreaseLunaKey(_J=qword)/CanMakeJumpingCharacterStatus/LoadPreviousPlayTime*/TransformOperation(TransformDBSubType,TransformDBData); `DBConn` = ODBC-слой (Init/AllocSQLPool/ExecuteInsert/Delete/AndAddLog/SetAutoCommit/Bind). Исходники-пути в exe: `d:\_build\src\shared\MemoryMan.h`, `IoCompletion.h`. Полный дизasm по именам — метод logd (objdump + .map).

**ШАНСЫ ОБНОВЛЕНЫ: ~90%** (было 80-85) — PDB+map+procs+RPC-словарь сняты, осталось только wire-формат фрейма (capture R1) и opcode-нумерация (дизasm dispatch по PDB-именам).

**Бонус authd-треку:** mmo-dev Auth.7z скачан (юзер выложил на 192.168.0.248:3923) — L2AuthD.exe x64 classic 162-287 (610 816Б, 2023) + config.txt (**BfKey=6B60CB5B82CE90B1CC2B6C556C6C6C6C — тот же static-key! ProtocolVersion=50721=0xC621, serverExPort=2106**, accountCachedPort нет) + **etc/SQLQuery3.sql = полная схема lin2db (userno/user_time/usn + 13 procs ap_GPwd/ap_GStat/ap_GStatEtc/ap_GUserTime/ap_SUserData/ap_SNewPwd/ap_LoginWithPoint/ap_LogoutWithPoint/ap_SetGameRestriction/ap_SetConcurrentUserStatistics/web_CreateAccount/l2p_TempCreateAccount/hauthd_login)** → `authd-ref/lin2db-classic-x64-schema.sql` (в гит). Бинарь → artifacts/mmODEV-auth (локально).

Готчи R0: ssh дефолт-шелл = PowerShell (&& запрещён, одиночные команды; cmd-пайпы через `cmd /c "..."`); **System.Data.SqlClient в PS5 НЕ коннектится к SQL2022 (TLS)** — только `sqlcmd` (sa/123 работает); sqlcmd -o + -y 0/-Y 0 для длинных полей; OBJECT_DEFINITION вместо sp_helptext (курсор+INSERT в #tmp — один проход); tar-pipe 117МБ за ~40с.
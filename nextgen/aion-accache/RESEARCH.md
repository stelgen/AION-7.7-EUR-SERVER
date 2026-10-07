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
Файл: `../accountcache-ref/rpc-opcodes-58.txt`. Ключевые:

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

Полный список процедур: `../accountcache-ref/db-procs-58.txt` (UTF-16 → открыть в utf-16).
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

### 6.1 Повторный свип (10.10.2026, приоткрыт R1) — ВЕРДИКТ НЕ ИЗМЕНИЛСЯ: публично путь до нас НЕ пройден

- **GitHub repo-search**: `AccountCacheServer` = 0, `aion accountcache` = 0, `aion cache server` = 0 репо.
- **Sourcegraph global (исчерпывающий code-search)**: 2 совпадения во ВСЁМ публичном коде, оба = enum
  `STR_L2AUTH_S_ACCOUNTCACHESERVER_DOWN(62)` в Java loginserver beyond-aion — т.е. эмуляторы знают ACS
  только как код ошибки логина, реализации ноль. (Кросс-пульс authd: наш authd-ответник может
  переиспользовать код 62 для «ACS недоступен» — совпадает с L2-наследием.)
- **Ловушки-символы** (`ACQ_VERSION_PACKET`, `ACP_VERSION_RESPONE`, `PutCmd_ACP`,
  `ACQ_FIRST_LOAD_ACCOUNT_INFO`, `ServerToAccountCached`, `AccountCacheServer.pdb`) — 0 хитов веб/код.
  PDB/декомп ACS публично НЕ существуют — наша добыча (pdb-big 92МБ) уникальна.
- **mmo-dev 18390 «Aion 4.6 PTS»** (RU, слив Инновы): ACS в гайдах запуска = старт **№3**
  (1AuthServer→2AuthGateServer→**3AccountCacheServer**→4CaptCharServer→5ChannelChatting→6ICServer→
  7CacheServer→8NPCServer→9MainServer), ODBC-набор = aion_accoutdb/aiongm/aionlog/aionworld_110/L2Conn
  DSN-набор сходится с нашим §2. ⚠ РАСХОЖДЕНИЕ ИСТОЧНИКОВ по порядку старта: 4.6-гайд mmo-dev
  ставит ACS **№3 (после AuthServer/AuthGate)**, тогда как наш §2 (RZ-гайды) = ACS №1 ДО authd —
  при R1-стенде не критично (fork-proxy), при R5-свитче задачи стартовать как ориг-задача AionAcc.
  Бонус: AuthServer-бинпатч single-instance (`FindWindowA`-проверка, je→jmp @4412BD) — кросс-пульс authd. Реверса ACS — нет.
- **RZ 7.7 PTS EU (1211744) / mmo-dev 28786**: комьюнити считает 7.7-паки «too broken», уходят на 4.6
  или Java-эмуляторы → мотивации реверсить ACS у рынка нет (гонки за нами нет).
- grep.app — за Vercel-чекпоинтом, недоступен без браузера (покрыто Sourcegraph).
- **Контрольный свип (07.10, сессия после R1)**: GitHub repo-search API `AccountCacheServer` = **0 репо** (переподтверждено); `aion cached server` упёрся в rate-limit API, веб-пусто; точные символы (`AccountCacheServer.pdb`, `ServerToAccountCached`, `ACQ_*`) в вебе = 0 хитов; RZ 1267905 (aion 2.7 pts, слив ~07.2026) — «tons of stuff missing or tampered», ACS-реверса нет; кандидат только на кросс-версионный бин-дифф ACS 2.7 (качает юзер). **ВЕРДИКТ ДЕРЖИТСЯ: публично путь до нас НЕ пройден.**

## 7. Артефакты (в гит: ../accountcache-ref/)

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

## 10. DISPATCH-ТАБЛИЦА СНЯТА ДИЗАСМОМ (07.10, до R1 — payload'ы читаются сразу)

Артефакты: `accountcache-ref/dispatch-77.md` (аннотированная таблица) + `dispatch-77.txt` (raw). Метод: `AC_Socket::OnRead` (@VA 0x1400799F0) → `ACPacketHandler` ctor (@VA 0x140078B40) разворачивает 2 таблицы (handler-слоты 0x14449ab10+ / 0x14449ae38+, параллельный массив имён UTF-16 0x141100cf8+), cmd = (slot-base)/8.

**WIRE-ФРЕЙМ (известен уже!):** `[u16 len_total_minus2 LE][u16 cmd LE][u8 0xEB][u16 ~cmd LE][payload]`; лимит 0x2000; cmd ≥ 0x6C = reject; dispatch `handler = table[cmd]` ([sock+0x1d8]); ответы = `PutCmd_ACP` (VA 0x14007D800) тем же фреймом; читалка cmd = `GetCmd_ACQ` (VA 0x14007E560).

**Таблица 1 (основной ACQ-диспетчер, cmds 0..39):** 0=UNUSED_0(default), **1=VERSION_PACKET**, 2=TEST_INSERT, 3=TEST_SELECT, **4=FIRST_LOAD_ACCOUNT_INFO** (handler зовёт LoadHiddenFatigueInfoByAccountId@AccountDb — СХОДИТСЯ 1-в-1 с proc aion_GetAccountData_20170428 полями!), 5/6=LOAD/UPDATE_BM_PACK, 7/8=LOAD/UPDATE_TRIAL_ACCOUNT_DATA, 9=SYNC_PACKET_TEST, 10/11=SAVE/LOAD_CAHR_CUSTOM, 12/13=LOAD_CHAR_CUSTOM_BY_ITEM/SAVE_CHAR_CUSTOM_TO_ITEM, 14=LOG_INFO, 15/16/17/18/19/20/21=CHAR_CREATED/LOGIN/LOGOUT/DELETE/DELETE_COMPLETED/INFO_REFRESH/LEVEL_CHANGED, **22=НЕ ЗАНЯТ**, 23/24=UPDATE/DELETE_LOGIN_EVENT_RECORD, 25=UPDATE_HIDDEN_FATIGUE, 26/27=LOAD/UPDATE_LUNA, 28/29=CONFIRM_LUNA_REWARD/DECREASE_LUNA_KEY, 30=UPDATE_BOARD_BM_STATE, 31=ASK_CAN_MAKE_JUMPING_CHARACTER, 32/33/34/35=LOAD/UPDATE_PREVIOUS_PLAYTIME(S)_FOR_POLLS, 36=RESET_LUNA_REWARD, 37=TRANSFORM_OPERATION, 38/39=MONSTER_CORE_UPDATE_VALUE/UPGRADE; 0x28..0x64 = default-резерв.

**Таблица 2 (второй канал, cmds 0..7):** 1/2=GEN_TEST_PACKET2/3, 3/4=MoveCharResult/MoveCharByServicePacket, 5/6=Set/DeletePromotionCoolTimePacket, 7=пусто. Семантику второго сокета уточнить capture'ом (подозрение: отдельный клиент-канал/ответный поток).

Payload-детали: декодеры (`Decode*@ServerToAccountCached`) вызываются непрямо — точные байтовые раскладки = R1 capture; НО состав полей уже известен из mangled-сигнатур publics (DecodeCharLogin: H charId + SpecialSvrTypeEnum + wchar* userId; DecodeDecreaseLunaKey: __int64+LunaParam; DecodeUpdateHiddenFatigue: I,H,I,I,I; EncodeFirstLoadAccountInfo_AddArg: 5×int = 5 колонок GetAccountData...). Каркас MVP пишется БЕЗ capture; capture только для байтовых офсетов.

Инструменты: парсер ctor = /tmp/acs_dispatch2.py + /tmp/acs_final.py (перенести в tools/analysis); дизasm /tmp/acs.asm 283к строк; publics = artifacts/pdb-big/AccountCacheServer/pdb-publics-8035.txt. Готча objdump: адрес границы функции может не существовать как метка (шаг 4Б) — искать ближайший префикс; VA = ImageBase + 0x1000 + public-off (секция .text).## 11. R1+R2.5: WIRE СНЯТ ПОЛНОСТЬЮ (10.10, capture + дизasm OnRead/GetCmd_ACQ — финальный канон)

Стенд: FORK-SPEC §2 mirror-copy (accmirror v1.1 :2220→:2221 + копия ACS-2221, патч common.xml 1 байта
@idx293, ориг MD5 DE2EC2B2 цел). ⚠ ACS single-instance (FindWindowA+CreateMutexW в импортах) — копию
стартовать ТОЛЬКО соло. ✅ Server64 переподключается сам ~3-4с и **переигрывает VERSION-handshake**.

### ФИНАЛЬНЫЙ WIRE (доказано: дизasm + 0 bad frames на всём capture)

```
[u16 N][u16 cmd][u8 marker][u16 ~cmd][payload]
```
- **N = ПОЛНАЯ длина кадра, включая само len-поле** (тело = N-2). ⚠ В dispatch-77.md поле звалось
  «lenMinus2» — неверная трактовка, из-за неё accmirror v1.0 и каркас R2 имели офф-бай-2 (исправлено
  в v1.1/internal/proto; тесты = golden-кадры из живого capture).
- **marker: C2S = 0xEB, S2C = 0xEC** (ответы ACS инкрементят маркер; PutCmd_ACP).
- ~cmd = побитовая инверсия cmd (u16 LE).
- Лимиты OnRead: тело ≤ 0x2000; **cmd ≥ 0x6C → close**; marker ≠ EB → только LOG (не drop!);
  N < 2 → close. State-машина: state0 ждёт len-поле (+2 байта), state1 ждёт тело (N-2), после
  обработки offset += N-2... = ровно наш proto.Reader.
- **ВТОРОГО ФОРМАТА КАДРОВ НЕТ** — «CTRL-кадры» из первого лога = артефакт офф-бай-2 логгера v1.0.
- Дизasm-бонус: payload[0] (байт, sign-extended) индексирует глобальную таблицу каналов
  (stride 0xB8 @ 0x14EAC7AB0) → выбор структуры канала до постановки в очередь.

### Наблюдённые cmds и раскладки (fixtures в internal/payload/payload_test.go — ЗЕЛЁНЫЕ)

| cmd | C2S (Server64→ACS) | S2C (ACS→Server64) |
|---|---|---|
| 1 VERSION | `03000000 01000000` | `03000000` |
| 4 FIRST_LOAD | `f2030000` (ch) | ch+u32 1+u32 0+u8+**u32 ts**+u16 (19Б) |
| 5 LOAD_BM_PACK | ch | ch+u16 (6Б) |
| 7 LOAD_TRIAL | ch + u32 charRef (`ea030002`) | — |
| 8 UPDATE_TRIAL | ch+ts+u32 0+u32 0x395+хвост | — |
| 16 CHAR_LOGIN | u32 1+u32 0+charRef+ch+**UTF-16LE z-stamp** («2026-10-07T13:35:55.860»)+u32 1+u32 2+u32 0 | ch+`ffffffff ffffffff`+нули (31Б) |
| 17 CHAR_LOGOUT | u32 1+charRef+utf16z-stamp+u32 1+u32 2+u32 0 (**без ch-префикса!**) | — |
| 25 UPDATE_HIDDEN_FATIGUE | ch+3×u32 0+**ts1(0x6AC62093)+ts2(0x6AC5DFE0)**+u32 0 (24Б) | — |
| 26 LOAD_LUNA | ch | ch+u32 ts+u32 0 |
| 27/28 LUNA upd/confirm | — | ch+u32+u16 (хвосты) |
| 31 ASK_JUMPING_CHAR | u8 01+ch+u16 0001 (**префикс с u8, не u32!**) | — |
| 37 TRANSFORM_OP | u32 0xF2+ch+u8 00 | — |
| push 13/15/20/21 | — | ответные/асинхронные пуши (13=`ae310864` после BM; 20=большой blob с ts) |

- **`f2030000` = u32 0x3F2 = 1010** — channel/ticket-константа сессии (не «порт authd» — совпадение).
  Исключения: VERSION (03...), CHAR_LOGOUT (нет префикса), ASK_JUMPING_CHAR (u8-префикс).
- **S2C нумерация = ТА ЖЕ enum ACQ (маркер 0xEC)** — «не-совпадение номеров» из первого лога = артефакт
  офф-бай-2. 1↔1, 4↔4, 5↔5, 26↔26 подтверждаются парами.
- В CHAR_LOGIN/LOGOUT Server64 передаёт свою **локальную метку времени UTF-16LE z-строкой**.

### Артефакты и состояние

- `accountcache-ref/capture-20261007/`: accmirror.raw.log, c.hex, o.hex, **frames.txt** (полный
  поимённый список кадров обоих направлений); `aion-accache/tools/accparse.py` (v2, канон).
- Каркас: `internal/proto` (len=total, DirC2S/DirS2C, ParseFrame) + `internal/payload` (типизированные
  раскладки) — тесты на живых golden-кадрах, ЗЕЛЁНЫЕ.
- Стенд ОСТАВЛЕН работать (accmirror v1.1 деплой 14:09, реконнект+handshake ok). Откат =
  `taskkill /F /IM accmirror.exe` + kill копии по PID + `schtasks /run /tn AionAcc`; задачи
  AionAccCopy/AionAccMirror (SYSTEM, включены на время capture).
- **R3-хвосты**: роли полей FatigueReq/TrialReq (PDB DecodeUpdateHiddenFatigue I,H,I,I,I), семантика
  push 13/15/20/21, ACP-номера через vtable (для не-зеркальных), T2-канал (второй listener) — по
  дозахвату (стенд живой, логин ×N даёт новые cmds: CUSTOM 10-13, PLAYTIME 32-35 и пр.).

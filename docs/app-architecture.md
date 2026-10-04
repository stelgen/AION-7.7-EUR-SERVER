# 🏛 AION 7.7 PTS EU — ПОЛНАЯ АРХИТЕКТУРА СЕРВЕРА ПРИЛОЖЕНИЙ (реверс-база)

> Мастер-документ. Сюда сводится ВСЁ: бинари, базы, конфиги, порты, логика, хронология движений.
> Обновление: 04.10.2026. Источник фактов: инвентаризация D:\AION_LIVE_SERVER + packet-логи authd + дифф бинарей + RaGEZONE 1211744.

---

## 1. КАРТА СЕРВИСОВ (кто с кем говорит, порты, зависимость запуска)

| # | Сервис | EXE | Порт(ы) | Откуда слушает | К кому подключается сам | Зачем нужен |
|---|--------|-----|---------|----------------|------------------------|-------------|
| 1 | AccountCacheServer | AccountCacheServer.exe (20,587,520) | **2220** | common.xml | SQL (AionAccountCacheD_rc через DSN aion_accoutdb/aiongm) | Кэш аккаунтов для authd/CacheD |
| 2 | CAPTCHAImageServer | CAPTCHAImageServer.exe (155,648) | **22206** (int16-баг: 43330=-22206) | config.ini | — | Капчи при логине |
| 3 | ICServer (Interchange) | ICServer.exe (21,521,408) | **2005 + 2305** | common.xml | CacheD (2006), Server64 (master/delivery 7 сокетов) | Транзакционный хаб: обмены, биллинг-передачи |
| 4 | PAServer (PortalAuth) | 01-PAServer7.7.exe | **10057 (только 127.0.0.1)** | config.txt (conn в AionAccounts) | SQL AionAccounts | Авторегистрация/портал-авторизация (L2Authd UsePAServer=true, PAConnectionCount=2) |
| 5 | Прокси-логгер (наш) | py\python.exe aionproxy.py | **2106** → форвард 2107 | — | AuthGateD 2107 | Логгер трафика (MITM-пассив). После него authd видит клиентов как 127.0.0.1! |
| 6 | LogServer64 | LogServer64.exe (19,952,640) | **2051** | common.xml (country=2) | SQL Aion_log (BCP bulk insert) | Логи мира (logd). Требует ACP=28591 в реестре |
| 7 | L2Authd (AuthServer) | L2Authd.exe (1,198,592) | **2104** (serverPort), **2110** (serverExPort для AuthGateD), 2108 (GM interactive), 10062 (QMAS) | etc\config.txt | SQL AionAccounts (L2Conn.dsn), AccountCache 2220, PA 10057 | Авторизация клиентов, регистрация миров, OTP |
| 8 | AuthGateD | AuthGateD.exe (249,856) | **2107** (serverPort; клиенты!), форвард на authd 2110 | etc\config.txt (loginType=2, companyCode=2, sessionTimeout=5) | L2Authd 2110 | Клиентский шлюз авторизации; RSA-ключ сессии; IPList BlockIPs |
| 9 | CacheD64 | CacheD64.exe (22,526,464) | **2006 + 2007** | common.xml (aionworld_new.dsn) | SQL _AionWorldNew114_rc, LogServer 2051 | Кэш БД мира: предметы/юзеры, RPC для Server64 |
| 10 | NPCSvr64 | NPCSvr64.exe (24,991,744) | **2002** (NPC↔World) | common.xml | SQL (мир), Server64 2002 | Спавны, AI, движение NPC, дропы, ConditionSpawn |
| 11 | Server64 (MainServer/мир) | Server64.exe (45,499,392 = #180-кряк) | **7777** (клиенты), **2002** | common.xml (authServerAddr 2104), реестр connStr | AuthD 2104, CacheD 2006, LogServer, ICServer | ВЕСЬ мир: игроки, квесты, бои, бандлы, guild, abyss |
| 12 | NPRelay64 | NPRelay64.exe (21,826,560) | — | common.xml (countryCode=2) | Server64 (NP/Warehouse relay) | NCoin/склад релей — опционален |
| 13 | RankingServer (.NET) | RankingServer.exe (157,696) + SuperSocket + EF | свой | app-конфиг | SQL | Абисс-рейтинги — НЕ запускался никогда |
| 14 | GMServer / GMShopServer / WebGM (IIS _AION_WEB) | GMServer.exe (442,880, .NET) | свой | exe.config + DBConfigEncrypt | SQL aion_log (схема **aiongm_ur**: TBL_GAME_SERVER_INFO, TBL_GAME_WORLD_INFO) | GM-панель/веб — есть в 4.6-ките |
| 15 | PetitionServer | PetitionD_Integration.exe (248,832, .NET) | **2107** ⚠️ конфликтует с гейтом! | AppConfigForServiceCode.xml + exe.config (DatabaseConnString шифруется Codec.exe) | SQL PetitionDB | Саппорт-тикеты 1:1 |
| 16 | ShopAgentServer | ShopAgentServer.exe (157,184) | **10100** + IDW 10130 + PS 10110 + Common 10115 | ShopAgentServer.xml (IDW DBCONNECTIONSTRING → **БД SADB**, нет у нас) | SQL SADB | Внутриигровой магазин/касса |
| 17 | ChannelChattingServer | ChannelChattingServer.exe (1,382,912) | **10254** (A2G), Ingame 10241, Gate 20241, Watchdog 10255, Mgmt 10261 | ChannelChattingServer.xml + ChatServerList_Server.xml | authd? LogServer? | Кастомные чат-каналы |

### Порядок старта (жёсткая зависимость)
```
SQL Server (сервис винды)
→ AccountCacheServer → CAPTCHAImageServer → ICServer → PA → Proxy(2106) → LogServer64
→ L2Authd → AuthGateD
→ CacheD64
→ NPCSvr64 (10-15 МИНУТ загрузка 14.9 ГБ)
→ Server64 (RunAsDate 04/06/2020 16:28:23 !) → мир собран, когда 16 NPC↔World ESTABLISHED на 2002
```
⚠️ Server64 и NPCSvr — **пара**: смерть одного ⇒ graceful-смерть второго. Рестартить всегда вместе.

---

## 2. БИНАРНАЯ ИСТОРИЯ MainServer (дифф сток vs #180)

| Файл | MD5 | Что это |
|---|---|---|
| MainServer64.exe | `caf1e340ed291c93f17d5d562eaca370` | **СТОКОВЫЙ оригинал** ( identical Server64.exe.orig, штамп 11.05.2022) |
| Server64.exe | `c515730263c2330c003b5d2d7ae1aca8` (SHA256 b000c6f5…) | **#180-кряк** (Angry Catster, аттач RaGEZONE 31.10.2024) |
| Server64.exe.orig | `caf1e340…` | бекап оригинала рядом с Server64.exe |
| Server64.exe.180-bak, prod-backups\Server64.exe-20261004.bak | `c5157302…` | наши бекапы кряка |

**Дифф сток↔кряк = ВСЕГО 4 байта в 2 регионах:**
```
file 0x00E7CA9B len=3:  B4 00 00 → FF FF FF   (внутри функции ?RGB2HSL@@YAXHHHAEAM00@Z, RVA+0x15B)
file 0x00E90F1E len=1:  00 → 02             (внутри ?EncodeAlive@LogClientToLogServer@@YAHPEAD_J@Z, RVA+0xE)
VA: 0x140E7D69B и 0x140E91B1E (ImageBase 0x140000000)
```
**ВЫВОД: #180-кряк НЕ снимает «authorization time» в PlayerConnect** — он только date-check старта. Поэтому мир на реальной дате (2026) отклоняет вход; **RunAsDate (2020-06-04) обязателен** для входов (доказано логинами вчера). FliesQQ «Crack authorization verification time» — именно это чинит; его бинарь несовместим с нашей парой (умирает мгновенно) — путь: Ghidra-патч нашего #180 (PDB есть) или запуск FliesQQ с его же окружением на тест-клоне.

**ScriptDLL64.dll (87,731,200)** — ОДНА И ТА ЖЕ (MD5 e7016eae…) для MainServer и NPCServer: скриптовый слой игровой логики (квесты/спавны/AI) — потенциально «вшито дохуя полезного», реверс-кандидат №2.

**ncguardserver64.dll (1,892,112)** — NC Guard (защита) — грузится Server64.

---

## 3. БАЗЫ ДАННЫХ (SQL Server 2022, D:\SQL, max mem 2048)

| БД | Размер | Кем используется | Ключевое |
|---|---|---|---|
| AionAccounts | 80 МБ | L2Authd, PA, Server64 (linked RC-AIONAUTHDB) | `server` (id=1 _MAIN, ip=81.25.59.194, **inner_ip=127.0.0.1 — НЕ ТРОГАТЬ! это IP мира для auth-связи**), user_account (uid/login/…), user_auth (MD5 binary16), ssn, userno, worldstatus, user_count, block_* |
| AionAccountCacheD_rc | 16 МБ | AccountCacheServer | max text repl size=65664 критичен |
| _AionWorldNew114_rc | 13,800 МБ | Server64 (прямые connStr), CacheD64 (aionworld_new.dsn), ScriptDLL | user_data (user_id NVARCHAR40 = имя аккаунта!, account_id INT = uid), user_item, guild, abyss, house_*, item_*; 1056 процедур |
| Aion_log | 13 МБ | LogServer64 (BCP), GMServer/WebGM (схема **aiongm_ur**: TBL_GAME_SERVER_INFO, TBL_GAME_WORLD_INFO) | Log_TblGameServerInfo_UpdateServerstatus/UpdateLogfreedisk, Log_TblGameWorldInfo_UpdateMainStatus (оригиналы восстановлены) |
| PetitionDB / BkPetitionDB | 2 ГБ | PetitionServer | тикеты |
| aionaccdeldb | 16 МБ | authd (aion_ProcessWithdrawAccount) | del_account |

### Процедуры-«дырки» (нет в БД, Server64/CacheD их зовут)
```
aion_GetItemCollection{List,LevelList,ExpiredList,CompleteTimeLimitList,CompleteList}(serverId,uid,0)
aion_LoadFameInfo(1002) / aion_LoadFameReduceTime / aion_UpdateFameInfo(1000,1,0,0)
aion_LoadReinventInfo(1002)
aion_DeleteItemByDate_20191206(4,10)
aion_getItemAttributeDeltaListAll_20190919(1002) + VendorDark/VendorLight(0)
Log_TblGameServerInfo_UpdateLogfreedisk(56,1)  ← ОРИГИНАЛ ВОССТАНОВЛЕН (aiongm_ur.TBL_GAME_SERVER_INFO)
```
Сигнатуры собираются из CacheServer\log\*.err `{call …}` строк.

---

## 4. КОНФИГИ (поля, которые имеют значение)

| Файл | Поля |
|---|---|
| MainServer\common.xml | authServerAddr/Port(2104), maxUser=500, maxUserLevel 80, DefaultEnableCheckIpFromAuth=False, expMultiple/AbyssPointMultiple (закомментированы!), petitionServerAddr/Port (**2107 = конфликт с гейтом!**), shopAgentServer 10100, mxChatS2S 10254 + mxChatKey/mxChatSignature (=ChannelChat-конфиг) |
| AuthD\etc\config.txt | serverPort=2104, serverExPort=2110, serverIntPort=2108, worldport=7777, UseLogD=false, UseIPServer=false, UsePAServer=true PAIP_1/2=127.0.0.1:10057 |
| AuthGateD\etc\config.txt | serverPort=2107 (перенесён с 2106 из-за прокси!), authPort=2110, loginType=2, sessionTimeout=5, tryCount=20, dumpPacket=true |
| LogServer\common.xml | country=2, serverPort=2051, BCPInterval=10000, UseBulkInsert=1 |
| CAPTCHAImageServer\config.ini | captchaServerPort=22206 (был int16-баг 43330=-22206) |
| ICServer\common.xml | 2005/2305 |
| Шлюзовой реестр | HKLM\SOFTWARE\NC Soft\AION\<Comp> — connStr blob (XOR 512B), сам пересоздаётся после первого логина; ODBC DSN в C:\DSN\ |
| MainServer\runasdate-x64 | DateTime=04-06-2020 16:28:23 (= билд-дата Server64) |

---

## 5. ХРОНОЛОГИЯ ДВИЖЕНИЙ (по chat-логам, укороченная)

| Дата | Событие |
|---|---|
| 02.10 | Установка: SQL 2022, БД, ODBC/DSN, дроп логов, ONSTART-задачи, первый полный старт |
| 03.10 утро | Логин-квест: Session id mismatched → **победа `-loginex -pwd16`** (08:36, uid 1010) |
| 03.10 10:20 | Рестарт Server64 (RunAsDate, юзером) → **10:35:44 ВХОД В МИР** (игра 10:35–12:01) |
| 03.10 12:05–12:16 | Дроп-деплой (XML дропов) + рестарт ПАРЫ → **после этого успешных входов не было** |
| 03.10 вечер-ночь | inner_ip-правка (фейл, откат 04.10), ребуты VM ×2, краши authd из задач, лаунчеры v2/v3 |
| 04.10 | Восстановление: RunAsDate-CLI Server64, пара пересобрана, дифф сток↔#180 (4 байта), все логи в память |

---

## 6. ОТКРЫТЫЕ ВОПРОСЫ (реверс-очередь)

1. **«Не авторизован» при входе**: код 02/03/04/05 в World→Auth — семантика не расшифрована. Крипта: welcome скремблирован, сессионный Blowfish-ключ генерирует authd (у authd/Server64 PDB ЕСТЬ — реверс через Server64-side проще, чем клиентский).
2. ScriptDLL64.dll 87.7 МБ — скриптовый слой (квесты/спавны) — реверс-кандидат.
3. RankingServer (.NET) — никогда не запускался.
4. Ночные рестарты NPCSvr (утечка Abyss) — roadmap Э3.
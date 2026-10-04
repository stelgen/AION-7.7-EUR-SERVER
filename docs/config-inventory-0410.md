# Полная инвентаризация конфигов диска D: VM AION-77-PTS-EU (04.10.2026)

READ-ONLY обход всего диска D. Артефакты: полный дамп 203 конфиг-файлов → `C:\Temp\cfgdump.txt` на VM + локальная копия `STELGEN/tmp/aion-vm/cfgdump_clean.txt`; инвентарь 675 файлов (без log/Map/unsent) → `C:\Temp\inv_main.csv` / локально `STELGEN/tmp/aion-vm/inv_main.csv`.

## Корень D:
- `D:\AION_LIVE_SERVER\` — прод-сервер (12+ компонент, см. app-architecture.md)
- `D:\AIONSVC\` — деплой ChannelChat/Petition/ShopAgent (эксперименты 03.10)
- `D:\SQL\` — файлы БД: прод (AionAccounts, AionAccountCacheD_rc, _AionWorldNew114_rc 20 ГБ, Aion_log, BkPetitionDB+PetitionDB по 1 ГБ, aionaccdeldb) + REF58_* (4 read-only) + **REF58L_* (НОВОЕ: AionAccountDB 6.7 ГБ mdf / 4.3 ГБ ldf, AionAccountCacheDB, Aion_gm 320 МБ — тяжёлые референс-БД 5.8 с данными, resto уже сделан)**
- `D:\_AION_log\` — batchlog + main-логи (logDirectory из common.xml)
- `D:\_REF58\` — ref-киты (46/, out/ 5.8, fliesqq/, procs/, prod-backups/)
- `D:\ISO\` — SQL ISO + `SQL2017.ini`/`SQL2022.ini`: SAPWD=Wutian520 (заводской), collation Latin1_General_CI_AS, INSTALLSQLDATADIR=D:\SQL, NETWORK SERVICE, SECURITYMODE=SQL
- `D:\Temp\` — исходный кит AION7.7SERVER(eu).rar 4.2 ГБ + AionAccounts.bak

## MainServer (Server64)
- `config.xml`: serverTitle **Gardarika Main Server**, serverId=1, serverName=Gardarika, **clientAcceptAddr=192.168.0.125:7777** (LAN bind), v45_update_date=2017-11-20
- `common.xml` (9647 B, country=7): maxUser=500, waitUserMax=250, cacheServer 127.0.0.1:2006, LogServer 2051, petitionServer**Port=2107 (конфликт с гейтом, лупер)**, captcha 22206, authServer 2104 (useOldAuthServer=FALSE), accountCached 2220, mxserveraddr_1=192.168.200.131:10241 (внешний чат из кита), mxChatS2S 127.0.0.1:10254, mxChatKey=0123456789ABCDEF0123456789ABCDEF, mxChatSignature=AION-WING-SKY, shopAgent 10100, GIP 30271 (useGIP=false), createCharNum=8, maxUserLevel 80, MaxAllowedClientCount=30, aboutToPlayTimer=30000, KickBotUser, expMultiple/AbyssPointMultiple/CraftExpMultiple=500 ЗАКОММЕНТИРОВАНЫ, mailReceiver=andrey.klykov@inn.ru (мэйл исходного владельца кита!), spawn_version=040014200
- `allow.cfg / deny.cfg / denycountry.cfg / internal.cfg / unlimited.cfg` — **ВСЕ ПУСТЫЕ (2 байта)** — access-контроль отключён, «неизученные cfg» теперь закрыты
- `runasdate-x64\RunAsDate.cfg`: DateTime=04-06-2020 16:28:23 → Server64.exe (legacy, не используется)
- `AIONErr.txt`: краш-шаблон «AION MainServer64 **77.20.0604.15625** (2020-06-04 16:28:23)» — версия бинарника

## AuthD (L2Authd)
- `etc\config.txt` (и `config2.txt` = байт-в-байт копия, mtime 03.10 09:40): serverPort=2104, serverExPort=2110 (гейт), serverIntPort=2108 (GM), worldport=7777, serverMiPort=10062, DBConnectionNum=64, **UsePAServer=true PAIP_1/2=127.0.0.1:10057**, UsePacketLog=32767 (ALL packet-логи — источник packet-золота), ProtocolVersion=50721, GameID=8, vmGameId=99, PacketSizeType=3, newEncrypt=1, CountryCode=2, UseForbiddenServerSetting=true, UseDormantFlag=true, UseBanAccount=true, checkTasStatusFromDb=true, AppId/ClientAppId/AuthdAppId GUID'ы, reportMail smtp.game.ncsoft, WSM 192.168.200.131:10051 (off), GPProxy 10.71.12.x (off), OTP RetryCount=3
- `fcmsdk.ini`: китайская FCM-антифатиг система, NeedFCM=1, IPAddr=61.172.247.235:7329 (SNDA China) — нерелевантно, артефакт
- `lin2db.sql` — полная схема AionAccounts (см. auth-server-internals.md)

## AuthGateD
- `etc\config.txt` (50902 B, комменты в корейском mojibake): serverPort=2107, authAddr=127.0.0.1, authPort=2110, numThread=32, numIOThread=96, acceptCallNum=200, socketLimit=2000, sessionTimeout=5, useForbiddenIPList=true, checkGameGuard=false, useGameGuard=false, **loginType=2**, **companyCode=2 (Innova)**, tryInterval=60/tryCount=20/tryBlockInterval=120 (brute-блок), dumpPacket=true, ggNumActive=50, useGCSideExtendAccount=true
- `csauth2.cfg` = «60» (одиночное значение)
- `etc\AllowIPs.txt` — пуст; `etc\BlockIPs.txt` = 220.231.14.1-255 (корейский диапазон, legacy)

## AccountCacheServer
- `common.xml` (621 B): serverTitle **AION ACS**, mailServer=192.168.0.125, **serverPort=2220**, country=7, numberOfDBThreads=10, autoShutdown, UseFullDump
- `config.xml`: serverBasis/serverId=1

## CacheServer (CacheD64)
- `common.xml` (1430 B, == оригинал 2023): country=7, numberOfDBThreads=10, serverPort=2006, **interactivePort=2007**, LogServer 2051, logDirectory=D:\AION_LIVE_SERVER\Log\Cache, mapDirectory=D:\AION_LIVE_SERVER\Map, alert=false, serverTimeOut=0, **deleteitembydate=true + deleteItemDurationDays=4** (вот откуда aion_DeleteItemByDate_20191206(4,10)!), ItemCache=FALSE, ICServer 2305 id 1, CanRecover=false
- `config.xml`: serverId=1
- `DBLogDetail` / `DBLogSummary`: HTML-отчёты эры SQL 2008 R2 2017-11-27, «Cache Version: AION CacheServer64 4514.0319.0820.8451 (2014-08-20)»
- `AIONErr.txt`: **живой кейс 04.10 05:41:45 «Deadlock Detected by CheckIOThreadDeadlock(), ThreadID: 7(IOThread), passedTick(191625)»** — CacheD лагал/дедлок IO-треда во время нагрузки

## LogServer
- `common.xml`: country=2, serverAddress 127.0.0.1:2051, logfilefolder=D:\_AION_log, **BCPInterval=10000, BCPThreadNum=4, UseBulkInsert=1, TimeSlice=10**
- `config.xml`: «NovicePTS Cache Server», serverName=Test, country=2 (китовый мусор-титул)

## ICServer
- `common.xml`: mainport=2005, cacheport=2305, numBeginnerServer=1, numIdentifiedServer=2, numGAb1Server=2, DistributeGAb1WorldRestrict=true, gabyssGroupIdType=3
- `config.xml`: mapDir=D:\AION_LIVE_SERVER\Map, country=7, PreferBeginnerSvr_1=1

## CAPTCHAImageServer
- `config.ini`: SrcPort=22206, SessionCount=1024, KeepAliveTimeout=0, LanguageCode=en (ko/en/th xml), Console+File лог, FileNamePeriod=60
- `default.xml`: captcha-рендер (dds/dxt1 128x32, шрифты, кривые)

## NPCServer (NPCSvr64)
- `common.xml`: registry=HKLM, mailServer, serveraddr 127.0.0.1:2002, cacheserver 2006, logserver 2051, **autoloot=false**, removedelay=15000, DelayQueueMask=1023, logDirectory=D:\AION_LIVE_SERVER\Log\NPC\, MaxNpcLevel=65, **country=2** (!), cashMultiple/dropMultiple=500 ЗАКОММЕНТИРОВАНЫ, spawn_version=040014200
- `config.xml`: serverId=1, logserver 2051

## NPRelayServer
- `common.xml`+`config.xml`: countryCode=2, dataCenter=1 (конфиг пуст — сервис нерабочий, известно)

## RankingServer (.NET, никогда не запускался)
- `RankingServer.exe.config`: .NET 4.5 + EF6, connection string → **(LocalDb)\MSSQLLocalDB / RankingServer.Model1** (дефолт-скэффолд, НЕ настроен на наш SQL — причина молчаливого exit)

## NcGuard
- `ncgsvrcfg.ini`: gameid=4, countryid=410, patterns, скан-расписания (MALWARE/PD 01:00/01:30), moduleflag: cryptopacket=1, randomprotocol=1, patchdetect=1, memscan/rsrcscan/kobjscan=1, **logpath=d:\AionServer64\MainServer\log\ (битый путь из кита!)**, uploadfileopt=16

## AIONSVC (деплой 03.10)
- `ChannelChattingServer\ChannelChattingServer.xml`: KEY=0123456789ABCDEF... (совпадает с mxChatKey!), SIGNATURE=AION-WING-SKY, ALLOWED_DIFFTIME=3000, NP_AUTHENTICATION APPID=F1A97986-.../APPSECRET=JNUktl4PnaYFlZFKIZtXxw==; `ChatServerList_Server.xml`: Game Id=27, CHAT_SERVER 127.0.0.1 IngamePort=10241 GatePort=20241; exe молча падает до логгера (см. loops-fix)
- `PetitionServer\AppConfigForServiceCode.xml`: Service Code=27 Aion, AuthType=External, AuthIp=127.0.0.1, **AuthPort=2108**, AuthCompanyCode=4; `PetitionD_Integration.exe.config`: GmServicePort=2109, **WorldServicePort=21055** (наша правка), Notice=2121, Monitor=2122, DatabaseConnString=base64-ШИФР (нужен Codec.exe GUI), AdminGate.Host=172.16.200.119:6601, log4net → D:\AiON_SERVER\_aion_log\Petition\PetitionD.log (битый путь кита); `DB\codec.txt`: **открытым текстом SERVER=172.16.200.119; DATABASE=BkPetitionDB/PetitionDB; UID=sa; PWD=U6SjJk3ZyQhrv5tq** (пароль исходного кита!)
- `ShopAgentServer\ShopAgentServer.xml`: IDW.dll/CommonShopAgent.dll/GameInterface.dll, IDW SERVER 10130 с DBCONNECTIONSTRING **SERVER=(local);UID=sa;PWD=123;DATABASE=SADB** (уже адаптирован под наш sa/123), COMMON_SHOP_AGENT 10100, GAME 27; требует несуществующую БД SADB

## tool\ (китовые утилиты)
- `gmdb\gm.txt`: `Server=.;Database=aion_log;User ID=aiongm_ur;Password=U6SjJk3ZyQhrv5tq` + `gm.cmd` = `gm gm.txt gm.bin` — **шифратор пароля aion_log/aiongm_ur → gm.bin (96 B)**, есть `sa=gm.bin` (80 B)
- `worlds\worlds.cmd` = `worlds gm.bin pub.xml` + `pub.xml` = **RSA-публичный ключ (XML Modulus/AQAB)** → генерит `1.DB_CASH_SERVER.bin / 1.DB_WORLD_REP_SERVER.bin / 1.DB_WORLD_SERVER.bin / 1.GM_SERVER.bin / 1.SOCKET_CACHE_SERVER.bin / 1.SOCKET_MAIN_SERVER.bin` (по 128 B) — типы серверов для aion_log-авторизации сервисов!
- `sql-server.txt`: `exec sp_change_users_login 'AUTO_FIX','aiongm_ur'` (фикс orphan-юзера) + ключи SQL Server (Enterprise 6GPYM-VHN83-PHDM2-Q9T2R-KBV83 / TDKQD-PKV44-PJT4N-TCJG2-3YJ6B, Developer 22222-..., Standard PHDV4-..., Web WV79P-...)
- `emed64注册码.txt` = WinRAR-лицензия DKAZQ-R9TYP-5SM2A-9Z8KD-3E2RK
- `cn_sql_server_2017...` — CN SQL2017 дев-дистрибутив

## Корневые батники кита (D:\AION_LIVE_SERVER)
- `关闭游戏.bat` (Stop game): полный kill-лист — **ShootMail.exe, ServiceMonitor.exe** (новые имена компонентов!), PetitionD_Integration, NPCSvr64, Server64, GMServer/GMServerSTB, CacheD64, AuthGateD, L2AuthD, ICServer, ChannelChattingServer, CAPTCHAImageServer, AccountCacheServer, LogServer64
- `单机登陆器1.bat` (одиночный лаунчер): `start bin64\aion.bin -ip:192.168.200.131 -port:2106 -cc:5 -noauthgg -charnamemenu -loginex -pwd16 -megaphone -ingamebrowser -ncping` — родной CN-клиент кита, оригинальная LAN IP кита 192.168.200.131, cc:5
- 9 .lnk лаунчеров (01-PAServer…09-NPCSvr)

## DSN (по 7 шт в каждой компоненте + Database\ODBC)
- `aionworld_new.dsn`: DRIVER=SQL Server (не SNAC!), UID=syncconn затёрт UID=sa, PWD=123, SERVER=., DATABASE=_AionWorldNew114_rc
- `L2Conn.dsn`: DRIVER=SQL Server Native Client 11.0, DATABASE=AionAccounts, **Trusted_Connection=Yes**, SERVER=.
- `aion_accoutdb.dsn` (алиас-опечатка), BkPetitionDB/PetitionDB/aiongm.dsn
- Бэкапы от наших правок: *.bak20261002 (6), *.bak-country7 (5), *.orig2 (3 — Database\ODBC оригиналы)

## Map (структура)
DynCodeBin 2, event 7, event_beginner_event 3, GameGuardCS 2, NcGuard 6, **Worlds 870 файлов, XML 16 517 файлов** (npcs_test.xml / CommonDropItems.xml — см. fixes-registry секц. H)

## Итог-статус чтения
Полностью прочитаны ВСЕ текстовые конфиги/данные <1 МБ вне log/Map/unsent/batch. Остались непрочитанными только: логи (отдельный анализ), бинарники, Map-XML геймдаты, REF-киты (референс), SQL-бинарники БД, RTF-лицензии SQL.

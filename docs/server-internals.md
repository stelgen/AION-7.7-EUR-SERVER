# Как работает сервер AION 7.7 PTS — внутреннее описание

Документ собран из реального расследования: логов, дампов, конфигов, живых тестов на VM 109.
Здесь только то, что проверено на практике. Где не копали — прямо так и написано.

## 1. Общая схема: кто к кому обращается

```
[Клиент]
   │  TCP 2106
   ▼
AuthGateD ──────► L2Authd (2110) ──────► PAServer (10057)
   (гейт логина)      (авторизация)          (PortalAuth, 2 моста)
      │                    │
      │                    ▼
      │              AccountCacheServer (2220) — кэш аккаунтов
      ▼
Server64 (7777 — мир, 2002 — внутренний)
   │        │          │            │
   │        │          │            └──► NPCSvr64 (входящее: NPCSvr сам подключается к 2002)
   │        │          └──► Captcha (22206), Petition (2107 — нет exe), ShopAgent (10100 — нет exe),
   │        │                ChannelChat (10254 — нет exe), ICServer (2005)
   │        └──► CacheD64 (2006) ──► ICServer (2305), LogServer (2051), SQL
   └──► SQL (мир-БД), LogServer (2051)

NPCSvr64 ──► LogServer (2051, alive-пакеты)
CacheD64 ──► LogServer (2051), SQL, ICServer (2305)
LogServer64 ──► SQL (Aion_log) + файлы D:\_AION_log\batchlog
```

Топология протоколов повторяет линейное наследие NCsoft (AuthGateD/Gate → AuthD → CacheD),
поэтому знакомые с L2-серверами быстро сориентируются.

## 2. Цепочка логина — раскопано полностью

Что происходит, когда игрок вводит логин/пароль (всё это видно в дампах и логах):

1. Клиент открывает TCP на `2106` (AuthGateD).
2. AuthGateD создаёт сессию со счётчиком `m_iSessionId` (первая сессия после старта имеет id 1).
3. Сервер шлёт приветственный пакет ~248 байт (в нём RSA-ключ и session id; RSA-пара генерируется при старте).
4. Клиент отвечает ~88 байт, сервер отвечает ~96 байт.
5. Клиент отправляет логин-пакет ~240 байт. В нём поле sessionId.
6. Сервер сверяет sessionId со своим. Несовпадение → `[WARN] Session id mismatched.(RecvLogin) sessionId:0, m_iSessionId:1` → сервер шлёт RST, клиент видит «вы были отключены».
7. При `loginType=1` сервер до сравнения пишет `[WARN] Disallowed gamesession login (current logintype: 1)` — то есть клиент шлёт именно gamesession-логин (тип 2).

Наблюдение: западный клиент (EU/US сборка `7720.0603.x`) шлёт sessionId=0 всегда — у него
сессию выдаёт портал (CEF-форма), а не AuthGateD. Поэтому с ним логин невозможен без
портал-эмулятора. Клиенты линейки CN/KR PTS (`7720.0601.x`) логинятся классически.

8. Успешный логин: AuthGateD пересылает данные в L2Authd (порт 2110). Счётчики попыток:
   `WebLoginTry : N, ClientLoginTry : N` (видно в `AuthD\etc\log\*.winlog`).
9. L2Authd проверяет/создаёт аккаунт в БД `AionAccounts` (таблицы `user_account`, `user_auth`).
   Авто-регистрация включена: нового игрока создаёт сама при первом входе.
   Схема `user_account`: `uid, account, pay_stat, login_flag, warn_flag, block_flag, block_flag2,
   last_login, last_logout, subscription_flag, last_game, last_world, last_ip, block_end_date,
   forbidden_servers`.
10. Относительно портала: `AuthD\etc\config.txt` содержит `UsePAServer=true`, `PAConnectionCount=2`,
    `PAIP_1=127.0.0.1:10057`, `PAIP_2=…:10057`. L2Authd держит два соединения к PAServer
    (в конфиге кита второй адрес был `127.0.0.2` — исправлен на `127.0.0.1`).
11. Далее L2Authd отдаёт клиенту список серверов: `ServerListPacket(6) has 1 pages` — список
    берётся из таблицы `Server` БД (`id, name, ip, inner_ip, port, region`). У нас: `_MAIN,
    192.168.0.125, порт 7777, region 2`.
12. Клиент подключается к миру: `ip:7777` (порт мира задаётся в `AuthD` строкой `worldport=7777`).
    Отсюда правило: если сервер смотрит наружу за NAT — IP в таблице `Server` должен быть
    внешним (или DDNS), иначе клиент из интернета не найдёт мир.

## 3. Мир: как Server64 и NPCSvr64 работают в паре

- Server64 стартует и ждёт NPCSvr: в `MainServer\log\*.err` сыпятся
  `CondSpwnTimeMgr::TimerExpired, NPCSvr hasn't connected. Waiting 10 second...` и
  `DynamicFieldMgr::TimerExpired() wait NPC Server`. Мир (спавны, поля) не включается,
  пока NPCSvr не подключится.
- NPCSvr64 грузится 10–15 минут, съедает до ~15 ГБ private RAM (фазы загрузки видны по
  `.memory_summary` — топ-пулы: ConditionGroupMaker, NPCMaker, WayPoint, LogBuffer).
- Загрузившись, NPCSvr сам подключается к Server64 на порт 2002 — рабочих каналов ~17
  (ESTABLISHED по netstat).
- Дальше идёт симуляция: `World::MoveNPC(... times called) :: World(DF3, 220040000) ... (User:0, Npc:41)`,
  `SetGoingToDie` для тренировочных манекенов и т.п. — мир живёт даже без игроков.
- Абисс-рейтинг обновляется циклом раз в ~60 секунд: Server64 `AbyssRankMgr::ReceiveAbyssRankUserInfo
  race:0/1 ... _SwapRankUserInfo()`, параллельно CacheD выполняет
  `sqlOrderingAbyssRanking(worldId, race, time, 1000, ...)` — это нормальный фон, но он
  пишет строки в `CacheServer\log\*.err` и раздувает файл.
- Разрывы/падения Server64 писали крэш-отчёты в `MainServer\AIONErr.txt` (стек с
  `SNDAMinorFatigueClient::~...`, `doexit` — были эпизоды убийства по нехватке commit-памяти).

## 4. Конфиги: что раскопано по файлам и параметрам

### 4.1 `MainServer\common.xml` (Server64) — главная карта портов
```
cacheServerPort      = 2006    CacheD64
logserverport        = 2051    LogServer64
petitionServerAddr/Port = 127.0.0.1:2107   (petition-сервера в ките нет)
disablePetitionFrom/To  = 0 / 0            (семантика «часы отключения петиции» — не проверена)
useCaptcha           = true
captchaServerAddr/Port  = 127.0.0.1:22206  (в ките было 43330 — баг int16 у Server64)
authServerPort       = 2104    L2Authd
accountCachedPort    = 2220    AccountCacheServer
mxserverPort_1       = 10241   внешний ChatServer (в ките IP 192.168.200.131 — мёртвый)
mxChatS2SPort        = 10254   канал сервер-сервер чата (нет exe)
mxChatKey / mxChatSignature             ключ/подпись чата
shopAgentServerPort  = 10100   магазин-агент (нет exe)
GIPServerPort        = 30271   (назначение не копали)
ICServerPort         = 2005    ICServer (Interchange)
```

### 4.2 `AuthGateD\etc\config.txt`
```
serverPort = 2106          порт для клиентов
authAddr/authPort = 127.0.0.1:2110     куда идти за авторизацией (L2Authd serverExPort)
numThread=32, numIOThread=96, acceptCallNum=200, socketLimit=2000
sessionTimeout = 5         минуты простоя сессии
useForbiddenIPList = true  читает etc\BlockIPs.txt (AllowIPs.txt — пустой, белого списка нет)
dumpPacket = true          дамп пакетов (включили при расследовании; в лог попадает WARN-строка)
tryInterval/tryCount/tryBlockInterval = 60/20/120   брут-блок: 20 попыток за 60 с → бан IP 120 с
checkGameGuard = false, useGameGuard = false          GG полностью выключен
useNotifyCSResult = true
loginType = 2              0/1/2 — режимы протокола логина; EU-флоу = 2 (gamesession)
companyCode = "2"          2=Innova, 3=Snda Channeling, 4=Snda
appLaunchBanDelay = 5
useGCSideExtendAccount = true
useReportMail = false      (reportMail* — почтовые алерты, выключены)
ggNumActive=50, ggUseUpdateTimer=false, ggUseLog=2, ggLogInvalid=true
logdport = 3999            LogD-приёмник, useLogd=0 (выключен)
```

### 4.3 `AuthD\etc\config.txt` (L2Authd)
```
serverPort   = 2104        слушает игровые серверы
serverExPort = 2110        слушает AuthGateD
serverIntPort = 2108       GM/interactive (в т.ч. управление GMCheckMode через "4\t0\n")
serverMiPort = 10062       QMAS/monitor
worldport    = 7777        порт мира, который L2Authd сообщает клиенту
DBConnectionNum=64, numServerThread=64, SocketTimeOut=180
UseIPServer=false (IPServer=127.0.0.1, IPPort=3113, IPInterval=60)
UsePAServer=true, PAConnectionCount=2, PAIP_1=127.0.0.1:10057, PAIP_2=127.0.0.1:10057 (фикс: было 127.0.0.2)
PAReconnectInterval=60
UsePBServer=false (PBServerIP=127.0.0.1, PBServerPort=3000)
GMCheckMode=false           true = пускает только GM (переключается интерактивным портом)
convertPortalEmailAccount=false (L2-наследие)
limitIPNumforEachGS=0       лимит IP на игровой сервер (0 = нет)
```

### 4.4 `ICServer\common.xml`
```
mainport  = 2005   для Server64
cacheport = 2305   для CacheD64
```

### 4.5 `CAPTCHAImageServer\config.ini`
```
[Network] ConcurrentThreadCount=4, WorkerThreadCount=8
[CAPTCHAImageServer] SrcIp=0.0.0.0, SrcPort=22206 (фикс; было 43330), SessionCount=1024, KeepAliveTimeout=0
[CAPTCHAImageManager] LanguageCode=en, ko=ko.xml (файлы en.xml/ko.xml/th.xml — словари CAPTCHA)
```

### 4.6 Прочее
- `AccountCacheServer\config.xml|common.xml` — есть, детально не разбирали (работает штатно).
- Конфигов NPCServer/LogServer64 как xml не видно — их поведение управляется переменными
  окружения/кодовой страницей (LogServer: ACP=28591, см. Фиксы).
- `MainServer\allow.cfg, deny.cfg, denycountry.cfg, internal.cfg, unlimited.cfg, GeoIP.dat` —
  фильтры подключений, НЕ копали.
- Реестр `HKLM\SOFTWARE\NC Soft\AION\<Component>` — зашифрованный connStr каждого компонента
  (XOR-blob, не DPAPI); сам создаётся после первого успешного подключения.
- Реестр `HKLM\SYSTEM\CurrentControlSet\Control\Nls\CodePage\ACP = 28591` — нужен LogServer64.
- RunAsDate (`MainServer\runasdate-x64`): подмена времени на 2020-06-04 16:28:23 для Server64;
  инжект-редиректор `dateinj01_64.dll`. Из-за этого логи Server64 лежат в `log\2020-06-04.err`.

## 5. Заглушки и фиксы БД (применённые)

Все скрипты идемпотентные, лежат в `scripts/sql/`.

| Заглушка/фикс | Что делает |
|---|---|
| `aion_GetDeletedCharList` + индекс `IX_delete_complete_date` | Удалённые персонажи: `user_data(delete_complete_date, delete_date) INCLUDE(char_id, user_id, account_id, account_name, guild_id, guild_rank)`; процедура зовёт таблицу с хинтом `index=IX_delete_complete_date` |
| Linked server `RC-AIONAUTHDB` → localhost | Процедура `aion_ProcessWithdrawAccount` ссылается на `RC-AIONAUTHDB.aionaccdeldb.dbo.del_account`; создали linked server (sa-логин) и БД-заглушку `aionaccdeldb` с таблицей `del_account(seq, account_id, stat)` — только те колонки, что реально читает процедура (seq — PK, stat=2 означает «выведен») |
| `Log_TblGameServerInfo_UpdateServerstatus(worldId, status, serverType)` | Заглушка в `Aion_log`: LogServer при старте вызывает её с `(1,1,5)`; обновляет таблицу `Log_TblGameServerInfo`, если та есть |
| `Log_TblGameWorldInfo_UpdateMainStatus` | Заглушка без параметров в `Aion_log` (из community-списка багов) |
| `IX_user_item_sealed_char_id` | Индекс на `user_item_sealed(char_id)` — из того же списка |
| `aion_SetItemMatterOption(itemId, stat_enchant0..5)` | Сохранение манастоунов: UPDATE `user_item_option` SET `stat_enchant_name0..5`. В нашей БД уже была (эталон из #180 хранится в `tools/ragezone-1211744/`) |
| Отсутствуют (нет тел, ждут сигнатур из логов) | `aion_GetItemCollectionList / LevelList / ExpiredList / CompleteTimeLimitList / CompleteList`, `aion_LoadFameInfo` (вызов `(3003)` = 1 int), `aion_LoadReinventInfo` |

Сейв персонажа `aion_SetCharInfo_20160818` / `aion_GetCharInfo_20160818` в нашей БД **есть** и
идентичен эталону сообщества (66 колонок UPDATE). БД `_AionWorldNew114_rc` полнее, чем
списки багов с чужих серверов — проверять факты у себя, не верить спискам вслепую.

## 6. Карта логов: где что читать

| Компонент | Файлы | Что искать |
|---|---|---|
| Server64 | `MainServer\log\2020-06-04.err` (дата RunAsDate!) | луперы Can't connect, крэши → `MainServer\AIONErr.txt` |
| NPCSvr64 | `NPCServer\log\*.err, *.leak, *.memory_summary, *.npcsvr` | симуляция мира, утечки, старт/шатдаун |
| CacheD64 | `CacheServer\log\*.err, *.NN.log, DBReport_Alert.DBErr` | SQL-ошибки процедур, abyss-циклы |
| LogServer64 | `LogServer\log\*.err, *.conn, *.leak, *.memory` | подключения logd/cached, ACP-проблемы |
| AccountCacheServer | `AccountCacheServer\log\*.account, *.err, *.NN.log` | пульс AccountData, работа с аккаунтами |
| L2Authd | `AuthD\etc\log\*.winlog, *.packet, *authd-in*.log` | ServerListPacket, счётчики логинов, PA-связь |
| AuthGateD | `AuthGateD\log\*.NN.log, *.mon` | сессии клиентов, Session id mismatched, brute-блоки |

Рост логов: CacheD при загрузке пишет ~120 МБ варнингов Strings DB (одноразово),
Server64 ~1–2 МБ/день на луперы. Стоп-батник удаляет все `*.err` (шаг [10/10]).

## 7. Что ещё НЕ копали (честный список)

- `MainServer\config.xml` (вторая половина конфига Server64 — детально не разбирали).
- `allow.cfg / deny.cfg / denycountry.cfg / internal.cfg / unlimited.cfg / GeoIP.dat`.
- Внутренности `AccountCacheServer\config.xml|common.xml`.
- `GIPServerPort 30271`, `mxserverPort_1 10241` — кто должен их использовать.
- Протоколы пакетов между Server64↔CacheD64↔NPCSvr (есть `.map`/`.pdb` файлы для Ghidra:
  `Server64.pdb`, `NPRelay64.pdb/.map/.i64`, `L2Authd.pdb`, `ScriptDLL64.map/pdb`).
- Параметры PAServer: бинарник почти без строк (возможно упакован), слушает `127.0.0.1:10057`,
  без config.txt в подвешенном минимальном состоянии. L2Authd к нему подключается.
- RankingServer (.NET + EntityFramework/Protobuf) и его схему БД.
- Целиком `l10n`/Strings DB (источник 570k варнингов при загрузке CacheD).

## 8. Клиентская сторона — кратко

- Запуск прямым бинарем: `bin64\Aion.bin -ip:<IP> -port:2106 -cc:<код> -noauthgg ...`.
- Западный клиент запускается через `AionLauncher.exe`, который читает `launcher.config`
  (одна строка аргументов; в ките был порт 2105 — нерабочий, правильный 2106).
- `cc.ini` (`cc="2"`), `config.ini` (VoiceChat/UpdateServer), `Pub.key`, `sc_renew*.dat` —
  клиентские файлы конфигурации.
- Клиенты западной линейки (`7720.0603.x`) логинятся только через портал → несовместимы
  с прямым AuthGateD (см. README «Клиенты»).

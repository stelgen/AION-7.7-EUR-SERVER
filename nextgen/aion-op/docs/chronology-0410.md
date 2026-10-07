# 📋 ПОЛНАЯ ХРОНОЛОГИЯ ДВИЖЕНИЙ — AION 7.7 PTS EU (04.10.2026)

> Каждый шаг этой сессии с результатом. Читай сверху вниз — видишь что сломалось, что починилось, что осталось.

---

## 1. ROOT-CAUSE ЦЕПОЧКА «НИ ОДИН ИГРОВОЙ СЕРВЕР НЕ БЫЛ ЗАКРЕПЛЁН»

```
Симптом: клиент авторизуется → видит список серверов → жмёт Подключиться
         → Server64 отвечает отказом → клиент: «не авторизован на сервере авторизации»

Фактор 1 (ОСНОВНОЙ): Server64 ДОЛЖЕН быть запущен через RunAsDate
  → Дата 2020-06-04 → authorization-time проверка в PlayerConnect ПРОХОДИТ
  → Без RunAsDate (дата 2026) → проверка ПАДАЕТ → отказ ВСЕГДА
  → #180-кряк снимает ТОЛЬКО date-check при СТАРТЕ, НЕ в PlayerConnect
  → Доказательство: дифф сток↔#180 = 4 байта (RGB2HSL+0x15B, EncodeAlive+0xE)
  → RunAsDate вернут → Server64 зарегистрируется → вход работает

Фактор 2 (КРИТИЧНОЕ ПРАВИЛО): inner_ip = 127.0.0.1 в таблице server НЕ ТРОГАТЬ
  → inner_ip = IP мира ДЛЯ AUTH-СВЯЗИ (Server64→authd)
  → Server64 подключается к authd с 127.0.0.1 (authServerAddr в common.xml)
  → authd сверяет источник с inner_ip → 127.0.0.1 = совпадает = registered
  → МОЯ ПРАВКА inner_ip=81.25.59.194 → СОВПАДЕНИЕ ПРОПАЛО → «non-registered world server»
  → ОТКАТАНО на 127.0.0.1, authd рестартнут

Фактор 3 (окно спавна): NPCSvr грузится 10-15 мин
  → Server64 принимает клиентские TCP ТОЛЬКО после «NPCSvr connection»
  → попытки логина во время загрузки NPCSvr → «not connected» (не ошибка, просто рано)

Фактор 4 (VPN-выход): клиентский трафик идёт через VPN-выход
  → 89.47.164.187 = датацентр Time4VPS (Литва) — порты свободны → вход работал
  → 184.160.77.85 = residential Videotron (Монреаль) — может фильтровать 7777
  → ПРОВЕРКА: Test-NetConnection 81.25.59.194 -Port 7777 с клиента
```

---

## 2. ЧТО СЛОМАЛОСЬ И КОГДА (хронология отказов)

| Время | Событие | Результат |
|---|---|---|
| 03.10 08:36 | Победа логина (`-loginex -pwd16`) | авторизация OK |
| 03.10 08:37 | World->Auth = **01** | принят |
| 03.10 08:49 | World->Auth = **02** | первая «не авторизован» |
| 03.10 10:20 | Рестарт Server64 (юзером RunAsDate) | |
| 03.10 10:32 | World->Auth = **01** | принят |
| 03.10 **10:35:44** | **ВХОД В МИР** (CacheD items загрузка) | ✅ играл 10:35–12:01 |
| 03.10 12:05–12:16 | **Дроп-деплой: рестарт ПАРЫ** (Server64+NPCSvr) | после этого — ни одного входа |
| 04.10 00:25+ | Рестарты authd/gate/логов, ребуты VM ×3 | authd стал умирать тихо |
| 04.10 02:12 | Server64 рестарт (RunAsDate CLI) | |
| 04.10 02:40, 02:58, 03:02 | Попытки юзера | World->Auth = 02/04/05 — отказ |
| 04.10 03:44 | Попытка юзера | World->Auth = **01** — принят! |
| 04.10 03:45 | AboutToPlayerTimer: Client(1010) not connected | клиент не дошёл TCP до 7777 |

---

## 3. ЧТО СДЕЛАНО В ЭТОЙ СЕССИИ (все изменения)

### БД (AionAccounts)
- `inner_ip`: 127.0.0.1 → 81.25.59.194 → **откат на 127.0.0.1** (финал)
- `port`: 7777 → 7778 → **откат на 7777** (финал)
- `ip`: 81.25.59.194 → 192.168.0.125 → **откат на 81.25.59.194** (финал)
- **ИТОГ: server table = вчера-рабочие значения, НЕ ТРОГАТЬ inner_ip!**

### Файлы
- `common.xml` authServerAddr: 127.0.0.1 → 192.168.0.125 → **откат на 127.0.0.1**
- `Server64.exe`: заменён FliesQQ (тест) → **откат на #180** (MD5 c5157302 сверен)
- Бекапы: `D:\_REF58\prod-backups\` (4 БД .bak + Server64.exe.bak + конфиги)

### Сервисы
- Лёгкие (Acc/Log/IC/CAPTCHA/PA/Proxy): ONSTART SYSTEM — поднялись после ребутов ✓
- L2Authd/AuthGateD: /IT-задачи (интерактивная сессия юзера) — поднялись ✓
- Тяжёлые (NPCSvr/Server64): батником с десктопа ✓
- authd/gate/log: поднимались SSH-стартом ✓

### Крипта (трай десериализации)
- Pub.key (клиент) = DER RSA-1024 PUBLIC — не секрет
- Blowfish-рутина: Game.dll offset 14,994,144 (по P-таблице)
- 68 кандидатов Blowfish-ключей по known-plaintext: 0 хитов
- welcome полностью скремблирован, статического ключа нет
- Сессионный Blowfish-ключ генерирует authd → Server64-side реверс проще (PDB есть)

---

## 4. БИНАРИ ИНВЕНТАРИЗАЦИЯ (D:\AION_LIVE_SERVER)

| Каталог | EXE | Размер | Статус | Назначение |
|---|---|---|---|---|
| AccountCacheServer | AccountCacheServer.exe | 20.5 МБ | ✅ запущен | кэш аккаунтов, порт 2220 |
| AccountCacheServer | RegisterAccount.exe | 14 КБ | — | регистрация аккаунтов (утилита) |
| AuthD | L2Authd.exe | 1.2 МБ | ✅ запущен | авторизация, порты 2104/2108/2110 |
| AuthGateD | AuthGateD.exe | 250 КБ | ✅ запущен | клиентский шлюз, порт 2107 |
| CacheServer | CacheD64.exe | 22.5 МБ | ✅ запущен | кэш БД мира, порты 2006/2007 |
| CaptCharServer | CAPTCHAImageServer.exe | 156 КБ | ✅ запущен | капчи, порт 22206 |
| CaptCharServer | CAPTCHAImageServerTest.exe | 70 КБ | не нужен | тест капчи |
| CaptCharServer | CAPTCHAImageServerTestGUI.exe | 94 КБ | не нужен | GUI тест капчи |
| CaptCharServer | CAPTCHAImageTest.exe | 115 КБ | не нужен | тест капчи |
| ChannelChattingServer | ChannelChattingServer.exe | 1.4 МБ | ❌ не стартует | чат-каналы, порт 10254 — реверс init |
| ICServer | ICServer.exe | 21.4 МБ | ✅ запущен | interchange/биллинг, порты 2005/2305 |
| LogServer | LogServer64.exe | 662 КБ | ✅ запущен | логи мира, порт 2051 |
| MainServer | **Server64.exe** | 45.5 МБ | ✅ запущен | МИР (кряк #180), порты 2002/7777 |
| MainServer | **MainServer64.exe** | 45.5 МБ | НЕ запущен | стоковый оригинал (= Server64.exe.orig) |
| NPCServer | NPCSvr64.exe | 25 МБ | ✅ запущен | спавны/AI/NPC, порт 2002 |
| **RankingServer** | RankingServer.exe | 158 КБ | ❌ НИКОГДА | абисс-рейтинги (.NET) |
| NPRelayServer | NPRelay64.exe | 21.8 МБ | ❌ не запущен | NCoin релей |
| **GMServer** (4.6) | GMServer.exe | 443 КБ | ❌ | GM-панель (.NET) |
| **GMShopServer** (4.6) | GMServer.exe | 3.7 МБ | ❌ | GM-магазин (.NET) |
| **PetitionServer** (4.6) | PetitionD_Integration.exe | 249 КБ | ❌ | саппорт-тикеты (.NET, порт 2107 ⚠️ конфликт с гейтом) |
| **ShopAgentServer** (4.6) | ShopAgentServer.exe | 157 КБ | ❌ | магазин (.NET, порт 10100) |

**ScriptDLL64.dll (87,731,200 байт)** — ОДНА для MainServer+NPCServer (MD5 e7016eae): скриптовый слой мира (квесты/спавны/AI) — **реверс-кандидат №2**

**ncguardserver64.dll (1,892,112)** — NC Guard защита в Server64

---

## 5. БАЗЫ ДАННЫХ

| БД | Размер | Сервес | Ключевое |
|---|---|---|---|
| AionAccounts | 80 МБ | L2Authd, PA, Server64 | `server` (inner_ip=127.0.0.1 НЕ ТРОГАТЬ!), user_account, user_auth (MD5), worldstatus, userno |
| AionAccountCacheD_rc | 16 МБ | AccountCacheServer | max text repl size=65664 |
| _AionWorldNew114_rc | 13,800 МБ | Server64, CacheD | user_data (user_id=NVARCHAR имя!), user_item, guild, abyss, house_*; **1056 процедур** |
| Aion_log | 13 МБ | LogServer64, GMServer/WebGM | схема **aiongm_ur**: TBL_GAME_SERVER_INFO, TBL_GAME_WORLD_INFO |
| PetitionDB/BkPetitionDB | 2 ГБ | PetitionServer | тикеты |
| aionaccdeldb | 16 МБ | authd | del_account |

**Дырки-процедуры (нет в БД):**
```
aion_GetItemCollection{List,LevelList,ExpiredList,CompleteTimeLimitList,CompleteList}(1002,1010,0)
aion_LoadFameInfo(1002) / aion_LoadFameReduceTime(1000) / aion_UpdateFameInfo(1000,1,0,0)
aion_LoadReinventInfo(1002)
aion_DeleteItemByDate_20191206(4,10)
aion_getItemAttributeDeltaListAll_20190919(1002) + VendorDark/VendorLight(0)
```
Оригиналы восстановлены: Log_TblGameServerInfo_UpdateServerstatus/UpdateLogfreedisk, Log_TblGameWorldInfo_UpdateMainStatus (в aiongm_ur схеме)

---

## 6. КОНФИГИ (ключевые поля)

### common.xml (MainServer)
```xml
<authServerAddr>127.0.0.1</authServerAddr>   ← НЕ ТРОГАТЬ (Server64→authd, inner_ip сверка!)
<authServerPort>2104</authServerPort>
<maxUser>500</maxUser>
<maxUserLevellight>80</maxUserLevellight>
<petitionServerAddr>127.0.0.1</petitionServerAddr>
<petitionServerPort>2107</petitionServerPort>  ← ⚠️ конфликт с гейтом!
<shopAgentServerAddr>127.0.0.1</shopAgentServerAddr>
<shopAgentServerPort>10100</shopAgentServerPort>
<mxChatS2SAddr>127.0.0.1</mxChatS2SAddr>
<mxChatS2SPort>10254</mxChatS2SPort>
<mxChatKey>0123456789ABCDEF0123456789ABCDEF</mxChatKey>
<mxChatSignature>AION-WING-SKY</mxChatSignature>
<DefaultEnableCheckIpFromAuth>False</DefaultEnableCheckIpFromAuth>
<useGIP>false</useGIP>
```

### AuthD\etc\config.txt
```
serverPort = 2104
serverExPort = 2110
serverIntPort = 2108
worldport = 7777
UseLogD = false
UseIPServer = false
UsePAServer = true
PAConnectionCount = 2
PAIP_1 = "127.0.0.1"
PAPort_1 = 10057
PAIP_2 = "127.0.0.1"
PAPort_2 = 10057
SocketTimeOut = 180
DBConnectionNum = 64
```

### AuthGateD\etc\config.txt
```
serverPort = 2107
authPort = 2110
loginType = 2
sessionTimeout = 5
tryInterval = 60
tryCount = 20
tryBlockInterval = 120
useGameGuard = false
checkGameGuard = false
dumpPacket = true
```

### LogServer\common.xml
```xml
<country>2</country>
<serverPort>2051</serverPort>
<UseBulkInsert>1</UseBulkInsert>
<BCPInterval>10000</BCPInterval>
```

### Таблица server (AionAccounts)
```sql
-- ФИНАЛЬНЫЕ ЗНАЧЕНИЯ (вчерашние рабочие, НЕ ТРОГАТЬ):
id=1, name='_MAIN', ip='81.25.59.194', inner_ip='127.0.0.1', port=7777, region=7
-- inner_ip = IP мира ДЛЯ AUTH-СВЯЗИ (Server64 подключается с 127.0.0.1)
-- НЕ менять inner_ip на внешний — authd потеряет мир!
```

---

## 7. ПОРТЫ И ЛОГИ

| Порт | Сервис | Лог куда |
|---|---|---|
| 2106 | proxy2106 (python) | `C:\Temp\proxylog.txt` (hex дамп auth) |
| 2107 | AuthGateD | `AuthGateD\log\YYYY-MM-DD.HH.log` + `.TXT` (dumpPacket) |
| 2104/2110 | L2Authd | `AuthD\etc\log\YYYY-MM-DD.*.winlog/.packet/.err/.dual/.use` |
| 2220 | AccountCacheServer | `AccountCacheServer\log\*.err` |
| 2051 | LogServer64 | `LogServer\log\*.err/.conn/.memory` |
| 2006/2007 | CacheD64 | `CacheServer\log\*.err` |
| 2002 | NPCSvr64↔Server64 | `NPCServer\log\*.err` |
| 7777 | Server64 (клиенты) | `MainServer\log\*.err` |
| 22206 | CAPTCHAImageServer | `CAPTCHAImageServer\*.log` |
| 10057 | PAServer | (нет файла-лога) |
| 10100/10254/2107 | ShopAgent/ChannelChat/Petition | луперы Server64.err (сервисов нет) |

---

## 8. LAUNCHERS (все на десктопе VM + гит scripts/launchers/)

| Батник | Действие |
|---|---|
| `AION-START-SERVER.bat` (v3) | Полный старт: 12 шагов с дедупом и port-чеками |
| `AION-STOP-SERVER.bat` (v3) | Полный стоп (только AION, SQL не трогает) |
| `AION-RESTART-WORLD.bat` | Рестарт пары Server64+NPCSvr (сброс счётчика) |
| `rad-restart.bat` (C:\Temp) | RunAsDate-CLI Server64 (без кнопки RUN) |

**Порядок стопа**: Server64 → RunAsDate → NPCSvr → CacheD → AuthGateD → L2Authd → LogServer → PA → ICServer → CAPTCHA → AccountCache → Proxy

---

## 9. ПЕРМАНЕНТНЫЙ ФИКС (очередь, по «го»)

1. **Ghidra-кряк authorization-time в Server64 #180** (PDB есть): найти PlayerConnect ветку → NOP дату-проверку → Server64 работает без RunAsDate навсегда
2. ScriptDLL64.dll 87.7 МБ реверс (квесты/спавны/AI — вшитая логика)
3. RankingServer подъём (.NET)
4. ChannelChat реверс (инициализация умирает в session 0)
5. Крипта: Server64-side лог Blowfish-ключа (PDB есть) → расшифровка траффика прокси

---

## 10. БЕЗОПАСНОСТЬ И БЕКАПЫ

- `D:\_REF58\prod-backups\`: AionAccounts.bak, AionWorld.bak, Aion_log.bak, AionAccountCacheD.bak + Server64.exe.bak + конфиги — снапшот 04.10
- Откат БД: RESTORE DATABASE из .bak
- Откат бинаря: copy .bak → Server64.exe
- inner_ip откат: UPDATE server SET inner_ip='127.0.0.1'
- ВСЕ изменения этой сессии откатываются за 5 минут
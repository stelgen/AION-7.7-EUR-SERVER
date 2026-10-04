# Мердж логов: успешный vs проблемный логин (04.10.2026, дополнение к authlog-analysis-0410.md)

READ-ONLY. Источники: `C:\Temp\proxylog.txt` (MITM 2106→2107, копится с 03.10 07:14 — локально `STELGEN/tmp/aion-vm/proxylog-0410.txt`), `AuthD\etc\log\*.{packet,winlog,dual,use}`, `MainServer\log\2020-06-04.err`. НОВЫЙ ФАКТ ЮЗЕРА: `Test-NetConnection 81.25.59.194 -Port 7777` = **TcpTestSucceeded: True** (tun0, source 172.19.0.1) — 7777 проброшен и достижим. Гипотеза «недостижимости 7777» СНЯТА.

## Формат прокси-лога
`G->C/C->G len=N hex=...` — фрейминг 2b LE. Расшифровке НЕ подлежит (blowfish, сессионный ключ) — но ДОСТУПНЫ длины и тайминги: welcome=194, client-hello=34, gate-ok=42, login=314 (LoginEx), auth-OK=74, serverlist-req=26, **server list=42**, **select=26 (C->G)**, ответ на select = **26 (норма) или 18 (отказ гейта)**.

## ✅ УСПЕШНЫЙ вход №1 — 03.10 10:35:32 (сессия 5182 с, юзер в мире до 12:01:54)

| t (03.10) | источник | событие |
|---|---|---|
| 10:20:xx | winlog | *new world server connection (Server64 рестарт 10:20, сокет 1928) |
| 10:32:31 | proxylog | new client 184.160.77.85 → 194/34/42/314/74/26(req)/**42 serverlist** |
| 10:32:35 | proxylog | C->G 26 select → G->C 26 ответ → **C->G closed В ТУ ЖЕ СЕКУНДУ** |
| 10:32:35 | authd packet | Gate->Auth(2) → World->Auth **f203000001** (принято) |
| 10:33:05 | authd dual/packet | LOGOUT + f2030000cb (мир: клиент не пришёл, 30 с) — ПОПЫТКА №1 ПРОВАЛ |
| 10:35:31 | proxylog | new client 89.47.164.187 → 194/34/42/314/74/26 → **42 serverlist** |
| 10:35:36 | authd packet | select → World->Auth **f203000002** |
| 10:35:36 | proxylog | G->C 26 ответ на select → **2106 ОСТАЁТСЯ ОТКРЫТЫМ 10 МИН (до 10:45:31)** |
| 10:35:43-44 | authd packet | фрагменты World->Auth f2030000… (сессионные) |
| 10:41:31…12:01 | authd packet | heartbeat'ы мира: `f203000001 ea030000(char 1002) 02000000 <ts>` каждые ~5-6 мин |
| 12:01:54 | authd dual | LOGOUT Stelgen (5182 c) + f2030000cb |

**Server list 03.10 (Auth->Gate(4)): `0100/0101 01 c0a8007d 611e 000000000000 f401 …02…` = IP 192.168.0.125:7777, f401=500(maxUser), байт#23=02.**

## ✅ УСПЕШНЫЙ вход №2 — 03.10 22:46:01 (579 с)

```
22:46:00 new client 184.160.77.85 → 194/34/42/314/74/26/42(serverlist)
22:46:05 C->G 26 select → G->C 26 → C->G closed В ТУ ЖЕ СЕКУНДУ
22:46:27 authd: World->Auth f203000001ea03000009000000… (чар 1002 в мире)
22:47-22:51 authd: 0100f401 пульсы; 22:55:40 LOGOUT (579 c)
```
Сервер list тот же: 192.168.0.125, байт#23=02.

## ❌ ПРОБЛЕМНЫЙ вход — 04.10 06:59:28 (пятая попытка подряд)

```
06:59:27 new client 89.47.164.187
06:59:28 194 welcome / 34 / 42 / 314 login / 74 auth-OK
06:59:28 C->G 26 (req) → G->C 42 (server list)
06:59:31 C->G 26 (SELECT) → G->C 26 (ответ: вход разрешён, уходи на world-адрес)
        …далее тишина; клиент НЕ уходит на 7777
07:00:01 authd dual: LOGOUT (32 c — aboutToPlayTimer=30000 истёк)
07:03:10 C->G closed (клиент висел в 2106 ещё 3.7 мин — диалог ошибки)
```
**Server list 04.10: `010101 51193bc2 611e …f401…07…` = IP 81.25.59.194:7777, байт#23=07.**
То же на 06:33:23 (закрыл 2106 +2 с), 06:46 (+3 с), 06:53, 06:55 — все 26-байтный ответ, TCP до мира не дошёл.

## Server64 (MainServer\log\2020-06-04.err) — конкретные строки

⚠️ Логи УСПЕШНЫХ ранов (10:20/12:12 03.10) уничтожены (STOP-батник чистит *.err) — прямых «строк успешного логина» Server64 не сохранилось. Сохранившееся (имя файла = RunAsDate-дата, часы заморожены на «16:28», ориентир — порядок строк):

Старт рана (последний, ~06:08 04.10, строки 102106+):
```
Can't connect to cache server at 127.0.0.1:2006
ChatAccuseSystem Reset Time (init) => 1591277303
AboutToPlayTimer: 30000 msec
ExpMultiple:100 CraftExpMultiple:100 AbyssPointMultiple:100 …
Config :==  <clientAcceptAddr>192.168.0.125</clientAcceptAddr>
Config :==  <mxserveraddr_1>192.168.200.131</mxserveraddr_1>
CondSpwnTimeMgr::TimerExpired, NPCSvr hasn't connected. Waiting 10 second...
Connected to AuthServer: auth server at 127.0.0.1:2104
```
Ошибка при проблемном логине (строки 131947+, сегодня; всего 5 шт = 5 попыток):
```
LogClientSocket::AlivePacket, LogServer responds to an alive packet.
AboutToPlayerTimer ::== Client(Acct: 1010) is not connected      ← мир ждал 30 с, TCP 7777 не пришёл
Can't connect to ChannelChat server at 127.0.0.1:10254           (фоновый лупер)
Can't connect to shop agent server at 127.0.0.1:10100            (фоновый лупер)
AboutToPlayerTimer ::== Client(Acct: 1010) is not connected
AuthSocket Close 13e7dc(50414c10)
WaitPlaySocketListMgr::DisconnectAll: size 0
Connected to AuthServer: auth server at 127.0.0.1:2104           (реконнект после рестарта authd 06:51)
Connect AuthServer Protocol Version authVersion:2017012601, protocolVersion:1, reconnectFlag:0
AboutToPlayerTimer ::== Client(Acct: 1010) is not connected      ← продолжается и после реконнекта
[6] Too slow function, execution time = 15171 mili second … ObjectCountViewer::TimerExpired  ← 15-сек лаг-спайк IO-треда
```
В ране 03.10 12:12 — те же 3 строки AboutToPlayer (12:19/21/23, окно спавна NPCSvr до 12:39) — та же сигнатура «принял и ждёт».

## authd winlog — события дня
- `*new world server connection from 127.0.0.1` — 04:39:41…04:45:34 каждые ~17 с (флап: Server64 реконнектится к умирающему authd), 04:52:45, 05:11:21, 05:28:31, 05:49:29, 05:53:52, **06:27:23, 06:51:55** (текущий).
- `[ERROR] *close connection from 127.0.0.1` + `ServerDown(1) charging process` — 05:20:09, 05:40:29, 05:54:53 (мир отваливался), 06:06:53 CPASocket 10057.

## Что ИЗВЕСТНО и что НЕТ
ИЗВЕСТНО: auth-цепочка и вердикты ОК; server list доходит (42б); select проходит (26б ответ); клиент рвёт 2106 через 0-3 с и НЕ открывает TCP к миру; мир честно ждёт 30 с. Пакет server list изменился в ДВУХ местах: IP (192.168.0.125→81.25.59.194) и байт#23 (02→07, похоже region из БД server).
НЕ ИЗВЕСТНО (нужны данные/эксперименты, ничего не меняя):
1. Какой `-ip:` в клиентском bat сейчас (если 192.168.0.125, а мир отдаёт 81.25.59.194 — клиент может отказываться от «чужого» адреса; на 03.10 login-IP и world-IP совпадали!).
2. Доходит ли SYN клиента до VM:7777 во время РЕАЛЬНОЙ попытки (TNC — это не игровой клиент). План: юзер делает 1 попытку, параллельно на VM читаем `netstat -ano | findstr :7777` (и, если юзер разрешит, короткий pktmon-фильтр на 7777).
3. Не блокирует ли Frost/клиент WAN-адрес мира (та самая причина, от которой лечит version.dll-патч — но 03.10 LAN-адрес проходил, значит проверка завязана на сравнение с login-IP либо на список).

## Быстрые проверки-кандидаты (по «го», порядок не меняет прод без разрешения)
- A. Вернуть в `AionAccounts..server.ip` = 192.168.0.125 (как весь 03.10) + рестарт пары → server list снова совпадает с login-IP юзера.
- B. Если юзер хочет внешний IP: поменять `-ip:` в клиентском bat на 81.25.59.194 (чтобы login-IP == world-IP) и повторить.
- C. Живой тест: попытка входа + netstat-наблюдение на VM → смотрим, рождается ли SYN на 7777.

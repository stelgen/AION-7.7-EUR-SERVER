# 🌐 Карта портов AION 7.7 PTS EU (VM)

Все порты должны быть доступны на **внутренней сети VM** (bridge/`vmbr0`, DHCP или статика `192.168.0.125`). Проброс наружу не нужен, если клиент в той же LAN.

## 📋 Слушающие порты (в порядке запуска)

| Порт | Процесс | Зачем |
|---|---|---|
| **2106** | `AuthGateD.exe` | Точка входа **клиентов** (login) |
| **2104** | `L2Authd.exe` (AuthD) | Связь с game-server (Server64 подключается сюда) |
| **2108** | `L2Authd.exe` | Interactive socket (GM tools) |
| **2110** | `L2Authd.exe` | Connection for AuthGateD |
| **2051** | `LogServer64.exe` | Log-сервер (коннект от CacheD64/MainServer) |
| **2006** | `CacheD64.exe` | Мир-кэш (Server64 подключается сюда) |
| **2007** | `CacheD64.exe` | Interactive port CacheD64 |
| **2220** | `AccountCacheServer.exe` | Кэш аккаунтов |
| **10062** | `L2Authd.exe` | QMAS socket (внутренний) |
| **7777** | `Server64.exe` | Игровой порт (клиенты подключаются сюда после логина) |
| **1433** | `MSSQLSERVER` (SQL) | SQL Server TCP |
| **1434** | SQL Browser (UDP) | SQL discovery |

## 🔀 Схема соединений (кто к кому цепляется)

```
Клиент (aion.bin) ──2106──▶ AuthGateD
                             │
                             └──127.0.0.1:2110──▶ L2Authd (AuthD)
                                                   │
                                                   ├──▶ SQL: AionAccounts (через L2Conn.dsn)
                                                   └──▶ AccountCacheServer:2220

Клиент (после логина) ──7777──▶ Server64
                                  │
                                  ├──▶ CacheD64:2006   (мир-кэш)
                                  ├──▶ LogServer64:2051 (логи)
                                  ├──▶ L2Authd:2104     (auth-проверки)
                                  └──▶ SQL: _AionWorldNew114_rc (мир-БД)
```

## 🧱 Firewall на VM

Стандартно включён. Проброс не нужен для локальной сети (LAN-клиенты достучатся напрямую по `192.168.0.125`). Если захочешь из другой подсети — открыть входящие `2106, 7777` + (опционально) `2104, 2110, 2006, 2051, 2220`.

## ⚙️ Связь с конфигами

| Конфиг | Что там о портах |
|---|---|
| `AuthD\etc\config.txt` | `serverPort=2104`, `serverExPort=2110`, `serverIntPort=2108`, `worldport=7777`, `serverMiPort=10062` |
| `AuthGateD\etc\config.txt` | `serverPort=2106`, `authAddr=127.0.0.1`, `authPort=2110`, `loginType=2` |
| `CacheServer\common.xml` | `serverPort=2006`, `interactivePort=2007`, `logserveraddr=127.0.0.1`, `logserverport=2051` |
| `AccountCacheServer\common.xml` | `serverPort=2220`, `mailServer=<VM_IP>` |
| `LogServer\config.xml` | `serverAddress=127.0.0.1`, `country=2` |
| `MainServer\config.xml` | `clientAcceptAddr=<VM_IP>` |
| SQL таблица `Server` (в `AionAccounts`) | `id=1, name=_MAIN, ip=<VM_IP>, port=7777, region=2` |

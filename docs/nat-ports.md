# 🌐 Проброс портов за NAT для AION 7.7 PTS EU

Если сервер будет доступен **из интернета** (не только LAN), на роутере нужно пробросить порты на VM `192.168.0.125`.

## ✅ Обязательные для игры (клиенты снаружи)

| Порт | Протокол | Сервис | Зачем |
|---|---|---|---|
| **2106** | TCP | AuthGateD | Логин клиента (вход в игру) |
| **7777** | TCP | Server64 | Игровой трафик (мир, персонажи) |

⚠️ Этого **достаточно** для подключения клиента извне. Всё остальное — внутренние порты на `127.0.0.1`, их наружу не пускать.

## 🔧 Внутренние порты (НЕ пробрасывать наружу!)

Эти порты работают только между компонентами на `127.0.0.1` внутри VM:

| Порт | Сервис | Зачем |
|---|---|---|
| 2104 | L2Authd | Связь AuthGateD ↔ AuthD |
| 2108 | L2Authd | GM tools (interactive) |
| 2110 | L2Authd | AuthGateD → AuthD |
| 2006 | CacheD64 | Server64 → мир-кэш |
| 2007 | CacheD64 | Interactive |
| 2051 | LogServer64 | Server64 → логи |
| 2220 | AccountCacheServer | L2Authd → кэш аккаунтов |
| 10062 | L2Authd | QMAS socket (внутренний) |
| 1433 | MSSQLSERVER | SQL Server (TCP) |
| 1434 | SQL Browser | UDP discovery |

## 🛠 Настройка роутера

Если у тебя TP-Link/Asus/MikroTik — правило такое:

```
Name:          AION-Login
External port: 2106
Protocol:      TCP
Internal IP:   192.168.0.125
Internal port: 2106
```

```
Name:          AION-Game
External port: 7777
Protocol:      TCP
Internal IP:   192.168.0.125
Internal port: 7777
```

## 🔒 Firewall на VM (Proxmox)

Firewall на VM 109 сейчас включён (`firewall=1` в сетевом интерфейсе). Для локальной LAN-игры ничего не трогай — порты уже доступны. Для внешнего доступа:

```powershell
# На VM (PowerShell от админа):
Get-NetFirewallRule -Name RemoteDesktop-* | Enable-NetFirewallRule -EA SilentlyContinue
New-NetFirewallRule -DisplayName "AION Game" -Direction Inbound -Protocol TCP -LocalPort 7777 -Action Allow
New-NetFirewallRule -DisplayName "AION Login" -Direction Inbound -Protocol TCP -LocalPort 2106 -Action Allow
```

Либо в Proxmox UI: VM 109 → Firewall → Add Rule:
```
Direction: IN, Action: ACCEPT, Protocol: TCP, Dest. port: 2106, Source: any
Direction: IN, Action: ACCEPT, Protocol: TCP, Dest. port: 7777, Source: any
```

## 🌍 Если игроки подключаются снаружи (динамический IP)

- **DDNS**: подними DuckDNS/No-IP на роутере → домен `yourname.duckdns.org`
- **Клиент меняет IP**: в ярлыке клиента вместо `-ip:192.168.0.125` ставят `-ip:yourname.duckdns.org`
- **Таблица `Server` в БД `AionAccounts`** тоже должна показывать внешний IP:
  ```sql
  USE [AionAccounts];
  UPDATE Server SET ip='yourname.duckdns.org' WHERE id=1;
  ```

## 🛡 Секретность (не палить наружу)

Не пробрасывай наружу:
- ❌ **1433** (SQL Server) — держи только на `localhost`/LAN
- ❌ **3389** (RDP) — только через WireGuard/VPN
- ❌ **22** (SSH) — только через WireGuard/VPN
- ❌ **2220, 2006, 2007, 2051, 2104, 2108, 2110, 10062** — все внутренние, работают на `127.0.0.1`

## ✅ Итого (если извне)

На роутере пробрось **только 2 порта TCP**: `2106` и `7777` → `192.168.0.125`. Больше ничего не надо.

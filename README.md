# AION 7.7 PTS EUR — сервер под Windows (Proxmox VM) 🛡️

**Полный пошаговый гайд** по поднятию leaked retail-сервера **AION 7.7 PTS (Europe)** на Windows Server 2022 в Proxmox VE — со всеми фиксами, тиками и граблями, которые встретились при реальной установке.

> ⚠️ **Лицензия /legal**: файлы сервера (`AION7.7SERVER(eu).rar`, клиент) — собственность NC Soft, здесь НЕ хранятся. Скрипты и документация в репо — наши. Где взять файлы — см. [docs/links.md](docs/links.md) (RaGEZONE-тред, свой NAS и т.п.).

## 📋 Что тут есть

| Файл/папка | Что это |
|---|---|
| [README.md](README.md) | **Главный гайд** — по шагам, от VM до входа в игру |
| [docs/fields.md](docs/fields.md) | Все поля всех диалогов: что куда вводить (таблицы) |
| [docs/errors.md](docs/errors.md) | Каждая встреченная ошибка → причина → фикс |
| [docs/ports.md](docs/ports.md) | Карта портов всего стека |
| [docs/links.md](docs/links.md) | Легальные ссылки на скачиваемое (MS ISO/SDK/VC++, archive.org ISO, RaGEZONE-гайды) |
| [docs/fixes-community.md](docs/fixes-community.md) | **Community-фиксы из RaGEZONE-треда (2024-2025)**: Server64 date-patch, manastones SQL, CacheD64-fix, version.dll, БД 5.8 |
| `scripts/prep-vm.ps1` | Твики VM: zram (Memory Compression), NCSI-off, тёмная тема, деблоат, DisableCAD |
| `scripts/install-prereqs.ps1` | Тихая установка: VC++ 2010–2022, 7-Zip, SQL Native Client 11.0, SSMS (опц.) |
| `scripts/install-sql2022.ps1` | Тихая установка SQL Server 2022 Developer (в обход бага 2017 RTM) |
| `scripts/restore-dbs.ps1` | Restore всех 6 баз `.bak` (в т.ч. твой `AionAccounts.bak`) на `D:\SQL` |
| `scripts/sql-tune.ps1` | `max text repl size = 65664` (обязательный фикс CacheD64) + max memory |
| `scripts/setup-odbc.ps1` | Раскладка DSN: в `C:\DSN\`, в Common Files, в каждую папку сервера + алиас `aion_accoutdb.dsn` |
| `scripts/config-patch.ps1` | IP-замены (`192.168.0.125`), country=2, region=2, hardlink `Map\XML\Europe` |
| `scripts/start-server.bat` | Автозапуск всего стека по порядку с задержками (то же, что `AION-START-SERVER.bat`) |
| `scripts/stop-server.bat` | Останов всех компонентов |

## 🚀 Быстрый старт (если ты уже в середине)

```powershell
# на VM (PowerShell от админа):
Set-ExecutionPolicy -Scope Process Bypass -Force
cd C:\Temp
Invoke-WebRequest https://raw.githubusercontent.com/stelgen/AION-7.7-EUR-SERVER/main/scripts/prep-vm.ps1 -OutFile prep-vm.ps1
.\prep-vm.ps1
```

Дальше по шагам из [README](#-шаг-за-шагом).

## ✅ Шаг за шагом

### Шаг 0 — Что нужно заранее
1. **Windows VM** (Proxmox): Server 2022 Desktop Experience, 8 vCPU / **28–32 ГБ RAM** / 128+128 ГБ диски (C:, D:).
2. **Файлы сервера**: `AION7.7SERVER(eu).rar` (4.2 ГБ) — положить в `D:\Temp\`.
3. **Драйверы VirtIO** ISO в CD-ROM VM (`virtio-win-0.1.302.iso`), **SQL ISO** — скачаем сами.
4. Патч-файл `AionAccounts.bak` — опционально (он дублирует комплектный).

### Шаг 1 — Пререквизиты (один скрипт)
```powershell
.\install-prereqs.ps1     # VC++ (2010/2012/2013/2015-2022 x64+x86) + 7-Zip + SNAC11 + SSMS
```
`SQL Server Native Client 11.0` — **обязателен** (драйвер, на котором написаны все DSN). Без него все коннекты валится `HY000/556 недопустимый файл DSN ""`.

### Шаг 2 — SQL Server 2022
```powershell
.\install-sql2022.ps1     # тихая установка, sa/123, Mixed mode, TCP+NP, data на D:\SQL
```
> ⚠️ **Почему 2022, а не 2017?** RTM-установщик SQL 2017 на Windows Server 2022 падает с «Ошибка при создании документа XML» (баг 2017 RTM, чинили только в CU). Restore 2017-бэкапов в 2022 работает без проблем.

> ⚠️ **Setup должен запускаться в интерактивной сессии**, не из SSH-канала! Через SSH — падение `CryptographicException` (DPAPI). Обход: `schtasks /Create /IT` + `/Run`, либо вручную в GUI.

### Шаг 3 — Restore баз
```powershell
.\restore-dbs.ps1         # 6 баз: AionAccounts, AionAccountCacheD_rc, _AionWorldNew114_rc, Aion_log, BkPetitionDB, PetitionDB
```
Файлы баз уедут в `D:\SQL\MSSQL16.MSSQLSERVER\MSSQL\DATA\` (MOVE-редирект из старых путей `C:\...\MSSQL14...` в бэкапах).

### Шаг 4 — Тюниг SQL (обязательный!)
```powershell
.\sql-tune.ps1            # max text repl size = 65664, max server memory = 4–12 ГБ
```
> **`max text repl size = 65664`** — CacheD64 **самоубивается** без этого (`Invalid 'Max text repl size'(1536) value 65536 (65664 or -1 expected)` → graceful shutdown через 5 сек).

### Шаг 5 — Раскладка DSN + правка конфигов
```powershell
.\setup-odbc.ps1          # DSN в C:\DSN\ + Common Files + каждая папка сервера; alias aion_accoutdb.dsn
.\config-patch.ps1        # IP + country=2 + region=2 + Map\XML\Europe hardlinks
```
> 🔑 **Ключевой хак**: поле `File DSN`/`File DB` в диалогах принимает **только ПОЛНЫЙ ПУТЬ**, и он должен быть **< ~76 символов** (длинный путь обрезается `\u0001`). `C:\DSN\<name>.dsn` — идеально.

### Шаг 6 — Запуск (в правильном порядке, с задержками)
```cmd
start-server.bat
```
Порядок: AccountCacheServer → L2Authd → AuthGateD → LogServer → **CacheD64** → NPCSvr64 (10 мин) → Server64 (через RunAsDate).

> 🕒 **RunAsDate** (`MainServer\runasdate-x64\`) подменяет время системным «04-06-2020 16:28:23» (дата билда Server64.exe) — иначе `invalid system time` → отказ старта.

### Шаг 7 — Клиент
См. [docs/fields.md](docs/fields.md) — полный ярлык запуска клиента, все параметры.

---

## 🧪 Легенда разведки (для тех, кто попал в середину)

Стек собирается так:
- **AccountCacheServer** (порт 2220) → держит кэш аккаунтов;
- **AuthD/L2Authd** (2104/2108/2110) → авторизация; читает **`AionAccounts`** через `L2Conn.dsn`;
- **AuthGateD** (2106) → точка входа клиентов, посылает в AuthD;
- **LogServer64** (2051) → пишет логи в SQL (база `Aion_log` через `aiongm.dsn`);
- **CacheD64** (2006/2007) → кэш мира/предметов, читает `_AionWorldNew114_rc` + `AionAccountCacheD_rc`;
- **NPCSvr64** → спавны/скрипты NPC (жрёт до 10 ГБ RAM при загрузке);
- **Server64** (7777) → игровое ядро, запускается через RunAsDate.

Все диалоги `SQL Login` / `L2 ODBC Connection Info` после **первого успешного** коннекта сохраняют зашифрованную `connStr` в `HKLM\SOFTWARE\NC Soft\AION\<Component>` — больше не спрашивают.

---

## 📜 Источники / легальные ссылки
- [docs/links.md](docs/links.md) — где легально качается всё (Microsoft ISO, VC++, SNAC, SQL ISO на archive.org, virtio-win с GitHub).
- RaGEZONE-гайд: [AION7.7pts (Europe) set up tutorial](https://forum.ragezone.com/threads/aion7-7pts-europe-set-up-tutorial-and-server-download-address.1211744/) (dAIdoom, 2023).
- RaGEZONE-гайд на 4.6 (основа): [AION 4.6 retail server](https://forum.ragezone.com/threads/aion-4-6-retail-server-file-re-post.1197401/).

## 🤝 Вклад
Добавляй свои находки через PR — особенно по отсутствующим хранимым процедурам (см. [docs/errors.md](docs/errors.md)).

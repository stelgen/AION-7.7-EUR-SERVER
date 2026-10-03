# AION 7.7 PTS EUR — сервер под Windows (Proxmox VM) 🛡️

**Полный гайд** по поднятию leaked retail-сервера **AION 7.7 PTS** на Windows Server 2022 в Proxmox VE — архитектура, все компоненты, порты, фиксы (что исправлено и когда применять), клиенты и грабли реальной установки.

> ⚠️ **Legal**: файлы сервера/клиента — собственность NC Soft, здесь НЕ хранятся. Скрипты и документация — наши. Где взять файлы: [docs/links.md](docs/links.md).

## 🗺 Содержание
1. [Архитектура стека](#-архитектура-стека) — кто на чём сидит (включая сервисы, о которых забывают)
2. [Порты](#-порты) — клиентские vs внутренние
3. [Шаг за шагом](#-шаг-за-шагом) — установка от VM до входа в игру
4. [Диалоги SQL при первом старте](#-диалоги-sqlodbc-при-первом-старте-компонентов) — что вводить
5. [Фиксы: что исправлено и когда применять](#-фиксы-что-исправлено-и-когда-применять)
6. [Клиенты](#-клиенты) — какой качать и почему (частая ловушка!)
7. [Эксплуатация](#-эксплуатация) — RAM, логи, рестарты
8. [Структура репо](#-структура-репо)

---

## 🏗 Архитектура стека

Поток игрока: **Клиент → AuthGateD(2106) → L2Authd(2110) → AccountCacheServer(2220)/PA(10057) → мир: Server64(7777)**

### Обязательные компоненты (порядок старта = порядок в таблице)

| # | Компонент | Exe | Порты | Что делает | Кому нужен |
|---|---|---|---|---|---|
| 1 | AccountCacheServer | `AccountCacheServer.exe` | 2220 | Кэш аккаунтов, авто-регистрация | L2Authd, Server64 |
| 2 | L2Authd (AuthD) | `L2Authd.exe` | 2104 (game srv), 2108 (GM), **2110 (для AuthGateD)** | Авторизация, акк-БД `AionAccounts` (через `L2Conn.dsn`), ходит в PA(10057) | AuthGateD |
| 3 | AuthGateD | `AuthGateD.exe` | **2106** | Точка входа клиентов (loginType=2, companyCode=2) | Клиент |
| 4 | LogServer64 | `LogServer64.exe` | 2051 | Пишет логи в SQL (`Aion_log`). Требует **ACP=28591** (см. Фиксы F1) | Server64, CacheD64, NPCSvr64 |
| 5 | CacheD64 | `CacheD64.exe` | 2006/2007/2009 | Кэш мира/предметов, БД `_AionWorldNew114_rc` + `AionAccountCacheD_rc` | Server64 |
| 6 | ICServer | `ICServer.exe` | 2005 (MainServer), 2305 (CacheD) | **Interchange/Channel** — без него лупер «Can't connect to Interchange» у обоих | Server64, CacheD64 |
| 7 | CAPTCHAImageServer | `CAPTCHAImageServer.exe` | 22206 | CAPTCHA. ⚠️ Порт в конфиге кита `43330` ломается (баг int16 у Server64: 43330−65536=−22206) — у нас исправлен на **22206** | Server64 |
| 8 | NPCSvr64 | `NPCSvr64.exe` | — | Спавны/скрипты NPC, DynamicField; **грузится 10–15 мин, до ~15 ГБ RAM** | Server64 (мир не стартует без него) |
| 9 | Server64 | `Server64.exe` | **7777**, 2002 | Игровое ядро. Запуск ТОЛЬКО через RunAsDate (время 04-06-2020 16:28:23) или готовый date-bypass патч | Клиент |

### Внутренние сервисы (участвуют в auth-цепочке — запускать!)

| Компонент | Exe | Порт | Роль |
|---|---|---|---|
| **PAServer (PortalAuth)** | `01-PAServer7.7.exe` | 10057 (loopback) | **Обязателен**: `AuthD\etc\config.txt` → `UsePAServer=true`, L2Authd держит 2 соединения к нему (`PAIP_1/2=127.0.0.1:10057`). Без PA у EU-клиентов — «login only after official portal» |
| NPRelayServer | `NPRelay64.exe` | — (исходящий) | NCoin/Warehouse-релей к MainServer. Не блокирует логин; тестировался, задачи в DISABLE |
| RankingServer | `RankingServer.exe` | .NET-сервис | Веб-рейтинг; некритично, требует своего config.xml |
| NPCRelay | `NPRelay64.exe` | — | см. NPRelayServer |

### Не входят в кит (опционально, луперы если не поднять — см. [fixes-pending](fixes-pending/))
`ShopAgent(10100)`, `Petition(2107)`, `ChannelChat(10254)` — exe отсутствуют; луперы event-driven и безвредны.

---

## 🔌 Порты

| Наружу (клиентам) | Внутри (localhost/LAN VM) |
|---|---|
| **2106** TCP — логин | 2220, 2104, 2108, 2110, 2051, 2006/2007/2009, 2005, 2305, 22206, 2002, 10057 |
| **7777** TCP — мир | 1433 (SQL), 3389 (RDP), 22 (SSH) — наружу НЕ открывать |

Проброс за NAT и правила брандмауэра: [docs/nat-ports.md](docs/nat-ports.md).

---

## ✅ Шаг за шагом

### Шаг 0 — Что нужно заранее
1. **Windows VM**: Server 2022 Desktop, 8 vCPU / **32 ГБ RAM** (28 — впритык: Server64 ~10 ГБ + NPCSvr ~15 ГБ + SQL), диски C: 128 / D: 128.
2. **Файлы сервера** `AION7.7SERVER(eu).rar` → `D:\Temp\`.
3. VirtIO ISO в CD-ROM, SQL ISO — скачаем.

### Шаг 1 — Пререквизиты
```powershell
.\install-prereqs.ps1   # VC++ 2010–2022, 7-Zip, SQL Native Client 11.0 (ОБЯЗАТЕЛЕН), SSMS
```
> Без SNAC11 все DSN падают `HY000/556 «недопустимый файл DSN ""»`.

### Шаг 2 — SQL Server 2022
```powershell
.\install-sql2022.ps1   # sa/123, Mixed mode, TCP+NP, data → D:\SQL
```
> ⚠️ SQL 2017 RTM на WS2022 падает («Ошибка при создании XML»). ⚠️ Установка — только в интерактивной сессии (из SSH: DPAPI `CryptographicException`).

### Шаг 3 — Restore баз
```powershell
.\restore-dbs.ps1   # AionAccounts, AionAccountCacheD_rc, _AionWorldNew114_rc, Aion_log, BkPetitionDB, PetitionDB → D:\SQL
```

### Шаг 4 — Тюнинг SQL (обязательно)
```powershell
.\sql-tune.ps1   # max text repl size = 65664 (иначе CacheD64 самоубивается), max memory 2–4 ГБ
```
> **max server memory = 2048 МБ** — проверено: при 4–12 ГБ стэк (Server64+NPCSvr) упирается в commit и процессы умирают молча.

### Шаг 5 — DSN + конфиги
```powershell
.\setup-odbc.ps1     # DSN в C:\DSN\ + Common Files + в каждую папку сервера; алиас aion_accoutdb.dsn
.\config-patch.ps1   # IP=192.168.0.125, country=2, region=2, hardlink Map\XML\Europe
```
> 🔑 Поле `File DSN` в диалогах принимает только **полный путь ≤ ~76 симв**; работает и **с расширением, и БЕЗ `.dsn`** (`C:\DSN\aion_accoutdb`) — если диалог не берёт файл, вводи путь без расширения.

### Шаг 6 — SQL-фиксы БД (до первого старта!)
```cmd
sqlcmd -S localhost -U sa -P 123 -i scripts\sql\fix-db-2026-10-02.sql
sqlcmd -S localhost -U sa -P 123 -i scripts\sql\fix-community-2026-10-02.sql
```
Подробнее: [Фиксы](#-фиксы-что-исправлено-и-когда-применять).

### Шаг 7 — Старт стека
```cmd
scripts\start-server.bat   (десктопный AION-START-SERVER.bat — то же самое)
```
- Проверяет «уже запущен» (не дублирует), стартует через задачи schtasks, в конце — сводка портов и RAM.
- Порядок: AccountCache → L2Authd → AuthGateD → LogServer → CacheD → **ICServer** → **CAPTCHAImageServer** → NPCSvr (10–15 мин) → RunAsDate → жми **RUN**.
- ⚠️ Не запускай компоненты через `Start-Process` из SSH-сессии — умирают при её закрытии. Только schtasks/консоль.
- STOP: `AION-STOP-SERVER.bat` — обратный порядок + шаг [10/10] **удаляет все `*.err`** (логи не растут гигами).

### Шаг 8 — Клиент
См. раздел [Клиенты](#-клиенты).

---

## 💬 Диалоги SQL/ODBC при первом старте компонентов

При первом запуске большинство компонентов показывает `SQL Login` / `L2 ODBC Connection Info`:

| Окно (компонент) | Поле | Формат, который принял компонент |
|---|---|---|
| «SQL Login» — AccountCacheServer | File DB: `aion_accoutdb` | полный путь **БЕЗ** `.dsn` (`C:\...\aion_accoutdb`) + `sa/123` — поле обрезается на ~76 симв, с `.dsn` путь становился 77+ и портился `\u0001`-байтом |
| «SQL Login» — CacheD64 | File DB: `aionworld_new` | так же — **БЕЗ** `.dsn` |
| «L2 ODBC Connection Info» — L2Authd | File DSN: `L2Conn` | полный путь **С** `.dsn` (путь короткий, влез в лимит) |

Правило простое: держи все DSN в `C:\DSN\` — тогда любой путь влезает даже с расширением. Если диалог не принимает файл с расширением — убери `.dsn` на конце.

Во ВСЕХ окнах вводим:
- **Login ID / User**: `sa`
- **Password**: `123`

Пароль сменён с заводского `Wutian520` на единый `123` — прошит во все DSN-файлы, конфиги и скрипты репо. Упоминания `Wutian520` в кит-гайдах/комьюнити-постах — история (заводской пароль оригинальных бэкапов).

- После первого успешного коннекта зашифрованный `connStr` сохраняется в `HKLM\SOFTWARE\NC Soft\AION\<Component>` и больше не спрашивается. Если диалог замучил — удали ключ компонента и введи заново.
- Компонент ↔ его БД: AccountCacheServer→`AionAccountCacheD_rc`(+`AionAccounts`), L2Authd→`AionAccounts`(`L2Conn.dsn`), CacheD64→`_AionWorldNew114_rc`+`AionAccountCacheD_rc`, LogServer64→`Aion_log`(`aiongm.dsn`), Server64→`_AionWorldNew114_rc`.

---

## 🔧 Фиксы: что исправлено и когда применять

**Полный реестр**: [docs/fixes-registry.md](docs/fixes-registry.md) · планы/очередь: [fixes-pending/](fixes-pending/README.md) · сообщество: [docs/fixes-community.md](docs/fixes-community.md)

### Применяются до первого старта (Шаг 4/6)
| Фикс | Зачем | Файл |
|---|---|---|
| `max text repl size=65664` | CacheD64 без него самоубивается | `scripts/sql-tune.ps1` |
| `max server memory=2048` | RAM-бюджет стека | там же |
| Индекс `IX_delete_complete_date` | алерты aion_GetDeletedCharList каждые 5 мин | `scripts/sql/fix-db-*.sql` |
| Linked server `RC-AIONAUTHDB` + БД `aionaccdeldb.del_account` | aion_ProcessWithdrawAccount | там же |
| Заглушка `Log_TblGameServerInfo_UpdateServerstatus` | LogServer64 без неё падает | там же |
| Заглушка `Log_TblGameWorldInfo_UpdateMainStatus`, индекс `IX_user_item_sealed_char_id`, `aion_SetItemMatterOption` | community bugs-лист | `scripts/sql/fix-community-*.sql` |

### Конфиги, исправленные при установке
| Фикс | Детали |
|---|---|
| **CAPTCHA int16**: `43330` → `22206` | Server64 парсит порт как signed int16 → «Captcha −22206». Правится в `MainServer\common.xml` + `CAPTCHAImageServer\config.ini` |
| **ACP=28591** | LogServer64 без него умирает; реестр `Nls\CodePage\ACP` + ребут |
| **AuthD `PAIP_2` 127.0.0.2 → 127.0.0.1** | второй PA-мост L2Authd в конфиге кита вёл в никуда |
| **RunAsDate** | подмена времени 04-06-2020 для Server64 (либо готовый date-bypass патч из #180 — проверен хешем) |
| **Каталог `D:\_AION_log\batchlog`** | LogServer без него «Invalid file path» |

### Уже в binary (ничего делать не надо)
`Server64.exe` в комплекте = date-bypass патч #180 (SHA256 сверён), RunAsDate — страховка.

### Что НЕ деплоить
- EN/EU-клиент как основной (см. Клиенты). Чужие ENIGMA-сборки MainServer (риск бэкдоров). `P7-clear.zip` (вайп БД, только справка).

---

## 🎮 Клиенты

**⚠️ Ловушка №1**: западный клиент (`euro_aion`/`AION Free-to-Play`, EN, билд `7720.0603.x`) логинится **через CEF-портал** — с ним сервер даёт `Session id mismatched (RecvLogin) sessionId:0` и мгновенное «отключен». Это не лечится конфигами — нужен клиент прямого логина.

| Клиент | cc (аргумент `-cc:`) | Вердикт |
|---|---|---|
| **AION_KR 7.7** | 0 | ✅ основной кандидат — та же PTS-семья, что сервер `77.20.0601.15608` |
| aion chs 7.9 | 5 | запасной (CN, но версия новее) |
| aion rus 7.7 (Innova/4game) | 7 | запасной (обход 4game-лаунчера) |
| euro_aion 7.7 / AION Free-to-Play | 2 | ❌ портал-логин, «official portal» |

- Проверка версии после установки: `(Get-Item bin64\Aion.bin).VersionInfo` — хотим семейство `7720.0601.x`.
- Запуск: `bin64\Aion.bin -ip:<IP_сервера> -port:2106 -cc:<см.таблицу> -noauthgg -megaphone -webpetition -ncping -f2p -win10-mouse-fix`. EU-клиент запускается через `AionLauncher.exe`, который читает `launcher.config` (правится там, **порт 2106**, в ките стоял нерабочий 2105).
- Аккаунт создаётся автоматически при первом логине.
- **Где качать клиентов**:
  - Публичный диск кита: <https://disk.360.yandex.ru/client/aa/d_8r46o43ZR7x-Lw> → папка `Clients/aion 7.x` (внутри: AION_KR 7.7, aion chs 7.9, aion rus 7.7/7.9, euro_aion 7.7, AION Free-to-Play 7.2/7.9, aion 7.5) — режим «только просмотр», скачивание через приложение Яндекс 360;
  - Зеркало-торренты: <https://github.com/MrHousek/AionClients> (cc-таблица автора: 0=KR, 2=EURO/F2P, 4=JP, 5=CHA, 7=RUS).

---

## 🩺 Эксплуатация
- **RAM**: лимит SQL 2048 МБ; старт Server64 и NPCSvr не одновременно; 32 ГБ RAM на VM — целевой минимум.
- **Логи**: `*.err` удаляются стоп-батником; живой Server64 пишет в `MainServer\log\2020-06-04.err` (дата RunAsDate!). CacheD пишет ~120 МБ одноразово при загрузке (Strings DB warnings) — норма.
- **Сервисы**: поднимать только через задачи/bat (SSH-`Start-Process` умирает с сессией); после повторных стартов проверять дубли процессов.
- **Watchdog/ночной рестарт NPCSvr** (утечка ~15 ГБ/сессию) — см. [docs/roadmap.md](docs/roadmap.md).

---

## 📁 Структура репо

| Путь | Что |
|---|---|
| [docs/fixes-registry.md](docs/fixes-registry.md) | **Главный реестр фиксов**: задеплоено/ждёт/баги, одним списком |
| [docs/fixes-known-issues.md](docs/fixes-known-issues.md) | Известные баги + что исправлено (по раундам) |
| [docs/roadmap.md](docs/roadmap.md) | Роадмап: этапы восстановления БД, Ghidra-патчи, ops |
| [docs/fields.md](docs/fields.md) | Все поля всех диалогов + launcher.config клиента |
| [docs/errors.md](docs/errors.md) | Ошибка → причина → фикс |
| [docs/server-internals.md](docs/server-internals.md) | **Как работает сервер**: кто к кому обращается, порты, конфиги, заглушки, что не копали |
| [docs/ports.md](docs/ports.md) / [docs/nat-ports.md](docs/nat-ports.md) | Карта портов / проброс за NAT |
| [fixes-pending/](fixes-pending/README.md) | Очередь фиксов по папкам (каждый двигается отдельно) |
| [tools/ragezone-1211744/](tools/ragezone-1211744/README.md) | Скачанные community-фиксы (Server64 #180, LogServer патчи, SQL) с SHA256 |
| `scripts/sql/` | SQL-фиксы (идемпотентные) |
| `scripts/client/AION-CLIENT.bat` | Лаунчер клиента с автопоиском bin64 |
| `scripts/restart-all-services.ps1` | Health-check: поднимает упавшее (schtasks-only), порт-сводка |

## 📜 Источники
- [docs/links.md](docs/links.md) — всё скачиваемое (MS ISO, VC++, SNAC, virtio-win).
- RaGEZONE: [AION7.7pts (Europe) tutorial](https://forum.ragezone.com/threads/aion7-7pts-europe-set-up-tutorial-and-server-download-address.1211744/) (dAIdoom) + [4.6 retail](https://forum.ragezone.com/threads/aion-4-6-retail-server-file-re-post.1197401/).
- Клиенты: диск кита (папка `Clients/aion 7.x`) + GitHub `MrHousek/AionClients` (торренты; cc: 0=KR, 2=EURO/F2P, 5=CHA, 7=RUS).

## 🤝 Вклад
Находки — через PR. Приоритет: отсутствующие хранимые процедуры (сигнатуры из `CacheServer\log\*.err`) и заглушки к ним.

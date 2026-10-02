# 📝 Все поля всех диалогов — что вводить

Все диалоги возникают **один раз** на компонент (после первого успешного коннекта сохраняют connStr в реестр `HKLM\SOFTWARE\NC Soft\AION\<Компонент>`).

## Креды (единые, после нашего тюнинга)

| Параметр | Значение |
|---|---|
| **Login / Login Name** | `sa` |
| **Password** | `123` (мы сменили с заводского `Wutian520`) |
| **SQL Server** | `localhost` / `.` |
| **Инстанс** | `MSSQLSERVER` (по умолчанию) |

## 🗺️ Диалог → поле `File DSN` / `File DB` → что вводить

⚠️ **Обязательно ПОЛНЫЙ ПУТЬ**, и путь должен быть **не длиннее ~76 символов** (иначе `\u0001`-обрезание и `недопустимый файл DSN ""`).

| Окно показывает в File DB | Файл | Полный путь для ввода |
|---|---|---|
| `aion_accoutdb` (опечатка в сборке!) | `aion_accountdb.dsn` | `C:\DSN\aion_accoutdb.dsn` |
| `aion_accountdb` | `aion_accountdb.dsn` | `C:\DSN\aion_accountdb.dsn` |
| `aionworld_new` | `aionworld_new.dsn` | `C:\DSN\aionworld_new.dsn` |
| `aiongm` | `aiongm.dsn` | `C:\DSN\aiongm.dsn` |
| `L2Conn` | `L2Conn.dsn` | `C:\DSN\L2Conn.dsn` |
| `BkPetitionDB` | `BkPetitionDB.dsn` | `C:\DSN\BkPetitionDB.dsn` |
| `PetitionDB` | `PetitionDB.dsn` | `C:\DSN\PetitionDB.dsn` |

Все файлы уже лежат в `C:\DSN\` (после `setup-odbc.ps1`) — 24-26 символов, запас огромный.

## 📊 Соответствие окно ↔ база SQL

| Окно (File DB) | База данных | Кто окно открывает |
|---|---|---|
| `aion_accoutdb` / `aion_accountdb` | `AionAccountCacheD_rc` | **AccountCacheServer** |
| `aionworld_new` | `_AionWorldNew114_rc` | **CacheD64** |
| `L2Conn` | `AionAccounts` | **L2Authd** (AuthD) |
| `aiongm` | `Aion_log` | **LogServer64** |
| `BkPetitionDB` | `BkPetitionDB` | сервис петиций |
| `PetitionDB` | `PetitionDB` | сервис петиций |

## 🖥️ Диалоги по компонентам (порядок появления при запуске)

1. **AccountCacheServer** — окно «SQL Login» `File DB: aion_accoutdb` → вводишь `sa`/`123` → OK
2. **L2Authd (AuthD)** — окно «L2 ODBC Connection Info» `File DSN: L2Conn` → `C:\DSN\L2Conn.dsn` + `sa`/`123` → OK
3. **AuthGateD** — обычно **без окна** (только подключение к AuthD на `127.0.0.1:2110`)
4. **LogServer64** — окно `File DB: aiongm` → `C:\DSN\aiongm.dsn` + `sa`/`123` → OK
5. **CacheD64** — окно «SQL Login» `File DB: aionworld_new` → `C:\DSN\aionworld_new.dsn` + `sa`/`123` → OK
6. **NPCSvr64** — грузится без окна (подключается к `_AionWorldNew114_rc` через уже сохранённый коннект)
7. **Server64** — **без окна**, но запуск **только через RunAsDate** (время = 2020)

## 🎮 Запуск клиента (с десктопа)

```cmd
cd "C:\Path\To\euro_aion_7.7\client"
start bin64\aion.bin -ip:192.168.0.125 -port:2106 -multithread -cc:2 -noauthgg -charnamemenu -loginex -pwd16 -megaphone -ingamebrowser -ncping
```

| Параметр | Зачем |
|---|---|
| `-ip:192.168.0.125` | IP твоего сервера-VM (AuthGateD слушает 2106) |
| `-port:2106` | Порт AuthGateD |
| `-cc:2` | **country code 2 = Europe** (важно! иначе «regional code is not compatible») |
| `-noauthgg` | Отключить GameGuard (в конфиге тоже useGameGuard: false) |
| `-pwd16` | Пароль в MD5-16 |
| `-loginex` | Расширенный логин |
| `-charnamemenu` | Меню выбора имени персонажа |
| `-ingamebrowser` | Встроенный браузер в игре |
| `-ncping` | NCPing-протокол |

Аккаунт **регистрируется автоматически** при первом входе (упрощённая авторизация сборки).

## 🗃️ RunAsDate конфиг (уже на месте)

Файл `MainServer\runasdate-x64\RunAsDate.cfg`:
```
Filename=D:\AION_LIVE_SERVER\MainServer\Server64.exe
DateTime=04-06-2020 16:28:23   (дата билда Server64.exe)
DateTimeMode=1, AddTimeUnit=4, AddTimeValue=-1
```
Жмёшь **RUN** — Server64 запускается с подменённым временем. Без этого — `invalid system time` → отказ старта.

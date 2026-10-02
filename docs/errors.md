# 🐛 Все встреченные ошибки → причина → фикс

Список собран по ходу реальной установки на Windows Server 2022 RU + SQL Server 2022 Developer. Каждая ошибка подтверждена логами.

## 1. `HY000 / pfNativeError=556 / недопустимый файл DSN ""` (в AccountCacheServer, CacheD64, L2Authd)

**Причины (находились последовательно):**
1. **Нет драйвера `SQL Server Native Client 11.0`** — оригинальные DSN-файлы указывают на него, а он не установлен в системе → обёртка NC Soft репортит как пустой DSN.
2. **FILEDSN-имя без пути не резолвится** — Windows 2022 ODBC Driver Manager не ищет `.dsn` по имени **ни в cwd, ни в Common Files, ни в DefaultSourceDir** (проверено нативным тестом через `SQLDriverConnect` из mingw-собранного `dsntest.exe`). Работает только **полный путь** или `DSN=<имя>` (System DSN из реестра).
3. **Поле File DB в диалоге обрезается** на ~76 символов: введённый путь `C:\Program Files\Common Files\ODBC\Data Sources\aion_accoutdb.dsn` (77 сим.) обрезался до `...aion_accoutdb.ds\u0001` → Manager не открыл файл. Требуется **короткий путь** `C:\DSN\<name>.dsn`.

**Фикс (3 действия):**
1. Установить SNAC11: `msiexec /i sqlncli_x64.msi IACCEPTSQLNCLILICENSETERMS=YES /qn` (регистр важен! параметр называется именно `IACCEPTSQLNCLILICENSETERMS`, не `IACCEPTMSNCLILICENSETERMS`).
2. Все DSN-файлы скопировать в `C:\DSN\` (короткий путь) + в Common Files + в каждую папку сервера.
3. В диалогах вводить `C:\DSN\<имя>.dsn` + `sa`/`123`.

## 2. `CacheD64: Invalid 'Max text repl size'(1536) value 65536 (65664 or -1 expected)` → graceful shutdown через 5 сек

**Причина**: CacheD64 требует конфиг SQL Server `max text repl size = 65664` (нестандартное значение!).
**Фикс**:
```sql
EXEC sys.sp_configure 'show advanced options',1; RECONFIGURE;
EXEC sys.sp_configure 'max text repl size (B)',65664; RECONFIGURE;
```
Скрипт: [scripts/sql-tune.ps1](../scripts/sql-tune.ps1).

## 3. `Server64: invalid system time` → отказ запуска

**Причина**: сборка рассчитана на запуск Server64.exe **через RunAsDate** с подменой системного времени на дату билда (`04-06-2020 16:28:23`).
**Фикс**: всегда запускать через `MainServer\runasdate-x64\RunAsDate.exe` → RUN. Конфиг уже сохранён (`RunAsDate.cfg`).

## 4. `Server64: Failed to load L10N NpcID data: map\XML\Europe\npcs_test.xml` → shutdown

**Причина**: Server64 ищет L10N-подмножества XML в папке **`Map\XML\Europe`** (region=2 = EU). В комплекте её нет — только `china`.
**Фикс**: создать `Europe` и **hardlink'и** на все файлы из `Map\XML` (605 файлов, ~0 байт на диске):
```powershell
# из scripts/config-patch.ps1:
Get-ChildItem 'D:\AION_LIVE_SERVER\Map\XML' -File | ForEach-Object {
  New-Item -ItemType HardLink -Path "D:\AION_LIVE_SERVER\Map\XML\Europe\$($_.Name)" -Target $_.FullName
}
```
Жёсткие копии (в контексте NTFS hardlink) — нулевая цена диска, полная функциональность.

## 5. `LogServer64: Invalid current ANSI code-page for system, please set code-page to 28591` → падение при старте

**Причина**: LogServer требует ANSI code page = **28591 (ISO-8859-1)** — как у западной Windows. На русской (ACP=1251) он не стартует.
**Фикс** (реестр, вступает после **ребута VM**):
```powershell
Set-ItemProperty 'HKLM:\SYSTEM\CurrentControlSet\Control\Nls\CodePage' -Name ACP -Value 28591
Set-ItemProperty 'HKLM:\SYSTEM\CurrentControlSet\Control\Nls\CodePage' -Name OEMCP -Value 850
```
⚠️ После этого русский текст в ANSI-приложениях может криво отображаться (обратимо).

## 6. SQL Server 2017 RTM setup на WS2022: `Ошибка при создании документа XML` (exit 0x84C40013, facility 1201)

**Причина**: известный баг RTM-носителя 2017 (14.00.1000.169) на Windows Server 2022.
**Фикс**: ставить **SQL Server 2022** (restore 2017-бэкапов в 2022 работает без проблем — совместимость вверх).
Параметр называется **`INSTALLSQLDATADIR`**, НЕ `INSTALLSQLDIR` (в 2017/2022 имя параметра сменили, ошибка «not recognized» если юзаешь старое).

## 7. SQL setup из SSH-канала: `CryptographicException` в `SqlSecureString.WriteXml` (DPAPI)

**Причина**: SQL setup сериализует защищённые строки через DPAPI, который **не работает из SSH-сессии** (нет интерактивного профиля).
**Фикс**: запускать setup через **`schtasks /Create /IT` + `/Run`** (интерактивная сессия), либо вручную в GUI.

## 8. `NTFS-компрессия на D:\` недопустима (`The directory D:\SQL\... is under a compressed drive`)

**Причина**: SQL Server не работает с сжатыми NTFS-папками. При создании VM Proxmox создал диск D: с компрессией NTFS.
**Фикс**:
```cmd
compact /U /S:"D:\" /F /I /Q
```

## 9. Русская Windows: `BUILTIN\Administrators` не существует

**Причина**: на RU-редакциях встроенная группа называется **«Администраторы»** (кириллицей), а не `Administrators` — все команды `icacls`/`grant` с `BUILTIN\Administrators` падают («Сопоставление... не было произведено»).
**Фикс**: юзать **SID** (не локализуется):
```cmd
icacls <file> /inheritance:r /grant "*S-1-5-18:F" /grant "*S-1-5-32-544:F"
```
(`S-1-5-32-544` = Администраторы, `S-1-5-18` = SYSTEM.)

То же в SQL: `SQLSYSADMINACCOUNTS="NT AUTHORITY\SYSTEM"` (безопасный вариант) либо через SID.

## 10. `MSSQLSERVER` не стартует после ребута

**Причина**: служба после установки не в автозапуске (стартует от Setup, но стартовый тип Manual).
**Фикс**:
```powershell
Set-Service MSSQLSERVER -StartupType Automatic
Start-Service MSSQLSERVER
```

## 11. Шифрованный blob `connStr` в реестре (512 байт)

**Причина**: после первого успешного логина компонент записывает зашифрованную `connStr` в `HKLM\SOFTWARE\NC Soft\AION\<Component>`. Если она повреждена (например, при прошлом падении), компонент не может её прочитать → диалог снова.
**Фикс**: удалить ключ — компонент переспросит:
```cmd
reg export "HKLM\SOFTWARE\NC Soft\AION\CacheD" backup.reg /y
reg delete "HKLM\SOFTWARE\NC Soft\AION\CacheD" /f
```
Формат — XOR-шифрование (не DPAPI), расшифровать нельзя. После ввода creds в диалоге blob перезапишется.

## 12. `aion_accoutdb` vs `aion_accountdb` (опечатка в сборке!)

**Причина**: AccountCacheServer ищет файл DSN с опечаткой — `aion_accoutdb` (без «c»). Настоящий файл называется `aion_accountdb.dsn`.
**Фикс**: `setup-odbc.ps1` создаёт алиас `aion_accoutdb.dsn` (копия `aion_accountdb.dsn`) во всех нужных папках.

## 13. `System.BadImageFormatException` / `Add-Type` сработал некорректно

**Причина**: не всегда Add-Type принимает PowerShell heredoc с C# кодом (спорные кавычки/подстановки).
**Фикс**: писать `.cs`-файл в C:\Temp через `[IO.File]::WriteAllText($path, $code, [Text.Encoding]::ASCII)` и подключать через `Add-Type -Path $cs`.

## 14. Из треда RaGEZONE (фьючерс-хинты)

- `Game server not registered with authentication server 6` — auth-сервер не зарегистрировал game-server: бэкап БД требует обновления (в обновлённой сборке с треда это пофикшено).
- `The client's regional code is not compatible with the game server` — `country` в конфигах не совпадает с `cc:` в клиенте (см. [fields.md](fields.md)).
- Отсутствующие хранимые процедуры (`aion_GetItemCollectionLevelList`, `aion_LoadFameReduceTime`, `aion_SetCharInfo_20160818`…) — **норма для этой сборки** (DB incomplete ~85%, по автору треда).
- `питомцы/спрайты`, `сохранение смены класса на 9 lvl`, `карты с песочными часами` — тоже известные баги, в обновлённой сборке с треда пофикшены частично.

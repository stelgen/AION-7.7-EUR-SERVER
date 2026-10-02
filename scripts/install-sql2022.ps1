# ================================================================
# AION 7.7 PTS EU — SQL Server 2022 Developer (тихая установка)
# Обход бага SQL 2017 RTM на Windows Server 2022 (Ошибка при создании XML, 0x84C40013).
# sa / 123, Mixed mode, TCP+NP, данные на D:\SQL, коллация Latin1_General_CI_AS.
# ЗАПУСКАТЬ ТОЛЬКО В ИНТЕРАКТИВНОЙ СЕССИИ (не через SSH) — DPAPI не работает в SSH!
# ================================================================
$OutputEncoding = [Console]::OutputEncoding = [Text.Encoding]::UTF8
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$isoUrl  = 'https://archive.org/download/sqlserver-2022-x-64-eng/SQLServer2022-x64-ENG.iso'
$isoPath = 'D:\ISO\SQLServer2022-x64-ENG.iso'
$sqlRoot = 'D:\SQL'
$iniPath = 'D:\ISO\SQL2022.ini'

New-Item -ItemType Directory -Force -Path 'D:\ISO', $sqlRoot | Out-Null
# NTFS-компрессия на D:\ недопустима для SQL Server!
compact /U /S:"D:\" /F /I /Q 2>&1 | Out-Null

# 1. Скачать ISO
if (!(Test-Path $isoPath)) {
  Write-Host "=== Скачивание SQL Server 2022 Developer ISO (1.1 ГБ) ===" -ForegroundColor Cyan
  & C:\Windows\System32\curl.exe -sSL --retry 3 -o $isoPath $isoUrl
  Write-Host ("ISO: " + [math]::Round((Get-Item $isoPath).Length/1MB,0) + " MB")
}

# 2. Смонтировать
$dismount = $null
$iso = Get-DiskImage -ImagePath $isoPath -EA SilentlyContinue
if ($iso -and $iso.Attached) { $drv = ($iso | Get-Volume).DriveLetter }
else { $drv = ((Mount-DiskImage -ImagePath $isoPath -PassThru) | Get-Volume).DriveLetter }
Write-Host "ISO mounted at ${drv}:"

# 3. Конфигурационный файл
$ini = @"
[OPTIONS]
Action="Install"
QUIET="True"
ENU="True"
FEATURES=SQL
UpdateEnabled=0
IACCEPTSQLSERVERLICENSETERMS="True"
INSTALLSQLDATADIR="D:\SQL"
INSTANCENAME="MSSQLSERVER"
INSTANCEID="MSSQLSERVER"
SQLSVCSTARTUPTYPE=Automatic
SQLSVCACCOUNT="NT AUTHORITY\NETWORK SERVICE"
SQLSYSADMINACCOUNTS="NT AUTHORITY\SYSTEM"
SECURITYMODE="SQL"
SAPWD="123"
SQLCOLLATION="Latin1_General_CI_AS"
TCPENABLED=1
NPENABLED=1
"@
[System.IO.File]::WriteAllText($iniPath, $ini, [Text.Encoding]::ASCII)
Write-Host "INI: $iniPath"

# 4. Запуск setup (критично: НЕ из SSH — только интерактивная сессия или schtasks /IT!)
Write-Host "=== Запуск setup.exe (10-25 мин) ===" -ForegroundColor Cyan
$p = Start-Process "$($drv):\setup.exe" -ArgumentList "/ConfigurationFile=$iniPath" -WindowStyle Hidden -PassThru
Write-Host "SETUP-PID: $($p.Id)"
Write-Host "Логи: C:\Program Files\Microsoft SQL Server\160\Setup Bootstrap\Log\"

# 5. Ожидание завершения
$p.WaitForExit()
Write-Host "SETUP-RC: $($p.ExitCode)"
if ($p.ExitCode -eq 0) {
  Write-Host "SQL Server 2022 установлен!" -ForegroundColor Green
  # Автостарт + max memory
  Set-Service MSSQLSERVER -StartupType Automatic
  Start-Service MSSQLSERVER -EA SilentlyContinue
  Start-Sleep 5
  $cn = New-Object System.Data.SqlClient.SqlConnection
  $cn.ConnectionString = "Server=localhost,1433;Database=master;User Id=sa;Password=123;Connect Timeout=15"
  $cn.Open()
  $c = $cn.CreateCommand()
  $c.CommandText = "EXEC sys.sp_configure 'show advanced options',1; RECONFIGURE;"
  $c.ExecuteNonQuery() | Out-Null
  # max server memory: 4096 МБ на 16 ГБ RAM → 12288 МБ на 32 ГБ (подставь своё)
  $c.CommandText = "EXEC sys.sp_configure 'max server memory (MB)',12288; RECONFIGURE;"
  $c.ExecuteNonQuery() | Out-Null
  Write-Host "max server memory = 12288 МБ (настроить под свою RAM)"
  $cn.Close()
} else {
  Write-Host "SETUP FAILED — смотри Summary.txt" -ForegroundColor Red
  $sm = Get-ChildItem 'C:\Program Files\Microsoft SQL Server\160\Setup Bootstrap\Log' -Directory | Sort-Object Name -Descending | Select-Object -First 1
  Get-Content (Join-Path $sm.FullName 'Summary.txt') -Encoding UTF8 -EA SilentlyContinue | Select-Object -Last 30
}

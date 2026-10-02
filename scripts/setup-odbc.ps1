# ================================================================
# AION 7.7 PTS EU — Раскладка DSN + creds
# Все DSN-файлы (оригинальные, с SNAC11) кладутся в:
#   1) C:\DSN\                            — короткий путь для диалогов (поле File DB ограничено ~76 символами!)
#   2) C:\Program Files\Common Files\ODBC\Data Sources\ — ODBC default dir
#   3) D:\AION_LIVE_SERVER\<Компонент>\   — рабочие папки процессов
# Плюс: алиас aion_accoutdb.dsn (опечатка в сборке), creds sa/123 вшиты в файлы.
# ================================================================
$OutputEncoding = [Console]::OutputEncoding = [Text.Encoding]::UTF8
$enc = [Text.Encoding]::GetEncoding(1251)

$src = 'D:\AION_LIVE_SERVER\Database\ODBC'
if (!(Test-Path $src)) { Write-Host "НЕ НАЙДЕНО $src — сначала распакуй AION_LIVE_SERVER!" -ForegroundColor Red; exit 1 }

$dstD = 'C:\DSN'
$dstC = 'C:\Program Files\Common Files\ODBC\Data Sources'
New-Item -ItemType Directory -Force -Path $dstD, $dstC | Out-Null

# 1. Копируем все оригинальные DSN в C:\DSN\ и Common Files
Get-ChildItem "$src\*.dsn" | ForEach-Object {
  Copy-Item $_.FullName $dstD -Force
  Copy-Item $_.FullName $dstC -Force
}

# 2. Алиас aion_accoutdb.dsn (опечатка в сборке AccountCacheServer!)
if (!(Test-Path "$dstD\aion_accoutdb.dsn")) {
  Copy-Item "$dstD\aion_accountdb.dsn" "$dstD\aion_accoutdb.dsn" -Force
  Copy-Item "$dstD\aion_accountdb.dsn" "$dstC\aion_accoutdb.dsn" -Force
}

# 3. Вшить creds (sa/123) во все DSN в C:\DSN и Common Files
$dbMap = @{
  'aion_accountdb.dsn' = 'AionAccountCacheD_rc'
  'aion_accoutdb.dsn'  = 'AionAccountCacheD_rc'
  'aionworld_new.dsn'  = '_AionWorldNew114_rc'
  'aiongm.dsn'         = 'Aion_log'
  'L2Conn.dsn'         = 'AionAccounts'
  'BkPetitionDB.dsn'   = 'BkPetitionDB'
  'PetitionDB.dsn'     = 'PetitionDB'
}
foreach ($n in $dbMap.Keys) {
  foreach ($dir in $dstD, $dstC) {
    $p = "$dir\$n"
    if (!(Test-Path $p)) { continue }
    $t = [IO.File]::ReadAllText($p, $enc)
    $db = $dbMap[$n]
    # полностью переписываем содержимое: SNAC11 + creds + нужная БД
    $content = "[ODBC]`r`nDRIVER=SQL Server Native Client 11.0`r`nUID=sa`r`nPWD=123`r`nSERVER=(local)`r`nDATABASE=$db"
    [IO.File]::WriteAllText($p, $content, $enc)
  }
}

# 4. Раскладываем по всем серверным папкам (для FILEDSN в cwd)
$folders = 'AccountCacheServer','AuthD','AuthGateD','CacheServer','CAPTCHAImageServer','ICServer','LogServer','MainServer','NPCServer','NPRelayServer','RankingServer'
foreach ($d in $folders) {
  $dir = "D:\AION_LIVE_SERVER\$d"
  if (Test-Path $dir) {
    Get-ChildItem "$dstD\*.dsn" | ForEach-Object { Copy-Item $_.FullName $dir -Force }
  }
}

# 5. Тест всех DSN через .NET OdbcConnection
Write-Host ""
Write-Host "=== Тест всех DSN (FILEDSN=<C:\DSN\...>) ===" -ForegroundColor Cyan
foreach ($n in $dbMap.Keys) {
  try {
    $conn = New-Object System.Data.Odbc.OdbcConnection
    $conn.ConnectionString = "FILEDSN=$dstD\$n"
    $conn.Open()
    $c = $conn.CreateCommand(); $c.CommandText = "SELECT DB_NAME()"; $r = $c.ExecuteReader(); $r.Read()
    Write-Host "OK    $n -> $($r[0])" -ForegroundColor Green
    $r.Close(); $conn.Close()
  } catch {
    $m = ($_.Exception.Message -replace "`r`n", ' ')
    if ($m.Length -gt 90) { $m = $m.Substring(0, 90) }
    Write-Host "FAIL  $n : $m" -ForegroundColor Red
  }
}

Write-Host ""
Write-Host "=== Регистрация SYS-DSN (fallback для DSN=<имя>) ===" -ForegroundColor Cyan
# SNAC11 блокирует UID/PWD в реестре (security) — регистрируем только Server+Database+Trusted
foreach ($n in 'aion_accountdb','aion_accoutdb','aionworld_new','aiongm','L2Conn','BkPetitionDB','PetitionDB') {
  Remove-OdbcDsn -Name $n -DsnType System -EA SilentlyContinue
  $db = $dbMap["$n.dsn"]
  Add-OdbcDsn -Name $n -DriverName "SQL Server Native Client 11.0" -DsnType System -SetPropertyValue @("Server=localhost","Database=$db","Trusted_Connection=Yes") -EA SilentlyContinue
}
Get-OdbcDsn -DsnType System | Format-Table Name, DriverName -AutoSize

Write-Host ""
Write-Host "=== DSN SETUP DONE ( creds sa/123, пути C:\DSN\, оригиналы в CommonFiles, алиас для опечатки создан) ===" -ForegroundColor Green

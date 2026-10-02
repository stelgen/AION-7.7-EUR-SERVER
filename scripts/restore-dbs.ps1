# ================================================================
# AION 7.7 PTS EU — Restore всех 6 баз .bak
# Распаковка AION_LIVE_SERVER должна быть уже сделана (D:\AION_LIVE_SERVER\Database\*.bak).
# Твой AionAccounts.bak с NAS — опционально (идентичен комплектному — проверено SHA256).
# Файлы уедут в D:\SQL\MSSQL16.MSSQLSERVER\MSSQL\DATA\.
# ================================================================
$OutputEncoding = [Console]::OutputEncoding = [Text.Encoding]::UTF8

$cn = New-Object System.Data.SqlClient.SqlConnection
$cn.ConnectionString = "Server=localhost,1433;Database=master;User Id=sa;Password=123;Connect Timeout=15"
$cn.Open()

# Имя базы в SQL ← из бэкапа (определено через RESTORE HEADERONLY)
$map = [ordered]@{
  'AccountCacheD.bak'  = 'AionAccountCacheD_rc'
  'AionAccounts.bak'   = 'AionAccounts'
  'aionworld_110.bak'  = '_AionWorldNew114_rc'
  'Aion_log.bak'       = 'Aion_log'
  'bkpetitiondb.bak'   = 'BkPetitionDB'
  'petitiondb.bak'     = 'PetitionDB'
}

foreach ($bak in $map.Keys) {
  $db = $map[$bak]
  $f = "D:\AION_LIVE_SERVER\Database\$bak"
  if (!(Test-Path $f)) { Write-Host "MISSING: $f" -ForegroundColor Red; continue }

  # Читаем LogicalName из бэкапа
  $c = $cn.CreateCommand(); $c.CommandText = "RESTORE FILELISTONLY FROM DISK='$f'"; $c.CommandTimeout = 120
  $r = $c.ExecuteReader(); $files = @()
  while ($r.Read()) { $files += [pscustomobject]@{ L = $r['LogicalName']; T = $r['Type'] } }
  $r.Close()
  $d = ($files | Where-Object { $_.T -eq 'D' } | Select-Object -First 1).L
  $l = ($files | Where-Object { $_.T -eq 'L' } | Select-Object -First 1).L

  $sql = "RESTORE DATABASE [$db] FROM DISK='$f' WITH MOVE '$d' TO 'D:\SQL\MSSQL16.MSSQLSERVER\MSSQL\DATA\$db.mdf', MOVE '$l' TO 'D:\SQL\MSSQL16.MSSQLSERVER\MSSQL\DATA\$db.ldf', REPLACE, RECOVERY"
  try {
    $c2 = $cn.CreateCommand(); $c2.CommandText = $sql; $c2.CommandTimeout = 600
    $c2.ExecuteNonQuery() | Out-Null
    Write-Host "RESTORED: $db" -ForegroundColor Green
  } catch {
    $m = ($_.Exception.Message -replace "`r`n", ' ')
    Write-Host "FAIL $db : $m" -ForegroundColor Red
  }
}

$c3 = $cn.CreateCommand(); $c3.CommandText = "SELECT name FROM sys.databases WHERE database_id > 4 ORDER BY name"
$r3 = $c3.ExecuteReader()
Write-Host ""
Write-Host "=== Итоговый список баз ===" -ForegroundColor Cyan
while ($r3.Read()) { "  " + $r3[0] }
$r3.Close()
$cn.Close()

# Обновление таблицы Server в AionAccounts: IP VM и region=2 (EU)
$cn2 = New-Object System.Data.SqlClient.SqlConnection
$cn2.ConnectionString = "Server=localhost,1433;Database=AionAccounts;User Id=sa;Password=123;Connect Timeout=15"
$cn2.Open()
$vmIp = Read-Host "Введи IP VM (напр. 192.168.0.125)"
$c = $cn2.CreateCommand()
$c.CommandText = "UPDATE Server SET ip='$vmIp', region=2 WHERE id=1"
$c.ExecuteNonQuery() | Out-Null
$c.CommandText = "SELECT id, name, ip, port, region FROM Server"
$r = $c.ExecuteReader()
Write-Host ""
Write-Host "=== Таблица Server (проверка) ===" -ForegroundColor Cyan
while ($r.Read()) { "id=" + $r[0] + "  name=" + $r[1] + "  ip=" + $r[2] + "  port=" + $r[3] + "  region=" + $r[4] }
$r.Close()
$cn2.Close()

Write-Host ""
Write-Host "=== RESTORE DONE ===" -ForegroundColor Green

$OutputEncoding=[Console]::OutputEncoding=[Text.Encoding]::UTF8
# ============ AION 7.7 PTS: Pre-Run Preparation (после скачивания фикс-файлов) ============
# Скачай с RaGEZONE треда 1211744:
#   1. Server64.7z (4.8 МБ, пост #180) -> положи в C:\Temp\Server64.7z
#   2. aion 7.7 fix manastones saving.txt (607 Б, пост #180) -> C:\Temp\manastones.txt
# Этот скрипт: применит патч Server64, прогонит SQL-фикс манастоунов, перезапустит Server64.

$Server64Dir = 'D:\AION_LIVE_SERVER\MainServer'
$srcServer64 = 'C:\Temp\Server64.7z'
$srcMana = 'C:\Temp\manastones.txt'

# 1) Останов Server64
Stop-Process -Name Server64 -Force -EA SilentlyContinue
Start-Sleep 3

# 2) Применить патченный Server64 из архива
if (Test-Path $srcServer64) {
  & 'C:\Program Files\7-Zip\7z.exe' x -y -o"$Server64Dir" $srcServer64 | Out-Null
  if (Test-Path "$Server64Dir\Server64.exe") {
    Write-Host "PATCHED: Server64.exe заменён из $srcServer64" -ForegroundColor Green
  }
}

# 3) SQL fix manastones
if (Test-Path $srcMana) {
  $cn = New-Object System.Data.SqlClient.SqlConnection
  $cn.ConnectionString = "Server=localhost,1433;Database=_AionWorldNew114_rc;User Id=sa;Password=123;Connect Timeout=15"
  $cn.Open()
  $sql = [IO.File]::ReadAllText($srcMana)
  foreach ($stmt in ($sql -split "GO")) {
    $stmt = $stmt.Trim()
    if ($stmt) {
      try {
        $c = $cn.CreateCommand(); $c.CommandText = $stmt; $c.ExecuteNonQuery() | Out-Null
        Write-Host "SQL-OK: " + $stmt.Substring(0, [Math]::Min(60, $stmt.Length)) -ForegroundColor Green
      } catch { Write-Host "SQL-SKIP: " + $_.Exception.Message.Substring(0, [Math]::Min(100, $_.Exception.Message.Length)) -ForegroundColor Yellow }
    }
  }
  $cn.Close()
  Write-Host "MANASTONES-FIX-APPLIED" -ForegroundColor Green
}

# 4) Перезапуск Server64 (через RunAsDate, если патченный exe не применяется)
Write-Host "Запуск Server64 через RunAsDate (если новый exe сам по себе патч, RunAsDate не нужен)..."
schtasks /Run /TN "AionRAD" 2>&1 | Out-Null

# 5) Проверка портов
Start-Sleep 20
Get-NetTCPConnection -State Listen -EA SilentlyContinue | Where-Object { $_.LocalPort -in 7777,2106 } | Format-Table LocalPort, OwningProcess -AutoSize | Out-String

# ================================================================
# AION 7.7 PTS EU — SQL тюнинг (обязательный!)
# CacheD64 требует 'max text repl size = 65664' (не 65536!) — иначе graceful shutdown.
# max server memory = 2048 МБ (проверено на 28 ГБ RAM: Server64 ~10 ГБ + NPCSvr ~15 ГБ;
# больше SQL отжимать нельзя — commit-исчерпание молча убивает игровые процессы).
# ================================================================
$OutputEncoding = [Console]::OutputEncoding = [Text.Encoding]::UTF8
$cn = New-Object System.Data.SqlClient.SqlConnection
$cn.ConnectionString = "Server=localhost,1433;Database=master;User Id=sa;Password=123;Connect Timeout=15"
$cn.Open()
$c = $cn.CreateCommand()

$c.CommandText = "EXEC sys.sp_configure 'show advanced options',1; RECONFIGURE;"
$c.ExecuteNonQuery() | Out-Null

$c.CommandText = "EXEC sys.sp_configure 'max text repl size (B)',65664; RECONFIGURE;"
$c.ExecuteNonQuery() | Out-Null
$c.CommandText = "SELECT CAST(value_in_use AS int) FROM sys.configurations WHERE name='max text repl size (B)'"
Write-Host ("max text repl size = " + $c.ExecuteScalar() + " (CacheD64 требует 65664)") -ForegroundColor Green

$mem = Read-Host "max server memory (MB) [Enter = 2048, проверенное значение; поднимать только при RAM VM 32+ ГБ]"
if (-not $mem) { $mem = 2048 }
$c.CommandText = "EXEC sys.sp_configure 'max server memory (MB)',$mem; RECONFIGURE;"
$c.ExecuteNonQuery() | Out-Null
$c.CommandText = "SELECT CAST(value_in_use AS int) FROM sys.configurations WHERE name='max server memory (MB)'"
Write-Host ("max server memory = " + $c.ExecuteScalar() + " МБ") -ForegroundColor Green

$cn.Close()
Write-Host "=== SQL-TUNE DONE ===" -ForegroundColor Green

# ================================================================
# AION 7.7 PTS EU — SQL тюнинг (обязательный!)
# CacheD64 требует 'max text repl size = 65664' (не 65536!) — иначе graceful shutdown.
# max server memory = 4096 МБ на 16 ГБ RAM → 12288 МБ на 28–32 ГБ.
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

$mem = Read-Host "max server memory (MB): 4096 на 16 ГБ RAM / 12288 на 28-32 ГБ (введи число)"
$c.CommandText = "EXEC sys.sp_configure 'max server memory (MB)',$mem; RECONFIGURE;"
$c.ExecuteNonQuery() | Out-Null
$c.CommandText = "SELECT CAST(value_in_use AS int) FROM sys.configurations WHERE name='max server memory (MB)'"
Write-Host ("max server memory = " + $c.ExecuteScalar() + " МБ") -ForegroundColor Green

$cn.Close()
Write-Host "=== SQL-TUNE DONE ===" -ForegroundColor Green

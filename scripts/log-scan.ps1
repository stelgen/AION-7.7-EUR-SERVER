$OutputEncoding=[Console]::OutputEncoding=[Text.Encoding]::UTF8
# ============================================================
# Полный скан логов AION 7.7 PTS EU — собирает ВСЕ ошибки
# по компонентам в один отчёт. Безопасен: только чтение.
# Результат: C:\Temp\log_scan_report.txt
# ============================================================
$A='D:\AION_LIVE_SERVER'
$out='C:\Temp\log_scan_report.txt'
$lines=@()
$lines+="===== AION 7.7 PTS EU — LOG SCAN REPORT ====="
$lines+="Generated: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
$lines+=""

$procs=@{
 'MainServer'='Server64'
 'NPCServer'='NPCSvr64'
 'CacheServer'='CacheD64'
 'AccountCacheServer'='AccountCacheServer'
 'AuthD'='L2Authd'
 'AuthGateD'='AuthGateD'
 'LogServer'='LogServer64'
 'ICServer'='ICServer'
 'CAPTCHAImageServer'='CAPTCHAImageServer'
}
foreach($comp in $procs.Keys){
  $dir="$A\$comp\log"
  if(!(Test-Path $dir)){ continue }
  $lines+="`n=============================================="
  $lines+="### COMPONENT: $comp (exe=$($procs[$comp]))"
  $lines+="=============================================="
  # свежие .err и .log файлы (последние 24ч)
  $files=Get-ChildItem $dir -File -EA SilentlyContinue|Where-Object{$_.LastWriteTime -gt (Get-Date).AddHours(-24) -and $_.Extension -in '.err','.log'}
  foreach($f in $files){
    $lines+="`n--- $($f.Name) (size=$([math]::Round($f.Length/1KB,1))KB) ---"
    # последние 50 строк
    Get-Content $f.FullName -EA SilentlyContinue|Select -Last 50|ForEach-Object{ $lines+=$_.Line }
  }
  # файлы логов вне log-папки (некоторые пишут рядом с exe)
  $files2=Get-ChildItem "$A\$comp" -File -EA SilentlyContinue|Where-Object{$_.LastWriteTime -gt (Get-Date).AddHours(-24) -and $_.Extension -in '.err','.log'}
  foreach($f in $files2){
    $lines+="`n--- ROOT: $($f.Name) (size=$([math]::Round($f.Length/1KB,1))KB) ---"
    Get-Content $f.FullName -EA SilentlyContinue|Select -Last 30|ForEach-Object{ $lines+=$_.Line }
  }
}
# Общие сводки ошибок по паттернам
$lines+="`n`n=============================================="
$lines+="### ERROR PATTERNS (агрегация)"
$lines+="=============================================="
$all=($lines -join "`n")
$patterns=@(
 @{n='MISSING PROCEDURE'; p="找不到存储过程 '(\w+)'"},
 @{n='INVALID PROC PARAMS'; p="为过程或函数 '(\w+)'"},
 @{n='MISSING TABLE'; p="对象名 '(\S+)' 无效"},
 @{n='MISSING INDEX'; p="索引 '(\w+)'"}
)
foreach($p in $patterns){
  $m=[regex]::Matches($all,$p.p)|ForEach-Object{ $_.Groups[1].Value }|Select -Unique
  if($m){
    $lines+="`n[$($p.n)] — найдено $($m.Count) уникальных имён:"
    $m|ForEach-Object{ $lines+="  - $_" }
  }
}
# NPCSvr memory_summary
$lines+="`n`n=============================================="
$lines+="### NPCServer MEMORY SUMMARY"
$lines+="=============================================="
$ms=Get-ChildItem "$A\NPCServer\log\*.memory_summary" -EA SilentlyContinue|Sort LastWriteTime -Desc|Select -First 1
if($ms){
  $lines+="file: $($ms.Name)"
  Get-Content $ms.FullName|Select -Last 20|ForEach-Object{ $lines+=$_.Line }
}
$lines|Set-Content $out -Encoding UTF8
Write-Host "REPORT: $out ($(Get-Item $out).Length bytes)"

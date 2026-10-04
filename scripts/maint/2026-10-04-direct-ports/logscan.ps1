$ErrorActionPreference = 'SilentlyContinue'
function Tail($p, $n) {
    if (Test-Path $p) { Write-Output ('=== ' + $p + ' (tail ' + $n + ')'); Get-Content $p -Tail $n | ForEach-Object { Write-Output ('  ' + $_) } }
    else { Write-Output ('=== ' + $p + ' : MISSING') }
}
function FreshErr($dir) {
    $f = Get-ChildItem $dir -Filter '*.err' -ErrorAction SilentlyContinue | Sort-Object LastWriteTime | Select-Object -Last 1
    if ($f) { Write-Output ('=== ' + $f.FullName + ' mtime ' + $f.LastWriteTime + ' size ' + $f.Length); Get-Content $f.FullName -Tail 8 | ForEach-Object { Write-Output ('  ' + $_) } }
    else { Write-Output ('=== ' + $dir + ' : no *.err') }
}
Write-Output '######## AuthGateD log dir'
Get-ChildItem 'D:\AION_LIVE_SERVER\AuthGateD\log' -ErrorAction SilentlyContinue | Sort-Object LastWriteTime | Select-Object -Last 3 | ForEach-Object { Write-Output ('  ' + $_.Name + ' ' + $_.LastWriteTime + ' ' + $_.Length) }
FreshErr 'D:\AION_LIVE_SERVER\AuthGateD\log'
Write-Output '######## AuthD winlog (new world conn?)'
$w = Get-ChildItem 'D:\AION_LIVE_SERVER\AuthD\log\winlog' -ErrorAction SilentlyContinue | Sort-Object LastWriteTime | Select-Object -Last 1
if ($w) { Write-Output ('=== ' + $w.FullName + ' mtime ' + $w.LastWriteTime); Get-Content $w.FullName -Tail 12 | ForEach-Object { Write-Output ('  ' + $_) } } else { Write-Output 'no winlog dir' }
Write-Output '######## NPCSvr err'
FreshErr 'D:\AION_LIVE_SERVER\NPCServer\log'
Write-Output '######## MainServer err'
FreshErr 'D:\AION_LIVE_SERVER\MainServer\log'
Write-Output '######## CacheD err'
FreshErr 'D:\AION_LIVE_SERVER\CacheServer\log'
Write-Output '######## PA dir logs'
Get-ChildItem 'D:\AION_LIVE_SERVER' -Filter '*PA*' -ErrorAction SilentlyContinue | ForEach-Object { Write-Output ('  ' + $_.Name) }
Get-ChildItem 'D:\AION_LIVE_SERVER\PAServer\log' -ErrorAction SilentlyContinue | Sort-Object LastWriteTime | Select-Object -Last 2 | ForEach-Object { Write-Output ('  PAlog: ' + $_.Name + ' ' + $_.LastWriteTime) }
Write-Output '######## tasks status'
schtasks /query /tn AionGate /fo csv | Select-Object -Last 1
schtasks /query /tn AionNPCit /fo csv | Select-Object -Last 1
schtasks /query /tn AionMainit /fo csv | Select-Object -Last 1
schtasks /query /tn AionProxy /fo csv 2>&1 | Select-Object -Last 1

$ErrorActionPreference = 'SilentlyContinue'
Write-Output '---conns2002---'
(netstat -ano | Select-String ':2002' | Select-String 'ESTABLISHED').Count
Write-Output '---ports---'
netstat -ano | Select-String 'LISTENING' | Select-String ':2104',':2106',':2110',':2002',':2006',':7777',':2220',':10057',':2051',':2005',':22206' | ForEach-Object { Write-Output ('  ' + $_.Line.Trim()) }
Write-Output '---winlog tail---'
$w = Get-ChildItem 'D:\AION_LIVE_SERVER\AuthD\etc\log' -Filter '2026-10-04*.winlog' | Sort-Object LastWriteTime | Select-Object -Last 1
Get-Content $w.FullName -Tail 8 | ForEach-Object { Write-Output ('  ' + $_) }
Write-Output '---mainserver err tail---'
Get-Content 'D:\AION_LIVE_SERVER\MainServer\log\2026-10-04.err' -Tail 8 | ForEach-Object { Write-Output ('  ' + $_) }
Write-Output '---npc err tail---'
Get-Content 'D:\AION_LIVE_SERVER\NPCServer\log\2026-10-04.err' -Tail 5 | ForEach-Object { Write-Output ('  ' + $_) }

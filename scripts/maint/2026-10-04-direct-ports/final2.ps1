$ErrorActionPreference = 'SilentlyContinue'
Write-Output ('conns2002=' + (netstat -ano | Select-String ':2002' | Select-String 'ESTABLISHED').Count)
netstat -ano | Select-String 'LISTENING' | Select-String ':2104',':2106',':2110',':2002',':2006',':7777',':2220',':10057',':2051',':2005',':22206' | ForEach-Object { Write-Output ('  ' + $_.Line.Trim()) }
$w = Get-ChildItem 'D:\AION_LIVE_SERVER\AuthD\etc\log' -Filter '2026-10-04*.winlog' | Sort-Object LastWriteTime | Select-Object -Last 1
Write-Output ('winlog=' + $w.Name)
Get-Content $w.FullName -Tail 6 | ForEach-Object { Write-Output ('  W ' + $_) }
Write-Output '---mainserver err---'
Get-Content 'D:\AION_LIVE_SERVER\MainServer\log\2026-10-04.err' -Tail 6 | ForEach-Object { Write-Output ('  M ' + $_) }

# Restore direct ports: gate 2106, MainServer 7777, petition 2107. Backups first.
$ErrorActionPreference = 'Stop'

# 1) AuthGateD config.txt: serverPort 2107 -> 2106
$p = 'D:\AION_LIVE_SERVER\AuthGateD\etc\config.txt'
Copy-Item $p ($p + '.bak-back2106') -Force
(Get-Content $p -Raw).Replace('serverPort=2107','serverPort=2106') | Set-Content $p -Encoding Ascii
Write-Output ('[gate config] ' + ((Get-Content $p | Select-String 'serverPort=' | ForEach-Object ToString) -join ' | '))

# 2) MainServer config.xml: clientAcceptPort 7778 -> 7777
$p = 'D:\AION_LIVE_SERVER\MainServer\config.xml'
Copy-Item $p ($p + '.bak-restore7777') -Force
(Get-Content $p -Raw).Replace('<clientAcceptPort>7778</clientAcceptPort>','<clientAcceptPort>7777</clientAcceptPort>') | Set-Content $p -Encoding Ascii
Write-Output ('[main config.xml] ' + ((Get-Content $p | Select-String 'clientAccept' | ForEach-Object ToString) -join ' | '))

# 3) MainServer common.xml: petitionServerPort 21055 -> 2107
$p = 'D:\AION_LIVE_SERVER\MainServer\common.xml'
Copy-Item $p ($p + '.bak-pet2107') -Force
(Get-Content $p -Raw).Replace('<petitionServerPort>21055</petitionServerPort>','<petitionServerPort>2107</petitionServerPort>') | Set-Content $p -Encoding Ascii
Write-Output ('[main common.xml] ' + ((Get-Content $p | Select-String 'petitionServerPort' | ForEach-Object ToString) -join ' | '))

# 4) gate.bat: wait/retry port 2107 -> 2106 + kill stale gate before start
$p = 'C:\Temp\gate.bat'
Copy-Item $p ($p + '.bak-2107') -Force
$c = (Get-Content $p -Raw).Replace(':2107',':2106').Replace('listening 2107','listening 2106')
$c = $c.Replace("@echo off", "@echo off`r`ntaskkill /F /IM AuthGateD.exe >nul 2>&1")
Set-Content $p $c -Encoding Ascii
Write-Output '[gate.bat] head:'
Get-Content $p | Select-Object -First 6

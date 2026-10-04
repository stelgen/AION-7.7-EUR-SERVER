$ErrorActionPreference = 'Continue'
# 1) gate.bat cosmetic fix
C:\Temp\py\python.exe C:\Temp\fix-gate-cosmetic.py

# 2) find desktop bat locations
$dirs = @('C:\Users\Public\Desktop', [Environment]::GetFolderPath('Desktop'), [Environment]::GetFolderPath('CommonDesktopDirectory'))
$dirs = $dirs | Select-Object -Unique
Write-Output '--- search AION-START-ALL.bat ---'
$found = @()
foreach ($d in $dirs) {
    Get-ChildItem $d -Filter 'AION-START-ALL.bat' -ErrorAction SilentlyContinue | ForEach-Object {
        Write-Output ('FOUND: ' + $_.FullName)
        $found += $_.FullName
    }
}
# 3) deploy v5 with backup
foreach ($f in $found) {
    Copy-Item $f ($f + '.bak-v4') -Force
    Copy-Item 'C:\Temp\AION-START-ALL-v5.bat' $f -Force
    Write-Output ('DEPLOYED v5 -> ' + $f)
    Get-Content $f | Select-String 'proxy|2106|7778' | ForEach-Object { Write-Output ('  check: ' + $_.Line) }
}
if ($found.Count -eq 0) { Write-Output 'NOT FOUND on desktops - check C:\Temp' }

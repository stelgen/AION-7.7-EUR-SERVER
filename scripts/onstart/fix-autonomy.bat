@echo off
REM === AION VM autonomy settings ===
REM 1) No Ctrl+Alt+Del at logon
reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System" /v DisableCAD /t REG_DWORD /d 1 /f >nul
REM 2) No shutdown-reason popup
reg add "HKLM\SOFTWARE\Policies\Microsoft\Windows NT\Reliability" /v ShutdownReasonOn /t REG_DWORD /d 0 /f >nul
reg add "HKLM\SOFTWARE\Policies\Microsoft\Windows NT\Reliability" /v ShutdownReasonUI /t REG_DWORD /d 0 /f >nul
REM 3) Event logs: capped + overwrite old
wevtutil sl Application /ms:52428800 /rt:false /bn:Application >nul
wevtutil sl System /ms:52428800 /rt:false >nul
wevtutil sl Security /ms:268435456 /rt:false >nul
echo [LOGS] new sizes:
wevtutil gli Application | findstr "size"
wevtutil gli System | findstr "size"
wevtutil gli Security | findstr "size"
REM 4) Screensaver OFF for Administrators account
for /f %%S in ('powershell -NoProfile -Command "(New-Object System.Security.Principal.NTAccount(''Администратор'')).Translate([System.Security.Principal.SecurityIdentifier]).Value"') do set ADM_SID=%%S
echo [SID] %ADM_SID%
reg add "HKU\%ADM_SID%\Control Panel\Desktop" /v ScreenSaveActive /t REG_SZ /d 0 /f >nul
reg delete "HKU\%ADM_SID%\Control Panel\Desktop" /v SCRNSAVE.EXE /f >nul 2>nul
REM 5) Power: display never off, no sleep
powercfg /change monitor-timeout-ac 0
powercfg /change standby-timeout-ac 0
powercfg /change hibernate-timeout-ac 0
REM 6) Cleanup one-shot/junk tasks
schtasks /delete /f /tn AionFixLight >nul 2>nul
schtasks /delete /f /tn AionFixKill >nul 2>nul
schtasks /delete /f /tn AionIntStart >nul 2>nul
schtasks /delete /f /tn AionKillLog >nul 2>nul
schtasks /delete /f /tn AionRAD >nul 2>nul
schtasks /delete /f /tn AionMain2 >nul 2>nul
schtasks /delete /f /tn AionPA2 >nul 2>nul
schtasks /delete /f /tn AionPA2105 >nul 2>nul
schtasks /change /tn AionGateTest /disable >nul 2>nul
schtasks /change /tn AionNetMon /disable >nul 2>nul
echo [TASKS] junk removed, GateTest/NetMon disabled
echo AUTONOMY-DONE
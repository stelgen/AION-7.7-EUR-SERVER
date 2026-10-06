@echo off
taskkill /F /IM aion-gate.exe >nul 2>&1
setlocal enabledelayedexpansion
REM === aion-gate prod v1: wait L2Authd(2104) -> run aion-gate on 2106 (foreground+log) ===
set TRY=0
:authw
netstat -ano | findstr ":2104" | findstr "LISTENING" >nul
if %errorlevel%==0 goto authok
set /a TRY+=1
if !TRY! GTR 40 (echo [FAIL] L2Authd 2104 not up & goto :eof)
ping -n 6 127.0.0.1 >nul
goto authw
:authok
cd /d D:\SAION\aion-gate
echo [%date% %time%] aion-gate prod start >> gate-prod.log
aion-gate.exe -config config-prod.yaml >> gate-prod.log 2>&1
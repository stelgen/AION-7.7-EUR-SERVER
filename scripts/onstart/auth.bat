@echo off
setlocal enabledelayedexpansion
REM === auth.bat v3: wait SQL + AccountCache(2220) -> start -> wait 2104 -> retry ===
set TRY=0
:accw
netstat -ano | findstr ":2220" | findstr "LISTENING" >nul
if %errorlevel%==0 goto accok
set /a TRY+=1
if !TRY! GTR 40 (echo [FAIL] AccountCache 2220 not up in 3.5min & goto :eof)
ping -n 6 127.0.0.1 >nul
goto accw
:accok
cd /d D:\AION_LIVE_SERVER\AuthD
start L2Authd.exe
set TRY=0
:authw
netstat -ano | findstr ":2104" | findstr "LISTENING" >nul
if %errorlevel%==0 goto authok
set /a TRY+=1
if !TRY! GTR 20 goto authkill
ping -n 9 127.0.0.1 >nul
goto authw
:authkill
set /a TRY+=1
if !TRY! GTR 3 (echo [FAIL] L2Authd 2104 not up & goto :eof)
echo [RETRY] L2Authd restart
taskkill /F /IM L2Authd.exe >nul 2>&1
ping -n 6 127.0.0.1 >nul
start L2Authd.exe
set TRY=0
goto authw
:authok
echo [OK] L2Authd listening 2104
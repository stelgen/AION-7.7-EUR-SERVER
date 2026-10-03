@echo off
setlocal enabledelayedexpansion
REM === logsrv.bat v3: wait SQL(1433) -> start -> wait 2051 -> retry ===
set TRY=0
:sqlw
netstat -ano | findstr ":1433" | findstr "LISTENING" >nul
if %errorlevel%==0 goto sqlok
set /a TRY+=1
if !TRY! GTR 60 (echo [FAIL] SQL not up & goto :eof)
ping -n 6 127.0.0.1 >nul
goto sqlw
:sqlok
cd /d D:\AION_LIVE_SERVER\LogServer
start LogServer64.exe
set TRY=0
:lw
netstat -ano | findstr ":2051" | findstr "LISTENING" >nul
if %errorlevel%==0 goto lok
set /a TRY+=1
if !TRY! GTR 20 goto lkill
ping -n 9 127.0.0.1 >nul
goto lw
:lkill
set /a TRY+=1
if !TRY! GTR 3 (echo [FAIL] LogServer 2051 not up & goto :eof)
echo [RETRY] LogServer restart
taskkill /F /IM LogServer64.exe >nul 2>&1
ping -n 6 127.0.0.1 >nul
start LogServer64.exe
set TRY=0
goto lw
:lok
echo [OK] LogServer listening 2051
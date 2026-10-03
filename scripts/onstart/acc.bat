@echo off
setlocal enabledelayedexpansion
REM === acc.bat v3: wait SQL(1433) -> start -> wait 2220 -> kill+retry if dialog-hang ===
set TRY=0
:sqlw
netstat -ano | findstr ":1433" | findstr "LISTENING" >nul
if %errorlevel%==0 goto sqlok
set /a TRY+=1
if !TRY! GTR 60 (echo [FAIL] SQL 1433 not up in 5min & goto :eof)
ping -n 6 127.0.0.1 >nul
goto sqlw
:sqlok
cd /d D:\AION_LIVE_SERVER\AccountCacheServer
start AccountCacheServer.exe
set TRY=0
:accw
netstat -ano | findstr ":2220" | findstr "LISTENING" >nul
if %errorlevel%==0 goto accok
set /a TRY+=1
if !TRY! GTR 20 goto acckill
ping -n 9 127.0.0.1 >nul
goto accw
:acckill
set /a TRY+=1
if !TRY! GTR 3 (echo [FAIL] AccountCache 2220 not up & goto :eof)
echo [RETRY] AccountCache hang - kill and restart
taskkill /F /IM AccountCacheServer.exe >nul 2>&1
ping -n 6 127.0.0.1 >nul
start AccountCacheServer.exe
set TRY=0
goto accw
:accok
echo [OK] AccountCacheServer listening 2220
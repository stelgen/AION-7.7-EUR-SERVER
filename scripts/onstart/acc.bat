@echo off
REM waits for SQL Server after boot, then starts AccountCacheServer with retry
ping -n 46 127.0.0.1 >nul
cd /d D:\AION_LIVE_SERVER\AccountCacheServer
start AccountCacheServer.exe
ping -n 11 127.0.0.1 >nul
tasklist /FI "IMAGENAME eq AccountCacheServer.exe" | find /I "AccountCacheServer.exe" >nul
if %errorlevel%==0 goto done
start AccountCacheServer.exe
ping -n 11 127.0.0.1 >nul
:done
tasklist /FI "IMAGENAME eq AccountCacheServer.exe" | find /I "AccountCacheServer.exe" >nul
if %errorlevel%==0 (echo [OK] AccountCacheServer) else (echo [FAIL] AccountCacheServer)
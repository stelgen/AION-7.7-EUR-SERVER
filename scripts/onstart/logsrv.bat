@echo off
REM waits for SQL after boot, then starts LogServer64 with retry
ping -n 51 127.0.0.1 >nul
cd /d D:\AION_LIVE_SERVER\LogServer
start LogServer64.exe
ping -n 11 127.0.0.1 >nul
tasklist /FI "IMAGENAME eq LogServer64.exe" | find /I "LogServer64.exe" >nul
if %errorlevel%==0 goto done
start LogServer64.exe
ping -n 11 127.0.0.1 >nul
:done
tasklist /FI "IMAGENAME eq LogServer64.exe" | find /I "LogServer64.exe" >nul
if %errorlevel%==0 (echo [OK] LogServer64) else (echo [FAIL] LogServer64)
@echo off
REM waits for AccountCache/SQL after boot, then starts L2Authd with retry
ping -n 61 127.0.0.1 >nul
cd /d D:\AION_LIVE_SERVER\AuthD
start L2Authd.exe
ping -n 11 127.0.0.1 >nul
tasklist /FI "IMAGENAME eq L2Authd.exe" | find /I "L2Authd.exe" >nul
if %errorlevel%==0 goto done
start L2Authd.exe
ping -n 11 127.0.0.1 >nul
:done
tasklist /FI "IMAGENAME eq L2Authd.exe" | find /I "L2Authd.exe" >nul
if %errorlevel%==0 (echo [OK] L2Authd) else (echo [FAIL] L2Authd)
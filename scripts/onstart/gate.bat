@echo off
REM waits for L2Authd after boot, then starts AuthGateD with retry
ping -n 76 127.0.0.1 >nul
cd /d D:\AION_LIVE_SERVER\AuthGateD
start AuthGateD.exe
ping -n 11 127.0.0.1 >nul
tasklist /FI "IMAGENAME eq AuthGateD.exe" | find /I "AuthGateD.exe" >nul
if %errorlevel%==0 goto done
start AuthGateD.exe
ping -n 11 127.0.0.1 >nul
:done
tasklist /FI "IMAGENAME eq AuthGateD.exe" | find /I "AuthGateD.exe" >nul
if %errorlevel%==0 (echo [OK] AuthGateD) else (echo [FAIL] AuthGateD)
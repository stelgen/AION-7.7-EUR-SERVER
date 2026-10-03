@echo off
cd /d D:\AION_LIVE_SERVER
start 01-PAServer7.7.exe
ping -n 6 127.0.0.1 >nul
tasklist /FI "IMAGENAME eq 01-PAServer7.7.exe" | find /I "01-PAServer7.7.exe" >nul
if %errorlevel%==0 (echo [OK] PAServer) else (echo [FAIL] PAServer)
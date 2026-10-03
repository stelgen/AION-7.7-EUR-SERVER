@echo off
cd /d D:\AION_LIVE_SERVER\ICServer
start ICServer.exe
ping -n 11 127.0.0.1 >nul
tasklist /FI "IMAGENAME eq ICServer.exe" | find /I "ICServer.exe" >nul
if %errorlevel%==0 (echo [OK] ICServer) else (echo [FAIL] ICServer)
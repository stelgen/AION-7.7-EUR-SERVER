@echo off
cd /d D:\AION_LIVE_SERVER\CAPTCHAImageServer
start CAPTCHAImageServer.exe
ping -n 11 127.0.0.1 >nul
tasklist /FI "IMAGENAME eq CAPTCHAImageServer.exe" | find /I "CAPTCHAImageServer.exe" >nul
if %errorlevel%==0 (echo [OK] CAPTCHAImageServer) else (echo [FAIL] CAPTCHAImageServer)
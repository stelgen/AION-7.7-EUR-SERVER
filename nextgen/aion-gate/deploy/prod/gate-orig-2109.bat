@echo off
taskkill /F /IM AuthGateD.exe >nul 2>&1
cd /d D:\AION_LIVE_SERVER\AuthGateD-2109
start AuthGateD.exe
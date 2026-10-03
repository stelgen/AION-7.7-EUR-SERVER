@echo off
cd /d C:\Temp
start "AionProxy" /min C:\Temp\py\python.exe C:\Temp\aionproxy.py
ping -n 8 127.0.0.1 >nul
netstat -ano | findstr LISTENING | findstr ":2106"
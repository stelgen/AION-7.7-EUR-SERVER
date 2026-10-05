@echo off
title AION START LOG - LogServer64
REM ==========================================================
REM  AION START LOG (05.10.2026) - zapusk LogServer64 s desktopa
REM  Port 2051. ZAPUSKAT PERVYM: DO CacheD / NPC / MAIN.
REM  Esli LogServer ne zapushen - CacheD lovit "Can't connect to
REM  log server" i mir ne sobiraetsya polnostyu.
REM ==========================================================

tasklist /fi "imagename eq LogServer64.exe" 2>nul | findstr /i "LogServer64.exe" >nul
if errorlevel 1 goto startlog
echo [SKIP] LogServer64 UJE RABOTAET - zapusk ne nuzhen
pause
goto :eof

:startlog
echo [START] LogServer64...
cd /d D:\AION_LIVE_SERVER\LogServer
start "A-LOG" LogServer64.exe

echo [WAIT] proverka porta 2051, max ~2 min...
set /a TRY=0
:waitloop
netstat -ano 2>nul | findstr ":2051" | findstr "LISTENING" >nul
if not errorlevel 1 goto logok
set /a TRY+=1
if %TRY% GTR 24 goto logslow
ping -n 5 127.0.0.1 >nul
goto waitloop

:logslow
echo [VNIMANIE] 2051 ne slushaetsya ~2 min - smotri okno LogServer i log:
echo   D:\AION_LIVE_SERVER\LogServer\log - svezhiy .err
pause
goto :eof

:logok
echo [OK] LogServer64 slushaet 2051
echo Poryadok dalee: AION-START-CACHE.bat - AION-START-NPC.bat - AION-START-MAIN.bat
pause
goto :eof

@echo off
title AION START NPC - NPCSvr64
REM ==========================================================
REM  AION START NPC (05.10.2026) - zapusk NPCSvr64 s desktopa
REM  Polnaya zagruzka mira 10-15 min, private ~15 GB RAM.
REM  Port 2002 slushaetsya rano, polnaya gotovnost - konec spawn.
REM  Vse ostalnye servisy dolzhny byt podnyaty (AION-START-ALL).
REM ==========================================================

tasklist /fi "imagename eq NPCSvr64.exe" 2>nul | findstr /i "NPCSvr64.exe" >nul
if errorlevel 1 goto startnpc
echo [SKIP] NPCSvr64 UJE RABOTAET - zapusk ne nuzhen
pause
goto :eof

:startnpc
echo [START] NPCSvr64 - polnaya zagruzka mira 10-15 min...
cd /d D:\AION_LIVE_SERVER\NPCServer
start "A-NPC" NPCSvr64.exe

echo [WAIT] proverka porta 2002, max ~2 min...
set /a TRY=0
:waitloop
netstat -ano 2>nul | findstr ":2002" | findstr "LISTENING" >nul
if not errorlevel 1 goto npcok
set /a TRY+=1
if %TRY% GTR 24 goto npcslow
ping -n 5 127.0.0.1 >nul
goto waitloop

:npcslow
echo [VNIMANIE] 2002 ne slushaetsya ~2 min - smotri okno NPCSvr i log:
echo   D:\AION_LIVE_SERVER\NPCServer\log - svezhiy .err
pause
goto :eof

:npcok
echo [OK] NPCSvr64 slushaet 2002 - mir dogruzhaetsya 10-15 min
echo Zatem zapusti AION-START-MAIN.bat
pause
goto :eof

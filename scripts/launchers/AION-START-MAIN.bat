@echo off
title AION START MAIN - Server64 7777
REM ==========================================================
REM  AION START MAIN (05.10.2026) - zapusk Server64 s desktopa
REM  BEZ RunAsDate - date-bypass #180 v binary rabotaet sam.
REM  PORYADOK: SNACHALA NPC (AION-START-NPC.bat), potom MAIN.
REM  7777 poyavitsya tolko posle polnoy zagruzki NPC 10-15 min.
REM ==========================================================

tasklist /fi "imagename eq Server64.exe" 2>nul | findstr /i "Server64.exe" >nul
if errorlevel 1 goto startmain
echo [SKIP] Server64 UJE RABOTAET - zapusk ne nuzhen
pause
goto :eof

:startmain
set NPCHERE=0
tasklist /fi "imagename eq NPCSvr64.exe" 2>nul | findstr /i "NPCSvr64.exe" >nul
if errorlevel 1 goto npcwarn
set NPCHERE=1
goto launch

:npcwarn
echo [VNIMANIE] NPCSvr64 NE zapushen - Server64 budet zdat NPC
echo Pravilno: snachala AION-START-NPC.bat i dozhdis polnoy zagruzki
echo Vsyo ravno zapuskayu Server64...

:launch
echo [START] Server64...
cd /d D:\AION_LIVE_SERVER\MainServer
start "A-MAIN" Server64.exe

if %NPCHERE%==1 goto npcinfo
echo [INFO] NPC ne bil zapushen - mir ne sobereetsya bez nego
echo Zapusti AION-START-NPC.bat, dozhdis 10-15 min i perezapusti MAIN
pause
goto :eof

:npcinfo
echo [INFO] NPC rabotaet. 7777 podnimetsya posle polnoy zagruzki mira.
echo Proverka cherez 10-15 min: netstat -ano - ishchi 7777 LISTENING
echo i 16 ESTABLISHED na 2002 = mir sobran, mozhno zahodit klientom
pause
goto :eof

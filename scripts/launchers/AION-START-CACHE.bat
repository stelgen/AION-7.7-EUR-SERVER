@echo off
title AION START CACHE - CacheD64
REM ==========================================================
REM  AION START CACHE (05.10.2026) - zapusk CacheD64 s desktopa
REM  Porty 2006+2007. Zapuskat DO NPC i MAIN.
REM  Norma pri zagruzke: ogromnyy spam "Strings DB unexpted id"
REM  v .err (~570k strok odnorazovo) - eto NE oshibka.
REM ==========================================================

tasklist /fi "imagename eq CacheD64.exe" 2>nul | findstr /i "CacheD64.exe" >nul
if errorlevel 1 goto startcache
echo [SKIP] CacheD64 UJE RABOTAET - zapusk ne nuzhen
pause
goto :eof

:startcache
echo [START] CacheD64...
cd /d D:\AION_LIVE_SERVER\CacheServer
start "A-CACHE" CacheD64.exe

echo [WAIT] proverka porta 2006, max ~3 min...
set /a TRY=0
:waitloop
netstat -ano 2>nul | findstr ":2006" | findstr "LISTENING" >nul
if not errorlevel 1 goto cacheok
set /a TRY+=1
if %TRY% GTR 36 goto cacheslow
ping -n 5 127.0.0.1 >nul
goto waitloop

:cacheslow
echo [VNIMANIE] 2006 ne slushaetsya ~3 min - smotri okno CacheD i log:
echo   D:\AION_LIVE_SERVER\CacheServer\log - svezhiy .err
pause
goto :eof

:cacheok
echo [OK] CacheD64 slushaet 2006 - dogruzka Item Info idi eshyo paru min
echo Poryadok dalee: AION-START-NPC.bat - 10-15 min - AION-START-MAIN.bat
pause
goto :eof

@echo off
title AION START ALL v6 (aion-logd)
REM ================= AION START ALL v6 (05.10.2026) =================
REM Otlichie ot v5: shag [3/12] - vmesto LogServer64 startuem NASH
REM   aion-logd (D:\SAION\aion-logd, config.yaml, capture_all+logdb).
REM   Vse ostalnoe = v5 (provereno 04-05.10, MD5 919c49dd).
REM ZAPUSKAT S RABOCHEGO STOLA: vse appki v session 1, NE SYSTEM/session 0.
REM Esli zadachu planirovshika registriruesh - tolko s /IT (interaktivno).
REM Original AionLog NE trogaem (ostaetsya Ready) = otkat:
REM   taskkill /F /IM aion-logd.exe + schtasks /run AionLog
REM Odno-razovo perevesti logd iz SYSTEM v sessiyu (iz admin-konsoli):
REM   schtasks /change /tn AionLogCap /disable

echo [0/12] Kill strays + stop SYSTEM-logd (esli bil)...
taskkill /F /IM Server64.exe >nul 2>&1
taskkill /F /IM NPCSvr64.exe >nul 2>&1
taskkill /F /IM CacheD64.exe >nul 2>&1
taskkill /F /IM L2Authd.exe >nul 2>&1
taskkill /F /IM AuthGateD.exe >nul 2>&1
taskkill /F /IM LogServer64.exe >nul 2>&1
taskkill /F /IM AccountCacheServer.exe >nul 2>&1
taskkill /F /IM ICServer.exe >nul 2>&1
taskkill /F /IM CAPTCHAImageServer.exe >nul 2>&1
taskkill /F /IM 01-PAServer7.7.exe >nul 2>&1
taskkill /F /IM python.exe >nul 2>&1
schtasks /end /tn AionLogCap >nul 2>&1
taskkill /F /IM aion-logd.exe >nul 2>&1
ping -n 4 127.0.0.1 >nul

echo [1/12] Wait SQL 1433...
call :waitport 1433 60 SQL
if errorlevel 1 goto fail

echo [2/12] AccountCacheServer (2220)...
start "A-acc" /D "D:\AION_LIVE_SERVER\AccountCacheServer" AccountCacheServer.exe
call :waitport 2220 30 ACC2220
if errorlevel 1 goto fail

echo [3/12] NASH aion-logd (2051) vmesto LogServer64...
netstat -ano | findstr ":2051" | findstr "LISTENING" >nul
if not errorlevel 1 goto logdup
start "A-logd" /D "D:\SAION\aion-logd" run.cmd
call :waitport 2051 30 LOGD2051
if errorlevel 1 goto fail
goto logdone
:logdup
echo   [SKIP] 2051 uje slushaetsya - logd rabotaet (eshe iz AionLogCap/SYSTEM)
:logdone

echo [4/12] ICServer (2005)...
start "A-ic" /D "D:\AION_LIVE_SERVER\ICServer" ICServer.exe
call :waitport 2005 30 IC2005
if errorlevel 1 goto fail

echo [5/12] CAPTCHAImageServer (22206)...
start "A-cap" /D "D:\AION_LIVE_SERVER\CAPTCHAImageServer" CAPTCHAImageServer.exe
call :waitport 22206 30 CAP22206
if errorlevel 1 goto fail

echo [6/12] PAServer (10057)...
start "A-pa" /D "D:\AION_LIVE_SERVER" 01-PAServer7.7.exe
call :waitport 10057 30 PA10057
if errorlevel 1 goto fail

echo [7/12] L2Authd (2104)...
start "A-auth" /D "D:\AION_LIVE_SERVER\AuthD" L2Authd.exe
call :waitport 2104 30 AUTH2104
if errorlevel 1 goto fail

echo [8/12] AuthGateD (2106, pryamoy port)...
start "A-gate" /D "D:\AION_LIVE_SERVER\AuthGateD" AuthGateD.exe
call :waitport 2106 30 GATE2106
if errorlevel 1 goto fail

echo [9/12] CacheD64 (2006)...
start "A-cache" /D "D:\AION_LIVE_SERVER\CacheServer" CacheD64.exe
call :waitport 2006 30 CACHE2006
if errorlevel 1 goto fail

echo [10/12] NPCSvr64 (load 10-15 min)...
start "A-npc" /D "D:\AION_LIVE_SERVER\NPCServer" NPCSvr64.exe
ping -n 21 127.0.0.1 >nul

echo [11/12] Server64 (7777, pryamoy port)...
start "A-main" /D "D:\AION_LIVE_SERVER\MainServer" Server64.exe

echo [12/12] Wait world: 7777 + 16 NPC conns (max 25 min)...
set /a TRY=0
:worldw
set N=0
for /f %%i in ('netstat -ano ^| findstr ":2002" ^| findstr "ESTABLISHED" ^| find /c /v ""') do set N=%%i
netstat -ano | findstr ":7777" | findstr "LISTENING" >nul
if %errorlevel%==0 if %N% GEQ 16 goto worldok
set /a TRY+=1
if %TRY% GTR 150 (echo [FAIL] world not assembled. conns=%N% & pause & goto :eof)
ping -n 11 127.0.0.1 >nul
goto worldw
:worldok
echo.
echo ==================================================
echo   MIR SOBRAN: conns=%N%, 7777 OK - MOZHNO LOGINITsYA
echo ==================================================
pause
goto :eof

:waitport
set /a WPTRY=0
:wp_loop
netstat -ano | findstr ":%~1" | findstr "LISTENING" >nul
if %errorlevel%==0 ( echo   [OK] %~2 & goto :eof )
set /a WPTRY+=1
if %WPTRY% GTR %~2 ( echo   [FAIL] %~2 not up & exit /b 1 )
ping -n 4 127.0.0.1 >nul
goto wp_loop

:fail
echo [FATAL] startup aborted
pause
goto :eof

@echo off
setlocal enabledelayedexpansion
REM =====================================================
REM  AION 7.7 PTS EU - START v2.2
REM  Лёгкие сервисы (Acc/Auth/Gate/Log/IC/CAPTCHA/PA) - always-on (ONSTART задачи).
REM  Этот батник поднимает ТЯЖЁЛЫЕ: CacheD -> NPCSvr -> Server64 (напрямую, без RunAsDate).
REM =====================================================
set SRV=D:\AION_LIVE_SERVER

echo [0/4] Проверка always-on сервисов (окна невидимы - это норма, работают в фоне)...
call :ENSURE "AionAcc"     "AccountCacheServer.exe"
call :ENSURE "AionAuth"    "L2Authd.exe"
call :ENSURE "AionGate"    "AuthGateD.exe"
call :ENSURE "AionLog"     "LogServer64.exe"
call :ENSURE "AionICSrv"   "ICServer.exe"
call :ENSURE "AionCAPTCHA" "CAPTCHAImageServer.exe"
call :ENSURE "AionPA"      "01-PAServer7.7.exe"

echo [1/3] CacheD64 (2006/2007, ~20 c)...
call :STARTTASK "AionCache" "CacheD64.exe" "%SRV%\CacheServer"

echo [2/3] NPCSvr64 (грузится 10-15 мин, ~15 ГБ RAM)...
call :STARTTASK "AionNPC" "NPCSvr64.exe" "%SRV%\NPCServer"

echo [3/3] Server64 НАПРЯМУЮ (без RunAsDate, date-bypass в бинаре #180)...
call :STARTTASK "AionMain" "Server64.exe" "%SRV%\MainServer"

echo.
echo ===== СВОДКА ПОРТОВ =====
netstat -ano | findstr LISTENING | findstr /C:":2220 " /C:":2104 " /C:":2106 " /C:":2051 " /C:":2006 " /C:":2005 " /C:":22206 " /C:":7777 " /C:":2002 "
echo.
echo Память:
wmic OS get FreePhysicalMemory,FreeVirtualMemory /value | findstr "="
echo.
echo Готово. NPCSvr догружается сам (10-15 мин). Лог мира: %SRV%\MainServer\log\ (реальные даты, без RunAsDate).
if /I not "%~1"=="AUTO" pause
goto :eof

:ENSURE
REM %1=задача %2=процесс
tasklist /FI "IMAGENAME eq %~2" 2>nul | find /I "%~2" >nul
if %errorlevel%==0 (
  echo   [ON]  %~2 работает
  goto :eof
)
echo   [FIX] %~2 не запущен - поднимаю задачей %~1...
schtasks /Change /TN %~1 /ENABLE >nul 2>&1
schtasks /Run /TN %~1 >nul 2>&1
ping -n 7 127.0.0.1 >nul 2>nul
schtasks /Change /TN %~1 /DISABLE >nul 2>&1
tasklist /FI "IMAGENAME eq %~2" 2>nul | find /I "%~2" >nul
if %errorlevel%==0 ( echo   [OK]  %~2 поднят ) else ( echo   [FAIL] %~2 не поднялся! )
goto :eof

:STARTTASK
REM %1=задача %2=процесс %3=каталог (пауза фикс ~30 c с проверкой раз в 3 c)
tasklist /FI "IMAGENAME eq %~2" 2>nul | find /I "%~2" >nul
if %errorlevel%==0 (
  echo   [SKIP] %~2 уже работает
  goto :eof
)
echo   [START] %~2 через задачу %~1...
schtasks /Change /TN %~1 /ENABLE >nul 2>&1
schtasks /Run /TN %~1 >nul 2>&1
set /a LEFT=10
:wait1
ping -n 4 127.0.0.1 >nul 2>nul
tasklist /FI "IMAGENAME eq %~2" 2>nul | find /I "%~2" >nul
if %errorlevel%==0 goto started
set /a LEFT-=1
if !LEFT! GTR 0 goto wait1
schtasks /Change /TN %~1 /DISABLE >nul 2>&1
echo   [FALLBACK] задачей не поднялся за 30 c - стартую напрямую из %~3
pushd %~3
start "" %~2
popd
ping -n 9 127.0.0.1 >nul 2>nul
:started
schtasks /Change /TN %~1 /DISABLE >nul 2>&1
REM дедуп: если процессов больше 1 - убить все кроме первого
set CNT=0
for /f %%P in ('tasklist /FI "IMAGENAME eq %~2" 2^>nul ^| find /I /C "%~2"') do set CNT=%%P
if !CNT! GTR 1 (
  echo   [WARN] %~2 запущено !CNT! шт - убираю дубли...
  set FIRST=1
  for /f "skip=3 tokens=2" %%P in ('tasklist /FI "IMAGENAME eq %~2"') do (
    if !FIRST!==1 ( set FIRST=0 ) else ( taskkill /PID %%P /F >nul 2>&1 )
  )
)
tasklist /FI "IMAGENAME eq %~2" 2>nul | find /I "%~2" >nul
if %errorlevel%==0 ( echo   [OK] %~2 работает ) else ( echo   [FAIL] %~2 НЕ поднялся - смотри log в %~3\log\ )
goto :eof
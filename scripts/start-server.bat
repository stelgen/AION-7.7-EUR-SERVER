@echo off
REM =====================================================
REM  AION 7.7 PTS EU - START (smart: не дублирует процессы)
REM  Запускать КАК АДМИНИСТРАТОР. Ордер критичен.
REM  В конце: NPCSvr грузится 10-15 мин, потом жми RUN в RunAsDate.
REM =====================================================
setlocal
set SRV=D:\AION_LIVE_SERVER

echo [1/9] AccountCacheServer (порт 2220)...
call :STARTTASK "AionAcc"  "AccountCacheServer.exe" "%SRV%\AccountCacheServer" 10

echo [2/9] L2Authd (2104/2108/2110)...
call :STARTTASK "AionAuth" "L2Authd.exe"            "%SRV%\AuthD"              8

echo [3/9] AuthGateD (2106 - точка входа клиентов)...
call :STARTTASK "AionGate" "AuthGateD.exe"          "%SRV%\AuthGateD"          8

echo [4/9] LogServer64 (2051)...
call :STARTTASK "AionLog"  "LogServer64.exe"        "%SRV%\LogServer"          8

echo [5/9] CacheD64 (2006/2007, грузит данные ~20 c)...
call :STARTTASK "AionCache" "CacheD64.exe"          "%SRV%\CacheServer"        25

echo [6/9] ICServer (Interchange/Channel 2005/2305)...
call :STARTTASK "AionICSrv" "ICServer.exe"          "%SRV%\ICServer"           8

echo [7/9] CAPTCHAImageServer (22206)...
call :STARTTASK "AionCAPTCHA" "CAPTCHAImageServer.exe" "%SRV%\CAPTCHAImageServer" 5

echo [8/9] NPCSvr64 (грузится 10-15 минут, ~15 ГБ RAM)...
call :STARTTASK "AionNPC"  "NPCSvr64.exe"           "%SRV%\NPCServer"          15

echo [9/9] RunAsDate для Server64...
tasklist /FI "IMAGENAME eq Server64.exe" 2>nul | find /I "Server64.exe" >nul
if %errorlevel%==0 (
  echo   [SKIP] Server64 уже работает
) else (
  schtasks /Change /TN "AionRAD" /ENABLE >nul 2>&1
  schtasks /Run /TN "AionRAD" >nul 2>&1
  schtasks /Change /TN "AionRAD" /DISABLE >nul 2>&1
  echo   [GO] Открыл RunAsDate. Если окно не видно - смотри консоль VM.
  echo   [RUN] Нажми кнопку RUN - DateTime 04-06-2020 16:28:23 - ПОСЛЕ загрузки NPCSvr!
)

echo.
echo ===== СВОДКА ПОРТОВ =====
netstat -ano | findstr LISTENING | findstr /C:":2220 " /C:":2104 " /C:":2106 " /C:":2051 " /C:":2006 " /C:":2005 " /C:":22206 " /C:":7777 "
echo.
echo Память:
wmic OS get FreePhysicalMemory,FreeVirtualMemory /value | findstr "="
echo.
echo Готово. NPCSvr догружается сам; Server64 стартует кнопкой RUN в RunAsDate.
pause
goto :eof

:STARTTASK
REM %1=задача %2=процесс %3=рабочий каталог %4=секунд ожидания
tasklist /FI "IMAGENAME eq %~2" 2>nul | find /I "%~2" >nul
if %errorlevel%==0 (
  echo   [SKIP] %~2 уже запущен - не дублирую
  goto :eof
)
echo   [START] %~2 через задачу %~1...
schtasks /Change /TN %~1 /ENABLE >nul 2>&1
schtasks /Run   /TN %~1 >nul 2>&1
timeout /t 8 /nobreak >nul
schtasks /Change /TN %~1 /DISABLE >nul 2>&1
tasklist /FI "IMAGENAME eq %~2" 2>nul | find /I "%~2" >nul
if %errorlevel%==0 (
  echo   [OK] %~2 поднят
  goto :wait
)
echo   [FALLBACK] задачей не поднялся - стартую напрямую из %~3
pushd %~3
start "" %~2
popd
:wait
timeout /t %~4 /nobreak >nul
tasklist /FI "IMAGENAME eq %~2" 2>nul | find /I "%~2" >nul
if %errorlevel%==0 ( echo   [OK] %~2 работает ) else ( echo   [FAIL] %~2 НЕ поднялся - смотри log в %~3\log\ )
goto :eof
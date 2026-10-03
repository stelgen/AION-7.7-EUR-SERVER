@echo off
REM =====================================================
REM  AION 7.7 PTS EU - STOP v2.2
REM  Останавливает ТОЛЬКО тяжёлые: Server64 -> NPCSvr -> CacheD.
REM  Лёгкие always-on сервисы НЕ трогает. SQL не трогает.
REM =====================================================
set SRV=D:\AION_LIVE_SERVER

echo [1/3] Server64 (мир; мягко, потом форс)...
call :STOPPROC "Server64.exe"

echo [2/3] NPCSvr64 (спавны)...
call :STOPPROC "NPCSvr64.exe"

echo [3/3] CacheD64 (кэш мира)...
call :STOPPROC "CacheD64.exe"

echo [4] RunAsDate добить (если висит)...
taskkill /IM RunAsDate.exe >nul 2>&1

echo [5] Чистка *.err логов (тяжёлые остановлены)...
set DELCNT=0
for /d %%D in ("%SRV%\*") do (
  if exist "%%D\log" for %%F in ("%%D\log\*.err") do (
    del /q "%%F" 2>nul
    if not exist "%%F" set /a DELCNT+=1
  )
)
echo   [OK] Удалено *.err: %DELCNT%

echo.
echo ===== Тяжёлые после стопа (должно быть пусто) =====
tasklist | findstr /I "Server64 NPCSvr64 CacheD64 RunAsDate"
echo ===== Always-on сервисы (остались работать - так и задумано) =====
tasklist | findstr /I "AccountCacheServer L2Authd AuthGateD LogServer64 ICServer CAPTCHAImageServer 01-PAServer"
echo (Остановить и их: schtasks-задачи AionAcc/AionAuth/AionGate/AionLog/AionICSrv/AionCAPTCHA/AionPA + taskkill)
if /I not "%~1"=="AUTO" pause
goto :eof

:STOPPROC
tasklist /FI "IMAGENAME eq %~1" 2>nul | find /I "%~1" >nul
if not %errorlevel%==0 (
  echo   [SKIP] %~1 не запущен
  goto :eof
)
echo   [STOP] %~1 мягко...
taskkill /IM %~1 >nul 2>&1
ping -n 11 127.0.0.1 >nul 2>nul
tasklist /FI "IMAGENAME eq %~1" 2>nul | find /I "%~1" >nul
if %errorlevel%==0 (
  echo   [FORCE] %~1 не закрылся - форсирую
  taskkill /F /IM %~1 >nul 2>&1
  ping -n 4 127.0.0.1 >nul 2>nul
)
tasklist /FI "IMAGENAME eq %~1" 2>nul | find /I "%~1" >nul
if %errorlevel%==0 ( echo   [FAIL] %~1 всё ещё жив! ) else ( echo   [OK] %~1 остановлен )
goto :eof
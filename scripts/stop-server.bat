@echo off
REM =====================================================
REM  AION 7.7 PTS EU - STOP (обратный ордер, мягко где можно)
REM  SQL Server НЕ трогаем.
REM =====================================================
setlocal

echo [1/9] Server64 (мир; сначала мягко, потом форс)...
call :STOPPROC "Server64.exe" 10

echo [1b]  RunAsDate (GUI-обёртка)...
taskkill /IM RunAsDate.exe >nul 2>&1

echo [2/9] NPCSvr64 (спавны)...
call :STOPPROC "NPCSvr64.exe" 10

echo [3/9] CacheD64 (кэш мира)...
call :STOPPROC "CacheD64.exe" 6

echo [4/9] LogServer64...
call :STOPPROC "LogServer64.exe" 5

echo [5/9] ICServer (Interchange)...
call :STOPPROC "ICServer.exe" 5

echo [6/9] CAPTCHAImageServer...
call :STOPPROC "CAPTCHAImageServer.exe" 4

echo [7/9] AuthGateD...
call :STOPPROC "AuthGateD.exe" 5

echo [8/9] L2Authd...
call :STOPPROC "L2Authd.exe" 5

echo [9/9] AccountCacheServer...
call :STOPPROC "AccountCacheServer.exe" 5

echo [10/10] Чистка *.err логов (процессы остановлены - файлы свободны)...
set SRV=D:\AION_LIVE_SERVER
set DELCNT=0
for /d %%D in ("%SRV%\*") do (
  if exist "%%D\log" for %%F in ("%%D\log\*.err") do (
    del /q "%%F" 2>nul
    set /a DELCNT+=1
  )
)
echo   [OK] Удалено *.err: %DELCNT% (создадутся заново при старте; растут гигами больше не будут)

echo.
echo ===== Остатки игровых процессов =====
tasklist | findstr /I "Server64 NPCSvr64 LogServer64 CacheD64 AccountCacheServer L2Authd AuthGateD ICServer CAPTCHAImageServer RunAsDate"
echo (пусто = всё чисто. SQL Server оставлен работать.)
pause
goto :eof

:STOPPROC
REM %1=процесс %2=секунд на мягкое завершение
tasklist /FI "IMAGENAME eq %~1" 2>nul | find /I "%~1" >nul
if not %errorlevel%==0 (
  echo   [SKIP] %~1 не запущен
  goto :eof
)
echo   [STOP] %~1 мягко...
taskkill /IM %~1 >nul 2>&1
timeout /t %~2 /nobreak >nul
tasklist /FI "IMAGENAME eq %~1" 2>nul | find /I "%~1" >nul
if %errorlevel%==0 (
  echo   [FORCE] %~1 не закрылся - форсирую
  taskkill /F /IM %~1 >nul 2>&1
  timeout /t 3 /nobreak >nul
)
tasklist /FI "IMAGENAME eq %~1" 2>nul | find /I "%~1" >nul
if %errorlevel%==0 ( echo   [FAIL] %~1 всё ещё жив! ) else ( echo   [OK] %~1 остановлен )
goto :eof
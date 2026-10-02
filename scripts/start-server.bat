@echo off
REM =================================================================
REM AION 7.7 PTS EU - Auto-start all server components (ordered)
REM Run as administrator! (Right-click -> Run as administrator)
REM Stack: AccountCacheServer -> L2Authd -> AuthGateD -> LogServer64
REM        -> CacheD64 -> NPCSvr64 (10 min) -> Server64 (via RunAsDate)
REM =================================================================
set SRV=D:\AION_LIVE_SERVER
set DSN=C:\DSN

echo [1/7] AccountCacheServer (port 2220)...
schtasks /Run /TN "AionAcc" 2>nul
if errorlevel 1 (cd /d "%SRV%\AccountCacheServer" && start AccountCacheServer.exe)
timeout /t 10 /nobreak >nul

echo [2/7] L2Authd / AuthD (ports 2104, 2108, 2110)...
schtasks /Run /TN "AionAuth" 2>nul
if errorlevel 1 (cd /d "%SRV%\AuthD" && start L2Authd.exe)
timeout /t 10 /nobreak >nul

echo [3/7] AuthGateD (port 2106 - client entry)...
schtasks /Run /TN "AionGate" 2>nul
if errorlevel 1 (cd /d "%SRV%\AuthGateD" && start AuthGateD.exe)
timeout /t 10 /nobreak >nul

echo [4/7] LogServer64 (port 2051)...
schtasks /Run /TN "AionLog" 2>nul
if errorlevel 1 (cd /d "%SRV%\LogServer" && start LogServer64.exe)
timeout /t 10 /nobreak >nul

echo [5/7] CacheD64 (ports 2006, 2007 - world cache)...
schtasks /Run /TN "AionCache" 2>nul
if errorlevel 1 (cd /d "%SRV%\CacheServer" && start CacheD64.exe)
timeout /t 30 /nobreak >nul

echo [6/7] NPCSvr64 (spawns/scripts, ~10 min loading, up to 10 GB RAM)...
schtasks /Run /TN "AionNPC" 2>nul
if errorlevel 1 (cd /d "%SRV%\NPCServer" && start NPCSvr64.exe)
timeout /t 90 /nobreak >nul

echo [7/7] Server64 via RunAsDate (system time override to 04-06-2020)...
schtasks /Run /TN "AionRAD" 2>nul
if errorlevel 1 (cd /d "%SRV%\MainServer\runasdate-x64" && start RunAsDate.exe)
timeout /t 10 /nobreak >nul

echo.
echo =====================================================
echo  Server launched!
echo  Game port: 7777, login: 2106
echo  Client start: start bin64\aion.bin -ip:VM_IP -port:2106 -cc:2 ...
echo  Stop: stop-server.bat
echo =====================================================
echo.
echo NOTE: if SQL Login dialog appears - enter:
echo   File DB: %DSN%\aionworld_new.dsn   (or other dsn name from dialog)
echo   Login:   sa
echo   Password: 123
echo.
pause

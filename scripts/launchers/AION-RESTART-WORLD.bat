@echo off
REM =====================================================
REM  AION - RESTART WORLD (Server64 + NPCSvr) v1
REM  Сбрасывает счётчик онлайна (фикс "не авторизован") и
REM  пересобирает мир. Порядок: kill пары -> NPCSvr -> Server64
REM  -> ждём 16 NPC-коннектов (полная сборка).
REM  Запускать С ДЕСКТОПА VM (тогда taskkill работает).
REM =====================================================
setlocal enabledelayedexpansion
set SRV=D:\AION_LIVE_SERVER

echo [1/5] Останов пары Server64 + NPCSvr64...
taskkill /F /IM Server64.exe >nul 2>&1
taskkill /F /IM NPCSvr64.exe >nul 2>&1
ping -n 8 127.0.0.1 >nul
tasklist | findstr /I "Server64.exe NPCSvr64.exe"
echo   (если пусто - пара остановлена)

echo [2/5] NPCSvr64 старт (грузится 10-15 мин, ~15 ГБ RAM)...
cd /d %SRV%\NPCServer
start NPCSvr64.exe

echo [3/5] Server64 старт (напрямую, без RunAsDate)...
cd /d %SRV%\MainServer
start Server64.exe

echo [4/5] Жду сборку мира (16 NPC-коннектов, до ~20 мин)...
set /a TRIES=0
:waitc
ping -n 30 127.0.0.1 >nul
set CONNS=0
for /f %%C in ('netstat -ano ^| findstr ":2002" ^| findstr ESTABLISHED ^| find /C /V ""') do set CONNS=%%C
set /a TRIES+=1
echo   ...коннектов: !CONNS!  (попытка !TRIES!/40)
if !CONNS! GEQ 16 goto ready
if !TRIES! LSS 40 goto waitc
echo   [WARN] 20 мин - коннектов 16 нет. Смотри логи %SRV%\MainServer\log\
if /I not "%~1"=="AUTO" pause
goto :eof

:ready
echo.
echo [5/5] МИР СОБРАН (16 NPC-коннектов). Счётчик онлайна сброшен.
echo       Заходи клиентом - вход должен пройти с первой попытки.
echo       После каждого полного выхода из мира - запускай этот батник заново,
echo       иначе счётчик "течёт" и получишь отказ "не авторизован".
if /I not "%~1"=="AUTO" pause
goto :eof
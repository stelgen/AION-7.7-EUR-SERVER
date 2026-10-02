@echo off
REM ===== AION 7.7 EU client -> наш сервер =====
REM Положить В КОРЕНЬ клиента (рядом с папкой bin64) и запускать двойным кликом.
REM PowerShell-ошибка "module bin64 could not be loaded" = забыли .\ перед путём.
cd /d "%~dp0"
if not exist bin64\aion.exe (
  echo НЕ ВИЖУ bin64\aion.exe здесь: %cd%
  echo Положи этот bat в корень клиента рядом с bin64.
  pause
  exit /b 1
)
start "" bin64\aion.exe -ip:192.168.0.125 -port:2106 -cc:2 -noauthgg
echo Клиент запущен: логин 192.168.0.125:2106, страна cc:2 (EU).
echo Аккаунт создаётся автоматически при первом входе (любой логин/пароль).
timeout /t 3 >nul
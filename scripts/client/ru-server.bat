@echo off
REM =====================================================
REM AION 7.7 — рабочий запуск ru-клиента (Innova/4game)
REM Проверено 03.10.2026: логин+регистрация работают!
REM Положить В КОРЕНЬ клиента (рядом с bin64\).
REM Критические флаги: -loginex -pwd16 (классический логин,
REM пароль = MD5-хеш 16 байт = user_auth.password binary(16)).
REM Без них клиент шлёт портал-сессию (sessionId=0) → гейт рвёт!
REM =====================================================
cd /d "%~dp0\bin64\Frost"
start Aion.exe ^
  -ip:81.25.59.194 -port:2106 -cc:7 -noauthgg ^
  -noweb -nobs -ls -charnamemenu -lbox -nologout ^
  -customizing -megaphone -nwp -ncping -f2p ^
  -localtime -rcdelay:5 -loginex -pwd16 -multithread ^
  -frostGame "..\..\bin64\aion.bin" -frostOptions 7 -frostGameNameType aion_live
timeout /t 3 >nul
@echo off
REM ===== AION 7.7 EU client -> 192.168.0.125 =====
REM Put this bat NEXT TO bin64 folder (client root). Auto-searches if misplaced.
cd /d "%~dp0"

if exist bin64\aion.exe goto run
if exist ..\bin64\aion.exe ( cd /d .. & goto run )
for /d /r "%~dp0" %%D in (bin64) do if exist "%%D\aion.exe" ( cd /d "%%D\.." & goto run )

echo [X] bin64\aion.exe NOT FOUND near: %cd%
echo     Find aion.bin via Explorer search, then put this bat in that folder.
pause
exit /b 1

:run
echo [OK] Client root: %cd%
start "" bin64\aion.exe -ip:192.168.0.125 -port:2106 -cc:2 -noauthgg
echo Client launched. Login: 192.168.0.125:2106, country cc:2 (EU).
echo Account auto-creates on first login (any login/password).
timeout /t 3 >nul
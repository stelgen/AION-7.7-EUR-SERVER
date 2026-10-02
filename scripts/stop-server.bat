@echo off
REM Stop all AION server components (reverse order)
echo Stopping AION servers...
taskkill /f /im Server64.exe 2>nul
taskkill /f /im RunAsDate.exe 2>nul
taskkill /f /im NPCSvr64.exe 2>nul
taskkill /f /im CacheD64.exe 2>nul
taskkill /f /im LogServer64.exe 2>nul
taskkill /f /im AuthGateD.exe 2>nul
taskkill /f /im L2Authd.exe 2>nul
taskkill /f /im L2AuthD.exe 2>nul
taskkill /f /im AccountCacheServer.exe 2>nul
taskkill /f /im ICServer.exe 2>nul
taskkill /f /im CAPTCHAImageServer.exe 2>nul
taskkill /f /im NPRelayServer.exe 2>nul
taskkill /f /im RankingServer.exe 2>nul
taskkill /f /im GMServer.exe 2>nul
echo done
pause

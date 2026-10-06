@echo off
rem aion-gate stand (AionGateStand task) - port 2108, prod 2106 untouched
cd /d D:\SAION\aion-gate-stand
aion-gate.exe -config config-stand.yaml >> gate-stand.log 2>&1
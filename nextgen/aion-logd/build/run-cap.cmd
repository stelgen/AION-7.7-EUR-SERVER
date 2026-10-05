@echo off
cd /d C:\logd-capture
aion-logd.exe -config capture.yaml > logd.log 2>&1

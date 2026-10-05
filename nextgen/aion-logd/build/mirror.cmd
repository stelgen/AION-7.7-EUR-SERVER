@echo off
cd /d C:\logd-capture
set AIONLOG_MIRROR_UP=127.0.0.1:2053
aion-logd-mirror.exe -config capture.yaml

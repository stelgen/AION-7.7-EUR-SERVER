#!/usr/bin/env bash
# agent-cli.sh — CLI-обёртка Agent API aion-op (S12). Source of truth = ЭТОТ ФАЙЛ в гите;
# рабочая копия в песочнице: ~/STELGEN/tmp/aion-agent.sh (обновляй из гита, не отдельно).
# Использование: bash -c 'source agent-cli.sh; aionrun "dir C:\aionop"'
#   aionrun "cmd"            — cmd на VM  → {exit_code,stdout,stderr,ms}
#   aionrun_ps "cmd"         — powershell на VM
#   aiongetb64 "D:/p" > out  — скачать файл (aionget — сырой JSON)
#   aionput LOCAL REMOTE     — залить файл
#   aionls "D:/SAION"        — ls
#   aionlog <svc> [N]        — хвост лога (svc = logs.files из конфига op)
# Токен: ~/.aion-agent-token (копия = D:\SAION\creds\CREDS.md на VM).
TOKEN_FILE="$HOME/.aion-agent-token"
AION=http://192.168.0.125:10200
aionhdr() { echo "X-Agent-Token: $(cat "$TOKEN_FILE")"; }
aionrun() { curl -s -m 180 -X POST "$AION/api/agent/run" -H "$(aionhdr)" -H 'Content-Type: application/json' -d "{\"cmd\":$(python3 -c 'import json,sys;print(json.dumps(sys.argv[1]))' "$1")}"; }
aionrun_ps() { curl -s -m 180 -X POST "$AION/api/agent/run" -H "$(aionhdr)" -H 'Content-Type: application/json' -d "{\"shell\":\"ps\",\"cmd\":$(python3 -c 'import json,sys;print(json.dumps(sys.argv[1]))' "$1")}"; }
aionget() { local p="$1"; curl -s -m 120 "$AION/api/agent/file?path=$(python3 -c 'import urllib.parse,sys;print(urllib.parse.quote(sys.argv[1]))' "$p")" -H "$(aionhdr)"; }
aiongetb64() { aionget "$1" | python3 -c 'import json,sys,base64;sys.stdout.buffer.write(base64.b64decode(json.load(sys.stdin)["content_b64"]))'; }
aionput() { local l="$1" r="$2"; local b64; b64=$(base64 -w0 "$l"); curl -s -m 300 -X POST "$AION/api/agent/file" -H "$(aionhdr)" -H 'Content-Type: application/json' -d "{\"path\":\"$r\",\"content_b64\":\"$b64\"}"; }
aionls() { curl -s -m 60 "$AION/api/agent/ls?path=$(python3 -c 'import urllib.parse,sys;print(urllib.parse.quote(sys.argv[1]))' "$1")" -H "$(aionhdr)"; }
aionlog() { curl -s -m 60 "$AION/api/agent/log?name=$1&tail=${2:-200}" -H "$(aionhdr)"; }
aionact() { # aionact start|stop|restart|restart_pair <svc> [confirm]
  local confirm=""; [ "$3" != "" ] && confirm=",\"confirm\":\"$3\""
  curl -s -m 180 -X POST "$AION/api/action" -H "$(aionhdr)" -H 'Content-Type: application/json' -d "{\"action\":\"$1\",\"id\":\"$2\"$confirm}"; }
aionstatus() { curl -s -m 15 "$AION/api/status"; }

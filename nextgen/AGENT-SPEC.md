# AGENT-SPEC — как агент-разработчик ходит в VM (стандарт, 07.10)

Роадмап R0–R4 выполнен. **SSH-консоль с PowerShell-кавычками больше НЕ инструмент
по умолчанию** — только крайний случай (загрузка нового бинаря op, если API лежит).

## Каналы

| Канал | Что | Когда |
|---|---|---|
| **Agent API** (основной) | HTTP/JSON на `http://192.168.0.125:10200/api/agent/*`, токен `X-Agent-Token` | все команды, файлы, логи |
| SSH `aion` (алиас) | ключ dimini-agent, юзер `Администратор` (кириллица!) | только деплой самого op |
| op UI | `http://192.168.0.125:10200` | человек (юзер) |

## Токен

- Значение: `D:\SAION\creds\CREDS.md` на VM (единое место по политике) + у агента
  в песочнице `~/.aion-agent-token` (chmod 600). В гит/память НЕ класть.
- Env-альтернатива на VM: `AIONOP_AGENT_TOKEN`.

## API

```http
POST /api/agent/run   {"cmd":"...","shell":"cmd|ps","timeout_sec":120,"cwd":"D:/..."}
                      → {"exit_code","stdout","stderr","truncated","ms","timeout"}
GET  /api/agent/file?path=&max_mb=4   → {"path","size","truncated","content_b64"}
POST /api/agent/file  {"path","content_b64","append":false} → {"ok","bytes"}
GET  /api/agent/ls?path=              → {"entries":[{name,dir,size,modified}]}
GET  /api/agent/log?name=<svc>&tail=  → {"path","lines"}   # svc = logs.files из конфига
```

- Лимиты: тело run ≤1МБ, файл ≤192МБ (put), чтение по умолчанию 4МБ (`max_mb` до 256),
  stdout/stderr капа 8МБ (`agent.max_out_mb`), таймаут ≤900с, параллельно ≤2 exec.
- Токен constant-time; каждый вызов пишется в лог op (`AGENT ... ok=...`).
- `shell:"cmd"` — cmd /c; `"ps"` — powershell -NoProfile -NonInteractive.
- По таймауту дерево процессов добивается `taskkill /F /T`.

## Готовая обёртка (в песочнице агента)

```bash
source ~/STELGEN/tmp/aion-agent.sh
aionrun "tasklist /fo csv /nh"
aionrun_ps "Get-Process Server64"
aionls "D:/SAION"
aiongetb64 "D:/AION_LIVE_SERVER/MainServer/config.xml" > config.xml
aionput ./file.txt "D:/SAION/tmp/file.txt"
aionlog main 100
```

## Правила

1. Новый шаг на VM = скрипт через `aionput` + `aionrun "powershell -File ..."`
   (НЕ инлайн-команды: экранирование/кодировки = источник ошибок).
2. ps1-файлы для VM — ASCII-only или UTF-8 с BOM; в репо — латиница.
3. Замена бинаря op: scp новым именем → taskkill → backup → rename → schtasks /run
   (см. DEPLOY.md); бинарь копится локально `GOOS=windows go build` из nextgen/aion-op.
4. Firewall: вход 10200 разрешён только из `192.168.0.0/24` (правило `aionop-agent-10200`).
5. R1 (JSON-режим op-status/op-act.ps1) поглощён Agent API — хелперы остаются
   человеко-читаемыми, НЕ дублировать.

## История

- 07.10: Agent API в бою (aionop-win.exe, PID проверен, exit_code/stdout JSON, 403 без токена,
  файлы/логи/ls прогнаны). Бэкапы старого: `C:\aionop\aionop-win.bak-*.exe`, `config-vm.yaml.bak-*`.

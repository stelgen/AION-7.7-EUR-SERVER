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

## Готовая обёртка

Source of truth = `nextgen/agent-cli.sh` (в гите); рабочая копия в песочнице:
`~/STELGEN/tmp/aion-agent.sh` (если отсутствует — скопировать из гита, не писать заново).

```bash
bash -c 'source ~/STELGEN/tmp/aion-agent.sh'
aionrun "tasklist /fo csv /nh"
aionrun_ps "Get-Process Server64"
aionls "D:/SAION"
aiongetb64 "D:/AION_LIVE_SERVER/MainServer/config.xml" > config.xml
aionput ./file.txt "D:/SAION/tmp/file.txt"
aionlog main 100                     # хвост лога по svc из logs.files
aionact restart captcha restart      # управление стеком через op (POST /api/action)
aionstatus                           # GET /api/status
```

⚠️ **aionput и большие файлы** (инцидент 10.10, деплой gate exe 5.95МБ): обёртка кладёт
base64 в АРГУМЕНТ curl → `Argument list too long` (лимит ядра на один argv-элемент).
Тот же Agent API через JSON-файл + `--data-binary`:
```bash
python3 -c 'import json,base64; json.dump({"path":"D:/SAION/x/x.exe","content_b64":base64.b64encode(open("x.exe","rb").read()).decode()}, open("put.json","w"))'
curl -s -m 300 -X POST "$AION/api/agent/file" -H "$(aionhdr)" -H 'Content-Type: application/json' --data-binary @put.json
# проверить: ответ {"bytes":N} == размер локального файла
```

## Мульти-вантадж и диагностика канала (08.10, инцидент «VM недоступна»)

- **Firewall: RemoteAddress = Any (0.0.0.0)** — решение юзера 08.10: VM за NAT, наружу
  10200 не проброшен, единственный рубеж = токен (правило `aionop-agent-10200`).
  Агенты с ЛЮБОГО вантаджа (LAN, WG 10.10.x, docker 172.16/12, чужие песочницы) — могут.
- **Audit-лог op** = `C:\aionop\op.log` (run.cmd redirect, fix 08.10): каждая попытка
  видна строкой `AGENT <method> <url> <ip:port> ok=...` — инцидент «недоступно/403»
  диагностируется пост-фактум (было: stdout задачи терялся).
- **Если у агента «VM недоступна»** — сначала диаг-пакет в СВОЕЙ песочнице (не вывод
  «VM мертва»): `curl -v --max-time 5 http://192.168.0.125:10200/api/status`; `ip -4 addr; ip route | head -5`;
  проверить env-прокси (`env | grep -i proxy`) — если HTTP(S)_PROXY глобальный,
  добавить `NO_PROXY=192.168.0.0/16,10.10.0.0/16,localhost` (LAN не должен идти в интернет-прокси);
  проверит токен (`ls ~/.aion-agent-token`); у упавшего = если в `op.log` нет его IP → сеть,
  есть `ok=false` → токен.
- **Токен-бутстрап** в новой песочнице: файла нет → получить по ssh (`ssh aion "type D:\SAION\creds\CREDS.md"`)
  или у юзера; НЕ выдумывать и не дублировать в гит/память.

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

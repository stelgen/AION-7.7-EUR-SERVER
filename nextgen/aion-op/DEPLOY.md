# 🚢 Деплой/обновление aion-op на VM — runbook (работает Dimini)

> Развёрнуто 05.10.2026. aion-op живёт на VM как SYSTEM-задача **AionOp** (onstart).
> ЗОНА БЕЗОПАСНОСТИ: трогаем ТОЛЬКО `C:\aionop\` и задачу AionOp. Игровой стек не трогать
> (пробы read-only: tasklist/netstat/quser/Get-Process/Get-Content).

## Текущее состояние прод-деплоя

| Параметр | Значение |
|---|---|
| Каталог на VM | `C:\aionop\` (aionop-win.exe, config-vm.yaml, run.cmd, aionop.db) |
| Задача | `AionOp` — schtasks, `/ru SYSTEM /sc onstart /rl HIGHEST`, автозапуск при буте |
| Режим | `vm.mode: local` (пробы локально на VM), `operator.mode: observe` |
| Безопасность | `bind: 127.0.0.1:10200` (доступ только через ssh-туннель), `dry_run: true` (действия только планируются), POST /api/action отсутствует в observe |
| pprof | `127.0.0.1:10201/debug/pprof/` |
| Данные | `C:\aionop\aionop.db` — SQLite WAL, retention 30 дней |

## Доступ с моей машины (Linux)

```bash
# UI через туннель:
ssh -L 10200:127.0.0.1:10200 -L 10201:127.0.0.1:10201 'Администратор@192.168.0.125' -N &
# → http://127.0.0.1:10200 (локально в браузере/через curl)

# статус одной командой:
ssh 'Администратор@192.168.0.125' "curl -s --max-time 8 http://127.0.0.1:10200/api/status"
```

## Обновление aion-op (самообновление, стек НЕ трогается)

```bash
# 1. Сборка windows-бинаря (локально):
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o build/aionop-win.exe .
sed '0,/mode: mock/s//mode: local/' config.yaml > build/config-vm.yaml   # если менялся конфиг

# 2. Останов ТОЛЬКО нашего сервиса:
ssh 'Администратор@192.168.0.125' "cmd /c \"schtasks /end /tn AionOp & taskkill /F /IM aionop-win.exe 2>nul\""

# 3. Замена (tar-pipe) и старт:
tar -cf - -C build aionop-win.exe config-vm.yaml | \
  ssh 'Администратор@192.168.0.125' "cmd /c \"tar -xf - -C C:/aionop & schtasks /run /tn AionOp\""

# 4. Верификация:
ssh 'Администратор@192.168.0.125' "cmd /c \"netstat -ano -p tcp | findstr 10200 & curl -s --max-time 8 http://127.0.0.1:10200/api/status\""
```

## Переход в operate (когда юзер скажет «можно управлять»)

1. Создать недостающие /IT kill-задачи (пароль из реестра VM, НЕ сохранять нигде):
   `schtasks /create /f /tn AionKickGate /tr "taskkill /F /IM AuthGateD.exe" /sc once /st 00:00 /it /ru Администратор /rp <pwd> /rl HIGHEST` (и AionKickAuth).
2. На VM в `config-vm.yaml`: `operator.mode: operate` + `operator.dry_run: false` (явно!).
3. Обновить aion-op (см. выше) — POST /api/action смонтируется; кнопки в UI оживут.
4. Пока dry_run: true — действия возвращают план и пишут audit, но НЕ исполняются.
5. Первые реальные действия — только из юзер-сессии VM (десктоп), пара — только через restart_pair.

## Откат

```bash
ssh 'Администратор@192.168.0.125' "cmd /c \"schtasks /end /tn AionOp & schtasks /delete /f /tn AionOp & rd /s /q C:\aionop\""
```
(На стек не влияет вообще — aion-op сторонний наблюдатель.)

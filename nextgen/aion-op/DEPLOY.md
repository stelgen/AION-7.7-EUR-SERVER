# 🚢 Деплой/обновление aion-op на VM — runbook (работает Dimini)

> Развёрнуто 05.10.2026. aion-op живёт на VM как SYSTEM-задача **AionOp** (onstart).
> ЗОНА БЕЗОПАСНОСТИ: трогаем ТОЛЬКО `C:\aionop\` и задачу AionOp. Игровой стек не трогать
> (пробы read-only: tasklist/netstat/quser/Get-Process/Get-Content).

## Текущее состояние прод-деплоя

| Параметр | Значение |
|---|---|
| Каталог на VM | `C:\aionop\` (aionop-win.exe, config-vm.yaml, run.cmd, aionop.db) |
| Задача | `AionOp` — schtasks, `/ru SYSTEM /sc onstart /rl HIGHEST`, автозапуск при буте |
| Kill-задачи | AionKickGate/AionKickAuth — /IT созданы 05.10 (пароль из реестра Winlogon, нигде не сохранён); AionKickMain/AionKickNPC2 были; AionCAPTCHA — ре-enable 05.10 (был найден Disabled) |
| Режим | `vm.mode: local` (пробы локально на VM), `operator.mode: operate` + `dry_run: false` — реальное управление включено 05.10 (решение юзера) |
| Безопасность | `bind: 0.0.0.0` (решение юзера: стек в локалке, наружу не торчит — NAT закрыт), `dry_run: false` + `operate` (реальные действия с confirm=restart на опасных; см. README «OP-FIRST») |
| pprof | `127.0.0.1:10201/debug/pprof/` |
| Данные | `C:\aionop\aionop.db` — SQLite WAL, retention 30 дней |

## Agent API (07.10, R2) — ОСНОВНОЙ канал агента

`/api/agent/{run,file,ls,log}` на :10200, токен `X-Agent-Token` (конфиг `agent:` /
env `AIONOP_AGENT_TOKEN`). Спека + обёртка: [../AGENT-SPEC.md](../AGENT-SPEC.md).
Firewall: правило `aionop-agent-10200` (TCP 10200 только из 192.168.0.0/24) — добавлено 07.10,
до этого UI ходил только через туннель.

```bash
# статус одной командой:
ssh 'Администратор@192.168.0.125' "curl -s --max-time 8 http://127.0.0.1:10200/api/status"
```

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

## SQL-наблюдение (aionop_ro)

- На VM создан SQL-логин `aionop_ro` (db_datareader в Aion_log+AionAccounts, VIEW SERVER STATE; DDL: `C:\aionop\sql-setup.sql`). Пароль НЕ хранится в гите/памяти — только в `C:\aionop\config-vm.yaml` (поле sql.conn) и локальном `build/` (gitignored).
- Вкладка SQL: CCU (миры+auth), blocked, compile-очередь (RESOURCE_SEMAPHORE — root ночи 04–05), top-waits дельты.
- Ротация пароля: перегенерить python-рендером conn → обновить config-vm.yaml на VM → рестарт AionOp.

## Статус operate (05.10)

- Кнопки оживут в UI (POST /api/action смонтирован): start — все не-locked; stop/restart — где есть kill_task;
  restart_pair — только парой, с typed-confirm «RESTART PAIR», заблокирован в окне загрузки (conns<16) и при деградации.
- E2E-тест на проде 05.10: `restart captcha` = OK/exec (taskkill → schtasks /run AionCAPTCHA → новый PID 6968, порт 22206 вернулся); reject-тесты: restart_pair без confirm → отказ; locked chat → отказ; пара осталась нетронутой (8/16).
- Готча исполнения: шаги-пометки («пауза Nс», «ждать …») в планах пропускаются как не-команды; fail-fast по шагам с записью в audit.

## Переход в operate (выполнено 05.10)

1. ~~Создать недостающие /IT kill-задачи~~ — созданы: AionKickGate, AionKickAuth (пароль из реестра, НЕ сохранён нигде).
2. ~~config-vm.yaml: operate + dry_run: false~~ — включено 05.10.
3. ~~Обновить aion-op~~ — сделано; POST /api/action работает, UI-кнопки активны.
4. Первые реальные действия — только лёгкие сервисы; пара — только через restart_pair с confirm; **мир/auth не рестартовать пока юзер в игре**.

## Откат

```bash
ssh 'Администратор@192.168.0.125' "cmd /c \"schtasks /end /tn AionOp & schtasks /delete /f /tn AionOp & rd /s /q C:\aionop\""
```
(На стек не влияет вообще — aion-op сторонний наблюдатель.)

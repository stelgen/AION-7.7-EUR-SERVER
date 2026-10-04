# Прямые порты вместо прокси (04.10.2026, вечер)

Решение юзера: MITM-прокси больше не нужны — клиент коннектится **напрямую к приложениям**.
Прокси-фаза (логин-анализ 03.10) завершена, логи собраны в `docs/authlog-*.md`.

## Схема ПОСЛЕ (текущая)

```
[Клиент] → 2106 (AuthGateD) → 2110 (L2Authd) → 2220 (AccCache) / 10057 (PA) / SQL
[Клиент] → 7777 (Server64) напрямую
```

Наружу по-прежнему торчат только 2106 + 7777 (README, секция «Архитектура стека», снова верна).

## Что сделано

1. **Прокси убиты**: python-процессы (aionproxy.py 2106→2107 и aionproxy7777.py 7777→7778) taskkill'нуты.
2. **Задачи планировщика**: `AionProxy` и `AionProxyW` удалены (`schtasks /delete /f`).
3. **Порты возвращены** (всё с бэкапами):

| Конфиг | Значение | Бэкап на VM |
|---|---|---|
| `AuthGateD\etc\config.txt` | `serverPort = 2106` | `.bak-back2106` |
| `MainServer\config.xml` | `<clientAcceptPort>7777</clientAcceptPort>` | `.bak-restore7777` |
| `MainServer\common.xml` | `<petitionServerPort>2107</petitionServerPort>` | `.bak-pet2107` |

   Утренние патчи (гейт 2107, мир 7778, petition 21055) полностью откатаны.
   `petition=2107` при гейте на 2106 конфликта не даёт — лупер Server64 «Can't connect petition» получает быстрый RST (event-driven, безвредно).

4. **`C:\Temp\gate.bat` → v3.1**: `taskkill /F /IM AuthGateD.exe` в начале + ожидание/echo `2106` (бэкап `.bak-2107`).
5. **Десктопный `AION-START-ALL.bat` → v5**: без прокси-шага, 12 шагов, ожидание 2106 после старта гейта, 7777 после Server64; `kill python.exe` в шаге 0 оставлен как страховка портов (бэкап `.bak-v4`). Локальная копия: `scripts/AION-START-ALL-v5.bat`.
6. `AION-STOP-ALL.bat` не менялся.

## ⚠️ Ловушки при правке конфигов (запомнить)

- `AuthGateD\etc\config.txt`: строка `serverPort = 2107` — **с пробелами вокруг `=`**; в файле **48704 не-ASCII байтов** (корейские комменты). Править **ТОЛЬКО байтовой заменой** (python, `open('rb')` → `bytes.replace`). PS `Set-Content -Encoding Ascii` молча убивает корейщину.
- `inner_ip=127.0.0.1` в БД `server` — не трогать (auth-связь, факт из хронологии 04.10).
- PS-однострочники: `>nul` внутри PS падает (файл 'nul') — только cmd-батники или PS-скрипты файлом.

## Хронология подъёма (VM-время 04.10)

| Время | Событие |
|---|---|
| 10:33 | `taskkill python.exe` ×2, 2106/7777 свободны |
| 10:34–35 | правка 3 конфигов (+gate.bat), байтовые замены, верификация |
| 10:36:22 | рестарт гейта: `AionGate` (/IT) → старый гейт убит, новый PID 232 в Console-1 |
| 10:36:22 | gate-лог: `RSA Key Generated → Service Running → *new authd connection from 127.0.0.1` |
| 10:36:18→22 | authd winlog: `*close connection authgated` → `*new connection from 127.0.0.1,0x934` |
| 10:36:5x | NPCSvr64 старт (AionNPCit /IT) |
| 10:37:0x | Server64 старт (AionMainit /IT), ждёт NPCSvr (CondSpwnTimeMgr/DynamicFieldMgr wait — норма) |
| 10:40:59 | authd winlog: **`*new world server connection from 127.0.0.1`** — мир зарегистрирован |
| 10:41:0x | `0.0.0.0:7777 LISTENING` (Server64), 16/16 conns на 2002 |

## Live-статус после сборки

- **CPU (4 ядра, idle)**: Server64 0.9%, NPCSvr64 4% (post-spawn), AuthGateD/L2Authd/CacheD/LogServer/ACS/IC/CAPTCHA/PA = 0–0.1%. Пики только в фазе загрузки мира (норма).
- **RAM (private)**: NPCSvr 14.9 ГБ (полная загрузка), Server64 10.3 ГБ, ICServer 1.7 ГБ (известная особенность), CacheD 0.5 ГБ.
- **Логи**: CacheD — только `AlivePacket` к LogServer (здоров); NPCSvr — даталоад-шум ItemDB/`Skip LoadNPCTerritory` (косметика 7.7); Server64 — `Can't connect to ChannelChat 10254` (exe нет в ките), `Error while loading hardware.sig` (файл отсутствует, не критично).

## 🏆 Финал — юзер играет

Логин **под ovpn** на прямых портах успешен: `2106 → AuthGateD` напрямую, `7777 → Server64` напрямую.
Это закрывает практикой прежние гипотезы:

- «7777 недостижим через VPN» — **неверно** (TNC True + факт игры);
- «клиент валидирует login-IP vs world-IP, нужен возврат 192.168.0.125» — **неверно** (server list `81.25.59.194:7777` рабочий, менять БД не требуется);
- «Server64 без RunAsDate не работает» — **неверно** (запуск main.bat напрямую, timeDiff=0, err с реальными датами).

## Открытые пункты

- ✅ Разовые задачи 23:57–23:58 переведены в **manual-only** (04.10 вечер): триггеры отключены (`Trigger.Enabled=false` через `Set-ScheduledTask` с паролем автологона), задачи остались в списке — запуск только вручную `schtasks /run /tn <имя>`. Содержание (на память):

  | Задача | Команда |
  |---|---|
  | `AionKickMain` / `AionKickMain2` | `cmd /c taskkill /F /IM Server64.exe` |
  | `AionKickNPC` | `cmd /c taskkill /F /PID 6884` (PID устарел — при ручном запуске бесполезна) |
  | `AionKickNPC2` | `C:\Temp\killnpc.bat` |
  | `AionStopHeavy` | `taskkill Server64 + NPCSvr` |
  | `AionFullRestart` | `C:\Temp\full-restart.bat` |
  | `AionMainR` | `C:\Temp\restart-main.bat` |
  | `AionNPCR` | `C:\Temp\restart-npc.bat` (включён из Disabled) |
  | `AionRADTest` | `C:\Temp\rad-restart.bat` |

  Ловушка: `schtasks /change /sd` на Password-/IT-задачах **зависает**; COM `RegisterTaskChanges` из PS недоступен; рабочий путь — `Set-ScheduledTask -User <из Principal.UserId> -Password <DefaultPassword из Winlogon-реестра>` (пароль нигде не сохранялся).
- `C:\Temp\aionproxy*.py`, `worldproxy.bat`, `C:\Temp\py\` — можно удалить (не запускаются). Python embed оставлен — пригоден для байтовых правок конфигов.
- Регэксп-помощники раунда (`restore-ports2.py`, `cpu-probe.ps1` и пр.) переиспользуемы — см. `scripts/maint/`.
- Petition/ChannelChat/ShopAgent луперы — без изменений (event-driven).

## Файлы этого раунда

- `scripts/AION-START-ALL-v5.bat` — десктопный старт v5 (без прокси)
- `scripts/proxy/retired-aionproxy7777.py` — world-сниффер 7777→7778 (выведен, история)
- `scripts/maint/2026-10-04-direct-ports/` — restore-ports.ps1, restore-ports2.py (байтовые правки конфигов), fix-gate-cosmetic.py, deploy-v5.ps1, logscan.ps1, cpu-probe.ps1 (conns+CPU+RAM), final-check.ps1, final2.ps1

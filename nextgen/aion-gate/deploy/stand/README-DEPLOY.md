# Деплой STAND aion-gate (порт 2109) — решающий тест живым клиентом

Прод (`AuthGateD.exe` PID на :2106, `D:\AION_LIVE_SERVER\AuthGateD\`) НЕ трогается.
Стенд = `D:\SAION\aion-gate-stand\`, задача `AionGateStand`, порт **2109**, authd реальный (127.0.0.1:2110).
⚠ Порт 2108 ЗАНЯТ самим L2Authd (второй листенер!) — НЕ использовать.

## Состав бандла
| Файл | Что |
|---|---|
| `aion-gate.exe` | кросс-сборка `GOOS=windows GOARCH=amd64` (ea52a69, тесты зелёные) |
| `config-stand.yaml` | зеркало config.txt 41007 (прод-значения) + serverPort 2108 + authReconnectInterval 30 |
| `gate-stand.bat` | запуск (ASCII+CRLF): `aion-gate.exe -config config-stand.yaml >> gate-stand.log` |
| `etc/BlockIPs.txt` | пустой блок-лист |
| `README-DEPLOY.md` | этот док |

## Сборка exe (локально)
```bash
cd nextgen/aion-gate && GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o aion-gate.exe .
```

## Деплой (выполнен 06.10 ~00:15, порт 2109)
```bash
ssh Администратор@192.168.0.125 "cmd /c mkdir D:\SAION\aion-gate-stand\etc"
scp aion-gate.exe config-stand.yaml gate-stand.bat README-DEPLOY.md 'Администратор@192.168.0.125:D:/SAION/aion-gate-stand/'
scp etc/BlockIPs.txt 'Администратор@192.168.0.125:D:/SAION/aion-gate-stand/etc/'
schtasks /create /f /tn AionGateStand /tr "D:\SAION\aion-gate-stand\gate-stand.bat" /sc onstart /ru SYSTEM /rl highest
schtasks /run /tn AionGateStand
```

## ✅ SELF-TEST (выполнен 06.10 ~00:10 с LAN-машины)
- TCP 192.168.0.125:2109 → welcome len=194, framing ✓
- ECB-dec(key1): dword0 = 0x00000100 → **[0]=0x00 ✓, sid=1 ✓**, хвост 188..191 = 00000000 ✓
- 34b мусор → ответ 42b (echo-канал handleAuthGG жив)
- asm-модель подтверждена end-to-end на живом стенде

## Решающий тест живым клиентом
1. Клиент направить на `192.168.0.125:2109` (клиентский override логин-сервера).
2. Критерий УСПЕХА (asm-модель верна): клиент **не рвёт** соединение после welcome,
   шлёт AUTH_GG (34b), получает 42b `[sid][28×0]`, идёт дальше (логин/сервер-лист).
3. Критерий ПРОВАЛА: клиент закрывает соединение сразу после welcome (или с ошибкой)
   → парадокс §7 не в capture: копать рантайм-ключ/provenance capture (см. док ночь-4).
4. Лог: `D:\SAION\aion-gate-stand\gate-stand.log`; события authgg.mismatch/authgg.blob в лог.

## Откат (полный)
```bat
schtasks /end /tn AionGateStand
schtasks /delete /f /tn AionGateStand
rd /s /q D:\SAION\aion-gate-stand
```
Прод-гейт при этом никогда не выключался — откат мгновенный и без последствий.

## Примечания
- `welcomePlainByte` в yaml больше не используется (asm: plaintext[0]=0x00 константа).
- Если живой клиент пройдёт welcome → следующий этап: ретаргет задачи AionGate по плану §6
  (`D:\SAION\aion-gate`, aion-op config, откат `C:\Temp\gate.bat`).
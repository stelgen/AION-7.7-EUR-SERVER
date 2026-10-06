# aion-gate — замена AuthGateD для AION 7.7 EU (Go)

**РЕЛИЗ 07.10.2026.** Прод: `192.168.0.125:2106`, `mode: authgate`, полный живой флоу
доказан на двух аккаунтах (`1/1` → accId 7; `stelgen` → accId 1010):
`welcome 194B → AUTH_GG 42b → CM_LOGIN → blob cbdb → authd type=3 → [03]74Б →
[05] relay → [04]42Б → [02] relay → [07]26Б → мир 7777`.

Версия релиза: коммит **7711567** (exe sha256 `e159d52c…`). Откат: `aion-gate.exe.bak-f43e350`.

## ✅ Канонические факты протокола (все live-доказаны, НЕ трогать)

| Факт | Значение |
|---|---|
| Welcome | 194B = `[u16 0xC2][ECB(key1) 192]`; pt 173+4: `[0]=0x00, [1:5]=fc=sid, [5:9]=V (authd [03]), [9:137]=ScrambleModulusServer(N), [137:153]=GG-нули, [153:169]=key2, [168:172]=65650072, [172]=0x00` |
| Крипта | Blowfish канон; key1 = LUT(0x04bd) static `6b60cb5b…`; EncryptPrimary (cumsum, csum@n', dword0 не трогается); EncryptSecondary/DecryptSecondary @0x417a80/0d |
| RSA | RSA-1024, **e=65537** (доказано оракулом m^65537 mod N == ct), пул ключей, серверный скрамбл swap→xorL→dword→xorU |
| CM_AUTH_GG | клиент 34b = EncryptSecondary([sid][20Б]); ответ 42b = EncryptSecondary([0b][sid][27×0]) |
| CM_LOGIN | **op=0x00 у живого EU-клиента** (эталон Mobius = 0x0B — НЕ для leak-клиента!); pt = [op][ct 128×k][tail ≤64]; loginex k≥2: user=combined[78:142], pwd=combined[206:238] (реально на [220:230]), otp=[238:242]=0 |
| Blob в authd | **asm-форма 191Б**: `[00][decbuf34=user14+pwd16+otp4][dword=pt[148:152]][tail=pt[152:] (152Б)]` — ровно как оригинал; authd игнорит содержимое dword/tail, но требует asm-размер (86Б → тишина!) |
| Релей authd (2110) | `[00][sid][IP-be]` / `[01][sid]` / `[02][sid][len][blob]`; назад `[03][sid]` (V=0xc621), `[02][id][len][type][payload]`; **клиентский опкод = тип authd** (prepend): type=3→`[03]+pad64`=74Б, type=4→42b (pt 32), type=7→26b (pt 16) |
| State-машина | CONNECTED{0x07→authgg, 0x08→UPDATE_SESSION relay} → AUTHED_GG{0x00‖0x0B→login} → AUTHED_LOGIN{0x05,0x02→relay}; unknown = лог, НЕ cc45 |
| Username | `TrimSpace+ToLower` перед decbuf (К-5) |
| LoginFail | cc/SM_LOGIN_FAIL = messageId AionAuthResponse: 0,1,2,4,5,6,7,8,10,11,12,15,16,18,21,22 (22=BAN_IP «заблокирован» — live); 45 = authgate-спец |

## 🔧 Режимы (`mode` в config.yaml)

- **authgate** — наш флоу 7.7 (ПРОД).
- **fork** — прозрачный прокси `клиент ↔ наш гейт ↔ ориг AuthGateD (forkOrigAddr:Port)`,
  байт-в-байт + shadow-сверка (FORK-строки в логе: скрамбл-модель, SAME/DIFF ответов).
  Использовался для сверки raw-байт: welcome/authgg = SAME vs ориг.
- **classic** — beyond-aion 4.8 флоу (standalone, SM_INIT-210).

## 🚀 Деплой

```bash
cd nextgen/aion-gate && go vet ./... && go test ./... \
  && GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o aion-gate.exe .
scp aion-gate.exe 'Администратор@192.168.0.125:D:/SAION/aion-gate/aion-gate.exe'
ssh Администратор@192.168.0.125 "cmd /c 'schtasks /end /tn AionGate & taskkill /F /IM aion-gate.exe'"
ssh Администратор@192.168.0.125 "cmd /c 'schtasks /run /tn AionGate'"
```

⚠️ Готчи: exe залочен живым процессом (kill до scp); после kill первый `/run` может
словить bind-fail — bat ретраит ~35с, проверять баннер; кириллический scp — только push
(pull через `ssh … powershell Get-Content`); L2Authd живёт ТОЛЬКО в session 1 (десктоп).

Верификация после деплоя: баннер `mode=authgate rsa_exponent=65537` + `RAW AUTHD A>G [03] 21c60000`
в `D:\SAION\aion-gate\gate-prod.log`; клиентский smoke — `cmd/forkprobe` (welcome-разбор +
AUTH_GG вердикт) или решающий логин.

## 🧪 Тесты и инструменты

- `go test ./...` — golden welcome/echo/фреймы + Фаза-1 (loginex-asm-blob, state-переходы,
  k=1 две раскладки 4.8/7.7, normalize, playOk Rnd, SplitLogin-op).
- `cmd/probe` — прямой клиент authd: `probe.exe 127.0.0.1:2110 stelgen 192.168.0.253 [sid] [tail47]`
  → type=3 = healthy (tail47 = blob-форма гейта 86Б, дефолт = asm 191Б).
- `cmd/forkprobe` — синтетический клиент 2106 (welcome→AUTH_GG→26b) с raw-вердиктами.

## ⚙️ Ключи конфига (config-prod.yaml на VM)

`serverPort:2106, authAddr/AuthPort:127.0.0.1:2110, mode:authgate, rsaExponent:65537,
loginDecbufLen:34, forkOrigAddr/Port:127.0.0.1:2109, worldIP:192.168.0.125, worldPort:7777,
welcomeTestCC:0, welcomeWaitAuthdMs:2000, serverID:1, smAuthGgWire:42 (live-форма;
50 = эталон Mobius для A/B), dumpPacket, blockIPsFile, tryInterval/Count/BlockInterval`.

## 📚 Доки

- `docs/fork-classic-deploy-20261006.md` — леджер поправок дня (№1–13: эталон ≠ live,
  asm-blob, session-1 стек) + runbook после ребута ВМ.
- `docs/mobius-77-flow-review-20261006.md` — сверка с эталоном Mobius (К-1..К-6, все закрыты;
  К-1 REVISED, К-3 live-поправка op=0x00).
- `docs/beyond-aion-48-…`, `docs/rsa-hunt-…` — АРХИВ (история гипотез, не актуальны).
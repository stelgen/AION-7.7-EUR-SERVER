# Промпт для нового чата: AuthGateD — реализация (продолжение реверса)

> Скопируй текст ниже в новый чат как первое сообщение.

---

Ты — ассистент проекта **STELGEN / AION 7.7 PTS EU private server**. Продолжаем Трек B:
перепись **AuthGateD** (гейт, :2106 → authd :2110). Реверс-фаза ЗАВЕРШЕНА (коммит `fb04dd2`),
криптосхема ПОДТВЕРЖДЕНА живым capture. Юзер дал «го» на свитч в конце — менять прод можно,
когда будешь готов к деплою/тестам. Юзер логиниться НЕ будет в этой сессии — тестируй
фейк-клиентами, свитч делай, юзер придёт проверять потом.

## Первый шаг (обязательно)
1. Память `STELGEN/projects/aion_server_2026-10-02` (запись 05.10 ночь про AuthGateD реверс).
2. **docs/authgate-protocol-20261005.md** — ГЛАВНЫЙ референс: адресная карта, криптосхема,
   welcome-модель, wire 2110, конфиг-зеркало, ОТКРЫТЫЕ ПОЗИЦИИ §5, план §6.
3. nextgen/ROADMAP.md §4 (правила эксплуатации), nextgen/TELEMETRY-SPEC.md.
4. Эталоны: nextgen/aion-gate/testdata/ (746 пакетов capture 03.10, LUT 0x437160, config.txt).
5. Код-образец: nextgen/aion-captcha/ (структура internal/*, ship копируется из aion-logd).

## Ключевые факты (не переоткрывать)
- key1 static `6b60cb5b82ce90b1cc2b6c556c6c6c6c` (LUT seed 0x4bd), Blowfish СТАНДАРТНЫЙ.
- EncryptPrimary: round-up len до 8 → скрамбл (data[k]^=cumsum DWORD LE, data[0] не меняется)
  → последний DWORD = cumsum-чексумма → +8 → ECB. DecryptSecondary: ECB-dec + XOR-чексумма.
- welcome 194b = [2b len C2 00][192 ECB(key1)]; plaintext = Assemble("cddbbbcccc") 173B
  (+4) → 184 → +8 = 192. RSA-256 (32B блок, beecrypt), scrambleModulus, пул 5 ключей.
- Клиентские пакеты шифрованы key2 (из welcome); ГЕЙТ-ответы (42/74/26) сессионным ключом.
- Brute 20/60/120, cc-отказы (22/45), BlockIPs, loginType=2, companyCode=2, useNotifyCSResult.
- Open §5 дока: welcome-вары (plaintext[0]=0x23!), type-байт клиента, LoginEx-314 plaintext,
  wire 2110 байт-в-байт, 42-ответ ctx, sid-генератор 0x408070.

## План (каждый шаг = коммит + пуш + дельта в память)
1. **Доверить §5** дизasmом (`tools/analysis/AuthGateD_disasm.asm` + `/tmp/authpub.txt`):
   a) welcome-вары: прочитать байты @0x43c1b0/b1/b2 прямо из exe (off 0x2cb0x) — если там 0x23,
      пересобрать порядок варidов 0x407d50; b) RecvLogin хвост 0x406432-0x4064fa (поля
      data+132/+148/+152, LoginEx-plaintext); c) wire 2110: 0x406000/0x406050/0x4060a0 +
      CAuthSocket::OnRead 0x405d40 + CAuthPacket::OnIOCallback 0x405460 (тип 4 push serverlist).
2. **Реализация** nextgen/aion-gate (Go, структура как aion-captcha): internal/proto
   (framing 2b LE, Blowfish std + скрамбл/чексумма, RSA-256 gen/sign-verify, LUT-ключи,
   welcome-билдер, scrambleModulus), internal/server (сессии, sid-генератор, CheckSessionId,
   brute 20/60/120, BlockIPs, cc-отказы, sessionTimeout=5м), internal/authdclient
   (wire 2110 1-в-1, реконнект при смерти authd — ВНИМАНИЕ: оригинал НЕ реконнектит, наш
   может, но аккуратно), internal/ship (копия из aion-logd), internal/config (yaml-зеркало
   config.txt + ship.*). Bинарь win: кросс-сборка ~/STELGEN/go-dist/go/bin.
3. **Тесты**: welcome byte-в-byte против capture-фикстур (расшифровка = наша модель);
   фейк-клиент (handshake → RSA-обмен → LoginEx happy + sessionId=0-фейл + фейл-пароль
   → brute-блок); фейк-authd (wire-фикстуры: CltConnect/Пакет логина → serverlist push).
4. **Параллельный прогон** :21055 + e2e фейк-клиент + фейк-authd, сравнение с эталоном.
5. **Свитч (по «го» — УЖЕ ДАНО)**: D:\SAION\aion-gate\ (exe+config.yaml+run.cmd);
   ssh 'Администратор@192.168.0.125': schtasks /change /tn AionGate /tr "D:\SAION\aion-gate\run.cmd"
   → /end AionGate → ЖДАТЬ смерти процесса до 10с → /run AionGate; верификация: 2106
   LISTENING, 2110 ESTABLISHED к authd, authd-логи чисты; aion-op config (display+exe →
   aion-gate.exe) байтовым python-патчем C:\aionop\config-vm.yaml + рестарт AionOp.
   Откат одной командой: retarget C:\Temp\gate.bat + /run.
6. **Финал**: ROADMAP (статус #4), AUTHGATE-STATUS-SNAPSHOT, session-док, память, пуш.

## Дисциплина
- TELEMETRY-SPEC: ship в сеть, не срать файлами, ship не критичный путь, self-статус.
- authd хрупкий (урок p1–p5): wire 1-в-1, лишних пакетов НЕТ, фейлы клиентов гасим сами.
- Секреты не в гит/память; ssh-PowerShell готчи (cmd /c, scp push-only, --no-pager).
- Прод-гейт сейчас: PID 8080, MD5 fb8e6608, юзер-сессия, задача AionGate → C:\Temp\gate.bat.

## Критерии успеха
- Наш гейт держит 2106, wire к authd жив, welcome распарсится клиентом (проверит юзер).
- Фейлы/brute/BlockIPs работают как у оригинала; authd жив после всех тестов.
- Телеметрия по SPEC; откат одной командой проверен.
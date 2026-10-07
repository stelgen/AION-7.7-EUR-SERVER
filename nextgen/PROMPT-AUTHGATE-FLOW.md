> ⚠ АРХИВ — задача ЗАКРЫТА (компонент в бою/релизе). Документ сохранён для истории. Открытые промпты: PROMPT-ACCACHE.md, PROMPT-AUTHD.md, PROMPT-CACHED.md, PROMPT-ICSERVER.md (шаблон: README-TEMPLATE.md).

# PROMPT — Фаза 1: закрытие флоу aion-gate по эталону Mobius_AionEmu 7.7

> Скопировать целиком в новый чат. Ревью-док: nextgen/aion-gate/docs/mobius-77-flow-review-20261006.md (КОСЯКИ К-1..К-6 там).

```text
Проект: nextgen/aion-gate (Go) — замена AuthGateD для AION 7.7 EU leaked-стека.
Статус: e=65537 доказана, серверный скрамбл доказан, прод на нашем гейте (mode:authgate :2106),
блокеров RSA больше нет. Осталось закрыть живой флоу по ревью против эталона.

ШАГ 0 — артефакты (сделать первым):
  git clone --depth 1 https://github.com/MobiusDevelopment/Mobius_AionEmu /tmp/m77
  Эталон: java/com/aionemu/loginserver/network/** (LoginConnection, ncrypt/*, clientpackets/*,
  serverpackets/*, factories/AionPacketHandlerFactory.java, SessionKey.java, AionAuthResponse.java).
  Код эталона НЕ копировать — реализовывать в Go по ревью-доку
  nextgen/aion-gate/docs/mobius-77-flow-review-20261006.md (там таблица сверок и 6 косяков К-1..К-6).

ПРАВКИ (порядок P0→P2):
  P0-1 (К-1, КРИТИЧНО): internal/server/server.go handleLogin — relay-хвост при loginex.
    Сейчас dword148=pt[148:152], tail=pt[152:] — для k≥2 это зона ШИФРТЕКСТА (ct=pt[1:1+k*128]).
    Фикс: tail брать из proto.SplitLogin (уже возвращается), dword:
    k=1 → как раньше (asm: data+148); k≥2 → dword из tail (позиция = соответствие live-хвосту
    [sid LE][нули][0x20][68ffdab3e2fda892][2d9cc7baa87e0d49][00000000]; кандидат = dword после
    0x20-блока ИЛИ константный 0 при неоднозначности — логировать ВСЕ кандидаты в первый прогон).
    blob = Assemble("cbdb", 0, decbuf, dword, tailFromSplit). НЕ рвать при сомнении — relay.
  P0-2 (К-2): state-машина. Session.State = CONNECTED|AUTHED_GG|AUTHED_LOGIN
    (CONNECTED→AUTHED_GG после handleAuthGG-эхо; →AUTHED_LOGIN после успешного login-decode).
    Диспетчер по (state,op) как AionPacketHandlerFactory: CONNECTED{0x07→authgg,0x08→UPDATE_SESSION
    relay с type=0x08 ИЛИ лог+relay-raw}, AUTHED_GG{0x0B→login}, AUTHED_LOGIN{0x05,0x02→relay}.
    Прочее = лог "unknown packet state=%s op=0x%02x" (НЕ cc45). Длины 32/24/≥184 оставить fallback.
  P1-3 (К-3): SplitLogin — вернуть op (pt[0]) и логировать; в AUTHED_GG вход в login только по
    op==0x0B (эталон 7.7), иначе лог+raw-relay.
  P1-4 (К-4): proto/login.go DecodeLoginPlain — при len(ms)==1: сначала 4.8-раскладка
    (@94/@108/otp@124), при не-printable — 7.7-эталон: user=m[64:96](32), pwd=m[96:128](32),
    otp=LE m[124:128]. Обе гипотезы в лог.
  P1-5 (К-5): username перед BuildLoginDecbuf: strings.TrimSpace(strings.ToLower(user))
    (эталон .trim().toLowerCase(); authd авто-создание чувствительно к регистру).
  P2-6 (К-6): build26ReplyPt эмуляция — playOk1/playOk2 = crypto/rand u32 (не хардкод 1/1010),
    serverId из config; checkLogin-семантика против Session key.
  P2-7: doc — добавить в README реестр кодов LoginFail = AionAuthResponse messageId:
    0 AUTHED,1 SYSTEM_ERROR,2 INVALID_PASSWORD,4 FAILED_ACCOUNT_INFO,5 FAILED_SOCIAL_NUMBER,
    6 NO_GS_REGISTERED,7 ALREADY_LOGGED_IN,8 SERVER_DOWN,10 NO_SUCH_ACCOUNT,11 DISCONNECTED,
    12 AGE_LIMIT,15 SERVER_FULL,16 GM_ONLY,18 TIME_EXPIRED,21 ALREADY_USED_IP,22 BAN_IP
    (cc-канал шлёт эти же коды; 22 = «аккаунт заблокирован» — live-подтверждено).
  P3 (опция): конфиг smAuthGgWire42|50 — эталонная форма SM_AUTH_GG (D sid+B35, wire 50)
    рядом с live-формой 42b. Default 42 (live-принят).

ТЕСТЫ (добавить, старые не удалять):
  1. TestLoginRelayTailLoginex: pt-304 (op+2 ct-чанка+tail 47) → blob содержит tail ИЗ ХВОСТА,
     dword НЕ из ct-зоны (assert байты blob vs pt[148:152] ≠ при известном ct-филлере).
  2. TestStateMachineTransitions: CONNECTED 0x07→AUTHED_GG; AUTHED_GG 0x0B→login→AUTHED_LOGIN;
     0x08 в CONNECTED → relay/raw (не cc45); unknown op → лог, соединение живо.
  3. TestSplitLoginOp: op возвращается, ct/tail не смещаются при op=0x0B.
  4. TestDecodeK1TwoLayouts: m с user@64 → детект 7.7-раскладки; m с user@94 → 4.8.
  5. TestUsernameNormalize: " SteLGeN " → "stelgen" в decbuf.
  6. TestSessionKeyRnd: два логина в эмуляции дают разные playOk1/2.
  Существующие golden-тесты welcome/echo/фреймов должны остаться зелёными.

ОГРАНИЧЕНИЯ:
  - welcome 194B/EncryptPrimary/key1, SM_AUTH_GG 42b, relay-контракт (prepend типа authd),
    DecryptSecondary-до-SplitLogin, ScrambleModulusServer, e=65537 — НЕ ТРОГАТЬ (все доказаны живьём).
  - Java эталона не копировать в гит; артефакты = только /tmp/m77 (шаллоу, можно снести).
  - Секреты/creds юзера в гит/память не класть.
  - После правок: go vet ./... && go test ./... (зелёные) → отчёт: список изменённых файлов.

ДЕПОЙ (после зелёных тестов, по отдельной команде юзера «го»):
  1) кросс-сборка: cd nextgen/aion-gate && GOOS=windows GOARCH=amd64 go build -trimpath
     -ldflags "-s -w" -o aion-gate.exe .
  2) прод-свитч: taskkill /F /IM aion-gate.exe → scp exe в D:\SAION\aion-gate\ (scp push работает,
     exe локчен пока жив — только после taskkill) → schtasks /run AionGate.
  3) проверка баннера mode=authgate rsa_exponent=65537 + tail gate-prod.log.
  4) решающий логин юзера 1/1 → ожидаем [03]74Б (LOGIN_OK) → [05] relay → [04]42Б → выбор сервера
     → [02] relay → [07]26Б → мир 7777.
  Откат: предыдущий exe (версия = коммит f43e350) — держать aion-gate.exe.bak-f43e350 рядом.
```

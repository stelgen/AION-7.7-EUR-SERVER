# 🛰 Wire 2110 — authd-сторона (R1-контент, 07.10.2026)

> Контракт authd↔gate. Источники: дизasm AuthGateD (0x406000/0x406050/0x4060a0/0x405d40),
> наш `aion-gate/internal/authdclient/client.go` (live-проверен), `cmd/probe` (live-эталон,
> ходил в оригинальный L2Authd), RAW-логи gate-prod.log. Плейнтекст — БЕЗ крипты
> (вся крипта клиентская живёт в гейте).

## 1. Фреймы

### gate→authd (вход нашего сервера)

| Фрейм | Форма | Примечание |
|---|---|---|
| CltConnect | `[00][sid u32 LE][ip u32 LE]` | 9Б. ip = LE от BE-значения октетов: для 192.168.0.253 гейт шлёт `FD 00 A8 C0`. Регистрирует сессию гейт-sid |
| CltDisconnect | `[01][sid u32 LE]` | 5Б. **Онлайн-флаг НЕ снимает** (live-факт 08.10: «[01] не снимает», TTL 2-6 мин) |
| CltPacket | `[02][sid u32 LE][len u16 LE][blob]` | len **самоинклюзивный** = len(blob)+2; blob ≤ 0x1ffb |

### authd→gate (выход нашего сервера)

| Фрейм | Форма | Примечание |
|---|---|---|
| Greeting | `[03][V u32 LE]` | **Сразу при accept** (probe-live: «приветствие есть»). Live V = `0x0000c621` — попадает в welcome клиента `[5:9]` |
| UnknownSession | `[01][sid u32 LE]` | negative-ack на `[02]`/`[01]` для sid без CltConnect (probe-live 07.10) |
| Packet | `[02][id u32 LE][len u16 LE][type][payload]` | len = body+2, body = 1+len(payload); id = гейт-sid; **type = клиентский опкод** (гейт его prepend'ит) |

## 2. Диспетчер blob (blob[0] = клиентский опкод, гейт релеит 1-в-1)

| op | Смысл | Наш ответ | Payload |
|---|---|---|---|
| `0x00` | CM_LOGIN-релей, blob = asm-форма **191Б** `[00][decbuf 34][dword u32][tail 152]` | type=3 (login-ok) или type=1 (фейл) или **тишина** | ниже |
| `0x05` | CM_SERVER_LIST | type=4 | 31Б |
| `0x02` | CM_PLAY | type=7 | 15Б |
| `0x08` | CM_UPDATE_SESSION | **тишина** (T3 не закрыт) | — |
| прочее | unknown | **тишина + лог** (НЕ рвать коннекцию) | — |

### 2.1 Логин (op 0x00) — live-факты 06-07.10

- **Пароль НЕ проверяется вообще**; ASCII-логин автосоздаёт акк (акк «1» = uid 7, «stelgen» = 1010).
- decbuf 34Б = `[user 14][pwd 16][otp 4 LE]` (asm arg3=0x22=34; otp `FFFFFFFF` = нет).
- dword/tail authd **игнорит по содержимому**, но требует asm-РАЗМЕР: **86Б → тишина** (probe-доказано).
- username: TrimSpace+ToLower (К-5; «StelGeN» ≠ «stelgen»).
- Единственный live-фейл: **не-ASCII/пустой логин** → type=1.
- Реlogin **онлайн**-акка → **тишина** (ориг молчит; гейт сам отдаст LoginFail(1) по
  authdTimeoutSec=15). Опция `reloginPolicy: fail7` — немедленный фейл(7).
- Онлайн-флаг: ставится на login-ok (type=3), живёт TTL 2-6 мин, `[01]` НЕ снимает.

Ответы логина:

| Исход | type | payload | гейт-клиент получит |
|---|---|---|---|
| ok | 3 | **52Б** `[accId u32][token u32 Rnd][16×0][maxUsers u32=2000][unk1 u32=0xa0c69f0b][20×0]` | 74Б (pt [03]+52+pad11 = 64 → EncryptSecondary) |
| фейл | 1 | **4Б** `[messageId u32 LE]` | 18Б SM_LOGIN_FAIL (pt [01]+4 = 5Б) |

messageId = реестр Mobius AionAuthResponse (уже в aion-gate `authfail.go`):
2 = INVALID_PASSWORD, 7 = ALREADY_LOGGED_IN, 8 = SERVER_DOWN, 22 = BAN_IP и т.д.
Наши дефолты: badUser=2, blocked=22, db=1.

⚠ unk1 (`0xa0c69f0b`) и хвостовые dword'ы 74b-расшифровки (в форк-дампе мелькал
`1a6bc068` на pt[56:60]) — семантика НЕ вскрыта; клиент толерантен к нулевому паду
(доказано эмуляцией 26b/42b). Байт-паритет — арбитр R5 fork-дифф O-vs-N.

### 2.2 Server-info (type=4, live 42Б = pt 32 = `[04]`+payload 31)

```
[01 01 01][worldIP 4Б][worldPort u16 LE = 611e (7777)][00 00 00 00]
[f4 01 01 01][00 00 00 02][01 00 01][7×0]
```

### 2.3 Play-ok (type=7, live 26Б = pt 16 = `[07]`+payload 15)

```
[pk1 u32 Rnd][pk2 u32 Rnd][serverID байт][6×0]
```

(эталон SessionKey: playOk1/playOk2 = Rnd; НЕ хардкод 1/1010.)

## 3. Что НЕ покрыто MVP (осознанно)

- **Порт 2104** (serverPort — Server64/world-канал: logout-события, userLoggedToGs —
  семантика C1 WorldSrvSocket) — роль в Aion уточняется R0-дизasmом; без него свитч
  прод невозможен (Server64 ходит в L2Authd).
- **2108 GM / 10062 QMAS / 2220 AccountCache** — вне MVP (QMAS мёртв?, GM не используем).
- **procs AionAccounts** — дефолтные SQL = C1-схема (user_account/block_msg); реальные
  имена сверяются R0 (sp_helptext), переопределяются конфигом.
- WriteLogD-лог (ap_SLog-эвент в logd) — store.LogLogin заглушка/UPDATE.

## 4. Golden-фикстуры (тесты)

- greeting = `03 21 c6 00 00` (RAW live «A>G [03] 21c60000»).
- CltConnect 192.168.0.253 → `00 07000000 fd00a8c0` (LE-инверсия IP на проводе).
- CltPacket: len поле = 193 для blob 191Б.
- login-ok payload 52Б; фейл payload 4Б; type=4 = 31Б; type=7 = 15Б.
- probe-e2e (живой бинарь aion-gate/cmd/probe против aion-authd:2110) — см. README §Smoke.

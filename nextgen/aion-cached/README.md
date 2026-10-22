# aion-cached — Go-фреймворк CacheD64 (reverse-engineering + эмулятор)

> **ЕДИНАЯ СТАТУС-ШАПКА (обязательный формат, первой строкой под заголовком):**
> `🟢 ~80% · 22.10.2026 (R2 plaintext; R3: диспетчер 765 опкодов + DB 781 procs + 2007 текстовый + GQ/GP NPRelay per-conn на 2006 + wire accache-канон)`

---

## 📊 Статус и фазы

| Фаза | Статус | Что сделано |
|---|---|---|
| R1 — Go-каркас wire-фреймворка | ✅ | 3 листенера 2006/2007/2009, ship, конфиг (каркас ПЕРЕПИСАН на accache-канон — теория #15) |
| R2 — крипта | ✅ 21.10 | DummyCrypt = NO-OP, wire = plaintext (L2 C1 сурсы + дизasm Aion крипто-фэмили = мёртвый код) |
| R3 — Go MVP | ✅ 22.10 | MVP-диспетчер (765 опкодов/8 протоколов из .profile), DB-слой (781/140/132 procs, CallWorldProc, ErrProcMissing A/B-маркер), 2007 текстовый interactive, GQ/GP NPRelay per-conn, wire accache-канон + ping/версия-каноны |
| R4 — A/B с Live CacheD64 | ⬜ | Живой capture (fork-сдвиг 2006→2016 по «го» / pktmon) → раскладки read-пути → A/B |
| R5 — свитч | ⬜ | Замена CacheD64.exe (common.xml cacheServerPort) |

**Блокеры:** нет. `go vet`/`go test ./...` зелёные; подробности фаз/теорий — [ROADMAP.md](ROADMAP.md) (Т1-Т24).

## 📟 Канон протокола (R2/R3, 22.10 — только проверенное)

### Wire (accache-формат — теория #15, РЕВОЛЮЦИЯ: L2 C1 канон НЕсовместим с golden-кадрами Aion)

```
[u16 len total][cmd u16 LE][0xEB C2S / 0xEC S2C][~cmd u16 = ^cmd][payload]  БЕЗ хвост-суммы
```
- len = ПОЛНАЯ длина кадра вкл. len-поле; golden-кадр: `0f000100ebfeff0300000001000000`.
- Крипты НЕТ (R2: DummyCrypt = NO-OP — L2 C1 сурсы пустое тело; крипто-фэмили Aion = мёртвый код).
- 2007 = ТЕКСТОВЫЙ interactive (строки до \n, packetId первым полем, теория #14).
- 2006 per-conn режим: Server64 (RQ_VERSION/RQ_TEST первый кадр) vs NPRelay64 (GQ_LOGON(2)/GQ_KICKOFF(1) → relay-диспетчер; RQ/GQ id-пространства ПЕРЕСЕКАЮТСЯ).

### Ключевые ID

RQ_LOAD=11, RQ_LOAD_ITEM=14, RQ_LEAVE_WORLD=15, RQ_U_SAVE=16, RQ_PING=108, RQ_VERSION=215; RP_PING=45 (not-found форма L2 packet144: `cdddddddd`); GQ_ITEMADD=3, GQ_MAKEBUILDER=7, GQ_CHECK_CHAR=20, GQ_SET_BUILDER_CHAR=22, GQ_KICK_CHAR=24, GQ_MAIL_SEND=34, GQ_BUILD_COMMAND=43.

### Семантика (hourly-логи, Т24 — cached-ref/logs-hourly/)

`[CharacterList]` на каждый чар-селект · `[char]enter/leave world` с персистентной позицией · LoadClientQuickBar · RP_LOAD_SKILL_COOLTIME (RP-словарь сверяется) · старт-цепочка 13 CleanUp-ов · VendorDb pool ≈90% RAM.

### ГОТЧА e2e (теория #17)

`body = frame[2:]` ⇒ payload@body[**5**] (HeaderWire=7 − len-поле 2 = 5; НЕ body[7]!); ts в p[16:20], нули p[20:32].

## 🧱 Структура / сборка

```
aion-cached/
├── go.mod                  # module aion-cached, go 1.22
├── main.go                 # точка входа (flag -config, server.Run)
├── internal/
│   ├── wire/
│   │   ├── wire.go         # Reader (state-машина), Encoder, DummyCrypt, Frame
│   │   └── wire_test.go    # roundtrip, encoder roundtrip, bad csum, bad size, truncated
│   ├── server/
│   │   ├── server.go       # Config, Server, acceptLoop, handleConn
│   │   ├── handlers.go     # map[uint16]Handler, Route, stub
│   │   └── handlers_test.go# admin 0x01 → хандлер вызван, Config.Load
│   └── ship/
│       └── ship.go         # JSON-телеметрия ship-событий
```

```bash
# сборка / тесты
export PATH=$HOME/go-toolchain/go/bin:$PATH
export GOPATH=$HOME/go
cd nextgen/aion-cached
go vet ./... && go test ./...
go build -o cached64 .    # бинарник
```

Конфиг (key=value, простейший формат R1):

```conf
server_port 2006
admin_port 2007
npcdb_port 2009
crypt_seed_hex 5F3A7C21
```

## 🧪 Тесты и верификация

| Тест | Что покрывает |
|---|---|
| `TestRoundtrip` | build → read → payload равен (2 кадра подряд, rolling key) |
| `TestRoundtripEncoder` | Encoder.Build → Decoder.Feed, бит-в-бит payload |
| `TestBadChecksumRejected` | csum испорчен → `ErrBadChecksum` |
| `TestBadSizeRejected` | packetSize=0 и packetSize > 8192 → `ErrBadSize` |
| `TestTruncated` | пустой/обрезанный кадр → `ErrShort` |
| `TestAdminHandlerInvoked` | admin 0x01 → хандлер вызван, GP-ответ |
| `TestServerStubHandlerInvoked` | server 0x00 → хандлер вызван |
| `TestUnknownOpcodeRejected` | неизвестный опкод → nil |
| `TestConfigLoad` | парсинг key=value конфига |

## 📦 Артефакты

| Что | Путь |
|---|---|
| код | `nextgen/aion-cached/` |
| wire-канон | `nextgen/cached-ref/l2-c1-cached/src/network/CServerSocket.cpp` |
| пакеты-хандлеры | `nextgen/cached-ref/l2-c1-cached/include/cached/network/packets/{serverHandlers,adminHandlers,npcDbHandlers}` |
| RPC-инвентарь | `nextgen/cached-ref/rpc-opcodes-and-procs.md`, `nextgen/cached-rpc-map.md` |

## 🗂 Сурсы-эталоны

| Сурс | Что из него берём |
|---|---|
| `cached-ref/l2-c1-cached/src/network/CServerSocket.cpp` | wire-канон (len, DummyCrypt, rolling, csum) |
| `cached-ref/l2-c1-cached/generated/Cached.c|Cached.h` | IDA-деконструкция (для R2) |
| `include/cached/network/packets/serverHandlers/` | имена и форматы хандлеров |
| `include/cached/network/packets/adminHandlers/` | GQ/GP опкоды |
| `include/cached/network/packets/npcDbHandlers/` | NPC-DB опкоды |

## 🚧 Блокеры / открытые вопросы

- R2: точная форма `DummyCrypt` (обрезка ключа, порядок аргументов) — нужна дизasm-верификация или capture-сверка.
- Rolling key: катится ли при браке csum (R1 реализует канон «катится всегда»; L2 код подтверждает — `m_key += m_packetSize` до проверки).

## ⏭️ Следующий шаг

R4-read-раскладки: живой capture 2006 — (а) fork-сдвиг common.xml cacheServerPort 2006→2016 + cdmirror (по «го» — рестарт CacheD), или (б) pktmon filter port 2006 (без рестарта); раскладки RQ_LOAD(11)/RQ_LOAD_ITEM(14)/RQ_U_SAVE(16) → Go-хендлеры → RAM-модели → A/B → R5. Текстовые ответы 2007 (`Send("0%d",3)`) и GQ/GP payload — из того же capture. Семантику порядка — из cached-ref/logs-hourly/ (Т24).

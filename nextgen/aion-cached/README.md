# aion-cached — Go-фреймворк CacheD64 (reverse-engineering + эмулятор)

> **ЕДИНАЯ СТАТУС-ШАПКА (обязательный формат, первой строкой под заголовком):**
> `🟡 каркас · ~55% · 21.10.2026 (R2 закрыт: wire = plaintext)`

---

## 📊 Статус и фазы

| Фаза | Статус | Что сделано |
|---|---|---|
| R1 — Go-каркас wire-фреймворка | ✅ | `wire.go` (Reader/Encoder, канон CServerSocket.cpp), `handlers.go` (map[uint16]Handler), `ship.go`, `server.go` (TCP 2006/2007/2009), конфиг |
| R2 — дизasm DummyCrypt | ⏳ | Требуется точная форма XOR-стрима (обрезка ключа/порядок), сверка с L2 C1 |
| R3 — Go MVP (реальные хандлеры) | ⬜ | Замена `stub` на рабочие RQ/RP, GQ/GP, NPC-DB |
| R4 — A/B тест с Live CacheD64 | ⬜ | Захват и сравнение пакетов |
| R5 — свитч (замена бинарника) | ⬜ | Полная замена CacheD64.exe |

**Блокеры:** нет. `go vet` и `go test ./...` зелёные.

## 📟 Канон протокола

Канон из `nextgen/cached-ref/l2-c1-cached/src/network/CServerSocket.cpp` (L2 C1 CacheD — единственный публичный reversed CacheD).

### Wire-формат кадра

```
[u16 LE self-inclusive len][opcode][payload...][2Б csum]
```

| Элемент | Формат | Описание |
|---|---|---|
| len | u16 LE, self-inclusive | Размер всего кадра (включая сам len-филд 2Б) |
| opcode | 1Б на проводе, `uint16` в Go | Тип пакета (map[uint16]Handler для future-proofing) |
| payload | переменная | Тело пакета |
| csum | 2Б хвост [preLast][last] | u16 LE от (packetSize + 2) = wireLen |

### State-машина Reader (canon CServerSocket.cpp)

```go
// Канон: m_packetSize = (buf[off+1]<<8) + buf[off] - 2
packetSize := int(binary.LittleEndian.Uint16(buf)) - 2

// DummyCrypt XOR-стрим, rolling key
for i := range data {
    data[i] ^= byte(key + uint32(i))
}
// rolling: m_key += m_packetSize (выполняется ДО csum-проверки, как в ориге)

// checkSumm = (preLastByte + (lastByte<<8)) - 2 == m_packetSize
// → u16 LE двух последних байт тела = packetSize + 2 = полная wire-длина
```

### Отклонения от канона (R1, осознанные)

- **L2 canon**: `reject при bad size / bad csum` → socket Close. **R1**: то же самое (canon соблюдён).
- **L2 canon**: `Send` НЕ шифрует ответы. **R1**: то же (outbound крипта отсутствует).
- **`Frame.Opcode` uint16 в Go** при 1 байте на проводе: канон соблюде, Go-тип расширен под `map[uint16]Handler`.

### Каналы

| Порт | Канон L2 C1 | Наши типы | Примеры opcodes |
|---|---|---|---|
| 2006 | `serverHandlers/` | RQ/RP (ServerToDb/DbToServer) | 0x00 CacheVersion, 0x01 LoadCharacter, 0x09 SaveCharacter |
| 2007 | `adminHandlers/` | GQ/GP (AdminToCache/CacheToAdmin) | 0x01 GQ_MAKEBUILDER, 0x12 GQ_ITEMADD |
| 2009 | `npcDbHandlers/` | NPC-DB (NPCSvr64) | 0x01 Version, 0x02 LoadNpc, 0x03 SaveNpc, 0x04 UpdateBoss |

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

R2 — дизasm DummyCrypt из `generated/Cached.c` / PDB publics, сверка с формулой `payload[i] ^= byte((m_key + uint32(i)) & 0xFF)`.

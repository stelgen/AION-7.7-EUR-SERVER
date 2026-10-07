# aion-main — перепись Server64/MainServer (игровое ядро 7777/2002) — Go, ДЕПРИОРИТ (последний в свитче)

> 🟡 **КАРКАС ЖИВОЙ, ПРОД НЕ ТРОНУТ** (08.10.2026, коммит `7a5dffd`): R0–R4 закрыты (протокол-реестр 637 пакетов live, крипта подтверждена live, Go-каркас с тенью :7778, мир-стейт по прод-раскладкам, tap-режим FORK-SPEC). Бинар ориг в бою (`Server64.exe` #180-патч, SHA `b000c6f5…`); наша тень = пассивна. Свитч = гейт после полного MVP-мира (R3.8–R4.4, см. [ROADMAP](ROADMAP.md)).
> Запуск чата: `WORKFLOW: main` ([../WORKFLOW.md](../WORKFLOW.md)). Стандарты S1–S12: [../README.md §4](../README.md). Тактический слой поверх ориг-бинаря: [../aion-binpatch/](../aion-binpatch/).

## 📊 Статус и фазы

| Фаза | Что | Статус (08.10.2026) |
|---|---|---|
| R0 | Разведка: PDB 284МБ (74164 publics), карта зависимостей/подсистем, конфиг-инвентарь, реестр 637 пакетов из 5 сурсов | ✅ (см. [RESEARCH.md](RESEARCH.md), [OPCODES.md](OPCODES.md)) |
| R1 | Wire capture: 3 сессии pktmon 7777 (логины юзера), ~7500 пакетов, **0 invalid**; крипта 7.7 EU подтверждена live; payload-разборы (SM_STRONGHOLDS 0x0182, 0x00D4, 0x0195, 0x0184) | ✅ |
| R2 | Go-каркас: `internal/{crypt,wire,ops}` + `ops.yaml` (637) + тень :7778 + golden-тесты на реальных фреймах | ✅ (06b0ac7) |
| R3 | MVP-хендлеры: VERSION_CHECK/TIME_CHECK/PING/MOVE/CHAT/TARGET/LEVEL_READY + Session; **два независимых ключа** (канон); E2E PASS | ✅ (2035a69) |
| R3.5 | Мир-стейт по capture-раскладкам: 13 прод-payload'ов (14962Б), InitSequence канон; E2E PASS байт-в-байт | ✅ (bd10861) |
| R3.6 | Динамические подмены раскладок (`layouts.yaml`: hp/mp/time, копия-не-мутируется); live-вердикт SM_STATUPDATE_HP | ✅ (6ab4509) |
| R3.7 | OID-механизм (ReplaceOID) + факт «OID не в раскладках» (персоно-агностичность, тест) | ✅ (62c4ea5) |
| R4 | tap-режим FORK-SPEC: пассивный разбор КОПИИ трафика (SM_KEY-синк, VERDICT vs реестр); тест на реальном capture: 2147 кадров, known=2146, C2S 100% | ✅ (6ab4509) |
| R3.8 | **Живой мир-стейт**: инвентарь/статы из cached-read (после aion-cache R1+R2/R3), дизасм-смещения полей | ⬜ |
| R3.9 | Чат-флоу полный (каналы/систем-месседжи) + движение полный (телепорты/переходы карт) | ⬜ |
| R4.1 | 2002-мир: NPCSvr-интеграция (требует созревания **aion-npc MVP** — кросс-зависимость) | ⬜ |
| R4.2 | cached 2006: read-путь + write-транзит (aion-cache R3) | ⬜ |
| R4.3 | A/B автоматический: симуляция полного флоу юзера vs ориг → auto SAME/DIFF | ⬜ |
| R4.4 | Стабильность: RAM-планка «чище ориг» (<1ГБ vs 10ГБ), долгие прогоны | ⬜ |
| R4.5 | Деплой тени на VM (`D:\SAION\aion-main\` + задача AionMainShadow + pktmon-fed tap) | ⬜ |
| R5 | **Свитч по «го»** (пара NPC+MAIN, op restart_pair); откат = ориг. ⚠ ГЕЙТ: только после R3.8–R4.4 (полный MVP-мир на живых данных + NPC-пара) | ⬜ |
| R6 | Наблюдение (RAM/хендлы — целевая планка чище ориг: 860k хендлов ориг недопустимы) | ⬜ |

**Работает сейчас**: тень :7778 — полный handshake (SM_KEY), дешифр C2S, диспетчер по имени, мир-последовательность из прод-раскладок, /status (:10221), tap-разбор копий. **НЕ работает**: живой мир-стейт (статика раскладок), NPC-мир, cached-интеграция (см. Блокеры).

## 📟 Канон протокола (только проверенные, «НЕ трогать»)

### Wire 7777 (крипта 7.x — live-подтверждено capture 08.10, 0 invalid/~7500 пакетов)

| Факт | Значение | Источник |
|---|---|---|
| Фрейм | `[u16 size LE self-inclusive][body]`; size НЕ шифруется | Mobius/ag78 Dispatcher + live |
| S2C тело | `[E u16][0x56][~E u16][payload]`, `E = (op + 0xD8) ^ 0xD9` | Crypt.encodeOpcodec + live |
| C2S тело | `[op u16][0x75][~op u16][payload]` | EncryptionKeyPair.validate + live |
| Крипта | XOR: `b[i] ^= staticKey[i&63] ^ key8[i&7] ^ prev`, prev = **шифрованный** байт (encrypt: после xor; decrypt: до xor) | Java + live |
| Ключи | **ДВА независимых** (keys[SERVER]/keys[CLIENT]), каждый = `[baseKey u32 LE][A1 6C 54 87]`, roll `+= len(body)` за пакет | Java + live |
| SM_KEY | Первый пакет, НЕ шифрован: `[E(0x48)][0x56][~E][falseKey u32]`; `base = (falseKey − 0x3FF2CCDF) ^ 0xCD92E4D9` | Crypt.enableKey + live |
| staticKey | 64Б `nKO/WctQ0AVLbpzfBkS6NevDYT8ourG5CRlmdjyJ72aswx4EPq1UgZhFMXH?3iI9` | Java (идентичен 7.5/7.8) |
| PacketSamurai 7.5 | `crypt="AionGame7_5_0_0"`, `checksumSize=3`, port 7777 | Game_7.5.x.xml |

### Реестр пакетов (637 в `ops.yaml`; источники: 5 реестров + live)

| Факт | Значение | Источник |
|---|---|---|
| Приоритет арбитража | FULL-согласие 3+ Java > тег «7.7 EU» > ag78 > xml75; арбитр = live capture | OPCODES.md §4 |
| XML-словарь Packet Samurai | частично неточен (21 расхождение vs согласованной Java, ex. CM_ATTACK 0x0FB vs 0x0F6) — Java прав | live + реестры |
| live-вердикты | `0x01B5=CM_HOUSE_OPEN_DOOR` (encom75 7.5 KR; «CM_MEGAPHONE 5.4 EU» в ag78 неактивен), `0x0111=SM_HOUSE_RENDER` (xml75 ошибочно SM_DELETE_HOUSE), `0x00F6=SM_FLY_TIME`, `0x0106=CM_MOVE` (xml75 ошибочно CM_LEGION_MODIFY_EMBLEM), `0x00DB=CM_PET_EMOTE`, `0x02FA=CM_SHOW_DIALOG` | live capture#2/#3b |
| Новые 7.7 EU S2C | `0x0184` (пустой сигнал), `0x0195` (1Б `00`) — вне эталонов; в реестре как SM_UNK0184/SM_UNK0195 | live #3b |
| SM_STRONGHOLDS | `0x0182`: bulk при входе в мир (151 шт), элемент ~180Б, ID u16 @+4, статусы u16 @+17..19; **только в Encom leak** | live payload-разбор |
| High-диапазон | 0x2F2–0x2FD существуют live (CM_PING 0x02F2, CM_EMOTION 0x02FD…); 0x3FD/0x3FC/0x3F2 = теги «7.7 EU TODO» (Mobius) | live |

### Каналы и подсистемы (PDB Server64: 74164 publics)

| Факт | Значение | Источник |
|---|---|---|
| RPC-семейства | `ServerToClient/ClientToServerS`=7777; `ServerToDb/DbToServer/ServerToDb_Update`=2006 (1-в-1 cached-ref); `ServerToNPCServer/NPCServerToServer`=2002; `ServerToIC`=2005; `RankMainProtocol`, `CNpRelaySocket` | PDB publics |
| Matchmaker | `MatchingMgr/World_BattleGround/World_IDArenaTournament/World_IDARENAPvP` в бинаре = InterSvrType=2 канон | PDB |
| Мир-объекты | `User 1326 / Creature 295 / Npc 281 / World 247 / WorldBase 93 / DynamicWorld 69` | PDB |
| Крипта-классы | `CPacket, CInvalidPacketBlocker, PacketProfiler, PacketProfileLogger, SyncPacketMgr, PacketArgInfo`; GG(GameGuard)+Blowfish+AES | PDB |
| SM_/CM_ имён в publics | НЕТ (семантика в handler-функциях) — маппинг опкодов только из эталонов+capture | PDB |
| proc-имена | НЕ строки в Server64.exe (динамика/упаковка) → каталог недостающего = op-алерты proc_missing | strings-скан |
| Конфиг-порты (прод) | 7777/2002/2006/2051/2104/2107/2220/10241/10254/10100/IC 2005; captcha буфер 10000; RunAsDate-корреляция `v45_update_date=2017-11-20` | common.xml/config.xml («Gardarika» 2023) |
| CM_MOVE структура | payload@0 = OID u32 (capture `0x443E29B5`), далее X/Y f32 | live payload-разбор |
| Раскладки | 13 прод-payload'ов персоно-агностичны по OID (OID не встречается — тест); InitSequence: ABNORMAL→SKILL_LIST→WAREHOUSE→INVENTORY→STATS_INFO→LUNA→SIEGE→STATUPDATE (кадры 16/40/86/87/101/106/172/427) | live |

### БД (тех-долг, отдельный док: [docs/tech-debt-db-20261008.md](../../docs/tech-debt-db-20261008.md))
База кита ~85% (pada8801, RZ 1205286 p.7 #121); наша `_AionWorldNew114_rc` = 174 NC-таблицы; 10 недостающих procs (коллекции×5/fame/reinvent/itemAttributeDelta×3) — донор = кит 5.8 PTS DB.

## 🧱 Структура / сборка

```
nextgen/aion-main/
├── ops.yaml                      ← реестр 637 пакетов (live-вердикты src: LIVE)
├── cmd/aionmain/main.go          ← тень :7778 (S4) / tap-режим / canary (S7) / raw-first (S3) / status :10221 (S2)
├── internal/
│   ├── crypt/                    ← rolling XOR + SM_KEY + encodeOpcodec (+golden-тесты на РЕАЛЬНЫХ фреймах)
│   ├── wire/                     ← фрейминг [size][E][0x56][~E][payload]
│   ├── ops/                      ← YAML-реестр (Lookup op/dir/name)
│   ├── handlers/                 ← MVP-хендлеры (эхо + Session) — R3.5+/R3.8 живой мир
│   ├── world/                    ← раскладки (13 прод-payload'ов) + InitSequence + LayoutPatch (R3.6) + ReplaceOID (R3.7)
│   ├── integrations/             ← cached2006 ping-only / accache/authd/npc стабы с каноном
│   └── tap/                      ← R4: VERDICT-разбор копий трафика
└── docs/                         ← opcodes-unified-0810.csv (688), OPCODES.md
```

Сборка: `go vet ./... && go test ./... && go build -o aionmain ./cmd/aionmain` (Go ≥1.22; linux+windows одним исходником). Конфиг-пример: `ops.yaml` + `layouts.yaml` (без секретов; S7: латиница-комменты, канарейка-баннер при старте).

## 🧪 Тесты и верификация
- **Golden** на РЕАЛЬНЫХ фреймах capture (SM_KEY + CM_VERSION_CHECK 0x00D6) — дешифр/валидация/roll.
- Roundtrip encrypt/decrypt (два ключа), фрейминг Split/Build, реестр YAML (SM_KEY/CM_MOVE присутствуют).
- **E2E smoke**: фейк-клиент → SM_KEY → шифрованный C2S → диспетчер → шифрованный SM-ответ (PASS: TIME_CHECK, мир-последовательность 9 фреймов 14687Б байт-в-байт).
- **Tap на реальном capture** (0810d): 2147 кадров, known=2146, C2S 100%, 0 invalid.
- `go vet ./... && go test ./...` — зелёные до пуша (S9).

## 📦 Артефакты — где лежат

| Что | Гит | VM | Локально |
|---|---|---|---|
| Код + ops.yaml + тесты | [nextgen/aion-main/](.) (этот каталог) | `D:\SAION\aion-main\` (R4.5) | — |
| PDB 284МБ + publics 74164 + прод-конфиги | ❌ (манифесты) | `D:\AION_LIVE_SERVER\MainServer\` | `aion_rev_2026-10-05/artifacts/pdb-big/Server64/` |
| Раскладки мира (13, 14962Б) | ❌ gitignore (приватность/прод-данные) | — | `internal/world/testdata/layouts/` |
| Capture pcapng + decrypt-log | ❌ gitignore | `C:\Temp\cap7777-0810*.etl` | `STELGEN/tmp/cap7777/` |
| Эталоны (7 шт, ~3.9ГБ) | ❌ | киты 2.7/4.6db/5.8 | `STELGEN/projects/aion_server_2026-10-02/reference/` |
| Дешифратор capture | [tools/analysis/cap7777_decrypt.py](../../tools/analysis/cap7777_decrypt.py) | — | tmp/cap7777/ |

## 🗂 Сурсы-эталоны (приоритет над дизасмом)

| Сурс | Где | Что берём |
|---|---|---|
| Encom leak 7.5–7.7 | reference/encom-leak-7577 | семантика пакетов 7.5 (CM_HOUSE_OPEN_DOOR 0x1B5 и др. encom-only!), GS↔LS 2104, Packet Samurai словарь |
| aion-germany 7.8 EU | reference/aion-germany | самый близкий регион; полный реестр; крипта-канон |
| Mobius 7.7 | reference/Mobius_AionEmu | наша версия; сервер-пакет реестр; «7.7 EU» теги |
| AionLightning 7.8.0 | reference/AionLightning | верификатор |
| beyond-aion 4.8 / ARP 4.7.5 | reference/aion-server, yoress-arp-475 | ретро-семантика |
| aion-encombase-58 | reference/aion-encombase-58 | 5.8-слой |
| PDB publics 74164 | artifacts/pdb-big/Server64 | подсистемы/RPC-семейства/структуры |
| Дизасм (objdump+.map) | — | ТОЛЬКО после исчерпания сурсов (WORKFLOW §5) |

## 🚢 Деплой и откат (R4.5 — по «го»)

1. `aionput /tmp/aionmain D:/SAION/aion-main/aionmain.exe` + `run.cmd` (`aionmain.exe -config ops.yaml -mode shadow -listen :7778`).
2. Задача `AionMainShadow` (schtasks /create — НЕ AutoStart: тень пассивна).
3. tap-фид: pktmon 7777 → etl2pcap циклично → `aionmain -mode tap`.
4. **ОТКАТ одной командой**: `schtasks /end /tn AionMainShadow && schtasks /change /tn AionMainShadow /disable` (тень пассивна, ориг не трогается — мир не зависит от тени).
5. R5 (свитч, по «го»): пара NPC+MAIN через op `restart_pair`; откат = `Server64.exe.etalon` (проверенный возврат).

## 📜 Логи
raw-first (S3): `[RAW-FAIL]` hex тела до парсинга; `[C2S]` имя+payload-hex; `[CANARY]` при старте (S7); `[WORLD]` init-факт; ship в rsyslog→Loki→Grafana — R4.5+ (S2: сейчас self-status :10221). Ориг пишет `MainServer\log\<дата>.err` (дата RunAsDate!) — тейлерит op.

## 🚧 Блокеры / открытые вопросы
1. **cached-RPC**: wire 2006 (7.7) не снят → блокирует R4.2/R3.8; первый шаг = aion-cache R1 (pktmon 2006). До него cached-клиент = ping-only.
2. **NPC-мир (2002)**: Server64-тень без мир-пары бессмысленна для свитча → aion-npc MVP должен созреть первым (R4.1).
3. **7777 открыт наружу**: сканеры (34.78.135.85 в логе) — закрыть firewall LAN-only (безопасность).
4. **Полный мир-масштаб**: эмулятор = 2783 java (services 93, skillengine 250, dataholders 129) — порт СЕМАНТИКИ по приоритетам live-подсистем (см. карту портирования ниже), не кода.
5. База ~85% (TD1–TD6): [docs/tech-debt-db-20261008.md](../../docs/tech-debt-db-20261008.md).

## ⏭️ Следующий шаг (для нового чата)
`WORKFLOW: main` → **R3.8**: живой мир-стейт (дизасм-смещения полей раскладок: PDB + cached-ref карты; после aion-cache R1 — прямые данные из кеша). Промпт: [PROMPT.md](PROMPT.md), план: [ROADMAP.md](ROADMAP.md). Кросс-пульс соседям обязателен (WORKFLOW §5).
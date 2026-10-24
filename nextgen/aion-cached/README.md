# aion-cached — Go-фреймворк CacheD64 (reverse-engineering + эмулятор)

> 🟢 **~90% · 23.10.2026** — R4-ФИНАЛ PASS (offline A/B + живой T39-acks 18/0/12), R4.6 мир-init Server64 ✅ байт-в-байт, R4.6.2 NPCSvr W30 ✅ (228 кадров replay), DB-генератор Этапы 0,1,3,5 ✅. R5 свитч #4 вердикт: v16c (c18a371c) встал на 2006/2007/2009 ✅, но мир 0/8 — Server64 прислал только полл 282 (ноль мир-init), откат по канону. **#5 = эмиссия мир-init при рестарте пары** (capture собран: worldinit-20261023.txt, 103 O-cmds).
> Прод-топология (23.10 вечер): **мир 0/8 — пара NPC+MAIN остановлена юзером**; cd2016+cdmirror+cachedshadow живы. При подъёме мира CacheD = копия-2016 за cdmirror (ориг), наша тень = пассив :2018/2019/2020. Из кеш-семейства в проде НАШ только aion-accache :2220.
> Запуск чата: `WORKFLOW: cached` ([PROMPT.md](PROMPT.md)) · Фазы/теории: [ROADMAP.md](ROADMAP.md) · Соседи: Server64 = [aion-main](../aion-main/README.md), NPCSvr = [aion-npc](../aion-npc/README.md).

## 📊 Фазы

| Фаза | Статус | Суть |
|---|---|---|
| R1 — Go-каркас wire | ✅ 10.10 | 3 листенера 2006/2007/2009, ship, конфиг (позже переписан на accache-канон — теория #15) |
| R2 — крипта | ✅ 21.10 | DummyCrypt = NO-OP, wire = plaintext (L2 C1 сурсы + дизasm крипто-фэмили Aion = мёртвый код) |
| R3 — Go MVP | ✅ 22.10 | 765 опкодов/8 протоколов (из .profile), DB-слой 781+140+132 procs, 2007 текстовый, GQ/GP per-conn |
| R3-хвосты — read-раскладки | ✅ 22.10 | Т28–Т35: session/LOAD/LOAD_ITEM/LEAVE/U_SAVE, RP cmd=1 (Т31), settings XML store (Т30/Т32), item blocks (Т33), RP-эмиттеры + ChainReplay |
| R4 — A/B vs live CacheD | ✅ 23.10 | offline PASS (self-replay 100%, cross-login 97.75%) + живой T39-acks 12 cmd×25Б: **18 PASS / 0 DIFF / 12 UNPAIRED**; accId session-applied подтверждён; полный цикл логин→чарлист→мир→выход через тень ✅ |
| R4.6 — мир-init Server64 | ✅ 23.10 | **ПРОРЫВ**: startup W5-бёрст = login world-flow chain байт-в-байт (118/120) + 2 кадра (0x7A#2, 0xBF) — стартовый протокол = login-path, НЕ отдельный RE; StartupTracker (новый X на 288 → повторная эмиссия = фикс корня крэш-лупа) |
| R4.6.2 — NPCSvr W30 | ✅ 23.10 | sessSet-гейт УБИТ (NPCSvr без 288 = мёртвый код, S=0 live-доказательство); npcsrv_w30.go: C277→O187×150, C62→45 кадров (40×O18), 179/224 fire-and-forget; 228 кадров replay байт-в-байт; v16c на tap ✅ |
| R5 — свитч CacheD64 | 🔴 | **#4 (23.10)**: v16c на 2006/2007/2009 ✅, но при рестарте пары мир 0/8 — Server64 прислал ТОЛЬКО полл op=282 (ноль мир-init 288/11/14), NPCSvr не дошёл до «NPC Server Started»; pktmon-2009 = 0 пакетов (npcDb при старте не используется). Откат выполнен. **#5 = эмиссия мир-init из собранного capture при рестарте пары** |

## 📟 Канон протокола (только live-доказанное)

### Wire 2006 (Т26 live, 332+ кадра; каркас = accache-формат, маркеры свои)

```
[u16 len total][cmd u16 LE][0xAE C2S / 0xAF S2C][~cmd u16 = ^cmd][payload]   plaintext, БЕЗ хвост-суммы
```
- len = полная длина кадра вкл. len-поле; инверсия ~cmd ✓ на 100% кадров; опкод u16 (RQ_VERSION=215 в потоке).
- Крипты НЕТ (R2): DummyCrypt = no-op (L2 C1 тело пустое; Aion крипто-фэмили = ноль ссылок).
- Маркер-пара ПО КАНАЛУ: accache EB/EC, cached AE/AF — каркас един.

### Три клиента 2006 + каналы

| Соединение | Фингерпринт | Флоу |
|---|---|---|
| Server64 | первый кадр RQ_VERSION(215)/RQ_TEST | логин-путь + тикер 179 (~6с)/277 + мир-init при старте |
| NPCSvr (Т27) | cmd=170-поток (UTF-16 `tl_dp_id:`/`tl_abyss_id:`), cmd=200 `[SAVE]v01` | 179/224/106/299 fire-and-forget |
| NPRelay64 | relay-кадры | **relay-wire = ЗАГЛУШКА** (дизasm 23.10: слот0 VERSION no-op, слоты1-5 NoPacket silent); флоу = RQ-канон (Т19 ⏳) |

- **2007 = текстовый interactive** (Т14 ✅): строки до `\n`, packetId первым полем (ReadSocketArgs "dsss"); GQ-словарь .profile = packetId-маппинг этого канала.
- **2009 npcDb**: словаря в .profile нет; при старте мира **0 пакетов** (pktmon 23.10) — канал спит.

### Read-путь логина (раскладки live 22.10, эмиттется тенью, Server64 принял живьём)

```
RQ_VERSION → 288 (сессия [X][a][accId][pad]) → 175 → cmd=90 CharacterList (UTF-16, charRef)
→ RQ_LOAD(11) [X][a][charId][accId] → RP cmd=1 (1156Б char struct)
→ RQ_LOAD_ITEM(14)×2 [X][a][charId][кат][accId][page] → RP cmd=0x0E ×3 (ZLIB UTF-16 XML settings)
→ RP cmd=109 ×12 (binary item blocks ~470Б/запись) → enter world (cmd=21 ⏳)
→ тикер 179/277 → RQ_U_SAVE(16) → RQ_LEAVE_WORLD(15) [acc][char][4×u32 pos]
```
- **RP cmd=1** (Т31): X echo, a, charRef@21, level = **single byte @payload[49]**, kinah (a08601=100000), имя UTF-16 @903:928 «Имя-<зона>», last_normal/resurrect координаты @428-456 (из user_data — НЕ текущая позиция).
- **RP cmd=0x0E** (Т32) = CHARACTER SETTINGS XML (zlib, page0=28578Б options / page1=3098Б quickbar+chainskill / page2=744Б favorites) — CacheD = settings store.
- **RP cmd=109** (Т33): item-записи по 470Б (item_id u32 LE @+0 и дубль @+22, amount @+4, slot u8 @+461, item-тип @+462, amount-дубль @+463, expired_time @268-276).
- **Т39**: RP-подтверждения 12 cmd × 25Б обязательны для паритета (x018/01D/02F/05B/063/080/084/08B/0BD/0E1/0E4/0EB-семейство).
- **Т38**: a @payload[4] НЕ константа (4→3 по сессии). **Т36/Т37**: accId@17 и last-login@703 заполняются со 2-й сессии.

### Мир-init (R4.6 + capture 23–24.10)

- Startup W5-бёрст = login world-flow chain (118 кадров) байт-в-байт + 0x7A#2 + 0xBF; кросс-оконные диффы = только сессионные поля (X BE @9:13 + a @13 + динамика @21).
- **StartupTracker**: новый X на 288 → сброс emittedWorldInit → повторная эмиссия (фикс корня крэш-лупа R5 #3/#4).
- Полный мир-init = 103 уникальных O-cmds; матерал: `cached-ref/logs-hourly/worldinit-20261023.txt` + `chain_worldstartup_20261023.bin`; **свежий рестарт 24.10**: `cached-ref/logs-hourly/cdmirror-20261024-worldstart.log`.
- **NPCSvr-init цепочки (канон 24.10, файлы npcsrv_*.bin в testdata)**: 201→28(1035Б) · 161→78 · 211×2→113(arg-echo)+53 · 248→142+143 · **215→16(27Б)+123(6321Б)** · 178 silent · 325→93+204+93+221×2(14500Б) · 233×2→129(echo) · 250→144 · C62→W30(40×O18+5) · **C36×4 ≈27КБ silent-UPLOAD (NPCSvr грузит данные В CacheD!)** · 46→162(arg) · 40→157(arg) · 47→164 ⏳ · 49→166 ⏳ · 311→213(ts!)+215(01) · 134→51 · 131→52 · 200/106/36 silent.
- **RP_VERSION в мир-init = 27Б** (`9c0f0300 7c000000 d8000000 00008000 00000000`) — НЕ логин-форма 850Б.
- **C282-полл (0x11A, ~5 мин период): ориг МОЛЧИТ всегда** (181 полл/2 дня без единого ответа) — наша тишина = коррект.
- Legacy-канон 22.10 подтверждён вторым рестартом: 215: 6348, 325: 14500.

### DB-генератор (DBGEN, Этапы 0,1,3,5 ✅)

- **O1** из `user_data` (_AionWorldNew114_rc): lev@49, last_normal_x/y/z@428/432/436, resurrect@448-456, charName@903:928 — 0 diff vs capture.
- **O109** из `user_item` (warehouse=0; склад НЕ шлётся): GenerateItemRecord 470Б — 96% match.
- **O14** settings store: zlib-XML per-charId (character_settings).
- Этап 5: `EmitWorldFlowDB` = chain → замена O1/O109 → мир-кадры passthrough; работает для ЛЮБОГО чара (доказано на 1002+1004).

### ГОТЧИ (жёсткие)

- `body = frame[2:]` ⇒ payload@body[**5**] (HeaderWire=7 − len-поле 2; НЕ body[7]!); ts в p[16:20], нули p[20:32] (Т17).
- payload-оффсеты в wire-блоках +7: frame[7] = payload[0].
- Exe лочится живым процессом → строго стоп-пуш-старт; `accstop.ps1` убивает ПРОД-accache — только op-кнопки / `cdshadowstop.ps1`.
- go-mssqldb НЕ понимает ODBC `{call}` → `EXEC dbo.proc @pN` (эталон aion-authd/accache).
- aionput exe >5МБ = ТОЛЬКО scp (API затирал файл в ноль при ok:true).

## 🧱 Структура / сборка

```
aion-cached/
├── main.go, go.mod              # module aion-cached, go 1.22
├── internal/wire/               # Reader/Encoder, маркеры AE/AF, golden-тесты
├── internal/server/             # Server, handleConn (session/relay/w30-детект), handlers_read
├── internal/rpc/                # opcodes(765), procs-инвентарь связь, replay/chain_replay, rp_emitter,
│                                #   dbgen, item_blocks/item_dbgen/item_slots, world_flow(+_db),
│                                #   npcsrv_chains/npcsrv_w30, acks(Т39), session, admin_text, relay_handlers
├── internal/db/                 # 781+140+132 procs, CallWorldProc (ErrProcMissing A/B-маркер), world_store
├── internal/ship/               # телеметрия
├── tools/analysis/              # gen_cached_opcodes.py, gen_cached_dbprocs.py
└── testdata/                    # capture-эталоны (rp/rp0e/rp109/npcsrv_*/chain_worldstartup/worldinit)
```

```bash
export PATH=$HOME/go-toolchain/go/bin:$PATH; export GOPATH=$HOME/go
cd nextgen/aion-cached && go vet ./... && go test ./... && go build -o cached64 .
```

Конфиг: key=value (`server_port`/`admin_port`/`npcdb_port` + `world_conn` строка _AionWorldNew114_rc + ship-блок). Баннер-канарейка при старте (S7).

## 🧪 Тесты (go test ./... — зелёные)

Golden wire (accache-каркас, маркеры AE/AF) · e2e: ping (RP_PING not-found байт-в-байт), admin-text 2007, relay-GQ-handshake · handlers_read по live-раскладкам · rp_emitter/dbgen (0 diff 1002, level=1 1004) · item_blocks 96% · world_flow cross-session 94% · npcsrv_w30 full-replay 228 кадров байт-в-байт · ab_live (T39 live-пары).

## 📦 Артефакты

| Что | Путь |
|---|---|
| Код + тесты | `nextgen/aion-cached/` |
| Опкод-карта 8 протоколов | `cached-ref/profile-opcode-map.md` (+ `mir_init_opcode_table.json` мир-init в корне компонента) |
| RPC-инвентарь | `cached-ref/rpc-opcodes-and-procs.md` |
| Verdict'ы A/B | `cached-ref/logs-hourly/ab-r4-verdict-20261022.md`, `ab-r4-verdict-20261023-live-t39.md` |
| Мир-init capture | `cached-ref/logs-hourly/worldinit-20261023.txt`; эталон-цепочка `internal/rpc/testdata/chain_worldstartup_20261023.bin` (120 кадров) |
| GQ/GP wire-вердикт | `cached-ref/gq-gp-wire-20261023.md` |
| L2 C1 эталон | `cached-ref/l2-c1-cached/` (CServerSocket.cpp, DummyCrypt, пакеты-хендлеры) |

## 🚢 Деплой / откат (тень и стенд)

- Тень v16c (c18a371c): `D:\SAION\aion-cached-capture\aion-cached.exe`, задача AionCachedShadow :2018/2019/2020 (конфиг = cached-capture.conf задачи; Т42). Стоп = op-кнопка или `cdshadowstop.ps1`.
- Стенд: cdmirror :2006→копия-2016 (задачи AionCDMirror/AionCD2016) — копия стартует ТОЛЬКО соло.
- R5-свитч: наш на 2006/2007/2009 (конфиг 2018→2006 и т.д.) + рестарт пары; **откат одной командой**: вернуть конфиг из bak + рестарт cdmirror/cd2016 → пара до 8/8.
- Выводы #4: Server64 валидирует кеш на старте (нулевой мир-init = мир не собирается); порядок подъёма после каскада: инфра → NPC (RAM-кривая 14→1ГБ ~10-15 мин) → main.

## 🚧 Блокеры / открытые

1. **R5 #4 вердикт** (главный): при рестарте пары Server64 шлёт только полл 282 — мир-init эмиссия не сработала. Гипотезы #5: (а) Server64 ждёт ответ на 282 (наша тишина); (б) NPCSvr падает до сетевой фазы (SQL/файлы); (в) хаос дёрганья при #4 исказил картину.
2. ⏳ Т19 NPRelay flow (NPRelay в проде STOPPED), Т21 cmd=21 мир-ID, cmd=179-тикер семантика, Т29 категории LOAD_ITEM динамические, источник exact charName+зоны O1.
3. Write-путь RQ_U_SAVE→SQL — не реализован (пока replay/passthrough).

## ⏭️ Следующий шаг

`WORKFLOW: cached` → R5 #5: (0) проверка копии-2016 + **restart_pair до 8/8** (прогрев мира, по «го» юзера) → (1) свитч-тест v16c с эмиссией мир-init из собранного capture (StartupTracker уже фиксит повторную эмиссию; worldinit-20261023.txt = 103 O-cmds → эмиттеры по образцу world_flow_db.go) → (2) критерий: «NPC Server Started» + мир 8/8 на нашем кеше → наблюдение 24ч. Промпт: [PROMPT.md](PROMPT.md).

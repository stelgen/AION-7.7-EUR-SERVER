# RESEARCH aion-main — Server64/MainServer: PDB-карта + сурсы/эталоны (08.10.2026)

> R0-ресёрч чат: докачка PDB с VM, досбор эталонов из интернета, первая сверка «бинарёв ↔ сурсы».
> Политика: СУРСЫ-ПРИОРИТЕТ перед дизасмом (WORKFLOW §5). Арбитр всегда = live-трафик.

## 1. PDB Server64 — СИМВОЛ-ИНВЕНТАРЬ (новое 08.10)

| Факт | Значение |
|---|---|
| Файл | `Server64.pdb`, 284,102,656 Б, MD5 `7d54300308f5cce4ef03f1b80413da33` = манифест ✅ |
| Локально | `STELGEN/projects/aion_rev_2026-10-05/artifacts/pdb-big/Server64/` (+ common.xml, config.xml из `aion_fixes/MainServer.rar`) |
| Publics | **74164** (vs CacheD64 14281 — ядро в 5× жирнее) |
| Реестр | `pdb-publics-74164.txt` (sec, off, flags, mangled-name) |

## 2. Карта подсистем (топ mangled-классов → назначение)

| Класс (publics) | n | Подсистема (наша карта каналов) |
|---|---|---|
| `User` | 1326 | Игрок — самый большой контекст-объект |
| `ServerToClient` | 535 | **Протокол клиентов :7777** (S2C-семейство) |
| `Creature`/`Npc`/`World`/`WorldBase`/`DynamicWorld` | 295/281/247/93/69 | Мир-симуляция (дубль обязанностей NPCSvr64 — иерархия NC: Server64 тоже знает мир) |
| `DbToServer`/`ServerToDb`/`ServerToDb_Update` | 206/197/37 | **CacheD-RPC :2006** (те же имена семейств, что в cached-ref!) |
| `ClientToServerS` | 174 | C2S-приём 7777 |
| `ServerToNPCServer`/`NPCServerToServer` | 86/62 | **NPCSvr :2002** |
| `Socket`/`CNpRelaySocket` | 151/39 | Сеть + **NPRelay** |
| `Item`/`EquipmentItem`/`Warehouse`/`UserBMPackMgr` | 133/103/37/36 | Инвентарь/склад |
| `AbyssMgr`/`Abyss`/`Abyss_GAb1Castle` | 128/115/29 | Бездна |
| `Guild`/`Alliance*`/`Party`/`Group` | 79/40+35+28/30 | Социальное |
| `SkillEffectMgr`/`Skill`/`SkillEffect` | 90/72/116 | Скилл-движок (XMLNode-парсинг, rapidjson) |
| `MatchingMgr`/`World_BattleGround`/`World_IDArenaTournament`/`World_IDARENAPvP` | 36/37/33/31 | **Matchmaker-миры (InterSvrType=2)** — канон aion-ic подтверждён из бинаря |
| `RankMainProtocol` | 39 | Рейтинг (RankingServer) |
| `ServerToIC` | 44 | **ICServer :2005** |
| `HousingDb`/`WorldDb`/`UserDB`/`DB` | 44/43/102/61 | DB-доступы |
| `NpcScriptMgr` | 30 | Скрипты NPC (ScriptDLL64) |

- Packet-инфраструктура: `CPacket`, `CInvalidPacketBlocker`, `PacketProfiler`, `PacketProfileLogger`, `PacketResultProcessManager`, `SyncPacketMgr`, `PacketArgInfo` (202 функции-обработчика с UPacketArgInfo).
- SM_/CM_ имён в publics **НЕТ** (0) — семантика пакетов у NC живёт в handler-функциях, не в именах символов → маппинг опкодов делаем по сурсам-эталонам + capture (R1), PDB даёт ПОДСИСТЕМЫ.
- Крипта 7777 (pubs): `GGEncryptPacket/GGDecryptPacket` (_GG_AUTH_DATA = GameGuard), `Blowfish_Encrypt/Decrypt`, `AES_Encrypt` + (из пакета Samurai для 7.5: `crypt="AionGame7_5_0_0"`, checksumSize=3).

## 3. Конфиг-инвентарь (реальные прод-файлы, 2023 «Gardarika»)

- `config.xml`: serverTitle/serverId=1/serverName, **clientAcceptAddr:7777**, logserver 127.0.0.1:2051, `v45_update_date=2017-11-20` (RunAsDate-корреляция!).
- `common.xml` (9795 Б): канон портов **подтверждён целиком** — cacheServerPort 2006, logserverport 2051, petitionServerPort 2107, authServerPort 2104, accountCachedPort 2220, mxserverPort_1 10241, mxChatS2SPort 10254, shopAgentServerPort 10100, ICServerAddr/Port 127.0.0.1:2005, captcha: useCaptcha=true, bufferSize 10000, addr 127.0.0.1:43330 (у нас CAPTCHA :22206 = внешний редирект — см. aion-captcha README).
- Мусор/параметры-кандидаты: useTcpNoDelay, useOldAuthServer, waitUserMax, vpTicTime/vpMinLogoutTime, limitType*Gold (торговые лимиты).

## 4. Эталон #7 — Encom leak 7.5–7.7 (новое 08.10, RZ #1196933)

- Что: утёкшие Java-сурсы Encom (lineage Aion-Extreme → Encom), «Aion Emulator 7.5-7.7.rar» (107 МБ, MEGA), распаковано 597 МБ.
- Где: `reference/encom-leak-7577/` (GameServer/LoginServer/ChatServer/Commons + tools).
- Реестр: **CM=264/SM=347**, пакеты `com.aionemu.*`; `network/loginserver/clientpackets` = **GS↔LS протокол 2104** (бонус для aion-authd R6!).
- `ServerPacketsOpcodes.java`: полный опкод-реестр серверных пакетов (комменты «7.0 KR» — база Encom = 7.0 KR, адаптация на 7.5); SM_KEY 0x48, SM_VERSION_CHECK 0x00, SM_TIME_CHECK 0x27, SM_ACCOUNT_PROPERTIES 0xf0, SM_CHARACTER_LIST 0xc8, SM_MAC_INFO 0x168 (2-байтовые опкоды!).
- 💎 **Packet Samurai** (сниффер) с `dist/protocols/`: **Game_7.5.x.xml = 11114 строк, 938 packet-записей** (имя пакета ↔ opcode, битсеты эффектов) + Game_5.8/6.5/7.0/7.2 + Chat_4.0/Login_4.0 XML — готовый словарь протокола под capture!
- Форум-факты [RZ 1196933]: leak = «Encom test files»; Voidstar держит SVN для 7.5–7.9 (доступ по PM); Robson26 (ex-Aion-Lightning) владеет Encom-сурсом с фиксами — контакт на будущее; minion-crash на 7.7 клиенте подтверждён сообществом (у нас фикс уже в aion-gm/docs/fix-minion-crash-20261005.md).

## 5. Сводка эталонов reference/ (7 шт, ~3.9 ГБ)

| Папка | Версия | CM/SM | HEAD |
|---|---|---|---|
| aion-germany | **7.8 EU Gameforge** + 5.8 | 481/666 | 562c5b2 |
| Mobius_AionEmu | 7.7 | 248/346 | d634851 tag 20260718 |
| **encom-leak-7577** | **7.5–7.7 (base 7.0 KR)** + Packet Samurai | 264/347 | RAR (MEGA leak) |
| AionLightning | 7.8.0 | 258/356 | 4427aa7 |
| aion-encombase-58 | 5.8 (Encom GitHub) | 237 | clone |
| aion-server (beyond-aion) | 4.8 | 233/285 | 81e409c |
| yoress-arp-475 (Aion-Core) | 4.7.5 | 227/282 | 6d670f1 |

## 6. Сверка «бинарёв ↔ сурсы» (первая итерация)

- RPC-семейства бинаря (ServerToDb/DbToServer/ServerToNPCServer/ServerToIC/RankMainProtocol) ↔ каноны cached-ref (RQ382/RP255), aion-npc (2002), aion-ic (2005) — имена 1-в-1, Shared-код NC подтверждён.
- Matchmaker-миры в бинаре (MatchingMgr/World_BattleGround/IDArena*) ↔ теория aion-ic «InterSvrType=2 = отдельный мир» — теперь **подтверждено из PDB**.
- Мир-объекты у Server64 (User/Creature/Npc/World) ↔ Java-эталоны (Player/Creature/Npc/World) — семантика 1-в-1, но wire-имена пакетов только в эталонах.
- Крипта бинаря (GG+Blowfish+AES) ↔ Packet Samurai crypt="AionGame7_5_0_0" — этап handshake 7777 закрывается capture'ом (R1).

## 7. Следующие шаги R0 (детализация ROADMAP)

1. Маппинг опкодов: наши 7.7 EU клиенты → aion-germany AL-Game 7.8 + Mobius 7.7 + Encom ServerPacketsOpcodes (7.5) — диффы реестров; арбитр = capture.
2. Wire capture 7777 (pktmon при логинах юзера) → сверка с Game_7.5.x.xml/эталонами (checksumSize=3, крипта GG/Blowfish).
3. Карта зависимостей финализировать (2002-коннекты NPC, 2006-RPC словари — ШАРИНГ с aion-cache/aion-npc).
4. Дизасм-углубление (CPacket handler'ы) — только после исчерпания сурсов.

## 8. Источники

- PDB: VM `D:\AION_LIVE_SERVER\MainServer\Server64.pdb` (tar-pipe 08.10, MD5 ok).
- Encom leak: RZ тред «Aion Emulator 7.5-7.7 with web and source files» (MEGA/GDrive), распакован unrar 7.12 (RAR5 v6-метод — 7z не берёт).
- Конфиги: `aion_fixes/MainServer.rar` (MainServer/common.xml, config.xml «Gardarika» 2023).
- Гист neon-dev (карта эмуляторов), GitHub (aion-germany/AionEncomBase).
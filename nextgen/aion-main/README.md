# aion-main — перепись Server64/MainServer (игровое ядро) — ДЕПРИОРИТЕТ, полный сервер бескомпромиссно

> ⬜ **НЕ НАЧАТ, ДЕПРИОРИТЕТ** (политика юзера: переписываем ВСЕ бескомпромиссно — низкий приоритет ≠ отмена; старт = последний, после мира-обвязки npc/cache). Тактический промежуточный слой — [../aion-binpatch/](../aion-binpatch/).
> Запуск чата: `WORKFLOW: main` ([../WORKFLOW.md](../WORKFLOW.md)). Стандарты S1–S12: [../README.md](../README.md) §4.

## 📊 Статус и фазы

| Фаза | Статус |
|---|---|
| R-сурсы: 2.7-кит Server64.exe (28МБ) на VM; PDB 284МБ; чужие эмуляторы | ✅ ЗАКРЫТ 08.10: 7 эталонов (~3.9ГБ) + [RESEARCH.md](RESEARCH.md) |
| R0: разведка (протокол клиента S/C-фреймы 7777, RPC CacheD 2006, NPC 2002, конфиги) | ✅ ЗАКРЫТ 08.10: [RESEARCH.md](RESEARCH.md) + [OPCODES.md](OPCODES.md) (688 пакетов, единая таблица) |
| R1: wire capture (pktmon 7777 при логинах юзера + 2006 из cache-трека — ШАРИРОВАТЬ с aion-cache!) | 🟡 LIVE#1 08.10: крипта подтверждена, 24 пакета проарбированы, 2 новых ([OPCODES.md §6](OPCODES.md)) |
| R2: протокол-док + Go-каркас (ops.yaml 637 пакетов, crypt/wire/dispatch, тень :7778, golden-тесты на capture) | ✅ 08.10 (06b0ac7) |
| R3: Go MVP — хендлеры-эхо + Session; интеграции-стабы (cached ping-only до aion-cache R1) | ✅ 08.10 (2035a69): E2E PASS |
| R3.5: мир-стейт по capture-раскладкам (13 SM-payload'ов прод, InitSequence канон) | ✅ 08.10 (bd10861): E2E PASS 9 фреймов байт-в-байт |
| R3.6: динамические подмены раскладок (layouts.yaml, hp/mp/time) | ✅ 08.10 (6ab4509): live-вердикт SM_STATUPDATE_HP |
| R4: fork tap-режим (пассивный разбор КОПИИ трафика, VERDICT) | ✅ 08.10 (6ab4509): тест на реальном capture 2147 кадров; полная A/B под миром = R4.5 |
| R4 A/B → R5 свитч (пара!) → R6 наблюдение | ⬜ |

## 📟 Канон (известное сейчас)

| Факт | Источник |
|---|---|
| Server64 = игровое ядро: **7777** (клиенты), **2002** (NPC-коннекты; критерий мира = 8 коннектов), клиент CacheD64 (:2006 RPC), ACS (:2220), лог (:2051) | live + cached-ref |
| Date-bypass **#180 уже в бинаре** (SHA256 доказано); RunAsDate = страховка | fixes-pending |
| RAM ~10 ГБ; хендлы 827k + 228/мин = кандидат утечки (watch-лист op) | PLAN §2 |
| `InterSvrType=1` live; **2 = matchmaker/beginner** («main difference», AKllX #26); matchmaker = отдельный мини-стек с портами 7778/aion_event=true | aion-ic/RESEARCH §2 |
| Матчмейкер арен = JZ→JNZ патч #108 (план); манастоны #111 (план) — до переписи | fixes-pending |
| Конфиги: `MainServer\common.xml` (captcha/useCaptcha, порты), `config.xml`, `common.xml` блок 3.0.1221 (ICServerAddr/Port/Id) | live + ic-RESEARCH |
| Клиент-протокол: семантика пакетов из Java-эталонов. **reference/ = 7 эталонов (~3.9ГБ)**: aion-germany 7.8 EU (481/666), Mobius 7.7 (248/346), **encom-leak-7577 = утёкшие сурсы Encom 7.5–7.7 (264/347) + Packet Samurai + Game_7.5.x.xml = 938 пакетов!**, AionLightning 7.8.0 (258/356), aion-encombase-58 (237), beyond-aion 4.8 (233/285), ARP/Aion-Core 4.7.5 (227/282) | reference/ + RZ #1196933 |
| PDB-карта Server64: 74164 publics; RPC-семейства ServerToDb/DbToServer/ServerToNPCServer/ServerToIC/RankMainProtocol = каналы 2006/2002/2005/ranking; крипта 7777 = GG(GameGuard)+Blowfish+AES; Matchmaker-миры (MatchingMgr/IDArena*) из бинаря = InterSvrType-канон | [RESEARCH.md](RESEARCH.md) |

## 🚧 Блокеры

- Самый большой объём протокола в проекте (мир-пакеты) — MVP только после npc/cache.
- Capture 7777 возможен только при логинах юзера (форк мира — дорого).

## ⏭️ Следующий шаг

`WORKFLOW: main` → **R1**: pktmon 7777 capture при логинах юзера → арбитраж 256 DIFF-опкодов + верификация 15×«7.7 EU TODO» (реестр: [OPCODES.md](OPCODES.md) + docs/opcodes-unified-0810.csv). Промпт: [PROMPT.md](PROMPT.md).

## 📦 Артефакты

| Что | Где |
|---|---|
| Промпт | [PROMPT.md](PROMPT.md) |
| PDB 284МБ + publics-реестр 74164 + common/config.xml | локально `aion_rev_2026-10-05/artifacts/pdb-big/Server64/` (MD5 ✅ 08.10) + VM |
| Java-эталоны (7 шт, ~3.9ГБ) | `reference/` (песочница): aion-germany 562c5b2, Mobius d634851, **encom-leak-7577** (MEGA-RAR 08.10, + Packet Samurai/protocols), AionLightning 4427aa7, aion-encombase-58, beyond-aion 81e409c, yoress-arp 6d670f1 |
| 2.7-кит | VM `D:\SAION\downloads\rz\unpacked\2.7\` |
| Ориг | VM `D:\AION_LIVE_SERVER\MainServer\` (задача AionMain; пара с AionNPC) |
| Креды/доступы | VM `D:\SAION\creds\` |

## 🗺 Карта портирования с эмуляторов (анализ 08.10)

Mobius 7.7 gameserver = **2783 java-файла**: serverpackets 314 / clientpackets 219 (= наш реестр 637 ✅), **skillengine 250** (самое мясо), **dataholders 129** (стат-данные), dao 74 (БД-паттерны → наши cached-RPC), model 46 пакетов, **services 93 файла** (ag78=94): приоритет портирования по live-пойманным сессиям: Account/Dialog/Exchange/Broker/Housing/CubeExpand/AutoGroup/GameTime/FindGroup → остальное. Портируем СЕМАНТИКУ (не код): Go-структуры = минимум (User/Item/Skill/Group ~15 структур), стат-данные = из клиентских 2.7/5.8 китов позже.

## 📜 Логи

Стандарт S3: raw-first io-дампы + fork C>/O>/N> ([../LOGGING-SPEC.md](../LOGGING-SPEC.md)); ориг пишет `MainServer\log\{{date}}.err` (дата RunAsDate!) — тейлерит op; наша перепись = .err-совместимый или перенос тейлеров.

📊 сосед узнал (08.10, от aion-npc RESEARCH): Abyss-логика у Java-эталонов (Mobius7.7/AL7.8, клоны в `../../reference/`) живёт в NPC-AI (`SimpleAbyssGuardHandler`), не в GS-ядре ⇒ поддерживает теорию «abyss-цикл 60с — NPCSvr»; мир Server64 в эталонах = один GameServer, мимикрия только подсистемная (world/creature/stats).
📊 сосед узнал (08.10, от aion-npc R0 netstat live): «8/8 conns на :2002» (критерий мира op) = 8 соединений **NPCSvr64→Server64** (NPC-канал, не игроки); Server64 дополнительно слушает **:2012** (⏳ назначение); Server64 держит 9 коннектов к IC:2005.
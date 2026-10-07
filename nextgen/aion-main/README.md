# aion-main — перепись Server64/MainServer (игровое ядро) — ДЕПРИОРИТЕТ, полный сервер бескомпромиссно

> ⬜ **НЕ НАЧАТ, ДЕПРИОРИТЕТ** (политика юзера: переписываем ВСЕ бескомпромиссно — низкий приоритет ≠ отмена; старт = последний, после мира-обвязки npc/cache). Тактический промежуточный слой — [../aion-binpatch/](../aion-binpatch/).
> Запуск чата: `WORKFLOW: main` ([../WORKFLOW.md](../WORKFLOW.md)). Стандарты S1–S12: [../README.md](../README.md) §4.

## 📊 Статус и фазы

| Фаза | Статус |
|---|---|
| R-сурсы: 2.7-кит Server64.exe (28МБ) на VM; PDB 284МБ; чужие эмуляторы (beyond-aion 4.8/Mobius 7.7 Java — семантика пакетов!) | ✅ частично (reference/ на песочнице) |
| R0: разведка (протокол клиента S/C-фреймы 7777, RPC CacheD 2006, NPC 2002, конфиги common.xml/config.xml/InterSvrType) | ⬜ |
| R1: wire capture (pktmon 7777 при логинах юзера + 2006 из cache-трека — ШАРИРОВАТЬ с aion-cache!) | ⬜ |
| R2–R3: Go `nextgen/aion-main` MVP: мир-пакеты (движение/чат/инвентарь) + интеграция aion-cache/aion-npc/aion-authd | ⬜ |
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
| Клиент-протокол: семантика пакетов из Java-эталонов. **reference/ = 5 эталонов** (веб-ресёрч 08.10, все свежие HEAD): Mobius_AionEmu 7.7 (CM 248/SM 346), **aion-germany 7.8 EU Gameforge + AL-Game-5.8 (CM 481/SM 666 — максимум!)**, AionLightning 7.8.0 (CM 258/SM 356), beyond-aion 4.8 (CM 233/SM 285), Yoress ARP = Aion-Core 4.7.5 open-source (CM 227/SM 282) | reference/ + гист neon-dev |

## 🚧 Блокеры

- Самый большой объём протокола в проекте (мир-пакеты) — MVP только после npc/cache.
- Capture 7777 возможен только при логинах юзера (форк мира — дорого).

## ⏭️ Следующий шаг

`WORKFLOW: main` → R0: конфиг-инвентарь + карта зависимостей + Java-эталоны сматчить к нашим опкодам — приоритет маппинга: **aion-germany AL-Game 7.8 (EU/Gameforge = наш регион!) + Mobius 7.7 (наша версия!)**, затем AionLightning 7.8 → cross-check; beyond-aion 4.8/ARP 4.7.5 = ретро-семантика (ШАРИРОВАТЬ с aion-npc/aion-cache!) → ROADMAP-детализация. Промпт: [PROMPT.md](PROMPT.md).

## 📦 Артефакты

| Что | Где |
|---|---|
| Промпт | [PROMPT.md](PROMPT.md) |
| PDB 284МБ | VM `D:\AION_LIVE_SERVER\MainServer\` (манифест manifest-pdb-big) |
| Java-эталоны (5 шт, ~3.2ГБ) | `STELGEN/projects/aion_server_2026-10-02/reference/` (песочница, вне гита): `Mobius_AionEmu` (HEAD d634851 tag 20260718 «Aion 7.7 update»), `aion-germany` (HEAD 562c5b2, **EU 7.8 Gameforge** + AL-Game-5.8, shallow), `AionLightning` (HEAD 4427aa7, ветка 7.8.0), `aion-server` = beyond-aion 4.8 (HEAD 81e409c), `yoress-arp-475` = Aion-Core 4.7.5 (HEAD 6d670f1). Опкоды встроены в Java-классы (отдельных opcode-файлов нет) |
| 2.7-кит | VM `D:\SAION\downloads\rz\unpacked\2.7\` |
| Ориг | VM `D:\AION_LIVE_SERVER\MainServer\` (задача AionMain; пара с AionNPC) |
| Креды/доступы | VM `D:\SAION\creds\` |

## 📜 Логи

Стандарт S3: raw-first io-дампы + fork C>/O>/N> ([../LOGGING-SPEC.md](../LOGGING-SPEC.md)); ориг пишет `MainServer\log\{{date}}.err` (дата RunAsDate!) — тейлерит op; наша перепись = .err-совместимый или перенос тейлеров.

📊 сосед узнал (08.10, от aion-npc RESEARCH): Abyss-логика у Java-эталонов (Mobius7.7/AL7.8, клоны в `../../reference/`) живёт в NPC-AI (`SimpleAbyssGuardHandler`), не в GS-ядре ⇒ поддерживает теорию «abyss-цикл 60с — NPCSvr»; мир Server64 в эталонах = один GameServer, мимикрия только подсистемная (world/creature/stats).
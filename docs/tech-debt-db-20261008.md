# 🧰 TECH-DEBT: база 7.7 кита (~85%) — аудит, источник слива, план восстановления (08.10.2026)

> Статус: 🟡 OPEN. Владелец-трек: aion-main R0-ресёрч (база — общий слой cache/main/npc; воркфлоу: кросс-пульс всем).
> Связь с known issues: [fixes-known-issues.md](fixes-known-issues.md) (баги кита), [fixes-registry.md](fixes-registry.md), [../nextgen/aion-main/RESEARCH.md](../nextgen/aion-main/RESEARCH.md).

## 1. Происхождение слива (RZ-ресёрч 08.10)

- Наш кит = слив **fyyre** (PTS 7.7 C++ + matching PDB), выложен subzeros в RZ тред **1205286 «Aion 7.7 C++ server files»** (10.05.2022, 7 страниц): «come with matching.PDB files… Thanks fyyre» [1].
- Сообщество с 2022 фиксирует неполноту базы: «need to rebuild aionaccounts database… missing» (#4 Mantios) [1], «PTS but missing some important database» (#10) [1], «you cant get it work without missing Database tables!» (#20 Master2012) [1].
- Ключевой факт от восстановителя БД: **pada8801 (page 7, #121, 26.12.2023): «my Aion 7.7 DB is about ~85% finished»** — наша БД = недоделанная реставрация pada8801, автор сам говорит про ~85% [2].
- Mantios: нужен 24+ ГБ RAM и «know how to rebuild database» (#123) [2]; готовых чистых фикс-сборок в публичном поле нет — «China have fixed and Russian but they are not shared» (#130 Mantios) [2].
- Полезные факты треда: jump-персонажи (instant 80) = таблица `jumping_character_config` в **AccountCacheD** (#126 klon22) [2].

## 2. Аудит нашей БД (08.10, live MSSQL VM)

- `_AionWorldNew114_rc` = **174 таблицы NC-схемы** (инвентарь в [scripts/sql](../scripts/sql/)); живые группы: user_* (профиль/инвентарь/скиллы/квесты), guild_*, house_* (6 таблиц!), abyss_*, vendor_*, luna_*, town_data.
- **Дома**: NC-таблицы `house_addrinfo/house_field/house_field_script/house_instant/house_instant_script/houseobject/houseobject_extdata` — ПРИСУТСТВУЮТ. Баг «дома после lvl 9» — НЕ БД, а логика бинаря (community: требуются IDA-патчи Server64, RZ 1211744 пост #201) → см. [fixes-known-issues.md §2](fixes-known-issues.md).
- **Миньоны**: NC-таблиц minion-семейства в схеме НЕ ВИДНО (`user_familiar/user_pet` = старые питомцы); алерты proc_missing миньонов в окне пока нет — проверить live-тестом (capture#2: действия с миньоном) → обновить этот док.
- **Недостающие stored-procs (подтверждено op-алертами proc_missing, 08.10): 8 уникальных**:
  `aion_GetItemCollectionExpiredList, aion_GetItemCollectionList, aion_GetItemCollectionCompleteTimeLimitList, aion_GetItemCollectionCompleteList, aion_GetItemCollectionLevelList, aion_LoadReinventInfo, aion_LoadFameInfo, aion_getItemAttributeDeltaListAll_20190919` (+ в ранних алертах: `aion_GetItemAttributeDeltaListAllVendorDark/Light_20190919, aion_DeleteItemByDate_20191206`).
  → Блокируют: коллекции предметов, слава (fame), реинкарнация (reinvent), дельта-атрибуты вендоров.
- ⚠️ Имена procs НЕ лежат строками в Server64.exe (strings ASCII/UTF-16 = 0 совпадений; динамическое построение/упаковка) → полный каталог недостающего строится ТОЛЬКО мониторингом `proc_missing`-алертов op при живом прогоне фич.

## 3. Эталоны vs наша схема (сравнение 08.10)

| Источник | Таблиц в дампах | Оценка |
|---|---|---|
| aion-germany **AL-Game 7.8** (sql/) | 89 | САМЫЙ близкий к 7.7: player_collections(+infos,+transform), player_minions, player_fame, houses(+bids+cooldowns+scripts), player_monsterbook, petitions, player_shugo_sweep, special_landing |
| Mobius 7.7 (dist/sql) | 96 | почти как 7.8 (minions/collections/fame/houses есть) |
| Encom leak 7.5–7.7 (GameServer/sql) | 87 | + player_transform_collections, competition_ranking |
| AL-Game-5.8 (al_server_gs.sql) | 76 | базовый слой 5.8 (Player_Minions.sql, MinionSkillPoints, MinionBirthdayFix) |
| beyond-aion 4.8 / ARP 4.7.5 | 61/62 | ретро (без minions/collections — их ещё не было) |

⚠️ **Политика миграции**: Java-дампы = MySQL-схемы эмуляторов, имена/структуры ≠ NC-схема (`user_*`). Напрямую в MSSQL НЕ переносимы. Использование: (1) семантика полей фичи (что сохранять у миньона/коллекции), (2) значения/дефолты, (3) наша Go-перепись (там схема наша).
- **БД 5.8 PTS кита** (VM `D:\SAION\downloads\rz\unpacked\5.8`) — NC-схема, ближе всех к 7.7 (community: RZ 1211744 #201 «БД 5.8 PTS ближе к 7.7 — оттуда можно подтягивать») → ГЛАВНЫЙ донор недостающих NC-procs/таблиц.
- PDB Server64 (74164 publics) — имя procs не в publics, но **сигнатуры вызовов ODBC-функций восстанавливаются дизасмом** по месту алерта (адрес вызова можно взять из лога-трейса) → фаза R2 aion-main.

## 4. Тех-долг (план восстановления, порядок)

| # | Задача | Источник решения | Оценка | Статус |
|---|---|---|---|---|
| TD1 | Каталог недостающих NC-объектов: ночной прогон всех фич + сбор proc_missing из op-алертов в единый список | op-алерты (канал готов) | часы | 🟡 частично (8+3 procs) |
| TD2 | Воссоздание procs: коллекции (5 шт) + fame + reinvent + itemAttributeDelta + DeleteItemByDate | кит 5.8 PTS DB (донор) + сигнатуры из PDB-дизасма + аттач #105 (SetCharInfo прецедент) | 2–4 дня | ⬜ |
| TD3 | Миньоны: live-тест (capture#2) → определить NC-таблицы/procs миньонов → восстановить/патч | capture + эталоны (Player_Minions.sql семантика) + кит a7741288 (#28) | 1–2 дня | ⬜ |
| TD4 | Дома после lvl 9: IDA-патч Server64 по референсу #201 (патч-план в aion-binpatch) | fixes-pending + IDA | 1–3 дня | ⬜ |
| TD5 | Сравнить схему кита 5.8 PTS DB с нашей (174 таблицы) полным диффом (таблицы+procs) | VM кит 5.8 | 1 день | ⬜ |
| TD6 | jump_character_config (jump 80) — включить в AccountCacheD-схему | тред #126 klon22 | часы | ⬜ |

## 5. Источники

- RZ 1205286 p.1 [1] и p.7 [2] (цитаты выше), наш инвентарь аттачей 1211744: [tools/ragezone-1211744/README.md](../tools/ragezone-1211744/README.md).
- Сканы: наша БД (sqlcmd sys.tables, 08.10), эталонные .sql (reference/), op proc_missing-алерты, strings Server64.exe (ASCII+UTF-16).
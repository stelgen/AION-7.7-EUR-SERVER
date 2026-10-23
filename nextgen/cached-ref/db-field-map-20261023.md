# DB-FIELD MAP — Этап 0 (23.10.2026): офсет↔колонка для O1 RP_LOAD

> Сверка capture-кадра O1 (1149Б payload, чар 1002 SteLGeN, sess1) с БД user_data 1002.
> Фикстура: `testdata/db-userdata-1002-1004.txt` (полный user_data + 166 user_item 1002 + 1004).

## ПОДТВЕРЖДЕНО байт-в-байт (payload-relative, Т31 расширенная)

| Офсет payload | Содержание | Значение capture | БД-источник | Статус |
|---|---|---|---|---|
| [0:4] | X echo | sess.X | C288 | ✅ Т31 |
| [21:25] | charRef LE | 0x020003EA | 1002\|0x02000000 | ✅ |
| [49] | **level SINGLE BYTE** | 0x50=80 | user_data.**lev**=80 | ✅ **точно** |
| [903:928] | charName UTF-16 | "SteLGeN-잉기스온" | имя чара + зона (잉기스온=Ингисон) | ✅ источник: charName+текущая зона ⏳ где зона в БД |
| [428] | float x | 1363.44 | user_data.**last_normal_xlocation**=1363.4438 | ✅ |
| [432] | float y | 2298.90 | **last_normal_ylocation**=2298.897 | ✅ |
| [436] | float z | 296.67 | **last_normal_zlocation**=296.67371 | ✅ |
| [448] | float x | 1363.20 | **resurrect_xlocation**=1363.2002 | ✅ |
| [452] | float y | 1513.50 | **resurrect_ylocation**=1513.4958 | ✅ |
| [456] | float z | 569.16 | **resurrect_zlocation**=569.15802 | ✅ |
| [408] | float ⏳ | 1410.52 | user_data.??? (не xlocation=2539.53!) | ⏳ |
| [412] | float ⏳ | 1508.74 | ⏳ | ⏳ |
| [416] | float ⏳ | 573.33 | ⏳ (dir? 78°→rad 1.36 нет) | ⏳ |
| [464] | float ⏳ | 573.33? нет: 569.16@456; тут др. | ⏳ | ⏳ |

## ВЫВОДЫ

1. **RP_LOAD несёт last_normal + resurrect точки, НЕ текущую позицию** (xlocation=2539.53 в capture отсутствует) — чар спавнится в last_normal точке. Логично.
2. **Текущая позиция для эмиссии** = last_normal_*, БД-источник прямой.
3. **user_data.lev** = уровень в [49] байт-в-байт.
4. **user_item 1002 = 166 строк** (id, char_id, name_id, slot_id, amount, slot, warehouse, tid, dynamic_property) — источник O109.
5. Остаток ⏳: [408/412/416] (3 float), статы 0x30-0x4D-зона, [464], точная семантика [903] имени+зоны.

## user_data колонки (схема снята)

Полный список в фикстуре. Ключевые для O1: lev, exp, abyss_point, xlocation/ylocation/zlocation/dir, last_normal_*, resurrect_*, now_hit, now_mana, name_id, race, class, cur_title_id, playtime, world, builder, guild_id, feat_* (внешность 1:1 как в capture), inventory_growth.

## user_item схема

`id, char_id, name_id, slot_id, amount, slot, warehouse, create_date, update_date, producer, tid, expired_time, buy_amount, buy_duration, main_item_dbid, dynamic_property, import_id, export_id, server_of_origin`

## SQL-доступ

aionop_ro: GRANT SELECT на _AionWorldNew114_rc ВЫДАН (23.10, «го» юзера). user_item 1002=166 строк, 1004=? (в фикстуре), user_data=1 строка/чар.

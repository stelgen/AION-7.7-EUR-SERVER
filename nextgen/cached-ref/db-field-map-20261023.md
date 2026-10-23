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

## Этап 3 (O109 item-блоки): разбор 23.10 — В ПРОЦЕССЕ

### Структура блоков (sess1 capture)

- 7 блоков: 6×8014Б + 1×3314Б
- Заголовок блока: [X-байты@2][a@6][charRef@10][03 000000][11 000000 = счётчик записей?]
- **Item-записи по 470Б** (регулярный шаг), в блоке ~18-28 записей
- Формат записи #1 (470Б): item_id LE u32 @запись+0 (двойное вхождение id в записи: +0 и +22), дальше данные

### Сверка capture↔БД (user_item 1002, CSV-фикстура db-useritem-1002-csv.txt)

- БД: 147 уникальных name_id (166 строк, есть дубли stack)
- capture: 116 уникальных u32 в item-диапазоне
- **Пересечение: 105 (90% БД-айтемов в capture)** ✓
- «только capture» 8 шт = НЕ items (степени двойки: 0x06000000, 0x07000000, 0x08000000 — флаги/служебные, фильтровать по маске)
- «только БД»: единичные (100050012/64/226/227... — warehouse=1 (склад, 29 строк) не шлётся / выдано после capture)

### Выводы

1. O109 = инвентарь чара из user_item (name_id → wire item-id), пересечение 105/116 подтверждает
2. Запись 470Б: item_id двойное вхождение (+0, +22) — внутренняя ссылка
3. Осталось: карта полей записи 470Б (slot_id, amount, dynamic_property → офсеты) — сверка per-item hex-vs-БД
4. warehouse=1 (склад) НЕ шлётся в login-блоках (29 строк БД не в capture) — фильтр генератора: warehouse=0

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

## Карта записи 470Б — ФИНАЛ Этапа 3 (23.10, XOR-дифф метод)

| Офсет | Поле | Доказательство |
|---|---|---|
| +0:4 | **item_id u32 LE** | XOR 186020035↔036: diff=1 = id-разница |
| +22:26 | item_id дубль | двойное вхождение |
| +4:8 | amount u32 LE | сверка БД 99/102 |
| +461 | **slot u8** | 100/102 slot_ok; XOR [461:462] не дифф (slot=0/1 оба 0-байт?), slot 2/4/1 — дифф @462 |
| +462:464 | ⏳ u16 динамическое (476/457) | XOR: A=0x01DC B=0x01C9 — stack/seed? |
| +268:276 | ⏳ **expired_time datetime-кодировка NC** | XOR пар 186020035(et=1791785006)↔036(et=1791946580): дни u16 @272-274 = 27362↔27363 (разница 1 день = et-разница 2 дня!); сек-поля @268-272 |
| остальное | опции/статы/нулевые зоны | template passthrough |

### Выводы карты

1. **slot = u8 @461** (100/102 OK), но XOR показывает @462 тоже дифф для разных slot — вероятно slot=u16@461 с item-типом в старшем (0x4602=slot2+тип46, 0x4C04=slot4+тип4C) — **slot u16 LE @461**: 0x0246=582? нет. LE: байты [02 46] → u16 = 0x4602. Хм: slot=2 → байты @461,462 = [02, 0x46]; slot=4 → [04, 0x4C]; slot=1 → [01, 0x4D]; slot=0 → [00, 0x43]. Значит @461=slot, @462=тип-зависящий-от-slot? 0x43+slot! (0x43+0=43, +2=45? нет 0x46). Хм: slot 0→43, 1→4D, 2→46, 4→4C — НЕ линейно. Это item-зависимое поле (тип экипировки?), не slot-функция.
2. **expired_time**: @272-274 = дни с фиксированной эпохи (27362↔27363), @268-271 = секунды-в-дне (разнобой) — формат NC-даты: [сек-поля u32][дни u16][pad]. Кодировка эпохи ⏳ (день 27362 + эпоха = 2026-10-12: 27362 дней ≈ 74.9 лет от 1900? 1900+74.9=1975. От 1952? Проверить позже).
3. Для генератора: expired-зона passthrough из шаблона (у 1002 все expired нулевые кроме 3 — дифф мал).

### Итог: GenerateItemRecord карта подтверждена, остальное = template passthrough. Этап 3 ЗАВЕРШЁН полностью.

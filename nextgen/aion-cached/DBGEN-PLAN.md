# DB-ГЕНЕРАТОР — план этапов (Вариант Б, 23.10.2026)

> GO юзера. Цель: RP-кадры мир-init из БД (для ЛЮБОГО чара/учётки), не из replay-шаблонов.
> Каркас эмиссии подтверждён live (sess4: SAME+nearSame на чужом чаре = структурная целостность).
> Предел replay: шаблон чара 1002 не может обслужить чар 1004 (данные чара = 215Б diff в O1).

## Этап 0: Инвентарь полей (офсет↔колонка) — ~2-3 часа

Метод: hex capture-кадров чара 1002 ↔ SQL-данные чара 1002 (сверка офсет-в-офсет).

| Кадр | Что содержит (гипотеза из Т31/R4) | Источник в БД |
|---|---|---|
| O1 (1149Б) | charRef@21, level@49, charName UTF-16@903:928, координаты@399:460, тайминги@468:477, статы 0x30-0x4D | global_user_data (user_level, class, race), user_data (позиция/статы) |
| O14 ×3 (zlib 7194Б) | CHARACTER SETTINGS XML 3 страницы (options 28578Б, quickbar 3098Б, favorites 744Б) | character_settings (уже пишем через SettingsDB!) |
| O109 ×7 (item-блоки 8014Б) | item-инвентарь чара: [ffff] паттерн, item IDs 100000000-199999999, слоты/кол-во | user_item (мир-БД _AionWorldNew114_rc) WHERE char_id |
| O108 (2630Б) | ⏳ разбор | ⏳ |
| O23 (1385Б) / O32 / O95 (148Б) | ⏳ | ⏳ |
| O97 (1115Б) / O242 (1926Б) / O244 / O239 / O245 | ⏳ | ⏳ |

Артефакт этапа: `nextgen/cached-ref/db-field-map-20261023.md` — таблица офсет↔колонка per-cmd.

## Этап 1: O1 RP_LOAD полный DB-генератор — 1 день

- rp_emitter.go: GenerateRPLoadStruct(ctx, ex, sessionX, charId, accId, charName) УЖЕ есть (Т31: X@0, charRef@21, level@49 SINGLE BYTE, charName@903:928)
- Дополнить: координаты (user_data spawn), статы 0x30-0x4D, тайминги
- SQL: SELECT из global_user_data + user_data (мир-БД) через aionop_ro
- Критерий: сгенерированный кадр чара 1002 vs sess1-capture = diff ≤ рантайм-зоны
- Критерий-2: сгенерированный кадр чара 1004 vs sess4-capture (другой чар = данные из БД должны совпасть!)

## Этап 2: O14 settings XML — полдня (каркас ГОТОВ)

- settings_store.go + settings_db.go уже реализованы (кэш zlib XML per-charId + персист в character_settings)
- DB-генератор = чтение из character_settings; если записи нет → эмитить last-known capture (transparent fallback) или дефолт
- Проверка: логин 1004 → тень пишет settings 1004 → следующий логин 1004 читает СВОИ настройки (not 1002!)

## Этап 3: O109 item-блоки из user_item — 1-2 дня

- item_blocks.go: ParseItemBlocks знает wire-формат (cmd=109 паттерн)
- Построить ОБРАТНЫЙ сериализатор: user_item rows → wire-кадры ×7
- SQL-доступ: вторая строка подключения к _AionWorldNew114_rc (гранты aionop_ro — добавить SELECT user_item)
- Критерий: инвентарь 1002 из БД → байты == sess1 O109×7

## Этап 4: Мир-кадры O108/O23/O32/O95/O97/O242/O244/O239/O245 — 2-4 дня

- Каждому: офсет-карта (этап 0) → DB-источник ИЛИ шаблон-с-подстановками (если контент статичен)
- Приоритет: O108 (2630Б, в каждом логине), O23 (1385Б), O242 (1926Б)

## Этап 5: Интеграция + live A/B на ЛЮБОМ чаре — 1 день

- EmitWorldFlowDB(sess, charRef) — ветка в case 11: DB-генерация вместо chain
- Флаг конфига: world_source: db|replay (replay = fallback)
- Деплой → логин ЛЮБОГО чара (1004!) → S-vs-O дифф → **nearSame ≥90% на ЛЮБОМ чаре = R5 вердикт**

## Этап 6: R5 свитч CacheD64 по «го» — откат одной командой

- common.xml cacheServerPort 2006→2016 (копия) | наш :2006
- cdmirror убирается из цепочки; откат = обратная правка + рестарт пары

## Правила

- Каждый этап: код + тест на capture-фикстуре + коммит (правило «исправил = удалил» для дашборда)
- SQL только read-only (aionop_ro); никаких INSERT в прод-БД из генератора
- Данные чара 1002/1004 = эталонные фикстуры в testdata (без секретов)

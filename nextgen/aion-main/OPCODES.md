# 🗺 OPCODES.md — единая карта опкодов Game-протокола (R0, 08.10.2026)

> Маппинг 7.7 EU. Источники: XML-словарь Packet Samurai 7.5 + 4 Java-реестра (Encom 7.5, Mobius 7.7, aion-germany 7.8, AL-Game-5.8).
> Полная таблица: [docs/opcodes-unified-0810.csv](docs/opcodes-unified-0810.csv) (688 строк). Регенерация: `tools/analysis/opcodes_map.py`.

## 1. Метод

- **XML75**: `Game_7.5.x.xml` (Encom-leak/Packet Samurai): `<packet id name>` — 475 имён.
- **Java-реестры** (два файла на репо): `factories/AionPacketHandlerFactory.java` (CM: `addPacket(new CM_X(0x…))`) + `aion/ServerPacketsOpcodes.java` (SM: `addPacketOpcode(SM_X.class, 0x…)`). Комменты тегируют версию (`// 7.5 EU`, `// 7.7 EU TODO`).
- Ключ = имя пакета; направление из префикса SM_/CM_. Опкоды u16.

## 2. Сводка (688 имён: SM 390 / CM 274)

| Метрика | Значение |
|---|---|
| Имена с данными в 4–5 источниках | 352 |
| **FULL-согласие** (все имеющиеся источники = один опкод) | **318** |
| DIFF (источники расходятся) | 256 |
| Только в одном источнике | 114 |
| Полнота источников: ag78 526 · ag58 499 · mobius77 503 · encom75 496 · xml75 475 | — |
| Тег «7.5 EU» в Java | 323 пакета |
| Тег «7.7 EU» (новое, Mobius «TODO») | 15 пакетов (CM_LUMIEL_TRANSFORM 0x3FD, CM_COMBAT_SUPPORT 0x3FC, CM_PLAYER_COLLECTION_REGISTER 0x3F2…) |
| Имена только в 7.7/7.8 (нет в 7.5-реестрах) | 95 (брокер/эчивмент/кубик/даэванион…) |
| Имена только в XML (Java их не знает) | 21 (CM_GG 0x13F, CM_MEGAPHONE 0x1B5, SM_AION_TV 0x146, SM_AETHERFORGING_* …) |

## 3. ⚠ Проверенный факт: XML-словарь частично неточен

21 пакет, где все Java-реестры согласны, а XML даёт другой/нет опкода (пример: CM_GG, CM_MEGAPHONE, SM_CASH_BUFF xml=0xFD vs Java 0xF6-семейство). При споре XML vs Java **арбитраж = Java-реестр** (Encom/Mobius/ag согласованы между собой: CM_ATTACK 0x0F6 во всех Java против 0x0FB в XML).

## 4. Правила арбитража для 7.7 EU клиента

1. **FULL-согласие 3+ Java-реестров** → опкод принят (базовый слой).
2. **Тег «7.7 EU» в Mobius** → принят как кандидат (TODO = не верифицирован live).
3. **Только в ag78/ag58** (пакеты после 7.5) → кандидат, приоритет ag78 (EU-регион).
4. **Только в XML** → кандидат низкого доверия (новее Java-базы), проверка capture'ом.
5. Арбитр всего = live capture 7777 (R1 pktmon при логинах юзера).

## 5. Вывод для переписи

- Базовый слой 7.5 EU (323 тега) + 15×7.7 EU + 95×7.8-новых = реестр наших клиентов 7.7 EU закрывается из эталонов на ~90%; остаток — неизвестные опкоды с wire (логируем raw, LOGGING-SPEC).
- Протокол-эволюция 7.5→7.8 мала по структуре (256 DIFF в основном от дрейфа чисел, не имён) → Go-диспетчер делаем таблицей «имя → handler», опкоды в YAML-конфиге (S7) для правок без пересборки.

## 6. ✅ R1 LIVE-АРБИТРАЖ #1 (08.10, capture 108 сек, юзер-сессия 192.168.1.6:49831)

- Capture: pktmon 7777 full-payload → pcapng (779 tcp-фреймов, 301КБ; игровых пакетов: S2C 1731 + C2S 89); файл `STELGEN/tmp/cap7777/cap7777-0810.pcapng`, дешифратор `tools/analysis/cap7777_decrypt.py`.
- 🔒 **КРИПТА 7.7 EU ПОДТВЕРЖДЕНА LIVE (0 invalid из 1820 пакетов!)** = эталонная схема 7.5/7.8 1-в-1: wire `[u16 size LE self-inclusive][тело]`; S2C тело `[E=(op+0xD8)^0xD9][0x56][u16 ~E][payload]`, C2S тело `[op][0x75][u16 ~op][payload]`; rolling key 8Б `[baseKey LE][A1 6C 54 87]`, key+=len(тела) за пакет; SM_KEY (0x48, первый пакет, НЕ шифрован) несёт `falseKey`, baseKey=(falseKey-0x3FF2CCDF)^0xCD92E4D9; staticKey 64Б `nKO/...?3iI9`.
- Live-поймано (op → имя → вердикт): S2C: 0x000E SM_NPC_INFO×349 ✅FULL, 0x00B8 SM_CUSTOM_SETTINGS×282 ✅, 0x0037 SM_MOVE×247 ✅, **0x0182×151 → кандидат SM_STRONGHOLDS (только encom75; в Mobius/ag78 выпилен — Encom leak единственный источник!)**, 0x00D1 SM_SIEGE_LOCATION_INFO×148 ✅, 0x0016 SM_DELETE×71, 0x007A SM_AUTO_GROUP×56 ✅, 0x00A8 SM_WAREHOUSE_INFO×46, 0x00A4 SM_WINDSTREAM_ANNOUNCE×46, 0x0031 SM_ABNORMAL_STATE×29, 0x002C SM_SKILL_LIST×29, 0x00ED SM_RIFT_ANNOUNCE×15, 0x0149 SM_LUNA_SYSTEM_INFO×14 (xml75 называл SM_LUNA_SHOP_LIST — дрейф имён), 0x001A SM_INVENTORY_INFO×12, 0x0152 SM_FLAG_INFO×11. C2S: 0x0106 CM_MOVE×28 ✅ (xml75 ошибочно CM_LEGION_MODIFY_EMBLEM — Java-арбитраж прав), 0x00E4 CM_TIME_CHECK×10 ✅, 0x02FD CM_EMOTION×8, 0x0100 CM_FILE_VERIFY×6, 0x0171 CM_CHAT_AUTH×5, 0x02F9 CM_CHARGE_ITEM×3, 0x02FC CM_GM_COMMAND_SEND×2 (DIFF vs xml75 CM_CLOSE_DIALOG), 0x00F7 CM_CASTSPELL×2, 0x0163 CM_MOVE_ITEM×2, 0x017D CM_FRIEND_STATUS×2, и по разу: CM_PING 0x02F2, CM_VERSION_CHECK 0x00D6, CM_L2AUTH_LOGIN_CHECK 0x0158, CM_MAC_ADDRESS 0x0180, CM_CHARACTER_LIST 0x0159, CM_MAY_LOGIN_INTO_GAME 0x018D, CM_SHOW_BLOCKLIST 0x0161, CM_SHOW_FRIENDLIST 0x01A9, CM_INSTANCE_INFO 0x0197, CM_ACHIEVEMENT_COMPLETE 0x01E9, CM_LUNA_SYSTEM 0x01DE, CM_GET_HOUSE_BIDS 0x01AD, CM_BLOCK_DEL 0x017E.
- 🔥 **Высокие опкоды 0x2F2–0x2FD у 7.7 EU клиентов СУЩЕСТВУЮТ LIVE** (CM_PING 0x02F2!) — семейство «второй» диапазона подтверждено; CM_LUMIEL_TRANSFORM 0x3FD в сессии не участвовал (не триггерился).
- **Новые кейсы**: C2S **0x00D4×1 — нет ни в одном реестре** (кандидат-сосед CM_TELEPORT-семейства; payload-анализ — следующий шаг).
- Итог: 1668 known / 152 unknown(=только 0x0182+0x00D4) / 0 invalid. Реестр OPCODES.md подтверждён live на ~92%.

## 7. ✅ R1 LIVE-АРБИТРАЖ #2 (08.10, capture 3200 tcp-фреймов, сессия юзера с действиями)

- 4426 игровых пакетов (S2C 3856 + C2S 570), **0 invalid** (крипта подтверждена повторно). Файл `tmp/cap7777/cap7777-0810b.pcapng`.
- **+20 новых live-подтверждений**: S2C: 0x0005 SM_ATTACK_STATUS×152, 0x0025 SM_EMOTION×76, 0x0003 SM_STATUPDATE_HP×57, **0x00F6 SM_FLY_TIME×56** (xml75 ошибочно SM_ALLIANCE_INFO), **0x00BB SM_PET_EMOTE×50**; C2S: **0x0107 CM_MOVE_IN_AIR×61** (ag58 ошибочно CM_ABYSS_LANDING), **0x00DB CM_PET_EMOTE×50** (ag58 ошибочно CM_VERSION_CHECK), 0x00E1 CM_TARGET_SELECT×13, **0x0108 CM_WINDSTREAM×13**, **0x02F8 CM_DIALOG_SELECT×4, 0x02FB CM_CLOSE_DIALOG×6, 0x02FA CM_SHOW_DIALOG×8** (ag58 ошибочно CM_HOUSE_DECORATE — диалог-NPC канал), 0x00E8 CM_EQUIP_ITEM×4, 0x02FE CM_PLAYER_LISTENER×2, 0x015B CM_TELEPORT_SELECT×2, **0x019F CM_AUTO_GROUP×2**, **0x02F3 CM_LEGION×1 (легион-окно!)**, 0x013F CM_GAMEGUARD×1, 0x0105 CM_BUY_ITEM×1, 0x01B3 CM_SELECT_ITEM×1.
- **Легион ✅ live (0x02F3); арен-интерфейс ✅ (0x019F + SM_AUTO_GROUP 0x007A×155); питомец-канал ✅ (0x00BB/0x00DB по 50); полёт/windstream ✅ (0x0107/0x0108/0x00F6)**.
- **Брокер НЕ триггерился** (CM_REGISTER_BROKER_ITEM нет в сессии — юзер не вносил предметы). **Дома НЕ триггерились явно** (целевой тест = capture#3). **Миньон-специфика (CM_MINION_*) не триггерилась** — юзер ткнул PET-канал; нужен целевой тест миньона (призовь/корми).
- proc_missing за сессию: НОВЫХ НЕТ (все те же 8: коллекции×5 + fame + reinvent + itemAttributeDelta) — PET-канал работает на живых таблицах; коллекции/фама остаются доб. (TD2).
- Unknown остаются: S2C 0x0182×151 (SM_STRONGHOLDS, encom75-only), C2S 0x00D4×1.

## 8. ✅ R1 LIVE-АРБИТРАЖ #3 (08.10, файл 0810d, 2148 S2C + 425 C2S, 0 invalid)

- 🔥 **МИНЬОН**: `0x01ED CM_MINIONS×21` — семейство миньонов live подтверждено (многочисленные призывы); SM-ответы миньонов в этой сессии не выделились.
- 🔥 **ДОМ**: `0x01B5×10 = CM_HOUSE_OPEN_DOOR (только encom75!)` — дверь открывалась ~10 раз + после GM-«принудительной регистрации» дом СТАЛ ТУС (dверь открылась при подходе!) → **house-логика частично жива**; `0x0111×34 SM_HOUSE_RENDER` (en/mo/ag; xml75 ошибочно SM_DELETE_HOUSE); GM-опции «регистрация дома» шли через `0x02FA CM_SHOW_DIALOG`/`0x02F8 CM_DIALOG_SELECT`.
- 🔥 **БРОКЕР**: полный сет: `0x0156 CM_REGISTER_BROKER_ITEM×1` (регистрация), `0x0140 CM_BROKER_REGISTERED_LIST×1`, `0x0154 CM_BROKER_SETTLE_ACCOUNT/SOLD_LIST×1`, `0x0157 CM_BROKER_CANCEL_REGISTERED×1` (снял с аукциона), `0x0138 CM_BROKER_SETTLE_LIST/START_REGISTER×1`.
- 🔥 **АУКЦИОН ДОМОВ**: `0x01A0×1 CM_PLACE_BID` (попытка купить дом за 1kk — «надо закончить задание» — quest-триггер) (ag58 ошибочно CM_CHALLENGE_LIST).
- **Часы/порталы Иннисона**: шли через NPC-диалог-канал (`0x003C SM_DIALOG_WINDOW×57`, 0x02F8/0x02FB) — отдельный опкод часов не выделен; повторить целевым прогоном позже.
- Прочее live: `0x00DF CM_LEVEL_READY×3` (телепорты), `0x00E7 SM_CHAT_INIT×26 + 0x0171 CM_CHAT_AUTH×26` (чат-канал), `0x02F9 CM_CHARGE_ITEM×3`, `0x0028 SM_LOOKATOBJECT×42`, `0x0019 SM_SYSTEM_MESSAGE×29`.
- ⚠️ **proc_missing выросло 8→10**: +`aion_getitemattributedelta_20190919` (сингл — при регистрации брокер-товара) и +`aion_GetItemAttributeDeltaListAllVendorLight_20190919` → TD2 пополнился.
- ⚠️ Безопасность: Server64.err = внешний сканер **34.78.135.85** стучался на 7777 («bad packet size 17733 Unknown[]» — отбит); 7777 открыт наружу → на проде закрыть firewall'ом (LAN-only).
- Тех-инфра дешифратора: tshark follow-режим надёжнее собственного reassembly (retransmissions ломают фрейминг; dup/overlap требуют вырезания) — cap7777_decrypt.py теперь поддерживает режим hex-файлов (s2c.hex/c2s.hex).

## 9. Следующий шаг

R1 продолжение (по желанию): payload-разбор 0x0182 SM_STRONGHOLDS и 0x00D4; целевой прогон коллекций/фама-окна (TD2 procs); переход к R2 (протокол-док финализация + Go-диспетчер).
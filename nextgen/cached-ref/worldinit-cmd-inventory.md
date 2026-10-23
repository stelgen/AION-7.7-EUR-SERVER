# МИР-INIT O-CMDS ИНВЕНТАРЬ (23.10) — ВСЕ 103 ИМЕНОВАНЫ, 0 НОВЫХ

> Сверка 103 уникальных O-cmds из мир-init capture (worldinit-20261023.txt)
> со словарём opcodes.go (765 опкодов) + rpc-opcodes-and-procs.md.
> РЕЗУЛЬТАТ: 103/103 именованы, 0 новых — мир-init = superset login-path RP-команд.

## Топ-нагрузка мир-init (по частоте)

| cmd | n | Имя | Комментарий |
|---|---:|---|---|
| 61 | 262 | RP_LOAD_WAREHOUSE | склад, крупнейший трафик |
| 187 | 151 | RP_CHALLENGE_TASK | мир-init специф |
| 170 | — | (C-кадр NPCSvr tl_) | NPCSvr сторона |
| 109 | 30 | RP_LOAD_ITEMS | item-блоки |
| 18 | 40 | RP_LOAD_FIELDHOUSE | хаус-данные |
| 14 | 18 | RP_LOAD_CLIENT_SETTINGS | 3 страницы XML |
| 124 | 27 | RP_DELETED_CHAR_LIST | |
| 90 | 6 | RP_CHARACTER_LIST | чарлист |
| 1 | 6 | RP_LOAD | RP cmd=1 (1149Б чар-структура) |

## Полный список (103 cmds, все именованы)

| cmd | n | Имя |
|---|---:|---|
| 1 | 6 | RP_LOAD |
| 4 | 6 | RP_GET_NEW_ITEM |
| 6 | 1 | RP_GUILD_INFO |
| 9 | 6 | RP_LOAD_PETS |
| 11 | 6 | RP_LOAD_BOOKMARK |
| 12 | 1 | RP_GUILD_MEMBER_INFO |
| 13 | 6 | RP_CAPTCHA |
| 14 | 18 | RP_LOAD_CLIENT_SETTINGS |
| 16 | 1 | RP_VERSION |
| 18 | 40 | RP_LOAD_FIELDHOUSE |
| 19 | 1 | RP_LOAD_FIELDHOUSE_OBJECT |
| 22 | 1 | RP_LOAD_FIELDHOUSE_CHARGE |
| 23 | 6 | RP_LOAD_FINISHEDQUEST |
| 24 | 6 | RP_LOAD_ABNORMAL_STATUS |
| 25 | 1 | RP_LOAD_HOUSE_SCRIPT |
| 27 | 1 | RP_LOAD_TOWN |
| 28 | 1 | RP_GET_WORLD_EXTCONDITION_LIST |
| 29 | 6 | RP_LOAD_SKILL_COOLTIME |
| 32 | 6 | RP_LOAD_WORKINGQUEST |
| 34 | 6 | RP_LOAD_CHAR_BM_PACK |
| 36 | 1 | RP_LOAD_ABYSS_OP_POINT_ALL |
| 37 | 1 | RP_LOAD_ABYSS_OP_SPAWN |
| 44 | 6 | RP_LOAD_BUDDY |
| 47 | 6 | RP_LOAD_GATHER_COOLTIME |
| 51 | 2 | RP_LIST_GOODS_INFO |
| 52 | 2 | RP_LIST_SPAWN_RANK |
| 53 | 1 | RP_ABYSS_GUILD_RANK |
| 59 | 6 | RP_VENDOR_SALESLOG |
| 61 | 262 | RP_LOAD_WAREHOUSE |
| 62 | 6 | RP_MAIL_BOXSIZELOAD |
| 76 | 6 | RP_FACTION_LIST |
| 77 | 6 | RP_LOAD_USERITEM_SIGNATURE |
| 78 | 1 | RP_LOAD_PVP_ENV |
| 90 | 6 | RP_CHARACTER_LIST |
| 91 | 6 | RP_LOAD_BLOCK_INFO |
| 93 | 2 | RP_ALL_ABYSS_INFOS |
| 94 | 6 | RP_LOAD_LEARNED_RECIPE |
| 95 | 1 | RP_GUILD_HISTORY |
| 97 | 6 | RP_GET_TITLE |
| 99 | 6 | RP_LOAD_COMBINE_COOLTIME |
| 100 | 1 | RP_GUILD_ITEM_LIST |
| 101 | 6 | RP_LOAD_BLOCK |
| 104 | 6 | RP_LOAD_INSTANCE_EXTRACOUNT |
| 108 | 6 | RP_LOAD_SKILL |
| 109 | 30 | RP_LOAD_ITEMS |
| 110 | 6 | RP_LOAD_MACRO |
| 113 | 2 | RP_ABYSS_RANKER |
| 122 | 12 | RP_GET_INACTIVE_ASSET |
| 123 | 1 | RP_SERVER_INFO |
| 124 | 27 | RP_DELETED_CHAR_LIST |
| 127 | 6 | RP_LOAD_ITEM_COOLTIME |
| 128 | 6 | RP_LOAD_EMOTION_LIST |
| 129 | 2 | RP_GET_ABYSS_USER_OWNER_LIST |
| 132 | 6 | RP_LOAD_PROMOTION_COOLTIME |
| 133 | 1 | RP_LOAD_FIELDHOUSE_OBJECT_EXTDATA |
| 139 | 6 | RP_LOAD_INSTANCE_DATA |
| 142 | 1 | RP_ALL_INSTANCE_DATA |
| 143 | 1 | RP_ALL_INSTANCE_CONDITION |
| 144 | 1 | RP_LOAD_DROP_CTRL_LIST |
| 147 | 3 | RP_FORBIDDEN_LIST |
| 150 | 6 | RP_CUSTOM_ANIMATION |
| 155 | 6 | RP_PEEK_INSTANTHOUSE |
| 157 | 2 | RP_HOUSING_AUCTION_LIST |
| 162 | 2 | RP_HOUSING_AUCTION_FILTER_LIST |
| 164 | 1 | RP_HOUSING_AUCTION_GRACE_LIST |
| 166 | 1 | RP_HOUSING_AUCTION_BETTING_LIST |
| 169 | 12 | RP_WEB_NOTIFY_CHECK |
| 176 | 6 | RP_LOAD_TRIAL_ACCOUNT_DATA |
| 178 | 6 | RP_INSTANCE_ACHIEVEMENT |
| 181 | 6 | RP_POLL_ANSWER_OF_USER |
| 184 | 6 | RP_QUERY_OFFLINE_BUDDY |
| 187 | 151 | RP_CHALLENGE_TASK |
| 189 | 6 | RP_LOAD_USER_RATE |
| 190 | 6 | RP_USER_LOGIN |
| 191 | 6 | RP_USER_LOGOUT |
| 196 | 6 | RP_HIGHEST_LEVEL_CHARACTER_INFO_ACK |
| 204 | 1 | RP_LOAD_LIMITED_SALES |
| 213 | 1 | RP_LEGION_DOMINION_OWNERS |
| 215 | 1 | RP_NOTIFY_LEGION_DOMINION_INFO_LOADED |
| 217 | 1 | RP_GUILD_JOIN_APPLICANT_LIST |
| 219 | 6 | RP_LOAD_USE_CP |
| 220 | 6 | RP_DISASSEMBLY_RETRY_LIST |
| 221 | 2 | RP_TOURNAMENT_CMD |
| 222 | 1 | RP_GET_AION_TV_CONTENTS_LIST |
| 225 | 6 | RP_SAVE_WARDROBE |
| 227 | 6 | RP_LOAD_LUNA_ABYSS_BOOST |
| 228 | 6 | RP_SAVE_LUNA_ABYSS_BOOST |
| 230 | 6 | RP_SKILL_SKIN |
| 232 | 6 | RP_LOAD_REFORM |
| 233 | 6 | RP_LOAD_VIP_ICON |
| 234 | 6 | RP_LOAD_EQUIPMENT_CHANGE_LIST |
| 235 | 6 | RP_QUNA_VENDOR_BUY |
| 238 | 1 | RQ_DELETE_PROMOTION_COOLTIME |
| 239 | 6 | RQ_SET_INSTANCE_DUNGEON |
| 240 | 6 | RQ_SET_CHARACTER_PAY_STAT |
| 241 | 6 | RQ_SHOP_GIFT_CHECK |
| 242 | 6 | RQ_EDIT_CHARACTER |
| 243 | 1 | RQ_CHANGE_CHARACTER_NAME |
| 244 | 5 | RQ_USER_SAVE_INSTANCE |
| 245 | 5 | RQ_CHANGE_COSMETIC |
| 247 | 2 | RQ_SAVE_INSTANCE |
| 248 | 6 | RQ_REQUEST_ALL_INSTANCE |
| 264 | 6 | RQ_UPDATE_TRIAL_ACCOUNT_DATA |

## ВЫВОД

**Мир-init = superset login-path RP-команд.** Словарь 765 опкодов покрывает 100%. Нет новых опкодов для reverse engineering. Эмиттеры строятся из:
1. **Chain replay** — существующие 118 кадров + расширенный chain (мир-init кадры из worldinit-20261023.txt)
2. **DB-генератор** — O1 из user_data, O109 из user_item (уже работают)
3. **SubstWorldFlow** — session-подстановки (уже работают)
4. **Новая работа** = только сборка цепочки для мир-init (порядок кадров + количество) — это chain-building задача, не RE

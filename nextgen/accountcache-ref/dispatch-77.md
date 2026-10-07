# AccountCacheServer 7.7 — dispatch-таблица опкодов (снято дизasmом, 07.10.2026)

Источники: `AccountCacheServer.exe` (MD5 33ada1f8) + `AccountCacheServer.pdb` (publics 8035) + `AccountCacheServer.map`.
Метод: `AC_Socket::OnRead` (@VA 0x1400799F0) → state-машина → `ACPacketHandler::ACPacketHandler` ctor (@VA 0x140078B40)
→ пары (handler-слот, имя-строка UTF-16 в .rdata) → номера cmd = (slot - base)/8.

## Wire-фрейм (УЖЕ ИЗВЕСТЕН, capture не нужен для каркаса)

```
[u16 len_total_minus2  LE]  ← state-машина OnRead, self-inclusive минус заголовок
[u16 cmd LE]                ← GetCmd_ACQ (VA 0x14007E560): cmd = word[0:2]
[u8  0xEB]                  ← маркер
[u16 ~cmd (NOT cmd) LE]     ← инверсия-подтверждение
[payload...]                ← размер = len_total_minus2 - 2 (лимит 0x2000)
```

- Dispatch: `handler = table[cmd]`, table ptr = `[sock+0x1d8]` (заполнена ctor'ом), cmd >= 0x6C → reject.
- Ответы ACS→Server64: `PutCmd_ACP` (VA 0x14007D800, buf+PacketArgInfo+len+cmd) — тот же фрейм-формат.
- Маркер 0xEB + ~cmd — двойная проверка целостности (как 0xBA/0xBB у LogServer).

## Таблица 1 — основной ACQ-диспетчер (Server64/мир → ACS), cmds 0..39

| cmd | Имя пакета | handler VA | Логика (вызовы из хендлера) |
|---|---|---|---|
| 0x00 | ACQ_UNUSED_0 | 0x14006c050 (default) | дефолт-хендлер (дроп/лог) |
| 0x01 | ACQ_VERSION_PACKET | 0x14006c230 | SendIOBuffer + CheckGlobalUserInServer |
| 0x02 | ACQ_TEST_INSERT | 0x14006c660 | DBConn ExecuteInsert |
| 0x03 | ACQ_TEST_SELECT | 0x14006c860 | DBConn FetchDbg ×2 |
| **0x04** | **ACQ_FIRST_LOAD_ACCOUNT_INFO** | **0x14006cbd0** | **LoadHiddenFatigueInfoByAccountId@AccountDb → SendIOBuffer** (= proc aion_GetAccountData_20170428: hidden_fatigue_point/updatetime/npckill/limit_play_*) |
| 0x05 | ACQ_LOAD_BM_PACK_LIST | 0x14006d4a0 | (DBConn) |
| 0x06 | ACQ_UPDATE_BM_PACK | 0x14006d680 | (DBConn) |
| 0x07 | ACQ_LOAD_TRIAL_ACCOUNT_DATA | 0x14006d890 | SendIOBuffer |
| 0x08 | ACQ_UPDATE_TRIAL_ACCOUNT_DATA | 0x14006db30 | SetTrialAccountData@AccountDb |
| 0x09 | ACQ_SYNC_PACKET_TEST | 0x14006ced0 | SendIOBuffer (эхо-тест) |
| 0x0a | ACQ_SAVE_CAHR_CUSTOM | 0x14006e010 | ExecuteInsert + SendIOBuffer |
| 0x0b | ACQ_LOAD_CAHR_CUSTOM | 0x14006eb40 | FetchDbg + SendIOBuffer |
| 0x0c | ACQ_LOAD_CHAR_CUSTOM_BY_ITEM | 0x14006f9a0 | FetchDbg + SendIOBuffer |
| 0x0d | ACQ_SAVE_CHAR_CUSTOM_TO_ITEM | 0x1400707b0 | (DBConn) |
| 0x0e | ACQ_LOG_INFO | 0x140071300 | (лог-запись) |
| 0x0f | ACQ_CHAR_CREATED | 0x140071510 | DBConn |
| **0x10** | **ACQ_CHAR_LOGIN** | **0x1400718f0** | **DBConn FetchDbg + SendIOBuffer ×2 + ResetHtmt** |
| 0x11 | ACQ_CHAR_LOGOUT | 0x1400724f0 | (DBConn) |
| 0x12 | ACQ_CHAR_DELETE | 0x140072850 | (DBConn) |
| 0x13 | ACQ_CHAR_DELETE_COMPLETED | 0x140072b60 | (DBConn) |
| 0x14 | ACQ_CHAR_INFO_REFRESH | 0x140072e70 | DBConn |
| 0x15 | ACQ_CHAR_LEVEL_CHANGED | 0x140073250 | (DBConn) |
| — | (cmd 0x16 = НЕ ЗАНЯТ, default) | | |
| 0x17 | ACQ_UPDATE_LOGIN_EVENT_RECORD | 0x140073ec0 | (DBConn) |
| 0x18 | ACQ_DELETE_LOGIN_EVENT_RECORD | 0x1400743f0 | (DBConn) |
| 0x19 | ACQ_UPDATE_HIDDEN_FATIGUE | 0x140074590 | (DBConn) |
| 0x1a | ACQ_LOAD_LUNA | 0x140074780 | FetchDbg ×2 + ResetHtmt |
| 0x1b | ACQ_UPDATE_LUNA | 0x140074b80 | (DBConn) |
| 0x1c | ACQ_CONFIRM_LUNA_REWARD | 0x140075220 | (DBConn) |
| 0x1d | ACQ_DECREASE_LUNA_KEY | 0x140075740 | (DBConn) |
| 0x1e | ACQ_UPDATE_BOARD_BM_STATE | 0x140076a00 | (DBConn) |
| 0x1f | ACQ_ASK_CAN_MAKE_JUMPING_CHARACTER | 0x140076d10 | (DBConn) |
| 0x20 | ACQ_LOAD_PREVIOUS_PLAYTIMES_FOR_POLLS | 0x140077360 | (DBConn) |
| 0x21 | ACQ_LOAD_PREVIOUS_PLAYTIME_FOR_ONE_POLL | 0x140077950 | (DBConn) |
| 0x22 | ACQ_UPDATE_PREVIOUS_PLAYTIME_FOR_ONE_POLL | 0x140077c50 | (DBConn) |
| 0x23 | ACQ_UPDATE_PREVIOUS_PLAYTIMES_FOR_POLLS | 0x140077f60 | (DBConn) |
| 0x24 | ACQ_RESET_LUNA_REWARD | 0x140075c70 | (DBConn) |
| 0x25 | ACQ_TRANSFORM_OPERATION | 0x1400766a0 | (DBConn) |
| 0x26 | ACQ_MONSTER_CORE_UPDATE_VALUE | 0x140078570 | (DBConn) |
| 0x27 | ACQ_MONSTER_CORE_UPGRADE | 0x140078890 | (DBConn) |
| 0x28..0x64 | НЕ ЗАНЯТЫ (default-хендлер) | 0x14006c050 | резерв (mап-заполнение дефолтом до слота ~69) |

## Таблица 2 — второй канал (слоты 0x14449ae38+, cmds 0..7)

Отдельная таблица хендлеров (второе соединение/канал AC_Socket — семантику уточнить capture'ом; подозрение: ответы/редкие операции):

| cmd | Имя | handler VA |
|---|---|---|
| 0x00 | ACQ_GEN_TEST_PACKET1 (дефолт) | 0x14006c050 |
| 0x01 | ACQ_GEN_TEST_PACKET2 | 0x140073870 |
| 0x02 | ACQ_GEN_TEST_PACKET3 | 0x140073a10 |
| 0x03 | ACQ_MoveCharResultPacket | 0x140073a20 |
| 0x04 | ACQ_MoveCharByServicePacket | 0x14006dd70 |
| 0x05 | ACQ_SetPromotionCoolTimePacket | 0x140073560 |
| 0x06 | ACQ_DeletePromotionCoolTimePacket | 0x140073a30 |
| 0x07 | (имя пустое) | 0x140073ce0 |

## Payload-параметрика (из PDB-сигнатур, Decode*@ServerToAccountCached)

Декодеры вызываются из слоя ниже хендлеров (непрямые вызовы) — точные байты берём capture'ом (R1),
но состав полей уже известен из mangled-сигнатур publics:

- `DecodeVersion(buf, len, out?)` — вероятно builder-номер, как у logd
- `DecodeFirstLoadAccountInfo(buf, len, id?, ...)` → ответ `EncodeFirstLoadAccountInfo_AddArg(PacketArgInfo, H, I, I, I, I)` — 5 полей (5 колонок proc GetAccountData!)
- `DecodeCharLogin(buf, len, H(charId), I, SpecialSvrTypeEnum, I, I, wchar*(userId), 1,1,1)` → ответ через AccGlobalCharInfo
- `DecodeCharDelete(..., H, I, I, I)` / `DecodeRefreshUserInfo(..., AccGlobalCharInfo, SpecialSvrTypeEnum)`
- `DecodeUpdateHiddenFatigue(..., I, H, I, I, I)` — 3 int-приращения
- `DecodeLoadLuna(I)` → `LunaParam` / `DecodeDecreaseLunaKey(..., __int64, LunaParam)`
- `DecodeUpdatePreviousPlayTimesForPolls(..., I, I, float)` — множитель-полл
- `DecodeTransformOperation(ProtocolType, TransformDBSubType)` / `_Load(..., I, float)` / `_Add_Arg(TransformDBData)`
- `DecodeLogInfo(..., AC_LOGTYPE, char*, short)` — лог-запись
- `DecodeMoveCharByServicePacket(..., I, I, H)` / ответ `EncodeMoveCharResultPacket(char*, I, I, H)`

## Полная сырая таблица

`dispatch-77.txt` (raw dump парсерным скриптом, 94 строки). Парсер: временный /tmp/acs_dispatch*.py
(перенести в tools/analysis при следующем заезде). Полные publics: локально
`aion_rev_2026-10-05/artifacts/pdb-big/AccountCacheServer/pdb-publics-8035.txt`.

## Вывод для MVP (aion-accache)

Каркас можно писать БЕЗ capture: фрейм+диспетчер+имена известны. Capture (R1) нужен только для
точных раскладок payload'ов каждого cmd (байтовые офсеты полей) и подтверждения формата ответов
(PutCmd_ACP + EncryptSecondary-подобной обёртки, если есть). Перекрёстная валидация с proc-телами
уже сходится (cmd 0x04 ↔ aion_GetAccountData поля, cmd 0x0a/0x0b ↔ cosmetic_data и т.п.).

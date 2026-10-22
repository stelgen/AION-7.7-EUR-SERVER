# capture-20261022 — accmirror live capture (полный прогон юзера)

**Дата:** 22.10.2026 06:16–07:12 · ** accmirror v1.1** (len=total канон) · **254 307Б**
**Формат:** C> = Server64→ACS (0xEB), O> = ACS→Server64 (0xEC) — hex кадры + RAW чанки
**Юзер:** логин stelgen, чар SteLGeN 80 (sid ea030002/eb030002 — 2 чара), мир 8/8

## Инвентарь кадров

| cmd | Имя | C> (запросы) | O> (ответы) | Статус R4 |
|---|---|---|---|---|
| 0x01 | VERSION_PACKET | 8 | 8 | ✅ байт-в-байт |
| 0x04 | FIRST_LOAD_ACCOUNT_INFO | 5 | 5 | ✅ байт-в-байт |
| 0x05 | LOAD_BM_PACK_LIST | 8 | 8 | ✅ байт-в-байт |
| 0x06 | UPDATE_BM_PACK | 3 | 3 | ✅ bм-pack push |
| 0x07 | LOAD_TRIAL_ACCOUNT_DATA | 3 | 3 | ✅ (S2C 0x06 ответ на 0x07) |
| 0x08 | UPDATE_TRIAL_ACCOUNT_DATA | 3 | — | ✅ тихий |
| 0x0D | ACP_13 push | — | 4 | ⚠ Т7: ts=0x640831AE (не в БД) |
| 0x0F | CHAR_CREATED push (ACP_15) | — | 3 | ⚠ Т8: [serverId][ts?][3] |
| 0x10 | CHAR_LOGIN | 7 | 7 | ✅ байт-в-байт |
| 0x11 | CHAR_LOGOUT | 3 | — | ✅ тихий |
| 0x14 | CHAR_INFO_REFRESH push (ACP_20) | — | 3 | ⚠ Т8: level=80 |
| 0x15 | CHAR_LEVEL_CHANGED | 5 | 5 | ✅ |
| 0x19 | UPDATE_HIDDEN_FATIGUE | 11 | — | ✅ Р2.5 раскладка подтверждена |
| 0x1A | LOAD_LUNA | 6 | 6 | ✅ ts=limit_play_reset_time |
| 0x1B | UPDATE_LUNA push | — | 3 | ✅ |
| 0x1C | CONFIRM_LUNA_REWARD push | — | 3 | ⚠ Т8 |
| 0x1F | ASK_CAN_MAKE_JUMPING | 7 | — | ✅ тихий |
| 0x25 | TRANSFORM_OPERATION | 4 | — | ✅ тихий |
| 0x09 | CHAR_LOGIN sid | — | 2 | ✅ |
| 0x02/0x03 | TEST | — | — | — |

**Статистика R4 replay:** 12/16 golden байт-в-байт (75%). Остаток 4 XFAIL (ACP_13/ACP_15/ACP_20/CONFIRM_LUNA_REWARD) — раскладки теперь выводимы из этого capture (2 чара × 3 логина = сравнение полей).

## Точки capture-обслуживания

- accmirror pid 4280, task AionAccMirror, слушает 0.0.0.0:2220 → back 127.0.0.1:2221
- ACS-2221 pid 5760, задача AionAcc2221User (**требует Console-сессию юзера**)
- Мир: CacheD64 + NPCSvr64 + Server64, 8/8 conns на 2002
- accmirror пишет БЕСКОНЕЧНО (каждый логин/чар-флоу добавляет кадры)

## Гочи инфра

- ACS-копия умирает в session-0 (0xC0000005) — живёт ТОЛЬКО в Console-сессии юзера (noVNC залогинен)
- acc.bat v3 запускает ориг из 2220-каталога — capture-схема использует копию -2221
- accmirror рестарт = taskkill accmirror.exe → Start-ScheduledTask AionAccMirror (Server64 переподключится сам)
- Double-consume pending[consumed:] в relay-ветке — не повторять
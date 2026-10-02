# RaGEZONE 1211744 — скачанные фиксы (аттачменты треда AION 7.7 PTS EU)

> Скачано 02.10.2026 под аккаунтом lnsimonovich (без reply-требований).
> Тред: https://forum.ragezone.com/threads/aion7-7pts-europe-set-up-tutorial-and-server-download-address.1211744/
> Реестр деплоя: [docs/fixes-registry.md](../../docs/fixes-registry.md)

## Инвентарь

| Файл | Пост | Автор | Дата поста | Назначение | SHA256 (первые 16) |
|---|---|---|---|---|---|
| `105-FileRecv.rar` | #105 | a7741288 | 10.11.2023 | SQL-процедуры `aion_SetCharInfo_20160818` + `aion_GetCharInfo_20160818` (распакованы рядом) | c899567b211b151b |
| `aion_SetCharInfo_20160818.sql` | ← | | | UPDATE user_data по 60+ полям (сейв персонажа) | текст |
| `aion_GetCharInfo_20160818.sql` | ← | | | SELECT полного состояния персонажа | текст |
| `106-LogServer_RU_EU_Region.rar` | #106 | TheReverend | 25.11.2023 | LogServer64.exe от 4.6 retail, EU-регион (667 КБ) | b1a8ed400d81d068 |
| `111-Mainserver64.rar` | #111 | TheReverend | 26.11.2023 | Server64.exe «Fix Manastones, Stats, Arenas, Server time» + `aion-main-10.reg`; ⚠️ сломан матчмейкер; только референс для Ghidra | 527357bffc77f33a |
| `117-LogServer64.rar` | #117 | TheReverend | 28.11.2023 | LogServer64.exe с codepage 1251 (19.9 МБ) | f96344bc93da5495 |
| `180-Server64.7z` | #180 | Angry Catster | окт 2024 | Server64.exe с date-check bypass (45 499 392 байта) | a8a3cd0e8814c67b |
| `180-aion-7.7-fix-manastones-saving.sql` | #180 | Angry Catster | окт 2024 | `aion_SetItemMatterOption` — сохранение манастоунов (USE AionWorld_110; у нас `_AionWorldNew114_rc`) | a3644c4bdce4afdf |
| `P7-clear.zip` | стр.7 | (кит) | 11.03.2023 | 清档.sql — SQL-вайп всех БД. ⚠️ НЕ ЗАПУСКАТЬ, только схема таблиц | b4ab4073d1c0f214 |
| `P8-bugs.txt` | стр.8 | (кит) | 2023 | Список недостающих процедур/индексов чужого сервера — чек-лист нашего аудита | 2ca54b5dc2b92e0e |

## Факты верификации (03.10.2026, VM 109)

- **Наш `D:\AION_LIVE_SERVER\MainServer\Server64.exe` = SHA256 `b000c6f52d140963ad0beb459dd28609…` = байт-в-байт аттач #180** (date-check bypass уже в работе). `Server64.exe.orig` = `a400cea6…` (чистый оригинал).
- Процедуры `aion_SetCharInfo_20160818` / `aion_GetCharInfo_20160818` / `aion_SetItemMatterOption` **уже существуют** в `_AionWorldNew114_rc` — файлы храним как эталон для восстановления.
- `aionaccdeldb.dbo.del_account` пересоздана нами (см. scripts/sql/fix-db-2026-10-02.sql).
- `IX_user_item_sealed_char_id` + заглушка `Log_TblGameWorldInfo_UpdateMainStatus` — scripts/sql/fix-community-2026-10-02.sql.

## Безопасность

Бинарники из треда проверены только на размер/хеш (никакой VT-проверки). Правило треда (#106–108): НЕ использовать чужие ENIGMA-обфусцированные сборки — риск бэкдоров. Наш работающий Server64 = чистый #180-патч + RunAsDate-страховка.
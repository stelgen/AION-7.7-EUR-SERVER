# 🔧 РЕЕСТР ФИКСОВ AION 7.7 PTS EU — один список по всем известным фиксам и багам

> Статусы: ✅ задеплоено и работает · 🟡 скачано/готово, ждёт деплоя · ⚪ не требуется (проверено на нашей конфигурации) · 🧾 справка · ❌ известный баг, просто не чинится
> Обновлено: 03.10.2026 00:10 VM. Источники: наши логи VM 109 + RaGEZONE 1211744 (стр. 1–11).

## A. Наши фиксы (из логов — задеплоены)

| # | Что | Файл/место | Статус |
|---|---|---|---|
| A1 | `max text repl size=65664` — CacheD64 без него самоубивается | SQL конфиг | ✅ 02.10 |
| A2 | SQL Native Client 11.0 + FILEDSN полные пути в `C:\DSN\` | setup-odbc.ps1 | ✅ |
| A3 | Индекс `IX_delete_complete_date` (aion_GetDeletedCharList) | scripts/sql/fix-db-2026-10-02.sql | ✅ 02.10, проверено |
| A4 | Linked server `RC-AIONAUTHDB` → localhost + БД-заглушка `aionaccdeldb.del_account(seq,account_id,stat)` | там же | ✅ 02.10, проверено |
| A5 | Заглушка `Log_TblGameServerInfo_UpdateServerstatus` (Aion_log) | там же | ✅ 02.10, проверено |
| A6 | **Баг int16**: `captchaServerPort 43330` → Server64 видел `-22206` | common.xml + config.ini → 22206, бэкапы `*.bak-captcha43330` | ✅, работает после рестарта Server64 |
| A7 | ACP=28591 (LogServer64 умирал на 1252) | реестр Nls\CodePage | ✅ 03.10 после ребута — LogServer жив |
| A8 | `D:\_AION_log\batchlog` каталог | создан | ✅ |
| A9 | ICServer не стартовал (луперы Interchange 2005/2305) | scripts/restart-all-services.ps1 | ✅ |
| A10 | Одноразовые schtasks-дубли (23:54–23:58) отключены | schtasks DISABLE | ✅ |
| A11 | **RAM-бюджет**: SQL max mem 12288→4096→**2048**; иначе Server64(8.6ГБ)+NPCSvr(15ГБ)+SQL = commit-исчерпание, процессы умирают молча | sp_configure | ✅ 03.10 |
| A12 | Заглушка `Log_TblGameWorldInfo_UpdateMainStatus` (Aion_log, 0 параметров) + индекс `IX_user_item_sealed_char_id` | scripts/sql/fix3 → scripts/sql/fix-community-2026-10-02.sql | ✅ 03.10 |

## B. Community-фиксы RaGEZONE 1211744 (скачано — tools/ragezone-1211744/)

| Пост | Файл | Что это | Статус |
|---|---|---|---|
| #105 (a7741288) | `105-FileRecv.rar` → `aion_SetCharInfo_20160818.sql` + `aion_GetCharInfo_20160818.sql` | SQL сейва/загрузки персонажа (эти процедуры в АionWorld_110 отсутствовали) | ⚪ **в нашей БД уже есть** (audit 03.10: вся семья 20140424…20160818 + Ext-версии). Храним как эталон-восстановление |
| #106 (TheReverend) | `106-LogServer_RU_EU_Region.rar` | LogServer64.exe от 4.6 retail с EU-регионом (667КБ) | 🟡 альтернатива нашему ACP-фиксу (A7 уже работает) |
| #111 (TheReverend) | `111-Mainserver64.rar` | Server64.exe «Fix Manastones, Stats, Arenas, Server time» + `aion-main-10.reg`, но сломан матчмейкер | 🟡 только как референс для Ghidra-переноса манастоун-фикса. НЕ деплоить (ломает матчмейкер) |
| #117 (TheReverend) | `117-LogServer64.rar` | LogServer64.exe с codepage 1251 (19.9МБ) | 🟡 альтернатива ACP (наш A7 работает) |
| #180 (Angry Catster) | `180-Server64.7z` | **Server64.exe с date-check bypass** (45 499 392 байт) | ✅ **УЖЕ ЗАДЕПЛОЕН: SHA256 нашего Server64.exe = b000c6f52d140963… = байт-в-байт этот файл**. `.orig` (a400cea6…) = чистый оригинал. RunAsDate оставлен как страховка |
| #180 | `180-aion-7.7-fix-manastones-saving.sql` → `aion_SetItemMatterOption` | сохранение манастоунов/заточек статов | ⚪ процедура **уже была** в нашей БД (audit 03.10) |
| стр.8 | `P8-bugs.txt` | список недостающих процедур другого сервера (ItemCollection×5, LoadFameInfo, LoadReinventInfo, Log_TblGameWorldInfo_UpdateMainStatus, SetLastTransformId/@last_collection_id, IX_user_item_sealed_char_id, aionaccdeldb.del_account) | 🟡 частично задеплоено: индекс ✅, UpdateMainStatus ✅, del_account ✅ (A4); остальные ждут сигнатур из наших логов (см. Г) |
| стр.7 | `P7-clear.zip` (清档.sql) | SQL-вайп всех БД (TRUNCATE-список) | 🧾 только справка/схема таблиц. НЕ запускать |
| #108 (AKllX) | (без аттача, метод) | Matchmaking на одном MainServer: `IsEventServer=true` + JZ→JNZ (Ghidra) | 🟡 не деплоено — если нужны арены без отдельного сервера |
| #189–194 | version.dll для PortalAuth | фикс connection error PA | ⚪ не нужен: авторегистрация/логин работают (проверено 02.10) |
| #119 (FliesQQ) | текст | PAServer config.txt (таблица аккаунтов), данжи без copy-server, снятие лимита онлайна, crack auth-time | 🟡 не проверено; PAServer у нас не запускается и не мешает |
| #106–108 предупреждение | — | чужие MainServer-сборки с ENIGMA-обфускацией = риск бэкдоров | 🧾 правило: только чистый бинарь + свои Ghidra-патчи |

## C. Известные баги сборки (просто не чинятся — нужен реверс/контент)

| Баг | Детали | Путь починки |
|---|---|---|
| ❌ Дома после lvl 9 | community-подтверждено | Ghidra-патч Server64 (PDB в ките есть) |
| ❌ Коллекции после 78 без рецептов | + отсутствуют `aion_GetItemCollection*` (5 шт) | заглушки по сигнатурам из логов → потом реализация |
| ❌ Дроп Heiron LF3 10–20 lvl нулевой | community-подтверждено | Ghidra/БД |
| ❌ «Песочные часы»: 2 новые карты недоступны | нужен integration server; даже автор кита не поднял | исследования |
| ❌ Питомцы / свитки призыва спрайтов | в updated-ките a7741288 чинились | сверить версию кита/бинарник |
| ❌ Манастоун-стекинг статов | «другой» MainServer фиксил (см. B/#111) | Ghidra-перенос на чистый бинарь |
| ❌ Утечка NPCSvr64 | Abyss.cpp(1211), ~600k блоков/сессию, RSS до ~15ГБ | ночной рестарт (roadmap Э3.1) |
| ❌ Нет сервисов: shop agent 10100, petition 2107, ChannelChat 10254 | exe в ките отсутствуют | игнор (некритично) или мини-эмуляторы |
| ❌ NPRelay/Ranking не запускались | нет config.xml в ките | генерить вручную (рецепт #113/#116) |

## D. Дальше «лопатами» по логам (текущая очередь)

1. **Собрать сигнатуры** отсутствующих процедур при первом срабатывании: парсим `{call aion_...}` из `CacheServer\log\*.err` → заглушки с точным числом параметров → потом реальные тела. (Список кандидатов: aion_GetItemCollectionList/LevelList/ExpiredList/CompleteTimeLimitList/CompleteList, aion_LoadFameInfo(1 int) — формат `(3003)` из bugs.txt, aion_LoadReinventInfo.)
2. **Проверить `aion_SetLastTransformId`**: в bugs.txt конфликт `@last_collection_id`; наша БД — своя сигнатура, сверить с вызовом бинарника при появлении ошибки.
3. **RAM 32 ГБ** (поднять в Proxmox) + статический pagefile 16 ГБ — иначе жить на грани (см. A11). Ночной бэкап/снапшот гипервизора ~23:50 давал disk Event 153 (IO-ресеты) — унести его от игровых часов или ограничить полосу.
4. **Первый клиентский тест**: регистрация → логин → мир → спавны → логаут/сейв → смотреть `*.err` на свежие ошибки = топливо для п.1.
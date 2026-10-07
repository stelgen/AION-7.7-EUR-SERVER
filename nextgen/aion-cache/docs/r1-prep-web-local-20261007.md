# R1-prep: веб + локальный ресёрч волна (10.10, чат WORKFLOW: cache)

Задача юзера: «кто продвинулся до нас, максимально скачать, разложить, анализ». Прод не тронут.

## 1. ВЕБ: кто продвинулся по CacheD (верdict)

**Aion CacheD: публичного переднего края по-прежнему НОЛЬ** (подтверждено 3-й раз):
- GitHub repo-search `L2CacheD` = 0, `l2cached` = 0, `cached+l2off` = 0, `aion cached` = шум (несвязные).
- Все Aion-эмуляторы (Mobius 7.7, AG 7.8, AL 7.8, beyond-aion 4.8, Aion-Core 4.7.5) пишут в БД DAO-слоем напрямую — CacheD-компонента у них нет.

**L2-наследие (кто реально лез в CacheD-бинарь — всё на L2 PTS, никем не эмулирован):**
| Кто | Что сделал | Источник | Польза нам |
|---|---|---|---|
| **l2shrine/MyExt64** | экстендер L2OFF: патчи L2 CacheD64.exe хуками (`Cached.cpp`: DisableSendMail NOP @0x4623C9/0x463066, HookStart @0x44C28D, подмена путей IMPORT_FILES→script) | github boomzabboy/L2OFF (`MyExt64/`, склонирован в `reference/l2off-boomzabboy/`, 7.8МБ, есть MyExt64.pdb) | техника патчей бинаря + рабочие адреса L2 CacheD (дифф-опора для R2) |
| **kick (mmo-dev 1164)** | «Метод добавления новых расширенных пакетов в CacheD»: hook Bind for Ex Packets @0x460547 (VirtualProtect + jmp на ExBindHook) | mmo-dev resource 1164 (HTML-копия в `cached-ref/ragezone/mmodev-1164-cached-ex-packets.html`, полн текст под логином — TODO из CREDS mmo-dev) | механизм добавления пакетов = подтверждение, что dispatch расширяем без сурцов |
| **MasterToma C1** | 95% реверс L2 CacheD C1 (у нас в гите `cached-ref/l2-c1-cached/`) | уже в гите | архитектурный шаблон 1-в-1 (каналы/wire) |

Никто в мире **не эмулировал CacheD** — мы первые (как и с ACS/authd). Источники: [1] boomzabboy/L2OFF, [2] mmo-dev 1164.

## 2. ЛОКАЛЬНО: главный прорыв чата — `.profile` логи = ГОТОВАЯ ОПКОД-НУМЕРАЦИЯ ВСЕХ КАНАЛОВ

`DBProfiler` CacheD64 каждый час печатает в `log/*.profile` ПОЛНЫЕ таблицы `id | имя | count | traffic` для **всех 8 протоколов**. Нумерация = НЕ НАДО дизasmить dispatch-ctor (цель R2 по нумерации — ЗАКРЫТА из логов).

| Секция | Ops | Наш канал |
|---|---|---|
| **DB2Server** (RP_) | 238 | 2006: ответы CacheD→Server64 |
| **Server2DB** (RQ_) | 381 | 2006: запросы Server64→CacheD |
| Log2Server (LP_) | 6 | 2051: CacheD→logd |
| Server2Log | 13 | 2051 |
| IC2DB | 11 | 2305: CacheD↔IC |
| DB2IC | 9 | 2305 |
| **NPRelay2Server** | 55 | NPRelay-канал (CacheD = участник!) |
| **Server2NPRelay** | 52 | NPRelay |

Артефакты: `cached-ref/profile-opcode-map.md` (полные 8 таблиц) + экстрактор `tools/analysis/cache_profile_parse.py` (агрегирует и топ-использование).

### 2.1 Топ реального использования (агрегат всех *.profile, 02–07.10 — это LIVE-профиль нагрузки прод-CacheD)

**Write-путь (Server2DB)**: RQ_ALERT_MSG 1 093 051 (133МБ!) · RQ_UPDATE_ABYSS_INFO 608 469 (54МБ, abyss-цикл) · RQ_SET_SERVER_INFO 487 842 (260МБ) · RQ_CHALLENGE_TASK 136 723 · RQ_U_UPDATE_COUNT 82 223 · RQ_U_UPDATE_COUNT_RELATIVE 50 094 · RQ_CAPTCHA 46 469 · RQ_UPDATE_SLOT_ID 42 029 · RQ_U_SAVE 37 875 (32МБ) · RQ_ADD_SKILL 33 642 · RQ_UPDATE_QUEST 28 913 · RQ_SET_WORLD_EXTCONDITION 28 480.

**Read-путь логина (кластер RP_LOAD_*×10 при каждом входе, ~4 346 логинов)**: RP_LOAD_WAREHOUSE 188 066 (21.8МБ) · RP_LOAD_FIELDHOUSE 36 440 (**283МБ** — самый жирный!) · RP_LOAD_ITEMS 16 650 (119МБ) · RP_LOAD_CLIENT_SETTINGS 13 038 (77.9МБ) · RP_CHARACTER_LIST 5 525 · RP_LOAD_SKILL/FINISHEDQUEST/WORKINGQUEST/PETS/TRIAL_ACCOUNT_DATA/VIP_ICON/ABNORMAL_STATUS + RP_USER_LOGIN/GET_TITLE/RANK_INFO — ровно по одному сету на логин.

→ **MVP read-путь подтверждён числами**: RQ_CHARACTER_LIST → RP_CHARACTER_LIST + пачка RP_LOAD_* (магазин/макро/клиент-сеттинги), затем RQ_U_* item-мутации. **MVP write-путь**: RQ_U_SAVE/U_UPDATE_*/ADD_SKILL/UPDATE_QUEST (+abyss-фон RQ_UPDATE_ABYSS_INFO/RQ_SET_SERVER_INFO — это фоновая нагрузка, не юзер-путь).

## 3. ЛОКАЛЬНО: `.err` = SQL-транзит с ПОЛНЫМИ параметрами (бесплатный материал R3)

`log/*.err` (155МБ/день) = DB-транзит: каждая строка `[SAVE] AbyssId:1011 OwnerRace:2 OwnerServerId:1 OwnerGuildId:0 DefTurnCount:82 ...` + полный `{call aion_SetAbyssInfoNew_20160520(1011, 1, 0, 2, 82, 0, ...)}`. → RPC-параметры многих write-команд восстановимы из логов БЕЗ capture. Abyss-цикл: пачки сохранений каждые ~3с bursts (внутри 60с-цикла).

## 4. ЛОКАЛЬНО: кросс-версионный дифф 5.8↔7.7 (append-only эволюция)

Извлечён 5.8 `Cached/CacheD64.exe` (22 303 232 Б, 2020-05-29, из `ref-kits/5.8-leak/58Server.rar`) → UTF-16 strings → дифф со словарём 7.7:

| Семейство | 5.8 | 7.7 | Δ |
|---|---|---|---|
| RQ | 351 | 382 | +31 |
| RP | 236 | 255 | +19 |
| GQ | 55 | 55 | 0 |
| GP | 52 | 53 | +1 |
| ACQ/ACP | 39/28 | 39/28 | 0 |

**only in 5.8 = ПУСТО** для всех семейств → словарь опкодов NC **append-only**: ничего не удаляется/не переименовывается, только добавляется → наш dispatch = superset, нумерация стабильна между версиями (верифицируется profile-id 7.7). Новое в 7.7: achievement/item-collection/fame/reinvent/quna_vendor/tournament/extslot/cursestate/matter_option — ровно семейства из proc_missing-алертов aion-main. Артефакт: `cached-ref/diff-dict-58-77.md`; конфиги 5.8 → `cached-ref/58-cached/` (exe 5.8 вне гита: `tmp/tmp58cached/`, перенести в artifacts по завершении чата).

## 5. L2 C1 сверка нумерации

L2 C1 handlers — те же нумерованные enum-файлы: serverHandlers 283 (packet000..packet282), adminHandlers 58, npcDbHandlers 7 (00_Dummy/01_NpcDbVersion/02_LoadNpcRequest/03_SaveNpcInfo/04_UpdateBossNpcValue). Подход NC к нумерации = стабильный enum → наша profile-нумерация = тот же механизм. (L2 283≠238+381 — Aion-каналы свои, маппинг только семантический.)

## 6. Влияние на ROADMAP

- **R2 сузился**: нумерация dispatch уже снята (§2); осталась семантика payload'ов (mangled-сигнатуры publics + .err-транзит) → R2 = сверка таблиц + раскладки, не поиск таблиц.
- **R1 остаётся**: wire-фрейминг 2006 (pktmon) — гипотеза `[u16 self-len][op][payload][2Б csum]` + rolling-XOR не закрыта ничем из этого материала (логи длину кадра не печатают).
- Теория «2009 = NPC-DB (NPCSvr64 клиент)» — ❌ (netstat live 08.10: NPCSvr ходит в 2006, 2009 без клиентов; леджер ROADMAP). Кто клиент 2009 — неизвестен (⏳, кандидат: IC/logd/админ-инструменты; в profile нет секции 2009-протокола — значит, 2009-канал не профилируется DBProfiler'ом или простаивает).
- Новые факты-каноны → README канон + «сосед узнал» в aion-relay (NPRelay 55/52), aion-logd (LP_ 6/13), aion-ic (IC2DB/DB2IC 11/9).

## 7. Скачано/положено (воркфлоу-раскладка)

| Что | Куда | Статус |
|---|---|---|
| `profile-opcode-map.md` (8 таблиц, 773 ops) | `nextgen/cached-ref/` | ✅ в гит |
| `diff-dict-58-77.md` | `nextgen/cached-ref/` | ✅ в гит |
| 5.8 конфиги+DBLog | `nextgen/cached-ref/58-cached/` | ✅ в гит |
| экстрактор профилей | `tools/analysis/cache_profile_parse.py` | ✅ в гит |
| mmo-dev 1164 HTML | `nextgen/cached-ref/ragezone/mmodev-1164-cached-ex-packets.html` | ✅ в гит |
| boomzabboy/L2OFF (MyExt64) | `reference/l2off-boomzabboy/` (вне гита) | ✅ клон 7.8МБ |
| 5.8 CacheD64.exe+apdll.dll | `/tmp/tmp58cached/` → переместить в `aion_rev/artifacts/` | ✅ вне гита |

Не скачано (нужен логин/вес): полн. текст mmo-dev 1164 (mmo-dev логин из CREDS — TODO), 5.8 PTS VM / 7.7 C++ pack с RZ (гигабайты, нецелесообразно из песочницы — остаётся юзер-задача §10.3 RESEARCH).

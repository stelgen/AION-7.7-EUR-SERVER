# accountcache-ref — референс-сурсы и артефакты ресёрча AccountCacheServer (:2220)

> Ресёрч: [aion-accache/RESEARCH.md](../aion-accache/RESEARCH.md) (~92%). Код: [../aion-accache/](../aion-accache/). План: [aion-accache/ROADMAP.md](../aion-accache/ROADMAP.md) (R1 capture = следующий).

| Файл | Что это |
|---|---|
| `dispatch-77.md` | **аннотированная dispatch-таблица 7.7** (снята дизasmом ctor VA 0x140078B40): wire-фрейм `[u16 lenMinus2 LE][u16 cmd LE][0xEB][~cmd LE][payload]`, T1 cmds 0..39 полная нумерация (FIRST_LOAD=4, CHAR_ 15..21, LUNA 26..29+36, CUSTOM 10..13, PLAYTIME_POLLS 32..35, MONSTER_CORE 38/39), T2 cmds 0..7, cmd22 = гэп |
| `dispatch-77.txt` | raw-вывод парсера (для перегенерации/сверки) |
| `db-procs-77-ref58.rpt` | **тела ВСЕХ 101 proc + 21 таблица БД AionAccountCacheD** (sp_helptext с VM REF58): account_data (hidden_fatigue), account_fatigue, account_luna(+reward), account_pack, aion_ranking_*, aion_server_data/serverlist, cosmetic_data, global_user_data, jumping_character_config, trial_account_data, user_board_bm(+dice/game), user_login_event_data(+daily/other/renewal), user_monster_core, user_promotion_cooltime, user_transform |
| `rpc-opcodes-77.txt` / `rpc-opcodes-58.txt` | словари ACQ_/ACP_*: 74 команды в 7.7 (+3 vs 5.8: MONSTER_CORE_UPDATE_VALUE/UPGRADE, TRANSFORM_OPERATION); 71 в 5.8 |
| `db-procs-58.txt` | procs-список 5.8 (версии-суффиксы = миграции) |
| `config.xml` / `perfmon.ini` / `AccountCacheServer.config` / `AccountCacheServer.common` | конфиги (7.7 с нашей VM + 5.8-кит): numberOfDBThreads=10, serverPort 2220, LogServer |
| `AIONErr.txt` | дамп deadlock-dumper оригинала (framework `Shared\IoCompletion.cpp`) |

## Ключевые факты

- Клиент :2220 на проде = **Server64** (ESTABLISHED 127.0.0.1); authd подключается лениво/по требованию.
- PDB: `AccountCacheServer.pdb` 92.4МБ (MD5 `17481fc5`) + .map 4.8МБ — на VM `D:\AION_LIVE_SERVER\AccountCacheServer\`; локально `aion_rev/artifacts/pdb-big/AccountCacheServer/` (publics 8035).
- Ответы ACP = тот же фрейм (`PutCmd_ACP`), номера ответных cmd TBD (indirect vtable) → R1 capture.
- НЕ путать с CacheD64 (:2006 — кэш мира; см. [cached-ref/README.md](../cached-ref/README.md)).

# АУДИТ LINKED SERVERS / LOOPBACK-ПАТТЕРНОВ (2026-10-05, read-only)

## Контекст
Ночь 04-05.10.2026: 4 краша CacheD64.exe (AV 0xc0000005 @+0x18c31f, «Intentional exception»
после CheckIOThreadDeadlock 181-193с). ROOT CAUSE: `aion_ProcessWithdrawAccount`
использовала LOOPBACK linked server `[RC-AIONAUTHDB]` внутри BEGIN TRAN —
self-deadlock compile-семофора + DTC-спин (command=NOP, 100% CPU 321с).
Исправлено тем же днём: `alter-processwithdrawaccount-noloopback-20261005.sql`.

## Связанные серверы (sys.servers)
| Имя | Data source | Тип |
|---|---|---|
| RC-AIONAUTHDB | localhost | linked (loopback!) |
| WIN-0883A4UBOEC | WIN-0883A4UBOEC | remote (legacy, имя старой машины) |

## Результаты скана всех пользовательских БД (sys.sql_modules, type=P)
| Паттерн | Найдено |
|---|---|
| `[RC-AIONAUTHDB]` в телах проц | **0** (была 1 — ProcessWithdrawAccount, устранена ALTER'ом) |
| `WIN-0883A4UBOEC` в телах проц | **0** (мёртвый remote server, кодом не используется) |
| OPENROWSET/OPENQUERY | 5 процедур (см. ниже) |

## OPENQUERY/OPENROWSET проц (мёртвый код шардовой архитектуры)
| БД | Процедура | Вызовы в логах 04-05.10 |
|---|---|---|
| AionAccountCacheD_rc | aion_PutUserDataFromServer | **0** |
| AionAccountCacheD_rc | aion_SyncUserDataFromServer | **0** |
| _AionWorldNew114_rc | aion_AddedService_Type4_MoveChar_CheckConn_ORS | **0** |
| _AionWorldNew114_rc | aion_AddedService_Type4_MoveChar_CheckOrgChar_ORS | **0** |
| _AionWorldNew114_rc | aion_AddedService_Type4_MoveChar_Process_ORS | **0** |

Семантика: шардовая архитектура NC (мир на отдельной машине) — `aion_PutUserDataFromServer`
читает `aion_serverlist` (datasource/database) и строит OPENROWSET('SQLOLEDB', login;pass)
к «серверной» user_data. В нашем деплое не вызывается НИ РАЗУ.

## Рекомендации
1. **Ничего не менять** — активных loopback-рисков после фикса не осталось.
2. `aion_serverlist`/OPENROWSET-процы: мёртвый код; деплой-кандидат «не трогать».
3. `WIN-0883A4UBOEC` remote server: можно снести (sp_dropserver) — нулевой риск,
   но и нулевая выгода. Опционально.
4. При появлении новых 2812/зависаний: тот же метод — CatchBlocked + sys.dm_exec_requests.

# authd-ref — референс-сурсы для aion-authd (замена L2Authd)

> Реестр S1–S7 = [../AUTHD-ROADMAP.md](../AUTHD-ROADMAP.md) §1. Ресёрч: [../AUTHD-RESEARCH.md](../AUTHD-RESEARCH.md).
> Приоритет источников при спорах: live-фреймы fork'а (S1) > сорс эталона (S5) > декомпил C1 (S2/S3) > дизasm PDB (S4).

| Каталог/файл | Что это | Что берём |
|---|---|---|
| `L2Auth-chaospaladin/` | полный компилируемый декомпил L2AuthD C1 build 40504 (13МБ) | архитектура authd: CAuthServer/CAuthSocket, WorldSrvServer (gs-wire), CAccount (ODBC-процедуры, block_msg, payStat), OneTimeLogOut/AutokickAccount, crypt-модули |
| `l2-c1-mastertoma/` | пакет MasterToma L2 PTS C1 (35МБ): L2Auth reversed/generated/src (FIXED-маркеры overflow CIOTimer/CJob, blockFlag_custom), L2LogD, CacheD, L2Core, PetitionD | 95% реверс C1 — кто дальше всех; сверка логики |
| `l2-c1-mastertoma/.../DBScript/` (`ReleaseAuthDBSchema.sql`) | схема БД authd: procs `ap_GPwd/ap_GStat/ap_GUserTime/ap_SLog/ap_SUserTime` + lin2comm 44 procs + lin2user/lin2log/lin2report | формат DB-слоя; сверять с прод-процами AionAccounts (sp_helptext, R0) |
| `lin2db-classic-x64-schema.sql` | схема x64 classic 162-287 (mmo-dev Auth.7z): userno/user_time/usn + 13 procs (ap_GPwd/ap_GStat/ap_GStatEtc/…) | вторичный DB-референс |
| `l2auth-legacy-2008/` | L2AuthD 2008 (yury-dymov) | вторичные проекции: IP-фильтры, режимы хостинга |
| `L2AuthHost-csharp/` | C#-хост L2Auth | вторичные проекции |

## Готовые факты (live)

- Wire 2110: `[00][sid][IP-be]` / `[01][sid]` / `[02][sid][len][blob 191Б asm-форма]`; назад `[03][V=0xc621]`, `[02][id][len][type][payload]` type=3/4/7 — см. `../aion-gate/docs/authd-wire-20261007.md` и `../aion-authd/docs/authd-wire-20261007.md`.
- Логика: пароль НЕ проверяется (авторегистрация), online-флаг TTL 2–6 мин ([01] не снимает), fail-реестр = AionAuthResponse ( Mobius S5).
- PA обязателен в live-пути оригинала (SYSTEM_ERROR 20 без него) — учитывать при R6-свитче.

> ⚠ ВЕРДИКТ ДОКУмента ОТМЕНЁН 07.10 (fork-доказательство): PA = ОБЯЗАТЕЛЬНЫЙ компонент стека —
> без живого PA (10057) ориг отклоняет ЛЮБОЙ логин мгновенно и молча (SYSTEM_ERROR(20))
> ДО SQL-этапа; старт-порядок: PA ДО authd (задача AionPA, op-кнопка pa). Что ВЕРНО из этого дока:
> PA не переписываем (не нужен в НАШЕМ пути авторизации — релей payStat портала); держим ориг живым.

# PA / PAServer (PortalAuth) — полный ресёрч и вердикт СКИП (06.10.2026)

**Вердикт**: `01-PAServer7.7.exe` НЕ ЗАПУСКАЕМ НАВСЕГДА (задача `AionPA` DISABLE).
Стек работает без него — проверено живьём (авторегистрация/логины/мир с 02.10 по 06.10).
Роль в оригинале: внешний релей пароль-авторизации между L2Authd и веб-порталом/биллингом NCsoft
(payStat/подписки/блокировки). У нас портала нет, authd держит аккаунты сам (L2Conn.dsn → AionAccounts).

## 1. Что за бинарь
- `01-PAServer7.7.exe` = **PAServer / PortalAuth**, порт **10057 (только loopback)**.
- Запакован, почти без строк (см. `docs/server-internals.md` §хвосты).
- `config.txt` в ките отсутствует; FliesQQ (RaGEZONE 1211744 #119): в нём задаётся connection-конфиг +
  **имя таблицы аккаунтов (по умолчанию `AionAccounts`)**; поддерживает автоматическое создание аккаунтов/паролей.
- Конфиг authd: `UsePAServer=true`, `PAConnectionCount=2`, `PAIP_1/2=127.0.0.1:10057` (наш фикс: PAIP_2 был 127.0.0.2),
  `PAReconnectInterval` (деф 60; комьюнити ставит 3).

## 2. Наши факты (почему скип)
- Не запускается и не мешает (`docs/fixes-registry.md` #119); задача `AionPA` DISABLE; NPRelay тоже.
- Авторегистрация/логин работают без PA (проверено 02.10; version.dll-фикс #189–194 не нужен).
- При потушенном PA authd терпит отсутствие мостов молча (event-driven, тишина в err).
- `CPASocket 10057` виден только в auth-логах при отвалах мира (`docs/authlog-merge-0410.md`).
- README-заявление «Обязателен / без PA EU-клиенты login only after official portal» = миф
  оригинальной портал-архитектуры, опровергнут живьём. (README поправлен этим же коммитом.)

## 3. Что знает комьюнити (RaGEZONE)
- Тред Aion 7.7 C++ server files (1205286, стр.6): «password authorization is not handled via Auth anymore in new bins,
  it is handled via an external relay … you need a PortalAuth implementation (PAServer in configs), **savior has an
  open source version released** … the Aion leak comes with pdb's» (pada8801). Git савиора **не публичный**.
- Тред AION7.7pts Europe (1211744): PA-секция конфига (`UsePAServer=true`, `PAConnectionCount=1`, `PAIP`,
  `PAPort=10057`, `PAReconnectInterval=3`); wern: «много процедур можно восстановить, т.к. есть .pdb файлы» →
  полный набор PDB в SVN-утечке (папка `Server-7.7`), у нас локально только сабсет.
- Тред L2 Classic 3.0 (1205208, стр.16–17) — та же PA-архитектура, **единственный публичный эмулятор PA**:
  «portal-auth-emulator» (Python + Docker, образ `pae`):
  ```bash
  docker build -t pae .
  docker run --detach --restart=always \
    --env="PAE_CONNECTION_STRING=DRIVER={ODBC Driver 17 for SQL Server};SERVER=<ip>;DATABASE=<db>;UID=sa;PWD=<pwd>;" \
    --env="PAE_CLIENT_APP_ID=<GUID из конфига authd/l2server>" \
    --name=pae-1 --publish=10057:10057 pae   # второй мост: --name=pae-2 --publish=10058:10057
  ```
  - Внутри: `.py` + SQL-проца **`pp_GetPortalUser`**; колонка `enc_flag` → в их БД переименована в `new_pwd_flag`
    (правили в py и в процедуре); нужна версия Docker 4.12.
  - Пример GUID (L2): `1c4b8992-def6-4ab3-9a17-041c5115e501`.
  - PA-лог «can't connect to database» при кривом conn-string (формат `Server=(local);Database=...;User Id=...;Password=...`).
  - Обход без PA в L2: `pay_stat = 0` прямо в БД (irk).
  - Сорс существует ТОЛЬКО как аттачмент RaGEZONE под логином: GitHub-поиск `portal-auth-emulator` /
    `pp_GetPortalUser` = 0 репозиториев (проверено API 06.10).
- Эмуляторы Aion (beyond-aion/aion-server 4.8, Mobius_AionEmu 7.7, AionGermany, AionLightning 5.8,
  Aion-Core 4.7.5) — **PA вообще не реализуют** (grep PortalAuth/PAServer/payStat = ноль; порт 10057 в сурсах — только ID айтемов).
- `ChaosPaladin/L2Auth` — реверс L2AuthD старых хроник (методология IDA→compilable src; к PA отношения не имеет).
- Китайские сурсы — только ENIGMA-упакованные бинари, PA никто не разбирал.

## 4. PDB
- Локально (`aion_rev/artifacts`): PAServer.pdb НЕТ (есть AuthGateD, L2Authd, MainServer, RankingServer / pdb-big: LogServer64, NPCSvr64).
- В полной утечке PDB есть (pada8801, wern). **Куда смотреть**: SVN-папка `Server-7.7` (TieLay), папка PAServer рядом с exe на VM.

## 5. Если однажды понадобится (триггеры пересмотра)
1. Появится веб-портал/биллинг с payStat-логикой → тогда: скачать аттачмент `portal-auth-emulator`
   (тред 1205208, пост renobizarro #307, нужен акк RaGEZONE), взять наши GUID'ы
   `ClientAppId/AuthdAppId` из `AuthD\etc\config.txt` (инвентаризованы в `docs/config-inventory-0410.md`),
   процы `ap_*`/`web_*` из `docs/auth-server-internals.md`.
2. Если authd на каком-то ребуте начнёт сыпать ошибками по PA или тормозить старт.

## 6. Ссылки
- RaGEZONE 1205286 p.6 — Aion 7.7 C++ server files (PA = external relay, savior, pdb's).
- RaGEZONE 1211744 p.5–6 — AION7.7pts Europe (PA-конфиг, FliesQQ #119, pdb's, ENIGMA-бин.main'ы).
- RaGEZONE 1205208 p.16–17 — L2 Classic 3.0 (portal-auth-emulator: docker/py/pp_GetPortalUser/pay_stat-обход).
- GitHub: `ChaosPaladin/L2Auth` (реверс L2AuthD), поиск `portal-auth-emulator`/`pp_GetPortalUser` = 0.
- Локальные: `docs/auth-server-internals.md` (процы/схема), `docs/server-internals.md` (PAServer в архитектуре),
  `docs/fixes-registry.md` (#119, #189–194), `docs/authlog-merge-0410.md` (CPASocket).
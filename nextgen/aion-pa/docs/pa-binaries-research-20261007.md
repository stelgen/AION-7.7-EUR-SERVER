# PA (PAServer/PortalAuth) — ресёрч бинарей и эмуляторов (07.10.2026)

**Контекст**: вердикт «скип» из `pa-research-20261006.md` ОТМЕНЁН 07.10 (fork-доказательство:
без живого PA:10057 ориг отклоняет любой логин SYSTEM_ERROR(20) ДО SQL). Задача ресёрча:
найти готовое (бинарь/эмулятор), чтобы не писать PA с нуля. Метод: полный веб-обход
(RaGEZONE, mmo-dev, GitHub, точные фразы).

## Вердикт
С нуля НЕ пишем. Есть 2 пути:
1. **pae (portal-auth-emulator)** — единственный публичный эмулятор PA, доказанно рабочий
   (Python+Docker). Аттачмент RaGEZONE 1205208 #307 (renobizarro) — **скачать под логином RZ**.
2. **Оригинальные бинари**: наш `01-PAServer7.7.exe` (+config.txt по схеме FliesQQ #119)
   и L2-ориг `pa server` из `L2_LIVE_CSERVER_SVN` — для кросс-версионного диффа.

## 1. Что нового (факты с веба)
### pae — полностью рабочий в L2-стеке (чек-поинты комьюнити)
- Установка: `docker build -t pae .` → два контейнера:
  ```bash
  docker run --detach --restart=always \
    --env="PAE_CONNECTION_STRING=DRIVER={ODBC Driver 17 for SQL Server};SERVER=<ip>;DATABASE=<db>;UID=sa;PWD=<pwd>;" \
    --env="PAE_CLIENT_APP_ID=<GUID из конфига authd/l2server>" \
    --name=pae-1 --publish=10057:10057 pae
  # второй мост: --name=pae-2 --publish=10058:10057 pae
  ```
- Docker **версии 4.12** (на новых билд падает; linux-контейнеры).
- authd-конф: `PAConnectionCount=1`, `PAIP=<LAN IP>`, `PAPort=10057`
  — **127.0.0.1 может не работать, ставили LAN-IP**.
- conn-string для локальной БД: `Server=(local);Database=...;User Id=...;Password=...`.
- Колонка `enc_flag` → переименована в `new_pwd_flag` в `.py` и `pp_GetPortalUser.sql`.
- **Итог blackpanther197 (#326/#328)**: PA без ошибок, authd без ошибок, authgated без ошибок,
  неверный пароль → дисконнект (проверка пароля реально идёт через pae).
- Оставшаяся у них проблема НЕ в PA: процы мира `lin_GetUserDataByCharId`/`lin_SaveBotReportConfirm`.
- Лог pae «can't connect to database» = кривой conn-string.

### Оригинальные бинари PA
- В утечке L2 есть **оригинальный pa server**: renobizarro поднимал полный сет
  L2_LIVE_CSERVER_SVN REV1 (Cached, l2server, l2npc, authd, authgated, **pa server**, l2comm).
  irk: «нужен отдельно созданный PaServer и оригинальный логин»; версии протоколов 162–287.
- Aion 7.7: полный набор PDB в SVN-утечке (`Server-7.7`); PAServer.pdb у нас локально НЕТ.
- savior (автор «open source PA»): git **не публичный**, версию постил в L2-тред —
  вероятно, это и есть источник pae; публичных зеркал нет.

### Негатив (где пусто)
- GitHub `q=PAServer+aion` = **0 репо**; `q=portal-auth` = шум (captive portal/wifidog).
- Точная фраза «portal-auth-emulator» вне RZ = **0 результатов** (только аттач в треде).
- mmo-dev.info по PAServer/PortalAuth/10057 = 0.
- Все известные эмуляторы Aion PA не реализуют (см. pa-research-20261006.md §3).

### Aion-конф (подтверждение источником)
- TAURUS (#90, тред 1211744): блок PortalAuth в authd-конфиге:
  `UsePAServer=true / PAConnectionCount=1 / PAIP="127.0.0.1" / PAPort=10057 / PAReconnectInterval=3`.

## 2. План внедрения в наш стек (Aion 7.7)
1. **Скачать** (нужен акк RaGEZONE):
   - аттач #307 (portal-auth-emulator) — тред 1205208;
   - `L2_LIVE_CSERVER_SVN` REV1 (#282 SWnet) — там ориг pa server для диффа;
   - (опц.) #283 patched l2server/cached, пароль `rag3z0n3`.
2. Разобрать pae: `.py` + `pp_GetPortalUser.sql` — wire-формат PA (opcodes payStat-семейства).
3. Подставить НАШИ GUID'ы `ClientAppId/AuthdAppId` из `AuthD\etc\config.txt`
   (инвентаризация: `../nextgen/aion-op/docs/config-inventory-0410.md`).
4. Адаптировать процу под нашу БД: у L2 — `lin2db`/`enc_flag`, у нас — `AionAccounts`
   (схема/процы `ap_*`/`web_*`: `../nextgen/aion-authd/docs/auth-server-internals.md`).
5. Альтернатива/эталон: поднять ориг `01-PAServer7.7.exe` (config.txt по схеме FliesQQ #119:
   connection + имя таблицы аккаунтов + автосоздание) и сравнить трафик с pae.
6. Порядок старта: **PA ДО authd** (см. заголовок pa-research-20261006.md).
7. Задача AionPA из DISABLE → ENABLE после подтверждения PA на 10057.

## 3. Ссылки
- RZ 1205208 p.15 — #282 L2_LIVE_CSERVER-r1, #283 patched bins (rag3z0n3), #290/#293 irk (PA+ориг логин, 162–287).
- RZ 1205208 p.16 — #301 irk (2 условия входа, pay_stat=0 хак), #303 renobizarro (полный сет + IDA-оффсеты серийников), #307 pae-туториал + аттач.
- RZ 1205208 p.17 — #326/#328 PA работает целиком (enc_flag→new_pwd_flag, docker 4.12, conn-string, PAIP=LAN).
- RZ 1205286 p.6 — #101/#104 pada8801 (PA=external relay, savior git не публичный, Aion leak с pdb).
- RZ 1211744 p.5 — #90 TAURUS: PA-блок конфига authd.
- GitHub API: search/repositories?q=PAServer+aion (0), q=portal-auth (шум).
- Локальные: `pa-research-20261006.md`, `../nextgen/aion-authd/docs/auth-server-internals.md`,
  `../nextgen/aion-op/docs/config-inventory-0410.md`, `../nextgen/aion-op/docs/authlog-merge-0410.md`.

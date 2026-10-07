# aion-pa — PortalAuth (PA, :10057) — ориг ОБЯЗАТЕЛЕН; наш эмулятор = адаптация pae, НЕ с нуля

> 🔬 **наш-эмулятор ~10% (деприор, адаптация pae) · ресёрч закрыт 08.10** ([docs/pa-binaries-research-20261007.md](docs/pa-binaries-research-20261007.md)): с нуля НЕ пишем — два пути: **(1) pae (portal-auth-emulator)** = единственный публичный эмулятор, доказанно рабочий в L2-стеке (Python+Docker, RZ 1205208 #307); **(2) оригинальные бинари** (`01-PAServer7.7.exe` + L2-PA из L2_LIVE_CSERVER_SVN) для кросс-версионного диффа.
> Прод: **ориг PA жив и обязателен** (задача AionPA, старт ДО authd): без живого PA ориг отклоняет ЛЮБОЙ логин SYSTEM_ERROR(20) молча (fork-доказательство 07.10; старый вердикт «SKIP» в [docs/pa-research-20261006.md](docs/pa-research-20261006.md) ОТМЕНЁН).
> Запуск чата: `WORKFLOW: pa` ([../WORKFLOW.md](../WORKFLOW.md)). Стандарты S1–S12: [../README.md](../README.md) §4.

## 📊 Статус и фазы

| Фаза | Статус |
|---|---|
| R-ресёрч (бинари/эмуляторы, полный веб-обход) | ✅ 08.10 ([docs/pa-binaries-research-20261007.md](docs/pa-binaries-research-20261007.md)) |
| R1: скачать аттачи RZ 1205208 #307 (pae) под логином → разобрать .py + pp_GetPortalUser.sql (wire payStat) | ⬜ СЛЕДУЮЩИЙ |
| R2: подставить НАШИ GUID'ы ClientAppId/AuthdAppId (config-inventory: `aion-op/docs/config-inventory-0410.md`) | ⬜ |
| R3: адаптировать процу под AionAccounts (схема `../aion-authd/docs/auth-server-internals.md`: ap_*/web_*; enc_flag→new_pwd_flag) | ⬜ |
| R4: тест против :10057 (fork-окно/соседний порт) → паритет ответов | ⬜ |
| R5: свитч по «го» (задача AionPA → pae-контейнер / наш адаптированный); откат = ориг | ⬜ |

**Прогресс ~15%** (ресёрч закрыт — фундамент; интеграция = отдельная работа, **низкий приоритет**: ориг PA стабилен и обязателен).

## 📟 Канон (проверенное)

| Факт | Источник |
|---|---|
| PA = релей payStat/подписок веб-портала NCsoft; при `UsePAServer=true` authd отказывает логин без живого PA мгновенно и молча | fork-доказательство 07.10 |
| authd-конф: `UsePAServer=true`, `PAConnectionCount=1`, `PAIP=<LAN IP>`, `PAPort=10057`, `PAReconnectInterval=3-60с`; наш `PAIP_2=127.0.0.2` — сверить (127.0.0.1 может не работать — комьюнити ставили LAN-IP!) | TAURUS #90 тред 1211744 + live |
| pae: Python+Docker (версия 4.12!), 2 контейнера 10057/10058, env `PAE_CONNECTION_STRING` (ODBC Driver 17) + `PAE_CLIENT_APP_ID` (GUID из конфига authd; L2-пример `1c4b8992-def6-4ab3-9a17-041c5115e501`) | pa-binaries-research §pae |
| Процедура `pp_GetPortalUser.sql`; колонка `enc_flag`→`new_pwd_flag` | там же |
| Проверка пароля реально идёт (неверный → дисконнект; blackpanther197 #326/#328) — в отличие от нашего authd, пароль которого не проверяется | там же |
| GitHub q=PAServer+aion = 0 репо; «savior has an open source version» (pada8801) — git НЕ публичный | веб-обход 08.10 |
| L2-ориг: полный сет `L2_LIVE_CSERVER_SVN REV1` (Cached/l2server/l2npc/authd/authgated/**PA**/L2Comm) — renobizarro #303; аттач #282 SWnet L2_LIVE_CSERVER-r1 | там же |

## 🚧 Блокеры

- Скачивание pae = RZ-логин (креды `D:\SAION\creds\CREDS.md` + cookies rz-cookies.txt).
- Docker на VM = ставить (версия 4.12; на новых билд падает; linux-контейнеры).
- RZ-скачанные exe из треда 2.7 — **малварь-стабы (VMProtect)** — вне изоляции НЕ запускать (урок ic-ресёрча).

## ⏭️ Следующий шаг

`WORKFLOW: pa` → R1: скачать аттачи pae (aionput на VM в `D:\SAION\downloads\pa\`) → распаковать → разбор .py/SQL → GUID'ы из `aion-op/docs/config-inventory-0410.md`. Промпт: [PROMPT.md](PROMPT.md).

## 📦 Артефакты

| Что | Где |
|---|---|
| Ресёрч-доки | docs/ (binaries-20261007, pa-research-20261006) |
| Ориг-бинарь | VM `D:\AION_LIVE_SERVER\` (`01-PAServer7.7.exe`, задача AionPA) |
| Промпт | [PROMPT.md](PROMPT.md) |
| Креды/доступы | VM `D:\SAION\creds\` (CREDS.md) |

## 📜 Логи

Стандарт S3/S2 ([../LOGGING-SPEC.md](../LOGGING-SPEC.md), [../TELEMETRY-SPEC.md](../TELEMETRY-SPEC.md)); инцидент-факт: PA мёртв = SYSTEM_ERROR(20) — op-кнопка pa (aionact pa start).
## 🔗 09.10 cross-pulse (authd R6): PA жив, наш authd его не зовёт

- PA 10057 остаётся запущенным (общая дисциплина стека), но наш aion-authd НЕ общается с PA
  (проверка пароля отсутствует как и у ориг) — login не зависит от PA на нашем пути.

# AUTHD-RESEARCH — ресёрч L2AuthD эмуляторов для своего authd (06.10.2026)

**Цель:** свой authd (замена `L2Authd.exe`, порты 2104/2110) для AION 7.7 EU.
**Метод:** атака аналогией — Aion `L2Authd.exe` = тот же NCSoft-компонент семейства L2AuthD
(L2 C1 → Classic → Aion; Hudson-джоба бинаря: `Platform-Auth-GameAuth-L2A1AuthD`).
Всё, что реверсили/переписывали L2AuthD — прямое методологическое топливо для нас.

⚠️ Код в `authd-ref/` = реверс-инжиниринг NCSoft (STUDY PURPOSE ONLY, как в README ChaosPaladin).
Бинари НЕ в гите (политика репо): `artifacts/authd-ref/` — локально, gitignored.

---

## 1. Кто был дальше всех (карта игроков)

| # | Кто | Что сделал | Статус/глубина | Где |
|---|-----|-----------|----------------|-----|
| 1 | **MasterToma** (mmo-dev) | 95% реверс L2 PTS C1 в IDA 6.8 (AuthD+CacheD+L2Server+NPC+LogD в составе), компилируемые сорцы (2019), за год пофиксил 7 багов CacheD + 1 в AuthD; C4 не закончил | **САМЫЙ ДАЛЬНИЙ.** Живой компилируемый код всего сервера C1, включая AuthD | mmo-dev.info/threads/l2-pts-c1-sources.10732 (+ аттач `l2_c1.7z` 4.5МБ от MrKirill1232, под логином) |
| 2 | **ChaosPaladin** (GitHub) | ПОЛНЫЙ декомпил + компилируемый код **только L2AuthD C1 build 40504**: Blowfish, DES, MD5, pwd-крипта (DB+юзеры), пакеты, INI, LogD, mail, crash-стек | Готовый эталон архитектуры authd. Скачан → `authd-ref/L2Auth-chaospaladin/` | github.com/ChaosPaladin/L2Auth |
| 3 | **010000** (mmo-dev, 01.2023) | AuthD classic off: апгрейд сорцов C1 → x64, Korean Classic 287 / off 162-287, без гейта, сменный BF-ключ, автозеркала серверов (id 111+), конфиг-минималки `[AuthD] UseAuthTimeManageSystem=false...` | Рабочий апгрейд той же ветки (поток: C1→classic). Аттач `Auth.7z` 609КБ под логином | mmo-dev.info/threads/authd-classic-off.22808 (13 стр.) |
| 4 | **AKllX** (ragezone, 08.2025) | Патч **НАШЕГО бинаря** (Aion L2AuthD.exe 3.5/4.6): FindWindowA-bypass для Win11 («сервер уже запущен»-баг) | Патч бинаря, не эмулятор. **Артефакт скачан под нашим логином** → `artifacts/authd-ref/L2AuthD-PATCHED-WINDOWS11.exe` (PE32, strings: `AccountDB.cpp`, `MobileOTP.validate` XML-RPC) | forum.ragezone.com/threads/l2authd-error.1251045 |
| 5 | **Guytis** | Публикатор C1-сорцов (авторство — MasterToma); свой **PaServer непубличен**, сам сидит на «PaServer + оригинальный Auth» | Транзитная точка шары | mmo-dev (см. треды выше) |
| 6 | **yury-dymov** | `legacy_l2auth` (2008): auth-сервер L2 PTS (C4/IL/CT1) на C++ с IP-фильтрацией | Старый, но самописный (не декомпил) | github.com/yury-dymov/legacy_l2auth → `authd-ref/l2auth-legacy-2008/` |
| 7 | **ChairmanYSL** | `L2AuthHost` (C#, 2023): tray-хост/обёртка над L2AuthD.exe (запуск/мониторинг) | Обёртка, не эмулятор; полезна схемой эксплуатации бинаря | github.com/ChairmanYSL/L2AuthHost → `authd-ref/L2AuthHost-csharp/` |
| 8 | **Ruk33** | `l2auth`: C4 login+game server на C «for fun» (клиентский wire: hosts → l2authd.lineage2.com) | Клиентская сторона логина C4; локально `STELGEN/tmp/authd-research/l2auth/` (106МБ, в гит не включён) | github.com/Ruk33/l2auth |
| 9 | **(anonym)** | portal-auth-emulator (Python+Docker, порты 10057/10058, procs `pp_GetPortalUser.sql`) — эмулятор **PA**, не authd | Аттач под логином rz 1205208 p16-17 (детали в памяти проекта) | forum.ragezone.com/threads/1205208 |

📌 **Патч «любой пароль + автосоздание акка»** (маркер глубокой изученности): для L2AuthD
классически существует в L2OFF-паках (patched-authd бинарные патчи). Для **Aion** — УЖЕ
встроено в наш живой authd и доказано живьём 06.10: **пароль не проверяется вообще**,
любой ASCII-логин автосоздаёт акк (fail только не-ASCII логин → 18b LoginFail).
⇒ наш стартовый бар выше, чем у любого L2-реверсера: у нас есть живой оракул + pdb + дизasm гейта.

## 2. Что внутри L2Auth (ChaosPaladin) — снимок архитектуры

```
src/
  network/   CAuthServer/CAuthSocket (клиенты 2104), WorldSrvServer/WorldSrvSocket
             (gameservers 2108), CLogSocket (LogD 3999), IPSessionDB, WantedPacket,
             packets/LoginPackets.cpp
  db/        CAccount (ODBC-процедуры! login-proc с OUTPUT uid + payStat),
             CDBConn/DBEnv (ODBC env/conn pool), block_msg select
  crypt/     Blowfish (ключи ниже), DES (foreign), PwdCrypt, OldCrypt, MD5
  config/    Config.cpp (INI-парсер), CIPList (BlockIPs.txt)
  threads/   CJob/CIOTimer/CRWLock (IOCP-инфраструктура)
  ui/, logger/, utils/ (SendMail — почта при краше, CoreDump)
etc/         config.txt, serverlist, BlockIPs.txt
generated/   IDA 6.8 Hex-Rays дамп (reference), reversed/ — промежуточный
```

**Ключевые факты из конфига C1 (`etc/config.txt`):**
```ini
serverPort=2104      # клиенты (в L2 клиент ходит в authd НАПРЯМУЮ)
serverExPort=2106    # external
serverIntPort=2108   # world/gs → authd (наш аналог 2110)
ProtocolVersion=30810, GameID=8
UseLogD=true, logdip, logdport=3999
OneTimeLogOut=true, AutokickAccount=true, UseOneIOCom=true
```
**Blowfish-ключи (per-pack, из README):**
```text
5F 3B 35 2E 5D 39 34 2D 33 31 3D 3D 2D 25 78 54 21 5E 5B 24   # дефолт C1
6E 63 73 6F 66 74 6C 69 6E 65 61 67 65 32 2E 63 6F 6D 20 20   # "ncsoftlineage2.com"
5B 3B 27 2E 5D 39 34 2D 33 31 3D 3D 2D 25 26 40 21 5E 2B 5D   # вариант
```
⚠️ У Aion наш гейт-статик BF `6b60cb5b...` — другая эволюция той же L2-криптосемьи.

**Прямые параллели с нашим стеком (C1 AuthD ↔ Aion L2Authd.exe):**
- ODBC + хранимки с `payStat` OUTPUT — ТА ЖЕ семантика (у нас L2Conn.dsn → AionAccounts, payStat=0-обход без PA);
- `block_msg`-таблица = наш блок-лист; `OneTimeLogOut` = наш онлайн-флаг/TTL;
- serverlist-файл = наш [02]-type-4 serverlist payload для 74b;
- LogD-канал = наш aion-logd трек (уже есть в nextgen!).

## 3. Шансы готовности своего authd (Go) после разбора этих сурсов

| Компонент | Готовность знаний | Комментарий |
|-----------|------------------|-------------|
| Архитектура/логика authd | **~90%** | Полный исходник C1 (той же кодобазы) + живой бинарь + pdb гейта |
| Wire authd↔gate (2110) | **~85%** | Уже вскрыт с нашей стороны (SmsSendConnect, [01]/[02]/[03], EncryptSecondary); L2Auth даст вторую проекцию (serverEx/Int-разделение) |
| DB-слой (procs, payStat, block_msg) | **~80%** | C1-процедуры названы в сорцах; наша схема (AionAccounts) уже частично известна, дизasm authd даст имена наших procs |
| Client-facing 2104 | **n/a** | В Aion клиенты НЕ ходят в authd напрямую (гейт держит 2106) — 2104 занят самим authd (второй листенер); выяснить роль дизasmом |
| OTP/ncguard/PA-ветки | **SKIP** | Отключаемы (UseNPLogin=false и аналоги; PA уже вырублен навсегда) |

**Итог: вероятность рабочего своего authd — ~85%**, сложность ≈ уровень aion-gate (уже сделан на ~90%).
Блокеры возможны только в редких [0x]-типах authd-wire — закрываются живым релеем (у нас уже
onAuthdPacket-роутер) + cdb/pdb дизasmом оригинала.

## 4. Роадмап «переписать по аналогии» (Go, по образцу L2Auth-архитектуры)

1. **Каркас**: `authd.exe` (Go): ODBC-pool на `L2Conn.dsn`-строке (позже — прямые SQL-процедуры), конфиг-yaml, io-пул как в aion-gate.
2. **Порт 2110 (gate-wire)** — ЯДРО: приём SmsSendConnect от гейтов, роутинг [01]/[02]/[03]-типов, serverlist-payload ([02]-type-4), payStat-ответы. Эталон: `WorldSrvServer/WorldSrvSocket` из L2Auth + наш живой log релея.
3. **Логика логина**: account-name → proc (uid, payStat) → автосоздание (INSERT) при отсутствии → блок-проверка (block_msg) → сессия (OneTimeLogOut-флаг) → payload [03] с V-константой (наш 0x0000c621).
4. **Порт 2104**: определить потребителя дизasmом (Server64?) — эмулировать при необходимости (готовый шаблон: CAuthSocket из L2Auth).
5. **LogD**: продолжить трек aion-logd (WriteLogD-формат уже виден в C1-сорцах: type, accName, ip, payStat, age, variant, uid).
6. **Паритет-тесты**: живой дифф наш-authd vs L2Authd.exe на фреймах гейта (метод fork-proxy из трека гейта).
7. **Свитч прод**: 2110 → наш (гейт уже умеет переподключение authReconnectInterval=30), открат — задача AionAuth.

## 5. TODO (докачать)
- [x] ~~`l2_c1.7z` (MasterToma C1 сорцы)~~ — СКАЧАН 06.10 (юзер дал с гейт-сервера 192.168.0.248:3923) → в гите `authd-ref/l2-c1-mastertoma/` (35МБ, 1140 файлов): `L2Auth/` (reversed 6М + generated 5.8М + src 780К, маркеры FIXED: overflow в CIOTimer/CJob, blockFlag_custom в CAccount), `L2LogD/`, `CacheD/`, `L2Core/`, `PetitionD/`, **`DBScript/` = ReleaseAuthDBSchema.sql (procs: `ap_GPwd`, `ap_GStat` ← payStat!, `ap_GUserTime`, `ap_SLog`, `ap_SUserTime`) + lin2comm.sql (44 procs) + lin2user/lin2log/lin2report/lin2world** — ГОТОВАЯ СХЕМА БД authd (все девелоперские ветки: legacy/develop-Extender C1/C4/C6, MSVC2013+). Полный пак (99МБ: + html 30М, CachedScript 29М, tests 5.7М) — локально `~/STELGEN/tmp/authd-research/artifacts/l2_c1/` + исходный `l2_c1.7z`.
- [ ] `Auth.7z` (classic x64, mmo-dev 22808) — нужна регистрация mmo-dev.
- [ ] RZ 1205208 p16-17: аттач portal-auth-emulator (если PA когда-нибудь понадобится — вердикт SKIP в силе).
- [ ] Погуглить Google-Drive changelog MasterToma (ссылка в его подписи на mmo-dev).

📌 Примечание: README ChaosPaladin/L2Auth и README MasterToma-пака совпадают почти дословно — это одна и та же шара (MasterToma-стрим 2019 → ChaosPaladin-репо). В гите лежат ОБА для трассировки.

## 6. Локальные артефакты (вне гита)
- `artifacts/authd-ref/L2AuthD-PATCHED-WINDOWS11.exe` (725КБ, PE32) + исходный zip — патч AKllX.
- Полные клоны: `~/STELGEN/tmp/authd-research/` (L2Auth, l2auth-Ruk33 106МБ, L2AuthHost, legacy_l2auth).
- Сессия ragezone: `/tmp/rz.txt` (cookie xf_user, логин Leonid_8952134).

# Промпт для нового чата: ПЕРЕПИСЬ L2Authd → свой authd (Трек B, шаг 4)

> Скопируй текст ниже в новый чат как первое сообщение.

---

Ты — ассистент проекта **STELGEN / AION 7.7 PTS EU private server**. Продолжаем Трек B: поэтапная замена NC-бинарей на свои (Go). Заменено и в бою: **aion-logd** (:2051), **aion-captcha** (:22206), **aion-gate** (:2106, замена AuthGateD — wire 2110 раскрыт нами 1-в-1). Метод отработан трижды. Теперь по плану **L2Authd → свой authd**.

## Первый шаг (обязательно, до любых действий)
1. Прочитай память по пути `STELGEN/projects/aion_server_2026-10-02` (хронология, готчи, прод-стек).
2. Прочитай в репо `~/STELGEN/projects/aion_server_2026-10-02/AION-7.7-EUR-SERVER/`:
   - `nextgen/AUTHD-ROADMAP.md` — план фаз R0-R6 и полный реестр сурсов S1-S7 (живой);
   - `nextgen/AUTHD-RESEARCH.md` — ресёрч эмуляторов L2AuthD (кто дальше всех, артефакты);
   - `nextgen/aion-gate/README.md` — раздел «Канонические факты протокола»: **вся наша сторона wire 2110** (фреймы, blob 191Б asm-форма, [03] V=0xc621, type=3/4/7, онлайн-флаг);
   - `nextgen/ROADMAP.md` + `nextgen/TELEMETRY-SPEC.md` (обязателен для переписи);
   - `nextgen/LOGD-REWRITE-ANALYSIS.md` — метод PDB→publics→дизasm (референс).
3. Сурсы-эталоны уже в репо: `nextgen/authd-ref/L2Auth-chaospaladin/` (полный декомпил L2AuthD C1), `nextgen/authd-ref/l2-c1-mastertoma/` (+ `DBScript/` — схема БД authd: procs `ap_GPwd/ap_GStat/ap_GUserTime/ap_SLog/ap_SUserTime`), `reference/Mobius_AionEmu/` (семантика фейлов/онлайн-флага).

## Цель
Своя замена **L2Authd.exe** (1,198,592 Б, NC, native C++; порты **2104** serverPort, **2110** serverExPort — его слушает наш aion-gate, 2108 GM, 10062 QMAS; конфиг `etc\config.txt`; БД `AionAccounts` через `L2Conn.dsn`; клиенты: AuthGateD 2110, AccountCache 2220; PA 10057 — DISABLE навсегда, SKIP). Живёт в `D:\AION_LIVE_SERVER\L2Authd\`, задача планировщика **AionAuth** (ритуал рестарта `C:\Temp\restart-auth.ps1`). Реверс-фундамент: **L2Authd.pdb малый уже скачан локально** (manifest-pdb-big.md) → метод pdbpub.py.

## Дисциплина (не нарушать)
- **Прод трогать ТОЛЬКО после явного «го» юзера.** Fork-фаза (R5) невидима для игрока: наш гейт продолжает ходить в ориг.
- Каждая замена переключаемая: бекап, откат одной командой; бекапы БД перед любым SQL ALTER (`D:\_REF58\prod-backups\`).
- **L2Authd ХРУПКИЙ: умирает от кривых пакетов** — в fork-режиме оригиналу НЕ СЛАТЬ ничего, кроме копий валидных фреймов гейта; смерть → рестарт-ритуал `C:\Temp\restart-auth.ps1` (/end+/run AionAuth → wait 2104 → рестарт AionGate).
- Probe-логины ЛОЧАТ акки на 2-6 мин (онлайн-флаг authd, TTL) — тестовые креды держать пулом (логины-цифры 1/2/3...), не долбить один.
- TELEMETRY-SPEC: телеметрия В СЕТЬ, пакет `internal/ship` копировать из `nextgen/aion-logd/internal/ship` как есть, конфиг-ключи `ship.*` единые.
- Своя апка: `D:\SAION\aion-authd\` (exe + config.yaml + run.cmd, ASCII+CRLF); автостарт из юзер-сессии 1 (паттерн AION-START-ALL-v6.bat).
- Готчи VM (192.168.0.125, `ssh 'Администратор@192.168.0.125'`): дефолт-шелл PowerShell; scp push да, pull нет (вниз через `cmd /c type`/PS base64); git `--no-pager`; Go тулчейн `~/STELGEN/go-dist/go/bin`; кириллица в конфигах — только байтовая замена; sqlcmd на больших XML глючит — SqlClient ExecuteScalar; TBL_GAME_* в схеме aiongm_ur; тела procs через sp_helptext.
- Секреты (connStr/пароли) — только в конфиге на VM, в гит/память не сохранять.

## План работ (фазы R0-R6 из AUTHD-ROADMAP.md; каждый шаг = коммит + пуш + дельта в память)
1. **R0 Разведка (read-only)**: инвентарь `D:\AION_LIVE_SERVER\L2Authd\` (etc/config.txt полный — зеркало в гит; логи winlog/packet/dual — снять хвосты); L2Conn.dsn → строка подключения (без сохранения секрета); netstat-карта портов 2104/2108/10062/2220 (кто реально держит/подключает); L2Authd.pdb → `pdbpub.py` → publics → дизasm ключевых функций (таблица диспетчера фреймов 2110, DB-вызовы); sp_helptext инвентарь procs AionAccounts vs `DBScript/ReleaseAuthDBSchema.sql` (C1-эталон).
2. **R1 Протокол-фундамент**: wire 2110 по golden-фреймам `D:\SAION\aion-gate\gate-prod.log` (RAW A>G/G>A уже там: [00]/[01]/[02]→вверх, [03]/[02][type]→вниз, blob 191Б) → док `docs/authd-wire-*.md` + golden-тесты. Проверить: шифруется ли 2110-wire (в логах видно).
3. **R2 Каркас** `nextgen/aion-authd/` (Go): main/config/ship/db/session, листенер 2110, replay-режим (отвечает golden-ответами из логов) — e2e с живым гейтом на стенде (второй порт!).
4. **R3 Логика**: порт логики C1 (`CAccount`): blob → user/pwd/otp → автосоздание (пароль НЕ проверяется — live-факт 06.10) → block_msg → online-флаг (TTL 2-6 мин, OneTimeLogOut) → [03] V=0xc621 → [02][type=3/4/7] (74Б serverlist / 42b / 26b). Семантика фейлов — из Mobius `AionAuthResponse` (уже в aion-gate authfail.go).
5. **R4 DB-слой**: реализация DB-вызовов поверх AionAccounts (вкладка ODBC/DSN — НЕ тащим, прямые SQL/procs).
6. **R5 fork-proxy A/B**: fork-proxy на 2110 → ориг (живой путь) + копия в наш; diff O-vs-N на каждый фрейм до 100% паритета; тест-прогон юзера на ориг + наш одновременно.
7. **R6 Свитч** по «го» → 2110 наш → наблюдение 24ч (логины, relogin, онлайн-флаг, ночные циклы) → финал: ROADMAP.md/README дельта + session-док + статус-снимок AUTHD-STATUS-SNAPSHOT.md.

Начни с R0 (разведка read-only) и дай план уточнений после осмотра. Ничего на проде не меняй без «го».

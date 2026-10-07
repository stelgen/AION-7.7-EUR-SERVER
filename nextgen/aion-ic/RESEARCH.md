# 🔎 RESEARCH aion-ic — внешние источники (RZ/mmo-dev/GitHub) — 07.10.2026

> Цель: найти готовые бинари/эмуляторы ICServer (Interchange 2005/2305), чтобы не работать с нуля. Итог: **публичного IC-эмулятора НЕ существует**; найдены документация протокола + чужие PTS-киты для кросс-версионного диффа. Логин RZ заведён, cookies обновлены, файлы скачаны на VM.

## 1. Вердикты по источникам

| Источник | Результат |
|---|---|
| GitHub repo-search `aion icserver` / `aion interchange` | **0 репозиториев** |
| Деревья beyond-aion, AionGermany, sunbsn/AionEmu, 2× AionLightning | 0 хитов `interchange\|icserver\|matchmak\|cached` — все Java LS/GS/CS, IC не реализован нигде |
| mmo-dev.info (Aion/Сборки) | IC не упоминается; общий шум 4.7.5 |
| MMOForge.dev | коммерческий сервис (3.5/4.6/5.8/7.7 retail + 4.8 emu) — платно, не паблик |
| RaGEZONE | PTS-киты с IC: 4.6 (AKllX/auglyn), 5.8 Leaked, 7.7; 2.7 PTS **без IC** |

## 2. Протокольные факты (открытый текст, пост AKllX #26, тред 1197955)

- `Server64/config.xml`: `<InterSvrType>1</InterSvrType>` = live; **`2` = matchmaker/beginner-сервер** («the main difference»).
- `Server64/common.xml` (блок `3.0.1221`): `ICServerAddr` / `ICServerPort` (**2005**) / `ICServerId`.
- Matchmaker = **отдельный мини-стек** (MainServer+NpcServer+Cache+Log) с непересекающимися портами; clientAcceptPort 7778; `aion_event=true` = вход только через matchmaker.
- 4.6 PTS: `matchmaker.xml` `<matchmaker_type>1</matchmaker_type>` → `0` — фикс стартовых ошибок ICServer/арен (тред 1250659).
- ICServer **опционален**: «First: it is not needed for work, you can not turn it on» (Mr. House, 1250659).
- Сходится с нашим `ICServer.common` 7.7: `numBeginnerServer=1`, `numIdentifiedServer=2`, `numGAb1Server=2`, `gabyssGroupIdType=3` — IC распределяет слоты/типы миров.
- Серийная защита 4.6/5.8/7.7 **одинакова** (marisa-chan: Themida-nop `0x90`) — применимо к нашему ICServer.exe.

## 3. Скачанное на VM (D:\SAION\downloads\rz\)

| Файл | Размер | MD5 | Источник |
|---|---|---|---|
| `2.7server(Dev_110907)_rel(111222).rar` | 71 875 802 | `91aad459783c464a840336c1589aadf2` | GDrive из треда 2.7 PTS (1267905) |
| `Server64_byAKllX.rar` | 3 656 140 | `9ea90d5d1b5a871d10a99f3f8274bf7c` | GDrive тред 1197955 (post #22) |
| `4.6 DB.rar` (имя юникод) | 274 537 164 | `90b12b28581fe9444f82a3e16720c476` | GDrive тред 1197955 (DB mirror TieLay) |
| `5.8static_data.7z` | 49 995 552 | `2623e9d921b3c187d0fdbae107156ad7` | GDrive тред 1234115 |
| `58Server.rar` (реально 7z) | 2 284 132 213 | `d9bb20e78186f5ac887ae77d1cf8360b` | GDrive тред 1234115 |
| `unpacked\2.7\` (CacheD/LogD/Main/NPC/AccountCacheD) | — | — | распаковано 7-Zip |

- **Кит 2.7 PTS (2011)**: `CacheD64.exe` 19.9МБ, `Server64.exe` 28МБ, `NPCSvr64.exe`, `LogServer64.exe`, `AccountCacheServer.exe`. **ICServer.exe НЕТ** — Interchange появился позже (блок конфигов помечен `3.0.1221`).
- **58Server.rar**: grep `ICServer|Interchange|matchmaker` = **0 хитов** (есть `Cached\CacheD64.exe` 22.3МБ 2020).
- GDrive `1FrkwWJu8...` (полный кит 4.6 AKllX?) — **мёртв (Error 404)**.
- Внимание: EXE-дампы из треда 2.7 (мега/GDrive #17 sunbsn) — по вердикту marisa-chan (#27) **малварь-стабы** (VMProtect, C2 1.94.118.214:8897) — не запускать вне изоляции.

## 4. Незакрытые ссылки (TODO, нужны Mega-инструменты или браузер VM; мусор `gd-t46-3.bin` — удалить)

- `mega.nz/folder/QB9DiBCY#sAqxYdxYc9D3TeoS7NwRXA` — **PTS5.8 Leaked полный кит** (тред 1234115, sunbsn) — единственный непроверенный кандидат на ICServer 5.8 PTS-варианта.
- `mega.nz/file/oFNQgC7Z#ZGl9...` — l2auth «accept any sn» патч (marisa-chan; для aion-authd, не IC).
- `mega.nz/file/WcVRnKyB#...`, `bZUhgYYZ#...`, `rdNXHaiQ#...` — зеркала 4.6 (DB/клиент CC=2/Server64_byAKllX).
- RZ-аттачменты (скрины кряка, webp) — качаются по cookies (`/attachments/...`), список в html-дампа `D:\SAION\downloads\rz\html\`.

## 5. Cookies-сессия RZ

- Старый jar умер → повторный логин 07.10 (XenForo `_xfToken` + POST `/login/login`) → **LOGIN_OK**, скрытые ссылки открыты на всех страницах.
- `D:\SAION\creds\rz-cookies.txt` (VM) + локальная копия `/tmp/rz.txt` — обе актуальны.

## 6. Выводы для aion-ic

1. Реверсить **свой** ICServer.exe 7.7 (есть в обоих китах) + PDB 104МБ — альтернативных сурсов нет.
2. Протокол-семантика из §2 — уже фиксируется в канон README (не ждать R0).
3. Для кросс-версионного диффа IC: ждать/искать 4.6 кит (GDrive мёртв; кандидат = PTS5.8 Leaked Mega-папка или RZ-поиск «aion 4.6 server files» повторно).
4. 2.7 кит полезен для **aion-cache** (CacheD-словари 2011 без обфускации), не для IC.

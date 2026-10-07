# 🧵 Сессия 05.10.2026 — финал логгера: Л1–Л4, подмены ×3, ship e2e, дизasm

> Контекст этой сессии (чтобы не потерять). Коммиты: `1e7c9d7` → `fe2976b` → `e6ede55` → финал (см. git log).

## Что сделано (хронология)

1. **Проверка гита/памяти/прода**: гит HEAD соответствовал памяти (свинарник — нет); на десктопе VM батник = v5 (MD5 919c49dd, бит-в-бит с гитом), логгер в бою SYSTEM-задачей AionLogCap.
2. **Батник v6** (`scripts/AION-START-ALL-v6.bat`, десктоп MD5 42ee98e6): шаг [3/12] = наш logd (`D:\SAION\aion-logd\run.cmd`), шаг [0/12] = end AionLogCap + kill aion-logd; всё стартует в юзер-сессии 1; v5 сохранён как откат. Инструкция: разово `schtasks /change /tn AionLogCap /disable` (иначе SYSTEM-инстанс вернётся на буте).
3. **Л1 type-9**: сняты свежие capture с прода + mirror-эталон; layout доказан: `[u32 id=928][{u32 key][wchar NUL]...][tail][SYSTEMTIME = последние 16 байт]`; entries = онлайн-сессии (1002,"SteLGeN")(1010,"Stelgen"); 223b = 1 сессия, 251b = 2. `internal/textlog` + fixture'ы (реальные пакеты) + тесты.
4. **Л2 сверка (read-only)**: TBL_GAME_* в схеме **aiongm_ur** (не dbo!); проц UpdateMainStatus/UpdateTotalMainStatus на проде ОТСУТСТВУЮТ → оригинал звал их в вечный 2812 и зона-счётчики не писал → наш logd уже на паритете; методы добавлены (позиционные {call}), вызов заглушен до деплоя REF58-проц + маппинга metric1-4.
5. **Л3**: InitializeCount на ServerStarted svc=3 (config `server.init_svc`); **Л4**: writer.Sweep (retention_days=14, старт + 30 мин).
6. **ship** (`internal/ship`): syslog RFC5424 udp/tcp octet-counted + HTTP ndjson + локальный ndjson (только явно); неблокирующая очередь + drop-счётчики + recover; self-статус ev=self; события start/stop/conn.up/down/version/server.started/status/text/parse.err/db/db.err/sweep/self. **Спека для всех переписей: ../../TELEMETRY-SPEC.md** (вшита в Трек B).
7. **Подмена ×1** (12:28): патч конфига питоном (init_svc, retention_days, ship-секция; без вывода секретов), backup exe+лога, swap. Лог: 3 клиента ESTABLISHED, InitializeCount(svc=3) ровно 1 раз, мир жив. Готча: после `/end` ЖДАТЬ смерти процесса (до 10с) — иначе copy «file in use».
8. **Косяк найден и исправлен**: координаты были int-биты (`pos=(1146813568.0)`) — `floatFrom` int→float вместо бит-реинтерпрета; фикс `math.Float32frombits`; подмена ×2 (12:36, PID 7220) → `pos=(952.5,1100.3,108.5)`.
9. **rsyslog e2e**: настоящий rsyslog 8.2504 в песочнице + наш linux-бинарь + фейк-клиент: UDP (12/12 событий) и TCP octet-counted (100%, включая ev=stop); PRI 14/12/11 корректные; self sent/dropped/ошибки. Приёмник убран после проверки (по требованию юзера). Прод до песочницы не дотягивается (docker-сеть) — на проде ship выключен до подъёма rsyslog на LAN Linux.
10. **Дизasm type-9** (LogServer64.exe+PDB локально): конвертер MsgId→wire-type @0x1400130ca: 0x644→4 Control, 0x645→5 Status, 0x646→6, 0x647→8, 0x648→9 TextLog; фрейминг общий Shared\LogClient.cpp; хвост type-9 набивается на call-sites Server64 (PDB 284МБ) — отложено. Anchor'и: LogBuffer ctor 0x14023290, CollectorThread 0x14023680, SendServerStarted 0x14026820, DecodeBotLog 0x140114f0. ⚠️ VA = ImageBase + off (off = RVA, НЕ sectionVMA+off); objdump БЕЗ `--start-address` (с фильтром адресов PE выдаёт пусто).
11. **Badstatus-шторм вскрыт и закрыт**: 2337 записей type-5 НЕ-194 (svc=701: `LDF5_Fortress_7011`, `LDF8_1993`, `ab1_1018` — ключи фортов/зон + поля n/n/d/d) падали в badstatus (у оригинала тоже — его старый лог уже имел 1898 таких) → `records.ParseVar` + writer per-svc .err + тесты на реальном сэмпле; подмена ×3 (PID 2704, MD5 7aca9dca) → 0 'status parse' ошибок, мир жив.
12. **Аудит доков**: ../../../docs/app-architecture.md исправлен (прокси-строки удалены, гейт 2106, RunAsDate не нужен, max mem 4096, порядок старта, aion-logd в таблице сервисов).

## Финальное состояние прода (05.10 ~13:00)

- aion-logd PID 2704 (SYSTEM-задача AionLogCap), :2051, 3 клиента ESTABLISHED, InitializeCount(svc=3) выполнен, 0 parse-ошибок, мир собран (7777 LISTENING, Server64 7348)
- ship: `enabled=false` (секция в конфиге готова)
- Откаты: логгер = `schtasks /run AionLog`; стек = v5 батник

## Инструменты этой сессии

- /tmp/t9analyze.py, /tmp/t9dump.py, /tmp/t9conn.py — анализаторы type-9
- /tmp/fake_logclient.py — фейк-клиент (handshake+type3/5/9/11+bad packet)
- /tmp/swap2.sh, /tmp/swap3.sh — процедуры подмены (с wait-loop)
- /tmp/patch_config.py (на VM: C:\Temp\patch_logdcfg.py) — патч конфига без секретов
- Сэмплы: ~/STELGEN/tmp/logd-io/t9/ (payload/capture/mirror), /tmp/badstatus.hex, /tmp/prodlogd.log
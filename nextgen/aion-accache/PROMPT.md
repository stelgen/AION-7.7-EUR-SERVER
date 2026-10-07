# ГОТОВЫЙ ПРОМПТ для следующего чата (копипаст первым сообщением)

AION aion-accache — ПРОДОЛЖЕНИЕ (прочитай память STELGEN/projects/aion_server_2026-10-02
+ ROADMAP.md + RESEARCH.md + README.md).

ГДЕ МЫ: каркас R2 готов и запушен (nextgen/aion-accache, Go: proto [u16 len-2][u16 cmd][0xEB][~cmd]
лимит 0x2000/cmd≥0x6C reject; dispatch T1 cmds 0..39 (cmd 22 не занят) + T2 0-7; RAM-кэш;
db-интерфейс {call}-обёртки; server с state-машиной OnRead; ship 2220; тесты зелёные).
R0 done: PDB 92МБ+map+exe (aion_rev artifacts/pdb-big/AccountCacheServer), тела 101 procs + 21
таблица REF58_AionAccountCacheD (accountcache-ref/db-procs-77-ref58.rpt), клиент 2220 = Server64
(authd ленивый), RPC-словарь 74, PDB publics 8035 (Decode/Encode-классы ServerToAccountCached/
AccountCachedToServer/DBConn с параметрикой).

ЦЕЛЬ ЧАТА = R1: capture-стенд :2220 (fork-proxy паттерн captcha-mirror/authd-R5).
(1) СХЕМА: копия ориг-ACS на :2221 (байтовая правка common.xml serverPort в КОПИИ каталога
D:\AION_LIVE_SERVER\AccountCacheServer-2221; ориг НЕ трогать; задача schtasks от SYSTEM) +
наш Go-прокси на :2220 (task AionAccProxy) → forward :2221 + hexdump per-frame C2S/S2C в лог
(WriteIO-паттерн logd). Проверить: Server64 переподключается сам (did в captcha/logd).
ОТКАТ одной командой: kill proxy + возврат common.xml (бэкап .bak-2221).
(2) КАПЧЕР: юзер логинится ×2-3 (Server64 сам наливает FIRST_LOAD/CHAR_LOGIN/SAVE_CUSTOM/LUNA/
PLAYTIME). Дамп тянуть tar-pipe, парсить локально (sh — без braces; UTF-16 строки).
(3) РАЗБОР: payload-раскладка per-cmd (арбитр = PDB-сигнатуры Decode*@ServerToAccountCached из
pdb-publics-8035.txt; EXE на VM и локально), ACP-номера ответов (SendIOBuffer = vtable indirect —
смотреть лока Store/x64dbg поCapture; или дифф: ответ приходит с cmd=X — снять), семантика T2
(кто второй клиент — netstat при live), длина payload каждого cmd.
(4) R2.5: вшить раскладки в nextgen/aion-accache (internal/payload + тесты на живых fixture
hex-кадрах), заполнять хендлеры по мере расшифровки; R3: SQLStore go-mssqldb (connStr секрет
на VM/env, логин aionop_ro+EXEC-гранты на AionAccountCacheD procs — см. logdb-гранты logd).
(5) R4: A/B дифф байт-в-байт (наш shadow против ориг на одном трафике, VERDICT=SAME/DIFF паттерн
forkauthd). Критерии в ROADMAP. R5-свитч ТОЛЬКО по «го» юзера.

ГОТЧИ ИНФРА (из памяти): ssh Администратор@192.168.0.125 ключ dimini-agent (~/.ssh/id_ed25519);
дефолт-шелл = PowerShell: && запрещён, одиночные команды, бинарный tar только cmd /c "tar -cf -";
sqlcmd sa/123 (SqlClient в PS5 НЕ работает — TLS); sqlcmd -o -y 0 -Y 0 + OBJECT_DEFINITION;
scp push работает; прод-стек поднят, Server64 жив — НЕ ронять (правила ROADMAP §4);
GOTCHA dispatch-77.md: cmd 22 пусто; номера ACP = TBD; carcass ответы = эхо-cmd (не деплоить).

РЕЖИМ: малые итерации — шаг = код/правка + тест + коммит+пуш + дельта в память. Креды/секреты
в гит/память НЕ класть. Прод-ACS не трогать без «го».

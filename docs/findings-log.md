# Журнал расследования логина — находки, теории, патчи (03.10.2026)

Хроника всего, что мы проверили в борьбе с «вы были отключены от сервера».
Каждый факт — из логов/перехвата/дизасма, не догадки (догадки помечены).

## 1. Хроника клиентских попыток

| Клиент | Версия | Запуск | Результат | Причина по логам сервера |
|---|---|---|---|---|
| euro_aion 7.7 (EN, CEF-портал) | 7720.0603.1118.16350 | AionLauncher + launcher.config | «Вы отключены от сервера» (мгновенно/10с) | `Session id mismatched.(RecvLogin) sessionId:0, m_iSessionId:1..4` → RST |
| euro_aion, -cc:7 тест | тот же | то же | «you can login only after official portal» + позже снова «отключены» | тот же mismatch; текст ошибки клиента менялся между попытками |
| aion rus 7.7 (Innova/4game, Frost) | 77.7.0909.28 (.inn.meta) | bin64\Aion.bin напрямую | «This program is unavailable in your country» | гео/Frost-чек при старте (VPN-выход IP); cc.ini 2 и 7 — не влияет |
| aion rus, Frost-обёртка (`bin64\Frost\Aion.exe -frostGame ..\..\bin64\aion.bin`) | тот же | cwd=bin64\Frost | Splash → но с VPN: снова country error; без VPN (cc=7): дошёл до РОДНОГО экрана логина | Frost инициализируется только из своей папки (cwd!) |
| aion rus, логин со своего IP | тот же | Frost-старт | отправил логин (186 байт) → гейт умер → клиент висит бесконечно | см. раздел патчей |
| AION_KR 7.7 (папка) | **79.21.0209.16665 (= 7.9!)** | bin64\Aion.bin | GameGuard error 1xx | KR-клиент требует GG без -noauthgg; версия 7.9 ≠ сервер 7.7 |
| наши TCP-пробы | — | /dev/tcp | соединение живёт, лог не пишет | AuthGateD не логирует пустые коннекты |

Дополнительные факты перехвата: клиент подключался с VPN-выходов (184.160.77.85, 89.47.164.187) — VPN-клиент перехватывает ВСЕ маршруты, включая локалку; после отключения VPN — коннект с реального IP.

## 2. Протокол AuthGateD (раскопано перехватчиком)

Фрейминг: 2 байта LE = длина пакета. Подтверждённые размеры: welcome **194** (0xC2),
клиент-ответ **34** (0x22), сервер-ответ **42** (0x2A), логин **186** (0xBA).
Ранее в pktmon-захвате (127-байтные срезы) видны 248/88/96/240 — разные фазы (да/разные попытки).

Хендшейк: welcome несёт RSA-модуль (гейт генерит RSA при старте: `RSA Key Generated...`).
Клиентский 34-байтный пакет = 32 байта payload = RSA-шифрованный ключ сессии (**256-бит RSA!**).
42-байтный ответ сервера содержит повторяющиеся 8-байтные блоки — блочный шифр (ECB-стиль).
После этого ВСЕ пакеты (включая логин 186b) зашифрованы → перехватчик видит шифр.

Ключевая строка лога гейта (UTF-16, VA 0x42ce38): `Session id mismatched.(%s) sessionId:%d, m_iSessionId:%d, IP:%08x`;
`(%s)` = имя обработчика (RecvLogin, VA 0x42c7e4).

## 3. Патчи AuthGateD (все верифицированы assert'ами исходных байтов)

Символьная база: AuthGateD.map (319 КБ) + AuthGateD.pdb (1.7 МБ) — есть в ките.

| # | Точка | Что меняли | Результат |
|---|---|---|---|
| p1 | VA 0x4079d0 (функция check: `mov eax,[ecx+0xfc]; mov edx,[esp+4]; cmp; je OK`; WARN при false) | заменено начало на `mov al,1; ret 8` (всегда OK) | старт OK, authd-канал умер через ~96с |
| p2 | VA 0x4063db (je в RecvLogin-обработчике @0x4063d4) | NOP×6 | висел на старте (под чужим именем в C:\Temp) |
| p3 | VA 0x4041b6 (`add eax,ecx` перед `mov [esi+0xfc],eax` — генератор id) | `xor eax,eax` (id=0) | под родным именем старт OK, но WARN показал m_iSessionId всё ещё 1 → генератор не тот путь |
| p4 | VA 0x4079dc (`je 0x407a05` → `jmp`, семантика p1 но с сохранением чтения полей) | 2 байта | под родным именем: authd-канал умер через ~118с, затем гейт исчез |
| p5 | = p2, но под родным именем/путём | NOP×6 | authd-канал ЖИВ; логин юзера прошёл проверку → гейт **тихо exited** во время обработки (клиент висел) |

**Разгадка зависаний**: запуск под чужим именем/из C:\Temp подвешивал процесс (NCsoft SharedGuard/самопроверка пути). p1-«всегда OK» ломает authd-канал (проверка вызывается и в authd-контуре — 7 call-сайтов: 0x4063d4, 0x406678, 0x4067d7, 0x406bc9, 0x406d39, 0x406e68, 0x4070e7).
**Разгадка смерти при p5**: после снятия сессионного замка логин уходит в парсер `0x407ac0` → там гейт тихо выходит (вероятно, детект аномалии/нехватка полей нового формата клиента) — нужен реверс парсера.

Текущее состояние: **оригинальный AuthGateD восстановлен** (бэкап: `AuthGateD.exe.orig-04079d0`; патчи лежат `C:\Temp\AuthGateD_p*.exe` на VM и `/tmp/AuthGateD_p*.exe` локально).

## 4. Клиенты: почему шлют sessionId=0 (теория, уровень уверенности высокий)

Все доступные клиенты — западной/новой линейки:
- euro_aion (EN, 0603): логин через **CEF-портал** (`AionCefProcess.exe` живёт в момент логина; сообщение «official portal») → session id должен приходить из портал-сессии.
- aion rus (Innova, 0909): 4game-портал (Frost-обёртка передаёт `-enaccount/-enpassword` — зашифрованные креды портал-логина, видно в `launcherUpdate.log` старой инсталляции) + Frost/гео-чеки.
- AION_KR «7.7» (фактически 79.21 = 7.9): GameGuard + более новый протокол.

Сервер кита (0601) рассчитан на клиента с классическим логином, который ЭХОИТ sessionId
из welcome-пакета. Ни один из доступных клиентов этого не делает.

## 5. БД-часть auth (раскопано полностью — это редактируемый слой!)

Procedures в `AionAccounts` (все с сигнатурами вытащены):
- **Регистрация**: `agent_CreateAccount(ggid uniqueidentifier, account varchar(14), password binary(16), email, mobile, question1/2, answer1/2 binary(32))`, `ap_AutoReg(@account)`, `web_CreateAccount(nickname, account, password binary(16), email, mobile, q1, q2, answers)`, `l2p_TempCreateAccount(@account, @ssn)`, `Pr_acc_act`
- **Логин/проверки**: `ap_GPwd11(@account, @pwd binary(16))`, `ap_GPwdWithFlag(+@flag, @otpflag)`, `ap_GStat/ap_GStat11/ap_GStatEtc` (статус: payStat, loginFlag, warnFlag, blockFlag, blockFlag2, subFlag, lastworld, block_end_date, forbidden_servers binary(16))
- **Серверы/мир**: `ap_GetServers`, `ap_SetServerStatus(gameServerNo, statusCode)`, `ap_SetConcurrentUserStatistics(serverNo, worldUser, limitUser, authUser, waitUser)` — CCU!
- **Персонажи/слоты**: `ap_GetGameAccountNo(gameAccount)`, `ap_GetAccountGameSlot`, `ap_GetAccountMaxLevelCharacter`, `ap_RegisterAccountGameCharacter`, `ap_GetRestriction`, `ap_SetGameRestriction`, `ap_GetSSN`, `ap_SetIllegalLoginTrace(gameAccountNo, IP, traceType)`, `ap_SetPasswordResetFlag`
- **Логи/сессии-время**: `ap_SLog(uid, lastlogin, lastlogout, LastGame, LastWorld, LastIP)`, `ap_SNewPwd(@account, @pwd, @encFlag)`, `ap_SUserData`, `ap_SUserTime/ap_GUserTime`, `ap_LoginWithPoint/ap_LogoutWithPoint` (портал-поинты)
- **Веб-портал API**: `web_FindAccount(@account → @uid, @password binary(16), @email, @credits)` — читает `user_auth.password` (binary16 = **MD5-хеш пароля**!) и `ssn.email`

Таблицы: `user_account` (uid, account, pay_stat, login_flag, warn_flag, block_flag, block_flag2, last_login, last_logout, subscription_flag, last_game, last_world, last_ip, block_end_date, forbidden_servers), `user_auth` (password = MD5), `ssn` (профиль: email, ssn char(13), телефоны, адреса, status_flag), `userno` (uid-счётчик), `user_info` (ses_num, kind, ip/mask_ip, bsoeday_time...), `account_data` (L2-наследие: id/name/password/access_level), `savormix_user_portal` (uid, login_name, account, birthdate, sex, block/notice/status codes, user_level, user_type_code, company_code, user_id binary16) — **портал-профили от эмулятора savormix (пустая)**, `worldstatus` (idx, server, status), `user_count` (CCU-история: record_time, server_id, world_user, limit_user, auth_user, wait_user, dayofweek), `item_code`, `gm_illegal_login`, `block_msg`, `block_reason_code`, `user_block/pay/stat/time/data`, `banancheg` (?).

Схема auth-БД из кита: `AuthD\lin2db.sql` (полный SQL-скрипт БД — источник для реконструкции/сравнения).

## 6. Уровень уверенности: переписываемость стека

- **Auth-логика аккаунтов** — в SQL-процедурах: редактируемо свободно (правила регистрации, блокировки, CCU).
- **Транспорт/сессии/парсинг пакетов** — в exe (AuthGateD): патчить можно (map/pdb есть), но нужно точечно (грубые always-OK ломают authd-контур: эта же проверка вызывается из 7 обработчиков).
- **Мир/спавны/квесты** — Server64 + ScriptDLL64 + NPCSvr + XML: частично редактируемо (XML-скрипты), частично патчи (PDB есть).
- **Вывод**: «переписать с нуля» никто не будет; целевые патчи + правки SQL/XML = рабочая стратегия.

## 7. Пути решения (по возрастанию трудозатрат)

1. **Клиент 0601-семейства** (CN/KR PTS 7.7 билд июня 2020): идеальный, без патчей. Проблема: в доступных источниках таких нет (все 0603/0909/7.9). Искать: kit-диск/торренты/архивы.
2. **Клиентский патч**: научить доступный клиент эхоить sessionId из welcome (aion.bin 6 МБ — реверсить тяжелее сервера; Frost/CEF осложняют).
3. **MITM-перехватчик с расшифровкой**: прокси уже стоит (2106→2107, hex-лог). Дальше: подмена RSA-ключа в welcome на свой → расшифровка 34-байтного пакета (ключ сессии) → открытый трафик → правка sessionId → шифрование к гейту. Альтернатива: факторизация 256-бит RSA (yafu/msieve) → пассивная расшифровка («вайершарк с ключом» — идея юзера, TLS-серты не нужны, это бинарный протокол).
4. **Реверс парсера логина гейта** (0x407ac0 + обработчики): понять, что убивает гейт после снятия сессионной проверки (p5-эксперимент показал: тихий exit в глубине); с .map/.pdb задача локализуется.
5. **Портал-эмулятор** (web_* API + savormix_user_portal): полноценный веб-портал, который выдаст клиентам сессию (как retail) — тогда клиенты западной линейки заработают «как у официальцев». Самый дорогой, но самый универсальный путь.

## 8. Артефакты

- VM: `C:\Temp\AuthGateD_p1..p5.exe` (патчи), `AuthGateD.exe.orig-04079d0` (оригинал), `C:\Temp\aionproxy.py + C:\Temp\py\python.exe (3.12.8 embed)`, `C:\Temp\proxylog.txt` (hex-дампы пакетов!), задачи schtasks: `AionGateTest` (гейт, сейчас оригинал), `AionProxy`, `AionNetMon`.
- Локально: `/tmp/AuthGateD.exe`, `.map`, `AuthGateD_p2/p3/p4/p5.exe`, `gate.asm` (полный дизасм 62 951 строк), `/tmp/aionproxy.py`.
- Перехваченные пакеты: welcome 194b ×2, клиент 34b, сервер 42b, логин 186b ×2 (hex в `C:\Temp\proxylog.txt`).

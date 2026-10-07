# Сервер авторизации AION 7.7 PTS — как работает (полное описание)

Документ по auth-слою: поток, процедуры БД (с сигнатурами), таблицы, где какая логика,
теория переписываемости и варианты решений. Все данные — из реального расследования
(перехват пакетов, дизасм AuthGateD, SQL-инспекция БД).

## 1. Схема авторизации

```
[Клиент]
   │ TCP 2106 (или через прокси-перехватчик)
   ▼
AuthGateD (гейт, exe 2013 г., L2-наследие)
   │  authd-канал: TCP 127.0.0.1:2110
   ▼
L2Authd (AuthD) ──┬──► AccountCacheServer (2220) — кэш аккаунтов в RAM
                  ├──► PAServer (10057) ×2 моста — PortalAuth-статусы
                  └──► SQL [AionAccounts] — ВСЯ auth-логика процедурами
                         user_account / user_auth (MD5) / ssn / userno / user_* / server / worldstatus

После логина L2Authd отдаёт клиенту список миров (таблица Server):
    id=1 _MAIN, ip=<внешний IP или LAN>, port=7777 (из worldport), region=2
Клиент подключается к Server64:7777 — дальше мир.
```

## 2. Хендшейк гейта (перехвачено, hex в C:\Temp\proxylog.txt)

Фрейминг: 2 байта LE = длина пакета (подтверждены: welcome 194=0xC2, клиент 34=0x22,
сервер 42=0x2A, логин 186=0xBA).

| Шаг | Куда | Размер | Содержимое |
|---|---|---|---|
| 1 | клиент ← гейт | 194 | welcome: session id + (скрытый/скремблированный RSA-1024 модуль + Blowfish-ключ сессии). ВСЕ байты меняются между сессиями (рандом scramble), маркеры aioncore (0xC621, 197635, 2097152) в открытом виде ОТСУТСТВУЮТ — формат 7.7 не как у aioncore 4.7.5 |
| 2 | клиент → гейт | 34 | 32 байта payload — RSA-шифрованный обмен (ключ сессии/рукопожатие) |
| 3 | клиент ← гейт | 42 | блочный шифр: видны повторяющиеся 8-байтные блоки (ECB-стиль), ключ сессионный |
| 4 | клиент → гейт | 186 | ЛОГИН (зашифрован) |
| 5 | гейт | — | проверка sessionId: `Session id mismatched.(RecvLogin) sessionId:0, m_iSessionId:N` при несовпадении → разрыв |

Пароли: в БД пароль = **binary(16) = MD5-хеш** (процедуры `ap_GPwd11(@account, @pwd binary(16))`).
Флаг клиента **-pwd16** включает именно этот режим (клиент шлёт MD5-хеш, а не сырой пароль/токен).
Флаг **-loginex** переключает протокол логина клиента (LoginEx) — у западных клиентов без
него включён портал-режим.

## 3. Дизасм проверки сессии (AuthGateD, VA → raw = VA − 0x400000)

```
0x4079d0: mov eax,[ecx+0xfc]   ; m_iSessionId (в объекте сессии гейта)
0x4079d6: mov edx,[esp+4]      ; sessionId из пакета
0x4079da: cmp edx,eax
0x4079dc: je 0x407a05          ; совпало → al=1, ret
0x4079de..4079fd: лог "[WARN] Session id mismatched.(%s)..." (строка UTF-16 @VA 0x42ce38, "(%s)" = имя пакета @VA 0x42c7e4)
0x407a00: xor al,al → ret 8    ; false
0x407a05: mov al,1 → ret 8     ; true
```

Вызовы проверки (7 мест = 7 обработчиков пакетов):
`0x4063d4` (RecvLogin-обработчик: перед ним push "RecvLogin"), `0x406678`, `0x4067d7`,
`0x406bc9`, `0x406d39`, `0x406e68`, `0x4070e7`.
После RecvLogin-проверки: `0x4063e1: call 0x407ac0` — парсер логина (account/password/…).

Счётчик m_iSessionId растёт 1,2,3,4… по сессиям (генератор: два писателя `[esi+0xfc]`
на VA 0x4041b8 — время+база, и VA 0x4076f3 — через call 0x408070). Клиенты западной
линейки шлют всегда 0 (сессию берут из портала, не из welcome).

## 4. Процедуры auth-БД (полный слепок с сигнатурами)

База: `AionAccounts`. Все auth-операции exe выполняют ПРОЦЕДУРАМИ:

### Регистрация
| Процедура | Сигнатура | Что делает |
|---|---|---|
| `agent_CreateAccount` | (@ggid uniqueidentifier, @account varchar(14), @password binary(16), @email varchar(50), @mobile varchar(20), @question1/2 varchar(255), @answer1/2 binary(32)) | полная регистрация (с GameGuard-id) |
| `ap_AutoReg` | (@account varchar(14)) | авто-регистрация по имени (то, что работает у нас: аккаунт создаётся при первом логине) |
| `web_CreateAccount` | (@nickname varchar(15), @account, @password binary(16), @email, @mobile, @question1/2, @answer1/2 binary(32)) | регистрация через веб-портал |
| `l2p_TempCreateAccount` | (@account, @ssn varchar(13)) | временная учётка (L2-наследие) |
| `Pr_acc_act` | — | активация аккаунта |

### Логин/проверки
| Процедура | Сигнатура | Что делает |
|---|---|---|
| `ap_GPwd11` | (@account, @pwd binary(16)) | сверка MD5-хеша пароля |
| `ap_GPwdWithFlag` | (@account, @pwd, @flag tinyint, @otpflag tinyint) | + флаги/OTP |
| `ap_GStat` / `ap_GStat11` | (@account, @uid, @payStat, @loginFlag, @warnFlag, @blockFlag, @blockFlag2, @subFlag, @lastworld, @block_end_date, @forbidden_servers binary(16)) | полный статус аккаунта (блокировки/подписка/мир) |
| `ap_GStatEtc` | (@account) | прочее состояние |
| `ap_LoginWithPoint` / `ap_LogoutWithPoint` | (@block_end_date, @last_login, @last_logout) | портал-поинтовый вход/выход |

### Состояние/логи
| Процедура | Сигнатура |
|---|---|
| `ap_SLog` | (@uid, @lastlogin, @lastlogout, @LastGame, @LastWorld, @LastIP) |
| `ap_SNewPwd` | (@account, @pwd binary(16), @encFlag tinyint) |
| `ap_SUserData` | (@account varchar(16), @possibleAccId int, @unkbin binary(16)) |
| `ap_SUserTime` / `ap_GUserTime` | (@useTime, @uid, @payStat, @loginTime) |
| `ap_SetIllegalLoginTrace` | (@gameAccountNo, @IP varchar(15), @traceTypeCode) |
| `ap_SetPasswordResetFlag` | (@gameAccountNo) |

### Серверы/CCU
| Процедура | Сигнатура |
|---|---|
| `ap_GetServers` | — (список миров для L2Authd) |
| `ap_SetServerStatus` | (@gameServerNo tinyint, @statusCode tinyint) |
| `ap_SetConcurrentUserStatistics` | (@serverNo smallint, @concurrentWorldUserCount, @concurrentUserLimit, @concurrentAuthUserCount, @concurrentAuthWaitCount) |
| `CCU_Stat` | — статистика онлайна |

### Персонажи/слоты
| Процедура | Сигнатура |
|---|---|
| `ap_GetGameAccountNo` | (@gameAccount varchar(16)) |
| `ap_GetAccountGameSlot` | (@accountId, @p2 datetime, @p3 datetime, @p4 varbinary(8)) |
| `ap_GetAccountMaxLevelCharacter` | (@unk1, @p2 tinyint, @p3..p5 int) |
| `ap_RegisterAccountGameCharacter` | (@unk1, @unk2, @unkCharId, @unk4, @unk5 datetime, @unk6, @unk7) |
| `ap_GetRestriction` / `ap_SetGameRestriction` | (@gameAccountNo) / (@uid, @flag tinyint) |
| `ap_GetSSN` | (@gameAccountNo, @ssn char(13)) |

### Веб-портал API (для внешнего портал-эмулятора!)
| Процедура | Что делает |
|---|---|
| `web_FindAccount(@account → @uid, @password binary(16), @email, @credits)` | ищет аккаунт: uid из user_account, password (MD5) из user_auth, email из ssn; credits=0. Возвращает 1/2/3 при недостающих данных |
| `web_CreateAccount` | см. регистрацию |

## 5. Таблицы auth-БД

| Таблица | Ключевые колонки | Роль |
|---|---|---|
| `user_account` | uid, account, pay_stat, login_flag, warn_flag, block_flag, block_flag2, last_login, last_logout, subscription_flag, last_game, last_world, last_ip, block_end_date, forbidden_servers binary(16) | аккаунты |
| `user_auth` | account, **password binary(16) = MD5** | пароли |
| `ssn` | ssn char(13), name, email, job, phone, mobile, reg_date, адреса, status_flag, master | профиль/SSN |
| `userno` | uid int | счётчик uid |
| `user_info` | account, create_date, ssn, status_flag, kind, ses_num, ip/mask_ip, code/ecode, bsoeday_time_start | инфо/сессии-статусы |
| `account_data` | id, name, password, access_level | L2-наследие (старая схема) |
| `savormix_user_portal` | uid, login_name, account, birthdate, sex, block_code, notice_code, status_code, block_reason_code, user_level, user_type_code, company_code, user_id binary(16) | портал-профили (эмулятор портала savormix; пустая) |
| `server` | id, name, ip, inner_ip, ageLimit, pk_flag, kind, port, region | список миров для клиентов |
| `worldstatus` | idx, server, status | статус миров |
| `user_count` | record_time, server_id, world_user, limit_user, auth_user, wait_user, dayofweek | CCU-история |
| `block_msg`, `block_reason_code`, `user_block`, `gm_illegal_login` | — | блокировки/следы |
| `item_code`, `user_pay`, `user_stat`, `user_time`, `user_data`, `banancheg` | — | прочее (не копали детально) |

## 6. Теория: где живёт логика и что переписываемо

| Слой | Где | Статус |
|---|---|---|
| Проверка пароля/создание аккаунта/блокировки/статусы | **SQL-процедуры** | полностью редактируемо (можно переписать свою логику: например, убрать блокировки/плату) |
| Сессии, парсинг пакетов, handshake, шифрование | **AuthGateD exe** | закрытый бинарь; патчится точечно (map/pdb есть), грубые патчи ломают authd-контур (7 вызовов проверки) и/или гейт тихо выходит в парсере логина (0x407ac0) |
| Список миров, статусы миров, CCU | SQL + L2Authd | редактируемо |
| Портал-сессии западных клиентов | внешний портал (CEF/4game/GG) | НЕ в БД кита (savormix_user_portal — только остатки API); нужен свой портал-эмулятор или клиент классической линейки |
| Мир/спавны/квесты | Server64 + ScriptDLL + NPCSvr + XML | часть — XML (редактируемо), часть — exe (Ghidra+PDB) |

Вывод: «дурская GUI-хрень» — нет; это полноценный связанный стек, где auth-аккаунтная
логика честно вынесена в SQL (переписываемо), транспорт — закрытый, но с символами для патчей.

## 7. Варианты решения логина (шансы)

1. **Флаги клиента `-loginex -pwd16`** (не были проверены!): pwd16 = клиент шлёт MD5 (сходится с БД), loginex = режим логина. Шанс: высокий, проверка минуты. Требуется полный флаг-набор из туториала кита.
2. **CN-клиент 7.7** (пара к киту; `-cc:5 -loginex -pwd16`): 70% — источники: CN one-click киты (sqybbs tid=3723), Baidu (pwd=d502), GDrive-зеркало, mmo-dev.
3. **Реверс парсера логина гейта** (0x407ac0 + 7 обработчиков): 50% — с map/pdb задача локализуется; грубые always-OK уже опробованы (p1..p5 — см. findings-log.md раздел 3).
4. **Портал-эмулятор** (web_* API): 20–30% — нужен реверс клиентских эндпоинтов (CEF/4game).
5. **Даунгрейд на 4.6-кит**: 90% (проверено сообществом), но потеря 7.7-контента.

## 8. Замечания по инфраструктуре (из практики)

- Запуск компонентов ПОД ЧУЖИМ ИМЕНЕМ/из Temp подвешивает процесс (SharedGuard/самопроверка пути) — только родной путь/имя.
- AuthGateD после потери authd-канала молчит и не переподключается (authReconnectInterval=0) — при рестартах L2Authd рестарти и гейт.
- authd-канал гейт-↔L2Authd живёт стабильно на оригинале; «всегда-OK» патчи проверки ломают его за ~2 мин (проверка используется и в authd-контуре).
- Скачок времени VM (NTP) может ронять таймеры сервисов — следить.
- Клиент через прокси видится сервером как 127.0.0.1 — IP-логика гейта/серверов слепа (GeoIP/блок-листы!) — для прод-прокси форвардить реальный IP (PROXY-протокол/расширение).

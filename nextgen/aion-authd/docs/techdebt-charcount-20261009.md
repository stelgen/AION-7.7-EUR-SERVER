# ТЕХ-ДОЛГ: «клиент RU 7.7 видит 0 персонажей на сервере» — 09.10

> Статус юзера: залогинился в игру успешно (fork-стенд, живой путь ориг).
> Клиент на экране выбора сервера ВСЕГДА показывает 0 персонажей.

## 1. Где счётчик по эталону (aion-germany 7.8 EU — AL-Login, SM_SERVER_LIST op=0x04)

```java
writeC(servers.size());
writeC(account.getLastServer());
for (server) {                       // per-server блок
    writeC(id); writeB(IP); writeD(port);
    writeC(0x00); writeC(0x01);      // age limit, pvp
    writeH(currentPlayers); writeH(maxPlayers);
    writeC(online?1:0); writeD(1); writeC(1);
}
writeH(maxId + 1); writeC(0x01);
writeB(new byte[49]);                // ← нулевая зона
for (i = 1..maxId) writeC(charactersCountOnServer.get(i) ?? 0);  // ← СЧЁТЧИК ЧАРОВ АККА
```

Источник счётчиков — GS-репорты в LS (`CM_GS_CHARACTER`, clientpackets/gameserver —
GS↔LS канал = наш 2104; в живом L2Authd-логе это события W→A type 35/3:
`uid + 01 + char_id + lev`).

## 2. Наш флоу (gate-prod.log 09:58, логин stelgen)

- Клиенту уходит **74Б = SM_SERVER_LIST**: `RAW G>C authd-pkt type=3 sid=4 len=74` —
  гейт строит его из authd type=3 (52Б) → EncryptSecondary: `[03][uid][token][16×0][2000][unk1][ЗОНА НУЛЕЙ][динамич dword][4×0]`.
- **Зона нулей в середине = зона счётчиков** (по Java-раскладке) — у нашего гейта она нулевая,
  у ориг AuthGateD, судя по всему, тоже → клиент всегда видит 0.
- 26Б type=4 = server-info (SAME ориг-vs-наш в fork) — счётчик НЕ здесь.

## 2.5 ✅ РАСШИФРОВКА 74Б (09.10, Blowfish ECB key2 — csum OK, LE-канон)

Сессия 09:58 (sid=4, stelgen): key2=`fcab045a81e5e53aa7f2854050f03bba` → payload 64Б:

```
off  0: 03              ← клиентский опкод SM_SERVER_LIST
off  1: f2030000        ← uid = 1010 (динамич)
off  5: f6373613        ← token (динамич)
off  9: 16×00           [9:25]
off 25: d0070000        ← 2000 (maxUsers)
off 29: 58c7d30b        ← unk1 (динамич)
off 33: 00 ×27          [33:60]  ← нулевая зона
off 60: 00 00 00 00     [60:64]  ← у ориг 06.10 здесь 1a6bc068 (динамич dword)
csum@64, pad@68 (EncryptSecondary-обёртка)
```

Сравнение с 52Б authd type=3: [uid][token][8×0][2000][unk1][28×0] → гейт ДОСТРАИВАЕТ 12Б
(опкод 03 + расширение нулевых зон) при сборке 74Б.

**Зона счётчиков = кандидаты [33:64] (нули)**; по Java-аналогии counts = конец пакета →
для 1 сервера = байт [63] (сейчас 0 → клиент видит 0). Динамич dword [60:64] у ориг
(06.10: 1a6bc068) НЕ похож на счётчик (LE = 0x68c66b1a — слишком большой) — вероятно
выравнивание/хвост-резидент.

## 3. ЭКСПЕРИМЕНТ РАЗВЁРНУТ (09.10, по «го» юзера)

- aion-gate: флаг `serverListCharCount` (коммит cd0fedf, `padLoginOK`): >0 → байт [63] 74Б = значение.
- VM: exe заменён (MD5 `4ac12728`), config-prod.yaml + `serverListCharCount: 7`, гейт рестартнут
  через op (AionGate, PID 5872, 2106 жив).
- ЖДЁМ перелогин юзера: на экране выбора сервера должно показать **7** вместо 0 → офсет [63] доказан.
- Откат: строка `serverListCharCount: 7` удалить/закомментировать + рестарт AionGate.
- След. шаг при успехе: реальный счётчик (число чаров акка из БД/2104-событий) вместо статики.

## 3.1 План закрытия (после R6)

1. Расшифровать 74Б-клиентский фрейм (key2 в gate-логе) → точные офсеты зоны счётчиков 7.7.
2. Источник counts: `user_data` по аккаунту (мир) / `ap_GetAccountGameSlot` (AionAccounts) /
   накопление из 2104-событий type 35/3 (по C1-семантике authd их агрегирует).
3. Заполнять зону в сборке 74Б (наш гейт) / передавать в type=3 payload (наш authd).
4. Сверка живым клиентом.

Не блокер R6 (косметика клиентского UI), но полезный паритетный бонус нашему authd.

## 5. ЭКСПЕРИМЕНТ №2 РАЗВЁРНУТ (09.10, ждём перелогин)

- Вердикт №1: байт [63] 74Б клиент ИГНОРИРУЕТ (колонка пустая) — 74Б = **SM_LOGIN_OK**
  (Packet Samurai Login_4.0: accountId/loginOk/?/?/1002/126282164/garbage47), не serverlist.
- Настоящий **SM_SERVER_LIST = наш type=4 (42Б wire)**; 26Б payload от authd разложился по XML:
  `[listSize=01][lastServer=01][id=01][ip][port d][age][pvp][cur h][max h][online=01][bits d]
  [brackets=02][countsSize h=01 00][autoConnect=01]` — и ОБРЫВ: **самих count-байтов НЕТ**
  (и у ориг!) → клиент ждёт 1 байт, не получает → колонка ПУСТАЯ.
- Правка: `serverListCharCount>0` → +1 байт в хвост type=4 (pt 28Б; **wire 42Б НЕ меняется**:
  roundup8(26)=roundup8(27)=32). Коммит `eaf52c9`, на VM MD5 `d5b9c708`, gate PID 1132.
- ОЖИДАНИЕ: перелогин юзера → колонка «Персонажи» = 7.
- При успехе: статику → реальный счётчик (user_data / ap_GetAccountGameSlot / 2104-события type 35/3).

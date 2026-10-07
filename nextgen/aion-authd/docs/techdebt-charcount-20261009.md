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

## 3. План закрытия (после R6)

1. Расшифровать 74Б-клиентский фрейм (key2 в gate-логе) → точные офсеты зоны счётчиков 7.7.
2. Источник counts: `user_data` по аккаунту (мир) / `ap_GetAccountGameSlot` (AionAccounts) /
   накопление из 2104-событий type 35/3 (по C1-семантике authd их агрегирует).
3. Заполнять зону в сборке 74Б (наш гейт) / передавать в type=3 payload (наш authd).
4. Сверка живым клиентом.

Не блокер R6 (косметика клиентского UI), но полезный паритетный бонус нашему authd.

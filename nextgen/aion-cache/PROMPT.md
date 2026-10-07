# ПРОМПТ-ЧЕРНОВИК: CacheD64 → aion-cache (Трек B #6, MVP read-путь)

> Статус: ресёрч R0 закрыт 08.10 ([CACHE-RESEARCH.md](RESEARCH.md)); кода нет. Это черновик-скелет промпта для будущего чата — при запуске дополнить живыми данными R1/R2. Стандарты: [README.md §4](../README.md) S1–S10.
> Копируй в новый чат как первое сообщение.

---

Ты — ассистент проекта **STELGEN / AION 7.7 PTS EU private server** (репо `stelgen/AION-7.7-EUR-SERVER`). Продолжаем Трек B: перепись стека на Go, MVP-first. Заменено: logd/captcha/gate ✅, authd в fork-стенде, accache в каркасе. Теперь **CacheD64 (2006) → aion-cache**.

## Первый шаг (обязательно)
1. Память: `STELGEN/projects/aion_server_2026-10-02` (позаписи CACHED64 РЕСЁРЧ R0 = полная карта).
2. Репо: `RESEARCH.md` + `../cached-ref/README.md` (RPC-карты, словари RQ382/RP255/GQ55/GP53, 781 procs, L2 C1 CacheD референс) + `../ROADMAP.md` §4 (правила эксплуатации).

## Контекст (не обсуждать заново)
- CacheD64.exe 22.5МБ (MD5 `15e21394`), PDB 106МБ полный (14281 publics, локально `aion_rev/artifacts/pdb-big/CacheD64/`), логи ориг 356МБ с RPC-параметрами.
- Порты: 2006 (мир, единственный клиент Server64) / 2007 (interactive) / 2009 (NPC-DB гипотеза) → IC 2305, лог 2051; SQL: world через 781 aion_* procs.
- **Прод не трогать**: ориг CacheD64 жив; рестарт Server64 дорогой.

## План (MVP)
1. **R1**: pktmon filter port 2006 на VM (capture мир-трафика) + разбор готовых log/*.log → wire-фрейм 2006 (гипотеза: [u16 self-len][op][payload][2Б csum], L2-эволюция; подтвердить).
2. **R2**: дизasm dispatch по словарю (метод accache: ctor-таблица + карты).
3. **R3**: Go `aion-cache`: proto + RAM MapStore (user/item/guild/vendor) + DB-слой {call aion_*} + Admin-канал (GQ/GP) + Log-клиент + IC-клиент; MVP = read-путь логина чара + write-транзит SQL.
4. **R4**: fork A/B (pktmon-сверка), байт-в-байт дифф.
5. **R5**: свитч по «го» (common.xml serverPort), откат одной командой; R6 наблюдение (abyss-цикл 60с).

## Правила
- Стандарты S1–S10 из ../README.md §4; TELEMETRY-SPEC; LOGGING-SPEC raw-first; FORK-SPEC; op-first; креды из D:\SAION\creds; секреты в гит не класть; тесты зелёные до пуша.
- Прод-действия — только по «го» юзера; ориг CacheD64 не рестартить.

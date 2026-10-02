# Луперы: shop agent 10100 / ChannelChat 10254 / petition 2107

**Статус: ИССЛЕДОВАНО 03.10.2026. Сервисов в ките НЕТ. Тишина в err — они event-driven.**

## Инвентаризация (что проверено)
- В ките НЕТ exe: чата/Maxx, ShopAgent, Petition-сервера (grep по всему D:\AION_LIVE_SERVER).
- **NPRelay64** (запускался тестом, ~206 МБ): NP/Warehouse-релей (NPRelayToMain, Ncoin),
  слушает ТОЛЬКО ничего — исходящий релей. НЕ закрывает наши луперы. Тестовая задача AionNPRelay DISABLE.
- **01-PAServer7.7.exe** (запускался тестом, 6 МБ): слушает только 127.0.0.1:10057 (portal-auth бэкенд).
  НЕ закрывает луперы. Задача AionPA DISABLE.
- ICServer (Interchange 2005/2305) — работает, к луперам отношения не имеет.

## Замеры роста логов (после чистки 03.10)
- Server64 err (`MainServer\log\2020-06-04.err` — RunAsDate-дата!): лупер ChannelChat ~12с ≈ 1–2 МБ/день.
- CacheD err: ~120 МБ ОДНОРАЗОВО при загрузке = `Warning while loading Strings DB : unexpted id for NNNNNN`
  (даталоад-варнинги, ~570k строк). НЕ постоянный поток. Чистится стоп-батником.
- Итог: постоянный рост мал; при каждом STOP сервера *.err удаляются (шаг [10/10] в AION-STOP-SERVER.bat).

## Варианты тишины (когда надоест)
1. **Ghidra silence-патч Server64** (рекомендуемый путь): в Server64.exe найти строки
   `Can't connect to ChannelChat server at` / `shop agent server` / `petition server` (.rdata) →
   xref → NOP-нуть условный переход перед retry-коннектом (или ветку логирования). Тест в копии MainServer_backup.
   PDB есть — символы помогут. Одна сессия Ghidra закроет все 3 лупера.
2. **Конфиг-эксперимент (дёшево, реверсивно)**: `MainServer\common.xml`
   `<disablePetitionFrom>0</disablePetitionFrom>` / `<disablePetitionTo>0</disablePetitionTo>` —
   похоже на «часы отключения петиции»; гипотеза: 0..24 = петиция выключена всегда → лупер 2107 стихнет.
   Проверка = рестарт Server64. НЕ применять до подтверждения семантики (строки-подсказки в Server64.pdb).
3. **Игнор**: луперы некритичны (chat в мире работает без внешнего ChannelChat — S2S нужен только
   для мульти-сервера; магазин PTS не нужен; петиций нет).

## Артефакты тестов
- NPRelayServer: config.xml (countryCode=2, dataCenter=1) + common.xml скопированы — рабочая связка,
  если понадобится revived (schtasks AionNPRelay, батник C:\Temp\nprelay.bat).
- PA: батник C:\Temp\pa.bat, задача AionPA.
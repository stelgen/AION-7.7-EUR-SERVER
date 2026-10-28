# PROMPT — aion-cached (копипаст для нового чата)

> Регламент: [../WORKFLOW.md](../WORKFLOW.md) — память → README хаба → README компонента → ROADMAP → этот промпт. Первое сообщение = «ГДЕ МЫ» + пульс. Прод-действия (старт мира, свитч) — только по «го» юзера.

```
WORKFLOW: cached

ГДЕ МЫ [aion-cached R5 🔴 ~90%]: R4-ФИНАЛ PASS + R4.6 мир-init + R4.6.2 W30 + DB-генератор
(O1/O109/O14, любой чар). Свитч-тест #5 (28.10) = FAIL: v18 на 2006/2007/2009 — Server64+
NPCSvr подключились к нашему, NPCSvr прошёл полный World::Load + IOThread-init, но «NPC
Server Started» НЕ напечатан (зависает ПОСЛЕ сетевой фазы — сужение гипотезы (б), см.
ROADMAP §R5 #5); Server64 висел в autoShutdown-зависании → откат, мир 8/8 на копии-2016
восстановлен 07:18 VM. Лог v18 = событийный (VERSION/поллы/tl_ молча) — raw-картины нет.
МИР 8/8 ЖИВ на копии-2016 (cd2016+cdmirror, тень v18 на 2018/2019/2020).

ШАГИ:
0. Прочитать README.md (канон) + ROADMAP.md (§R5 #6 вердикт v19 RAW) + память aion-cached.
1. Статус VM (aionstatus): мир 8/8? тень v19 на 2018-2020? (не трогать мир без «го»).
2. R5 #7 ОФФЛАЙН (без VM): разложить capture-окна по коннектам — фингерпринты
   C282=Server64-полл, C201=NPCSvr-handshake — в worldinit-20261023.txt +
   cdmirror-20261024-worldstart.log; найти ПЕРВЫЙ O-кадр Server64-коннекта (CacheD-пуш
   при accept Server64, кандидат O16 RP_VERSION 27Б) — гипотеза замка «Server64 ждёт
   Cache Version ⟷ NPCSvr ждёт Server64:2002» (§R5 #6).
3. v20 = пуш при accept Server64 (по найденному канону) + проверка NPCSvr-accept-пуша
   (C201→O28 в NpcSrvReplay есть) → vet/test → деплой scp (exe лочится).
4. Свитч-тест повтор (по «го»): bak-канон конфига 2006/2007/2009 → пара → RAW-лог
   (v19 уже пишет) → критерий «NPC Server Started» + 8/8 + ЛОГИН юзера. Откат = bak-r5c
   + cdmirror/cd2016 + пара (канон в ROADMAP #5/#6).
ГОТЧИ: exe лочится (стоп-пуш-старт), accstop убивает прод-accache (только op),
aionput exe >5МБ = scp, body=frame[2:] ⇒ payload@body[5], X LE as-is (не реверс),
restart_pair блокируется в LOADING-окне — обход руками stop main/start npc/start main,
дубли процессов проверять после повторных стартов (28.10: LogServer64+ICServer дубли).
```

Артефакты для старта: [README.md](README.md) · [ROADMAP.md](ROADMAP.md) ·
[../cached-ref/logs-hourly/worldinit-20261023.txt](../cached-ref/logs-hourly/worldinit-20261023.txt) ·
[../cached-ref/logs-hourly/ab-r4-verdict-20261023-live-t39.md](../cached-ref/logs-hourly/ab-r4-verdict-20261023-live-t39.md)

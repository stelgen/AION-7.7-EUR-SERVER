# PROMPT — aion-cached (копипаст для нового чата)

> Регламент: [../WORKFLOW.md](../WORKFLOW.md) — память → README хаба → README компонента → ROADMAP → этот промпт. Первое сообщение = «ГДЕ МЫ» + пульс. Прод-действия (старт мира, свитч) — только по «го» юзера.

```
WORKFLOW: cached

ГДЕ МЫ [aion-cached R5 🔴 ~90%]: R4-ФИНАЛ PASS (offline + живой T39-acks 18/0/12),
R4.6 мир-init Server64 ✅ байт-в-байт, R4.6.2 NPCSvr W30 ✅ (228 кадров), DB-генератор
Этапы 0,1,3,5 ✅ (O1/O109/O14 из WorldDB, любой чар). Свитч #4 (23.10): v16c c18a371c
встал на 2006/2007/2009 ✅, но Server64 при рестарте пары прислал ТОЛЬКО полл op=282
(ноль мир-init 288/11/14) → мир 0/8 → откат по канону. Мир сейчас 0/8 (юзер пару погасил).
Capture мир-init собран: cached-ref/logs-hourly/worldinit-20261023.txt (2828 кадров,
103 уникальных O-cmds) + chain_worldstartup_20261023.bin.

ШАГИ:
0. Прочитать README.md (канон) + ROADMAP.md (теории/леджер) + память aion-cached.
1. Статус VM через op (aionstatus): мир 0/8? cd2016/cdmirror/cachedshadow живы?
2. По «го» юзера: restart_pair до 8/8 (прогрев на копии-2016, юзер может играть).
3. R5 #5: эмиттеры мир-init из worldinit-20261023.txt по образцу world_flow_db.go
   (топ O-cmds: 61×262, 187×151, 109×30, 18×40, 124×27 — большинство в opcodes.go).
4. Гипотезы #4-фейла по порядку: (а) ответ на полл 282 (наша тишина) — реализовать;
   (б) NPCSvr падает до сетевой фазы; (в) чистый прогон после (а).
5. Свитч-тест v16c (по «го»): наш на 2006/2007/2009 → рестарт пары → критерий
   «NPC Server Started» + мир 8/8 → наблюдение 24ч. Откат = bak-конфиг + cdmirror/cd2016
   + пара (канон в README §Деплой).
ГОТЧИ: exe лочится (стоп-пуш-старт), accstop убивает прод-accache (только op),
aionput exe >5МБ = scp, body=frame[2:] ⇒ payload@body[5], X LE as-is (не реверс).
```

Артефакты для старта: [README.md](README.md) · [ROADMAP.md](ROADMAP.md) ·
[../cached-ref/logs-hourly/worldinit-20261023.txt](../cached-ref/logs-hourly/worldinit-20261023.txt) ·
[../cached-ref/logs-hourly/ab-r4-verdict-20261023-live-t39.md](../cached-ref/logs-hourly/ab-r4-verdict-20261023-live-t39.md)

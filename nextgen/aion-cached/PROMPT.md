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
0. Прочитать README.md (канон) + ROADMAP.md (§R5 #8 вердикт) + память aion-cached.
1. Статус VM (aionstatus): мир 8/8? тень v20 на 2018-2020? (не трогать мир без «го»).
2. R5 #9 ГЛАВНОЕ: PDB publics Server64 (VM, manifest-pdb-big.md) → имена функций
   кластера 0x4F970E-0x4FA100 (wait-функции «Waiting for …», xref insnVA 0x4F9A02,
   печать call 0x4f51f0 «Waiting for %s») + гейт-глобалов 0x2A703B4/0x5EAFA60 →
   ТОЧНОЕ условие выхода из «Waiting for Cache Version».
3. Альтернативный живой тест (по «го»): логин юзера на копии-2016 при RAW-тени
   (v20 уже пишет) → сверить Server64-init-кадры с #6.
4. v21-кандидаты: ответ на первый кадр Server64 «ack+version» пачкой (O196-стиль),
   или эмуляция IC-коннектов этапа (Connecting to Interchange master+db×7 — Server64
   их не начинает). Откат = bak-r5d + cdmirror/cd2016 + пара (отработан 3 раза).
5. ГОТЧИ: exe лочится (стоп-пуш-старт), accstop убивает прод-accache (только op),
   NPC-schtasks может не подняться с первого раза — старт повторять, дубли процессов
   проверять после рестартов, op-детект тяжелых врёт — верить netstat+tasklist.
ГОТЧИ: exe лочится (стоп-пуш-старт), accstop убивает прод-accache (только op),
aionput exe >5МБ = scp, body=frame[2:] ⇒ payload@body[5], X LE as-is (не реверс),
restart_pair блокируется в LOADING-окне — обход руками stop main/start npc/start main,
дубли процессов проверять после повторных стартов (28.10: LogServer64+ICServer дубли).
```

Артефакты для старта: [README.md](README.md) · [ROADMAP.md](ROADMAP.md) ·
[../cached-ref/logs-hourly/worldinit-20261023.txt](../cached-ref/logs-hourly/worldinit-20261023.txt) ·
[../cached-ref/logs-hourly/ab-r4-verdict-20261023-live-t39.md](../cached-ref/logs-hourly/ab-r4-verdict-20261023-live-t39.md)

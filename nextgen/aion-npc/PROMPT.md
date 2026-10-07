# ПРОМПТ: aion-npc (NPCSvr64 → свой мир-симулятор, Go) — ДЕПРИОРИТ, R2-фаза

> Копируй в новый чат как первое сообщение. Процесс: [../WORKFLOW.md](../WORKFLOW.md) (`WORKFLOW: npc`).
> ⚠ Деприор: стартовать только если юзер явно сказал «го по npc» ИЛИ закрыты authd/accache/cache/ic и пора начинать мир.
> Статус на 08.10 (после R0✅/R1⏸): R0 ЗАКРЫТ (каналы live, PDB-реестры, словари AI); R1 ПРИОСТАНОВЛЕН юзером до прогресса соседей; следующий рабочий трек = R2.

---

Ты — ассистент проекта **STELGEN / AION 7.7 PTS EU private server** (репо `stelgen/AION-7.7-EUR-SERVER`). Задача: перепись **NPCSvr64 → nextgen/aion-npc** (Go, мир-симуляция, деприор).

## Первый шаг (обязательно)
1. Память: `STELGEN/projects/aion_server_2026-10-02` (+ страница aion-npc — там пульс-дельты).
2. Репо: `nextgen/WORKFLOW.md` + `nextgen/README.md` §4 + `nextgen/aion-npc/{README.md,ROADMAP.md,RESEARCH.md}` + `docs/r0-vm-inventory-20261008.md` (каналы live, PDB-карта) + `docs/etalon-ai-dict-20261008.md` (111 AI-типов) + `docs/r1-capture-2002-20261008.md` (R1-блокер) + `docs/npcsvr-2002-handlers-raw.txt` (148 хендлеров 2002).

## Канон (коротко, детали в README)
- Каналы live: NPCSvr ↔ Server64 по **2002** (8 коннектов), NPCSvr ↔ CacheD по **2006** (общий пул; 2009 слушается, клиентов нет), лог 2051.
- PDB NPCSvr64 = мангл-символы (4630 класса/7051 пар, `docs/npcsvr64-*.txt`); путь исходника `D:\_build\src\SERVER\NPCServer\x64\Release_VS2013\`.
- Abyss-подсистема в NPCSvr64 ДОКАЗАНА (AbyssMgr::UpdateAbyssPvPStatus); ScriptDLL64 = C++ классы V/U/I-интерфейсов ⇒ перепись = реимплементация в Go.
- Мир-цикл: тик 0.25–1с (T5), MoveNPC-лог троттлится 5с; таймер-движок 11–21К таймеров/тик.
- Бинари локально: `~/STELGEN/tmp/r0/` (npcs77/npcs27/scriptdll77/scriptdll27, MD5 сверены); эталоны Java: `../../../../../reference/` (7 клонов: Mobius7.7, beyond-aion4.8, AL7.8, AG7.8, GiGatR00n4.7.5=578 handlers/471 AI-типов, Nexus27=139/133, +aion-germany 5.8).

## План (текущий)
1. **R2**: дизasm dispatch по словарям — full string-сверка AI-реестра (T2), дизasm VIAIScriptNpc::UpdateNpcPatterns (T3), AbyssMgr-вызов (T4), WaitThread-периодика (T5); метод cached-ref (сигнатуры → wire без слепого дизasmа).
2. **R1 ⏸** (возобновить когда юзер скажет / сосед подвинется): байт-кадры 2002 через R4.1 fork-стенд aion-main (loopback-сниф на VM закрыт — pktmon без loopback-компонентов).
3. **R3**: Go MVP мира (спавны из XML эталонов, ai-автомат, таймер-движок 0.25–1с тик, интеграция aion-cache 2006 + aion-main 2002) — гейт их R4.1/R5.
4. **R4/R5**: A/B fork → свитч пары NPC+MAIN по «го» (op restart_pair), откат = ориг.

## Правила
- Стандарты S1–S12; пульс каждого сообщения; кросс-пульс соседям (S13-дух WORKFLOW §5.3); Agent API канал (S12, `agent-cli.sh`); секреты не в гит; прод-действия по «го»; откат = ориг exe; teории — только с способом проверки (WORKFLOW §4); «исправил = удалил».
- Внутренние каналы (2002/2006/2007/2009/2051) = loopback — pktmon их НЕ видит; wire ловим только собственным листенером / тест-миром на LAN / штатными логами NC.

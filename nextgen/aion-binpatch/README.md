# aion-binpatch — точечные Ghidra-патчи мира (Server64/NPCSvr64/ScriptDLL64)

> ⬜ **ТАКТИЧЕСКИЙ ТРЕК ПАТЧЕЙ** поверх стратегической переписи мира: фулл-сервер переписываем ВСЕ бескомпромиссно ([../aion-npc/](../aion-npc/), [../aion-main/](../aion-main/) — деприор), а пока мир жив на ориг-бинаре — точечные Ghidra-патчи методом #180 (PDB-джекпот делает их реальными).
> Запуск: `WORKFLOW: binpatch`. Стандарты: [../README.md](../README.md) §4.

## 🧭 Граф кода (S13)

- Код компонента навигируется AST-графом `graphify`, НЕ чтением всего пакета: `query`/`explain`/`path`/`god-nodes` — правила и команды: [../WORKFLOW.md §6.1](../WORKFLOW.md).
- ПЕРЕД правкой символа — `graphify affected <имя>` (кто зависит = что сломается); ПОСЛЕ правки — `graphify update .` в этой папке; `graphify-out/` = кэш (в гит не кладётся).
- Граф = навигатор, не арбитр: семантика протокола = канон этого README + live capture.


## 📊 Статус и фазы

| Патч | Статус | Артефакты |
|---|---|---|
| **#180** date-bypass (RunAsDate-зависимость) | ✅ УЖЕ В БИНАРЕ (доказано SHA256); RunAsDate = страховка | root README «Уже в binary» |
| **#108** матчмейкер арен | 🟢 **ПАТЧ-АРТЕФАКТ ГОТОВ 22.10**: AuthSocket-гейт найден через конфиг-ветки (g_isEventSvr → глобал 0x142bc9158 → маска 0x142bc88c8/бит 0x2000 → 5 тестов; гейт @0x140302173 test→je→IsEventMember→jne→REJECT); патч 6 байт @0x30157D `0f8495000000→e99600000090` (je→jmp: reject недостижим, логин открыт, логика не тронута); копия MD5 `ec3b6310`; **стенд-тест = финальный гейт** | [docs/recon-108-111-20261022.md](docs/recon-108-111-20261022.md) |
| **#111** манастоны | ❄️ **FROZEN 22.10** — [ISSUE](docs/ISSUE-111-manastones-FROZEN.md) `заморожено-до-aion-main-MVP`: вердикт + карта реверса + инструменты зафиксированы (тег `freeze/111-manastones-until-aion-main-mvp`); unfreeze = aion-main MVP или решение юзера на семантический реверс | [docs/ISSUE-111-manastones-FROZEN.md](docs/ISSUE-111-manastones-FROZEN.md) |
| **Ghidra-silence** луперов 10100/10254/2107 (NOP условного перехода у «Can't connect» строк — ОДИН проход закроет все три) | 📋 исследовано | fixes-pending/loops-shopagent-channelchat-petition/ |
| Порог CheckIOThreadDeadlock / authorization-time (уйти от RunAsDate окончательно) | ⬜ идея PLAN §6.1 | — |

## 📟 Канон

| Факт | Источник |
|---|---|
| Метод: Ghidra+PDB → локализовать → байтовый патч в КОПИИ → стенд MainServer_backup_20261002_212213 → тест → прод только с бекапом `Server64.exe.etalon` | fixes-pending README |
| Server64.pdb 284МБ на VM; дизasm-инструменты: objdump + tools/analysis/pdbpub.py (метод logd) | память |
| Смерть Server64 каскадно убивает NPCSvr (graceful); пара рестартится только вместе (op restart_pair) | live |
| NPCSvr грузится 10–15 мин; утечка Abyss ~600k блоков/сессия → ночной рестарт пары (op Phase 1.5) | PLAN §2 |

## 🚧 Блокеры

- Работы стендовые (не трогаем прод): приоритет низший против переписей; стартовать только по «го».

## ⏭️ Следующий шаг

`WORKFLOW: binpatch` → Ghidra-проект Server64.pdb+exe → поиск Can't connect-строк (silence, дешёвый) → JZ→JNZ матчмейкера (стенд) — по порядку юзера.

## 📦 Артефакты

| Что | Где |
|---|---|
| Промпт | [PROMPT.md](PROMPT.md) |
| Планы патчей | ../fixes-pending/{108-matchmaker-arenas,111-mainserver64-manastones,loops-shopagent-channelchat-petition}/ (в корне репо) |
| PDB/бинари | VM `D:\AION_LIVE_SERVER\`; локально `aion_rev/artifacts/` |
| Креды | VM `D:\SAION\creds\` |

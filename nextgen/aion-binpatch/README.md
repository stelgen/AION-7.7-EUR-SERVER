# aion-binpatch — точечные Ghidra-патчи мира (Server64/NPCSvr64/ScriptDLL64)

> ⬜ **ТАКТИЧЕСКИЙ ТРЕК ПАТЧЕЙ** поверх стратегической переписи мира: фулл-сервер переписываем ВСЕ бескомпромиссно ([../aion-npc/](../aion-npc/), [../aion-main/](../aion-main/) — деприор), а пока мир жив на ориг-бинаре — точечные Ghidra-патчи методом #180 (PDB-джекпот делает их реальными).
> Запуск: `WORKFLOW: binpatch`. Стандарты: [../README.md](../README.md) §4.

## 📊 Статус и фазы

| Патч | Статус | Артефакты |
|---|---|---|
| **#180** date-bypass (RunAsDate-зависимость) | ✅ УЖЕ В БИНАРЕ (доказано SHA256); RunAsDate = страховка | root README «Уже в binary» |
| **#108** матчмейкер арен (IsEventServer=true + JZ→JNZ; НЕ трогать логин-ветку) | 🔍 **РЕКОН ✅ 22.10** — якоря найдены ([docs/recon-108-111-20261022.md](docs/recon-108-111-20261022.md): глобал 0x142bc9158, маска 0x142bc88c8/бит 0x2000, геттер 0x1400b4d10, CEventServerMgr в PDB); ветка JZ→JNZ = Ghidra-сессия + стенд-тест | fixes-pending/108-matchmaker-arenas/ + docs/recon |
| **#111** манастоны | 🧊 **БЛОКЕР 22.10**: чужой exe 31.9МБ ≠ наш 45.5МБ (разные сборки) — байт-дифф невалиден; путь = Ghidra-дифф по функциям (наш PDB + locate-by-prologue в чужом) ([docs/recon-108-111-20261022.md](docs/recon-108-111-20261022.md)); ENIGMA НЕ деплоить | fixes-pending/111-mainserver64-manastones/ + docs/recon |
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

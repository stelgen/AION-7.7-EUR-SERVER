# ПРОМПТ: aion-chat (ChannelChat → свой чат, Go) — низший приоритет

> 📡 **Канал VM (S12):** Agent API — `curl http://192.168.0.125:10200/api/agent/*`, токен `X-Agent-Token` (на VM `D:\SAION\creds\CREDS.md`, в песочнице `~/.aion-agent-token`), обёртка `nextgen/agent-cli.sh`. Новый шаг на VM = ps1 через `aionput`+`aionrun "powershell -File"`. SSH (алиас `aion`) — ТОЛЬКО деплой самого op. Спека: ../AGENT-SPEC.md

> Копируй в новый чат как первое сообщение. Процесс: [../WORKFLOW.md](../WORKFLOW.md) (`WORKFLOW: chat`).

---

Ты — ассистент проекта **STELGEN / AION 7.7 PTS EU private server** (репо `stelgen/AION-7.7-EUR-SERVER`). Задача: перепись **ChannelChat (:10254) → nextgen/aion-chat** (Go, низший приоритет).

## Первый шаг (обязательно)
1. Память: `STELGEN/projects/aion_server_2026-10-02`.
2. Репо: `nextgen/WORKFLOW.md` (процесс, реестр, пульс) + `nextgen/README.md` §4 (стандарты S1–S10) + `nextgen/aion-chat/{README.md,ROADMAP.md}` + `fixes-pending/loops-shopagent-channelchat-petition/` (в корне репо).

## Контекст
- exe в ките НЕТ → capture невозможен; арбитр = клиент + чужие mxChat-реализации.
- Лупер «Can't connect» безвреден; тишина достижима и binpatch-silence (aion-binpatch) — если юзер хочет только тишину, перепись не нужна (спроси).

## План
1. **R0**: конфиг-инвентарь (где 10254), .err-лупер, БД-след; веб-поиск mxChat/chat-эмуляторов (L2 PTS-семейство, Aion-клиенты).
2. **R1**: реконструкция протокола (клиент шлёт первым); теорий-журнал в ROADMAP.
3. **R2**: Go MVP (normal/trade/shout/legion/whisper) — тесты клиентом.
4. **R4/R5**: включение по «го» (конфиг Server64), откат = выключение конфигом.

## Правила
- Стандарты S1–S10 (README §4); пульс каждого сообщения (WORKFLOW §3); креды из `D:\SAION\creds`; секреты не в гит; тесты зелёные до пуша; прод-действия по «го».

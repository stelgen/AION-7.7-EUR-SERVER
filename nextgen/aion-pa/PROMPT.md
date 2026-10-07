# ПРОМПТ: aion-pa (PortalAuth → адаптация pae) — деприор, НЕ писать с нуля

> Копируй в новый чат как первое сообщение. Процесс: [../WORKFLOW.md](../WORKFLOW.md) (`WORKFLOW: pa`).

---

Ты — ассистент проекта **STELGEN / AION 7.7 PTS EU private server** (репо `stelgen/AION-7.7-EUR-SERVER`). Задача: довести **aion-pa** до рабочего эмулятора PA на базе чужих сурсов (НЕ писать с нуля).

## Первый шаг (обязательно)
1. Память: `STELGEN/projects/aion_server_2026-10-02` (+ страница pa-server).
2. Репо: `nextgen/WORKFLOW.md` + `nextgen/README.md` §4 + `nextgen/aion-pa/{README.md,ROADMAP.md,docs/pa-binaries-research-20261007.md,docs/pa-research-20261006.md}` + `nextgen/aion-op/docs/config-inventory-0410.md` (GUID'ы) + `nextgen/aion-authd/docs/auth-server-internals.md` (схема auth-БД).

## Контекст (не обсуждать заново)
- Ориг PA жив и ОБЯЗАТЕЛЕН (старт ДО authd; без него SYSTEM_ERROR(20)).
- pae (portal-auth-emulator, RZ 1205208 #307) = доказанно рабочий; Docker 4.12; env PAE_CONNECTION_STRING + PAE_CLIENT_APP_ID; колонка enc_flag→new_pwd_flag.
- Сурсы-приоритет: сначала чужие сурсы (pae + ориг-PA из L2_LIVE_CSERVER_SVN), потом свой дизасм — только если всё исчерпано.

## План
1. **R1**: скачать аттачи (креды из `D:\SAION\creds\CREDS.md`; cookies rz-cookies.txt) → `D:\SAION\downloads\pa\` → разобрать .py/SQL → wire payStat.
2. **R2**: GUID'ы из config-inventory-0410.md.
3. **R3**: адаптация процы под AionAccounts.
4. **R4**: стенд (соседний порт/fork) — арбитр = ориг PA (логины с оригом живы).
5. **R5**: свитч по «го»; откат = ориг exe.

## Правила
- Стандарты S1–S12 (README §4); пульс (WORKFLOW §3); Agent API канал (S12); секреты не в гит; прод-действия по «го»; откат всегда готов.
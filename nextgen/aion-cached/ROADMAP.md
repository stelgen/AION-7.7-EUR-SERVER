# ROADMAP — aion-cached (Go-фреймворк CacheD64)

> Журнал теорий / леджер решений для трека reverse-engineering CacheD64.
> Фазы: R1 каркас → R2 дизasm → R3 Go MVP → R4 A/B → R5 свитч.

---

## Фазы

| Фаза | Статус | Дата | Что сделано |
|---|---|---|---|
| R1 — Go-каркас wire-фреймворка | ✅ | 10.10.2026 | wire.go (Reader/Encoder, канон CServerSocket.cpp), handlers.go (map[uint16]Handler), ship.go, server.go (TCP 2006/2007/2009), конфиг, 9 тестов зелёные |
| R2 — дизasm DummyCrypt | ⏳ | — | Требуется точная форма XOR-стрима из generated/Cached.c + PDB publics |
| R3 — Go MVP (реальные хандлеры) | ⬜ | — | Замена stub на рабочие RQ/RP, GQ/GP, NPC-DB |
| R4 — A/B тест с Live CacheD64 | ⬜ | — | Захват и сравнение пакетов (probe/forkprobe/fake-клиенты) |
| R5 — свитч (замена бинарника) | ⬜ | — | Полная замена CacheD64.exe на проде |

---

## Журнал теорий / леджер решений

| # | Гипотеза | Способ проверки | Статус |
|---|---|---|---|
| 1 | Wire-канон L2 C1 CacheD применим к Aion CacheD64 (2006/2007/2009 — 1-в-1 каналы) | Сопоставление packetNNN_*.h серверных листенеров L2 C1 с нашими каналами | ✅ канон (R1 каркас построен) |
| 2 | DummyCrypt = `payload[i] ^= byte((m_key + uint32(i)) & 0xFF)` | Дизasm из `generated/Cached.c` или capture-сверка | ⏳ R2 |
| 3 | Rolling key катится при браке csum (m_key += m_packetSize до проверки) | Чтение CServerSocket.cpp: m_key += m_packetSize идёт до csum-check | ✅ канон |
| 4 | Outbound крипта отсутствует (L2 C1 Send НЕ шифрует ответы) | Чтение CServerSocket::Send — нет вызова DummyCrypt.Encrypt | ✅ канон |
| 5 | Opcode на проводе — 1 байт (L2 C1), Go-тип Frame.Opcode — uint16 | Соответствие L2 C1 wire-формату | ✅ канон |
| 6 | map[uint16]Handler для future-proofing (Aion CacheD64 может использовать 2-байтные опкоды) | Требование ТЗ R1 | ✅ принято |
| 7 | Frame.Opcode = 1 байт на проводе, но Go-тип uint16 | Не наруушает wire-канон, расширяет Go API | ✅ принято |
| 8 | Конфиг = key=value текст (не yaml/json), R1 минимализм | ТЗ: «Минимум для R1» | ✅ принято |

---

## Следующий шаг

R2 — дизasm DummyCrypt из `generated/Cached.c` / PDB publics, сверка с формулой `payload[i] ^= byte((m_key + uint32(i)) & 0xFF)`.

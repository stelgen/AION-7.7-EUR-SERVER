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

## R2 ЗАКРЫТ (21.10): DummyCrypt = NO-OP, wire = plaintext ✅

**Два независимых доказательства:**

1. **L2 C1 сурсы (MasterToma)** — `L2Core/src/crypt/DummyCrypt.cpp` (символ L2CacheD 0x00435050):
   тело `Decrypt(uint8_t* data, uint64_t key, int length)` **ПОЛНОСТЬЮ ПУСТОЕ**
   (`UNUSED(data); UNUSED(key); UNUSED(length);`) — «Dummy» в буквальном смысле.

2. **Aion 7.7 CacheD64.exe (дизasm по .map, exe+pdb локально)** — крипто-фэмили Utils.obj:
   | Символ | VA | |
   |---|---|---|
   | ?Decrypt@@YAXPEAEPEBKH@Z | 0x1401b0360 | Decrypt(uint8_t*, const uint32_t* key, int) |
   | ?Encrypt@@YAXPEAEPEBKH@Z | 0x1401b03e0 | |
   | ?Encrypt@@YAXPEAEPEBEPEBKH@Z | 0x1401b0440 | Encrypt(uint8_t*, const uint8_t*, const uint32_t*, int) |
   | ?Decrypt2@@YAXPEAEPEBKH@Z | 0x1401b04c0 | |
   | ?Encrypt2@@YAXPEAEPEBKH@Z | 0x1401b0570 | |

   **Ноль ссылок** на всю семью в бинаре (сканы: E8 rel32, E9 thunks, FF15 call [rip],
   lea rip-rel, ptr VA в .rdata+полный .data 19МБ) = **мёртвый код, не в wire-пути**.

**Бонус-факт (алгоритм мёртвой семьи, на случай реанимации в будущих версиях):**
дизasm Decrypt показал XOR-цепочку: `data[i] = staticTable[i&0x3F] ^ key[i&0x7] ^ data[i] ^ data[i-1]`
(staticTable 64Б ASCII-алфавит "nKO/WctQ0AVLbpzf...?3iI9" @0x1402dccb0; i=0 — только key[0]).
Связь с m_key (scalar в L2) НЕ найдена — Aion Decrypt берёт key-УКАЗАТЕЛЬ.

**Вывод для aion-cached:** wire 2006/2007/2009 = plaintext; единственная защита кадра =
csum-хвост (u16 LE wireLen). DummyCrypt в wire.go = no-op (API сохранён, seed = vestigial).
Конфиг seed теперь не влияет на провод — ОСТАВИТЬ (не ломать канон L2-эталона).

**Открытое:** позиция rolling m_key += m_packetSize в Aion OnRead не проверялась
(нет decrypt-вызова → вероятно rolling тоже отсутствует) — непринципиально (key не используется).

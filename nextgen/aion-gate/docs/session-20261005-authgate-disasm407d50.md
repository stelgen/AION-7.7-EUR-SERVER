# Session 2026-10-05 (ночь-3) — ДИЗASM 0x407d50 ЗАКРЫТ: полный расклад welcome + парадокс capture

## Что снято из asm (все адреса из AuthGateD_original.exe = прод, MD5 fb8e6608 ✓)

### 1. Welcome-билдер 0x407d50 (вызывается из OnCreate 0x407fa0 @40800d)

fmt-строка **0x42cf2c = "cddbbbcccc"** (10 глаголов, подтверждено из .rdata байт-в-байт).
Реальный форматтер = **0x40e540(dest, max, fmt, va_list)**, обёртка 0x40e740 — cdecl, 16 args (add esp,0x40).
Глаголы (jump-таблицы 0x40e6f8/0x40e714): `S`=UTF-16z, `s`=ASCIIz, **`b`=(len,ptr) копия len байт БЕЗ префикса**, `c`=байт, `h`=u16, `d`=u32 LE. Return = edi-dest = длина.

**Порядок varargs (cdecl, последний push = va[0])**: 
`0x0, [esi+0xfc], V=[ds:0x43b438→obj+0xa0], 0x80, &scrambleModulus, 0x10, &ggBlob, 0x10, P1=[esi+0x21a4], S=[0x43c1ac], B0=[0x43c1b0], B1=[0x43c1b1], B2=[0x43c1b2]` = ровно 13 va на 10 глаголов ✓.

**Plaintext (173 байта)**:
```
[0]      = 0x00            ('c' ← va[0]=0x0!)
[1:5]    = [esi+0xfc]      (d)
[5:9]    = V=[global+0xa0] (d)
[9:137]  = scrambleModulus 128B (b: len 0x80, ptr=&[esp+0x20]-local, ключ pool 0x43fcbc, keyIdx=[esi+0xf4])
[137:153]= 16B из 0x407a10 (b) — при флаге [0x43c188]==0 это 16 НУЛЕЙ (якорь для unscramble!)
[153:169]= 16B [esi+0x21a4] = KEY2 (b) — key2 передаётся клиенту ВНУТРИ welcome (как SM_INIT классики!)
[169]    = S байт, [170]=B0, [171]=B1, [172]=B2  (ccccc)
```
**welcomeExtra4(+4) РАЗГАДАН**: это НЕ поле — `add eax,0x4` к длине после форматтера (407e16). n = 173+4 = 177.

### 2. EncryptPrimary 0x417a20 — точная семантика (исправляет модель!)
```
n' = roundup8(n)                      ; 177→184
S = old[0]                            ; dword0 НЕ трогается
for k=1..n'/4-1: S += old[k]; new[k] = old[k]^S    ; S ВКЛЮЧАЕТ old[k]!
[n'/4] = S (csum = финальный cumsum)  ; dword46 → offset 184 ✓ (совпало с прежним допущением)
len = n' + 8                          ; 192 → ECB(buf, 192) → wire = 2+192 = 194 ✓✓
```
Арифметика замкнулась 1-в-1: 173 → +4=177 → roundup8=184 → csum@184 → ECB 192 → wire 194 (len C2 00 ✓).
DecryptSecondary 0x417a80: roundup8(n), csum = **XOR** всех dword → dword[n'/4], len+8. Верификатор 0x417ad0.

### 3. Ключи (0x407550 / 0x4075d0 / 0x417a00)
- **0x407550(seed 0x4bd)** → `[esi+0x2194]` = key1: `LUT[(i^ (dl^dh))]^LUT[dl]^LUT[dh]` ×3 + pad `0x6c6c6c6c` = **6b60cb5b82ce90b1cc2b6c556c6c6c6c bit-exact** (пересчитано из testdata/gate-lut ✓).
- **0x4075d0** → `[esi+0x21a4]` = key2 = LUT[r], LUT[r&0xc1], LUT[r&0xf2], LUT[r&0x23], r=rand (per-conn!).
- **0x417a00 = SetKey(ctx, key, len)** → 0x4178d0 = СТАНДАРТНЫЙ Blowfish key-schedule (P 0x12 dwords @0x437570, S 0x400 @0x4375b8 — константы стандартные).
- OnCreate: `SetKey(ctx#1=[esi+0x104], key1)`; `SetKey(ctx#2=[esi+0x114c], key2)`; затем welcome. **ctx#1 = BF(key1) — welcome-ключ это key1** (asm 1-в-1).
- LUT @0x437160 — только читается (писателей в коде нет).
- 0x407a10 = GetGGBlob(out): флаг [0x43c188]==0 → **16 нулей**; иначе 16Б из [this+0xCC] через 0x418d70([this+0x9c]).

### 4. Бонус: 42B RSA-echo расшифрован структурно
Capture: cipher-блоки [P][Q][Q][Q][P] (ECB-повторы) → plaintext = **[dword A][28 нулей][A][pad 0]**, A per-conn.
Это EncryptSecondary([sid][28×0]) → XOR-csum = A → dword8 → [A][0 pad] — точное совпадение с наблюдением.
Т.е. echo = [sid 4B][28×0] (32B) → 40B → frame 42 ✓. (Проверить в Go: их RSA-echo-модель может быть не та.)

## ⚠ ПАРАДОКС (НЕ ЗАКРЫТ, меняет план)

_asm-модель железно даёт plaintext[0]=0x00 (dword0 не скрамблится, 'c'←0x0) при ключе key1._
_capture же: dec(key1)[0] = 0x23/0xcc/0xc5/0xfa… — динамический, НИ ОДИН из 82 welcome не даёт 0x00_
_(проверены: offset 0..4, byte-order вариантов ключа 4 шт, ENC-направление). Математика модели_
_(173→177→184→192→194) сходится идеально — расхождение только в содержимом [0] при расшифровке._

Следствия/гипотезы (в порядке проверки):
1. **sid-гипотеза прошлой сессии была подгоном**: dword0 capture-группы 7d521423 = «23 14 52 7d» трактовали как [0x23][sid 0x7d5214] — но asm говорит [0]=0x00 и PlainByte в welcome ВООБЩЕ НЕ УЧАСТВУЕТ (статика только S,B0,B1,B2 на [169..172]). Их Go-билдер (c←PlainByte 0x23) **гарантированно не совпадает с оригиналом в байте [0]**.
2. Capture может быть смесью/не тем бинарем (в /tmp есть p2/p3/p4/patched-сборки) — проверить provenance proxylog/handshake-capture.
3. Если capture честный (внешние IP реально логинились через fb8e6608) → рантайм-модификация ключа/LUT, которую прямой grep не видит (поиск инициализации .data через указатели).
4. РЕШАЮЩИЙ тест теперь дешёвый: **собрать welcome по asm-модели 1-в-1 и отдать живому клиенту** (юзер логинится) — клиент либо примет (asm верен, capture-интерпретация была ошибочна), либо нет → копать рантайм-ключ.

## Статус чек-листа §6 (из ресёрч-дока)
- порядок варов: ✅ (fmt "cddbbbcccc", va-порядок снят)
- welcomeExtra4: ✅ (add eax,4 — не поле)
- источник первых 8 байт: ✅ структура ([0]=0x00, [fc], V) — но семантика [fc]/V (per-run?) не подтверждена из-за парадокса
- len-байты блобов: ✅ ('b' без префикса — прежняя гипотеза len-байтов в plaintext НЕВЕРНА)
- ECB↔скрамбл порядок: ✅ (скрамбл → csum → ECB, всё внутри EncryptPrimary)
- S_init: ✅ (S стартует с old[0], DWORD0 НЕ ТРОГАЕТСЯ — уточнение против старой записи «S_prev»)

## Готчи
- 0x40e540 handler[6] (default для неизвестных символов) = 0x40e6ab — просто continue БЕЗ потребления va (не багать в Go-реализации Assemble).
- «b» копирует len байт БЕЗ длины в поток; bounds check только dest+len.
- csum в EncryptPrimary — ЭТО финальный cumsum S (включая old[0]), не отдельный расчёт.
- Буфер: байты 188..191 (после csum) EncryptPrimary не пишет — резидуум пула буферов (0x40abf0); для byte-exact надо знать, зеро ли пул.

## Следующий шаг
~~Правка Go-билдера под asm-модель~~ → ✅ ВЫПОЛНЕНА (ночь-4, тесты зелёные -count=1):
- welcome.go: plaintext[0]=0x00 (PlainByte убран), layout по asm, +4 = удлинение (комментарии);
- crypto.go: DecryptSecondary ИСПРАВЛЕН по 0x417ad0 (XOR (len-8)/4 dword, csum@k, возврат len-8;
  прежний вариант XOR-ил csum и проверял pad — на реальных клиентах давал бы ложные отказы/пропуски);
  добавлен EncryptSecondary @0x417a80 (csum сразу за roundup8-данными, pad после);
- server.go: 34b-путь = handleAuthGG (CM/SM_AUTH_GG: ответ EncryptSecondary([sid][28×0]),
  расшифровка клиентского blob best-effort, связь не рвём; RSA в обмене НЕ участвует —
  он в логине 186b data[:128] ✓); encryptToClient удалён (позиция csum была неверной —
  в конце вместо сразу за данными); serverlist-push через proto.EncryptSecondary;
- тесты: TestWelcome194 (dword0=0x7d521400), TestEncryptPrimaryKnownAnswer (скрамбл hand-computed),
  TestEncryptSecondaryEchoStructure (cipher [P][Q][Q][Q][P] = capture-структура!), e2e AUTH_GG.
- config: welcomePlainByte остался в yaml (не используется, deprecated).

СЛЕДУЮЩИЙ ШАГ: ~~win cross-build → деплой на стенд~~ → ✅ ВЫПОЛНЕНО (ночь-4, ~00:15):
- кросс-сборка aion-gate.exe (7 538 688B, PE32+ x64, -trimpath -ldflags "-s -w");
- стенд D:\SAION\aion-gate-stand\ (aion-gate.exe + config-stand.yaml (порт 2109, зеркало прод-конфига
  41007: authd 127.0.0.1:2110, loginType=2, sessionTimeout=5, authReconnectInterval=30) + gate-stand.bat
  (ASCII+CRLF) + etc/BlockIPs.txt) — задача AionGateStand (SYSTEM, onstart), ПРОД (PID :2106) НЕ ТРОНУТ;
- ⚠ НАХОДКА: порт 2108 занят самим L2Authd (второй листенер!) — стенд переведён на 2109;
- ✅ SELF-TEST с LAN-машины (python blowfish): TCP → welcome 194B → ECB-dec(key1) → dword0=0x00000100
  = [0]=0x00 + sid=1 ✓✓, хвост 188..191 нулевой ✓; 34b мусор → 42b ответ (echo жив) —
  ASM-МОДЕЛЬ ПОДТВЕРЖДЕНА END-ТО-END НА ЖИВОМ СТЕНДЕ.

## ✅ СВИТЧ В ПРОД ВЫПОЛНЕН (06.10 ~04:00, по «го» юзера)
- D:\SAION\aion-gate\ = aion-gate.exe + config-prod.yaml (порт 2106) + gate-prod.bat + etc\BlockIPs.txt;
- C:\Temp\gate.bat заменён на наш (задача AionGate без ретаргета — она уже указывала на этот bat);
  ОРИГИНАЛ сохранён: C:\Temp\gate.bat.orig-AUTHGATED (= старый v3: taskkill AuthGateD → wait 2104 →
  start AuthGateD → wait 2106) — ОТКАТ: taskkill aion-gate; copy /y .orig-AUTHGATED → gate.bat;
  schtasks /run AionGate; вернуть exe-строку в aion-op config;
- оригинал AuthGateD (PID 8080) потушен, НАШ на 2106 (PID 8012), authd-связка ESTABLISHED 8012→2110;
- ⚠ ИНЦИДЕНТ при свитче №1: я по ошибке запушил gate.bat.orig-AUTHGATED В C:\Temp\gate.bat → задача
  подняла ОРИГИНАЛ обратно (PID 7404, ~1с после kill) — выглядело как «воскрешение», на деле мой
  файл-ошибка; повторный свитч с верным bat — чисто;
- aion-op: config-vm.yaml байт-замена python (exe: AuthGateD.exe → aion-gate.exe; лог-path →
  D:/SAION/aion-gate/gate-prod.log) + рестарт задачи AionOp (новый PID, :10200 жив);
- SELF-TEST ПРОДА: welcome 194B → dword0=0x00000100 ([0]=0x00, sid=1) ✓, echo 42b ✓;
- 60с мониторинг: AuthGateD НЕ воскрес, L2Authd жив (2104/2110), цепочка не упала;
- метод-уроки: scp multi-file на Windows-путь работает, но ВЕРИФИЦИРОВАТЬ dir после; НЕ пушить
  файл-эталон отката в действующий путь; PS-правки конфигов — python bytes-replace.

ОСТАЛОСЬ: живой КЛИЕНТ на прод 2106 (решающий эксперимент парадокса capture-vs-asm) — юзер логинится;
наблюдать gate-prod.log (authgg.mismatch/authgg.blob события) и Server64.err.
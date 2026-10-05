# Session 2026-10-05 (вечер/ночь) — AuthGateD: скелет aion-gate + чекпоинт byte-exact

## Что сделано (коммиты: 36594cf → bde0fa8)

1. **Актуализация** (36594cf): план §6 подтверждён, код не начат.
2. **§5.1 закрыт без дизasmа** (de80752): байты @0x43c1b0/b1/b2 из exe = UTF-16 «er\0» —
   0x23 НЕ из этой зоны.
3. **§5.3/§5.4 закрыты** (4e8181d): Assemble расшифрован (jump-table из exe:
   c=1B/h=u16/d=u32/b=(len,ptr)/s/S), fmt-строки .data ("cdd"/"cd"/"cdh"/"cbdb"/"cc");
   wire 2110 байт-в-байт (len самоинклюзивный u16 с обеих сторон); RecvLogin поля
   +128/+132/+148/+152, cc 22/45.
4. **Скелет aion-gate** (b98e4f3): internal/proto — свой Blowfish (константы из exe,
   кросс-чек python-blowfish), EncryptPrimary/DecryptSecondary, Assemble, LUT-key1
   (6b60cb5b… bit-exact из формулы), welcome-билдер, RSA-256 пул; internal/authdclient.
5. **server+config+main** (71cba68): сессии/brute/BlockIPs/cc/sessionTimeout, yaml-зеркало
   config.txt 41007, e2e fake-authd+fake-client зелёный (26b serverlist как в capture).
6. **0x417b60 закрыт** (5daa5e5): beecrypt rsapricrt — вход 128Б BE (ведущие нули),
   ключ pool[keyIdx]=[sock+0xf4], результат i2osp-выровнен → decbuf 32Б. handleLogin
   делает реальный RSA-dec.
7. **ship** (6fb8520): копия из aion-captcha (≡logd), события
   start/stop/conn.up/down/authd.up/down/cc/login/serverlist/parse.err; main: go sh.Run(ctx).

## ⚠ БЛОКЕР СВИТЧА — byte-exact welcome НЕ закрыт (bde0fa8, док §7)

- ECB-dec(key1=6b60cb5b) по 82 welcome → 49 групп (d0,d1): plaintext dword0/1
  константны в рамках запуска, случайны между запусками. Только proxylog-группа
  d0=7d521423 даёт plaintext[0]=0x23; в остальных 0xcc/0xc5/0xfa… → «0x23» —
  НЕ константа, динамический вар.
- Скрамбл из asm (0x417a20, прочитан лично: new[k]=old[k]^(S_prev+old[k]),
  csum=cumsum, ECB n+8) нарушает capture-тождество y_0==a_0 (0xd8≠0x23 у группы
  7d521423). GG-зона нулей (4+ равных dword в dec) не найдена ни на одном смещении.
- **Вывод:** неверна гипотеза layout'а варов/буфера welcome (порядок «c d d b b b c c c c»
  и welcomeExtra4(+4) — гипотетические). Solver: x^(a+x)=y решается линейно через
  переносы (c_{i-1}=y_i^a_i), см. tools/analysis/diag*.py.

## Следующий шаг (новый чат)

**Дизasm `0x407d50` (welcome-билдер)**: точный порядок варов Assemble, размер/смещение
буфера (где +4), что уходит в EncryptPrimary (возможен offset-заголовок или порядок
ECB↔скрамбл). КРИТИЧНО: наш welcome клиент не расшифрует без byte-exact — НЕ ДЕПЛОИТЬ.
После: byte-exact тест против capture → фейк-клиент с реальным plaintext LoginEx →
кросс-сборка win → деплой по плану §6 (D:\SAION\aion-gate, AionGate retarget, aion-op
config, откат C:\Temp\gate.bat). «го» на свитч дано заранее (юзер логинится потом).

## Готчи сессии

- replace_file_content на CRLF-файле может применить чанки ЧАСТИЧНО до ошибки —
  после мульти-правок go vet+тесты+grep критических блоков.
- net.Pipe синхронная — фейк-authd читатель обязателен; инъекция: srv.SetAuthd(DialConn(srv.AuthdHandler())).
- Срез блока Blowfish: buf[off:off+8], НЕ buf[off:].
- Экономный реверс: IAT-имена через парсер import table; секции PE: VirtualSize/VirtualAddress/
  SizeOfRawData/PointerToRawData — в этом порядке.

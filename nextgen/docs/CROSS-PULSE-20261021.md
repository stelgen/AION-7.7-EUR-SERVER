# 🔄 CROSS-PULSE: 2026-10-21 — accache R3.6 (wire-regression FIX)

## Статус
**Коммит:** `22cc4cf` (HEAD -> main) — test(aion-accache): FIX wire-regression callArgs prefix slicing (+11 -> +10), all tests green.
**Ветка:** main. Тесты: `go vet` 0 issues, `go test ./...` — ВСЕ ЗЕЛЁНЫЕ (5.518s).

## Что сделано (шаги)
1. **R3.6 хендлеры:** реализованы 12 silent-процедур (`handlers_r3x.go`) — luna 0x1b/1c/1d, board 0x1e, polls 0x20–0x23, monstercore 0x26/0x27, login-event 0x17/0x18.
2. **Wire-regression тесты:** `wire_regression_test.go` — W2/W3 покрытие, off-by-2 защита, silence-проверка.
3. **FIX деградации:** `callArgs` резал `q[:len(procName)+11]`, но префикс `"{call dbo."` = 10 символов → офф-бай-один-символ, сравнение всегда проваливалось. Исправлено на `+10`.
4. **fakeQErr:** для exec-only процов (без result-set) Query возвращает ошибку, форсируя Exec-путь `db.CallProc`.

## Правки в файлах
- `nextgen/aion-accache/internal/server/handlers_r3x.go` (NEW, 12 хендлеров)
- `nextgen/aion-accache/internal/server/wire_regression_test.go` (NEW, тесты W2/W3)
- `nextgen/aion-accache/internal/server/server.go` (регистрация R3.6)
- `nextgen/docs/REPORT-20261021.md` (NEW, отчёт)

## Следующие шаги (roadmap)
- [ ] **R4**: A/B fork-прогон — копия трафика в наш, ориг НЕ трогать, байт-в-байт дифф.
- [ ] **R5**: свитч по «го» (retarget `AionAcc` на `D:\SAION\aion-accache\run.cmd`).
- [ ] **CacheD64 R1**: wire-парсер 2006 из готовых `log/*.log` (356МБ).
- [ ] **op Phase 1.5**: событийный watchdog, async-ожидания, R6-конфиг OP-1..OP-6.
- [ ] **514/UDP**: заблокировано NAT (VM -> песочница), ожидание открытия порта юзером.

## Правила соблюдены
- ✅ Silent-политика (ответы только при доказанной capture)
- ✅ Wire-канон `len=total`
- ✅ Коммит + логирование шагов
- ✅ Отсутствие секретов в коде (conn string = плейсхолдер)

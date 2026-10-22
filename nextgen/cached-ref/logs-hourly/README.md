# hourly-логи CacheD64 — образец формата (2026-10-22, ~360 строк)

Источник: VM `D:\AION_LIVE_SERVER\CacheServer\log\YYYY-MM-DD.HH.log` (hourly, маленькие).
**НЕ wire-раскладки** — семантические следы хендлеров + служебные отчёты.

## Формат
`YY-MM-DD HH:MM:SS:<текст>`

## Семантические следы read-пути (канон порядка, Т24)
- `[CharacterList] ServerId:1 (AccID:1010) (charID:1003) (charCount:2)` — на КАЖДЫЙ чар-селект
  (после logout-флоу ACS и при входе на экран выбора). charID в логе = последний контекст (⏳).
- `[char]enter world 1010/1002 lev80 110010000(x,y,z)` — вход в мир; ID+координаты =
  персистентная позиция из БД (последнего выхода).
- `[char]leave world 1010/1002 <ID>(pos)` — выход из мира.
- `LoadClientQuickBar ::==  characterId(100X)` — quickbar.
- `thahn client setting:RP_LOAD_SKILL_COOLTIME...` — RP-имена сверяются со словарём .profile ✓.

## Служебное
- Старт-цепочка: CheckDatabase → CleanUp ×13 (Deleted Item/SellToNpc/Extracted/...) →
  VendorDb::CreateIndex → Poll → IOThread → `*interactive socket ready 2007/2009` →
  `*cache server ready on port 2006` → Connected to log server.
- Память: VendorDb.cpp pool = ~90% TOTAL (196МБ из 218МБ) — вендор-данные в RAM.
- Профиль-отчёты: `perf\YYYY-MM-DD.HH.MM.SS.profile` (те же .profile, что генерили opcodes.go).

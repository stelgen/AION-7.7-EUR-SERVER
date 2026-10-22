# ❄️ ISSUE #111-FROZEN: Манастоны (стекинг статов) — ЗАМОРОЖЕНО до aion-main MVP

> **labels:** `заморожено-до-aion-main-MVP` · `binpatch` · `blocked-external-architecture` · `verdict-final`
> **state:** FROZEN (22.10.2026) · **автор вердикта:** агент-сессия 22.10 (коммиты `4fd5cb1`/этот)
> **unfreeze-условие:** первый рабочий MVP мира в [aion-main](../../aion-main/) (логика зачарования переписывается в Go)

## Вердикт (финальный, доказательный)

Байтовый перенос манастоун-фикса из чужого бинаря (`111-Mainserver64.rar`, MD5 `b60a1a4b`,
31 890 432Б) в наш Server64 (#180, MD5 `c5157302`, 45 499 392Б) **НЕВОЗМОЖЕН**:

1. **Разные сборки** — размеры не совпадают, построчный дифф невалиден офсаетно.
2. **Тела enchant-функций структурно пересобраны** в чужой сборке (якорный дифф
   доказал: уникальные прологи есть, но тела — другой поток, другая арифметика
   (`imul 0x1F85EB51` — быстрый делитель), другие вызовы) — это правка исходников
   чужой стороной, а не байт-патч.
3. ENIGMA-сборки не деплоятся (политика бэкдор-риска) — чужой exe как целое тоже нельзя.

**Правильное решение по стратегии проекта** (README «бескомпромиссно»): фикс манастоунов
делается в переписи [aion-main](../../aion-main/) (Go) — там логика зачарования будет наша,
и «бесконечный стекинг статов» либо не воспроизведётся, либо будет починен на уровне логики.

## Карта реверса (что уже снято — переиспользуется при unfreeze)

### Наш бинарь (Server64 #180, PDB publics 74164 — якоря manastone-пути)

| Функция (PDB) | puboff | VA | Роль |
|---|---|---|---|
| `CanMatterOption@Item` | 0xce460 | 0x1400CF460 | проверка слота манастоуна |
| `UpdateDBMatterOption@Item` | 0x4da240 | 0x1404DB240 | запись option в БД |
| `SendUpdateMatterOption@DB` | 0x3ea670 | 0x1403EB670 | RPC к клиенту |
| `EncodeUpdateMatterOption@ServerToDb_Update` | 0xc52c0 | 0x1400CD2C0 | wire-кодирование |
| `ItemEnchantTool` ctor/dtor | 0xf2da0/0xf2f60 | 0x1400F3DA0/0x1400F3F60 | инструмент зачарования |
| `EnchantItem@User` | 0x4328e0 | 0x1404338E0 | главный вход зачарования |
| `defaultEnchantSuccess@User` | 0x431c60 | 0x140432C60 | обычный успех |
| `exceedEnchantSuccess@User` | 0x430390 | 0x140431390 | exceed-успех |
| `CalcEnchantCritical@User` | 0x434190 | 0x140435190 | крит зачарования |
| строки `stat_enchant_name0..5`, `manastone` | .rdata | xref-через-строку | доп. якоря |

### Чужой exe (результат якорного диффа, 512Б тел)

| Функция | Prologue в чужом | Тела |
|---|---|---|
| `CanMatterOption@Item` | NO-PROLOGUE | пересобрана |
| `UpdateDBMatterOption@Item` | generic ×46 | ложные якоря |
| `SendUpdateMatterOption@DB` | generic ×675 | ложные якоря |
| `CalcEnchantCritical@User` | generic ×675 | ложные якоря |
| `EncodeUpdateMatterOption` | NO-PROLOGUE | пересобрана |
| `ItemEnchantTool` ctor | NO-PROLOGUE | пересобран |
| `exceedEnchantSuccess@User` | NO-PROLOGUE | пересобрана |
| `EnchantItem@User` | **UNIQUE** @file 0x827f90 | **структурно другой код** (diff 463Б кластером, не байт-в-байт) |
| `defaultEnchantSuccess@User` | **UNIQUE** @file 0x168a25 | **структурно другой код** (`imul 0x1F85EB51`, другой поток) |

### Чего НЕ хватает для unfreeze-реверса (если решим патчить, а не переписывать)

- Полный реверс семантики чужих `EnchantItem` / `defaultEnchantSuccess` /
  `exceedEnchantSuccess` (часы дизasm-работы) → спецификация манастоун-логики.
- Ghidra-ре-имплантация спецификации в наш бинарь (аллокация кодовой пещеры,
  перелинковка вызовов CanMatterOption/UpdateDBMatterOption) — недели, риск высокий.
- Стенд-тест манастоуна (надеть → стат не стекается → релог сохраняет) — как у #108.

## Ссылки на инструменты (рабочие скрипты, коммитятся в tools/analysis/)

| Скрипт | Что делает |
|---|---|
| [`tools/analysis/prologue_diff.py`](../../../tools/analysis/prologue_diff.py) | якорный дифф сборок (размеры → prologue → тела → вердикт). Конфиг pairs.json. |
| [`tools/analysis/find_callers.py`](../../../tools/analysis/find_callers.py) | все ссылки на VA: E8/E9/FF15/FF25/lea/mov/ptr-таблицы (обкатан на 45МБ exe) |
| [`tools/analysis/pdb_nearest.py`](../../../tools/analysis/pdb_nearest.py) | ближайший PDB-символ к VA (формат `sec\toff\tsize\tname`, **off = parts[1]**!) |
| [Рекон-док #108/#111](recon-108-111-20261022.md) | полная методология + гейт #108 (патч готов) + PE-карты секций |
| `tools/analysis/pdbpub.py` | извлечение publics из PDB (метод logd) |
| `/home/user/bin/unrar` | RAR7-распаковка (7z не берёт RAR7; rarlab-curl заблокирован) |

## Условия разморозки

1. `aion-main` MVP мира жив (мир-логика в Go) → манастоуны в переписи — ИЛИ
2. Осознанное решение юзера на семантический реверс чужих функций (оценка: дни Ghidra-работы
   + риск регрессии enchant-механики в нашем бинаре).

До этих условий — **не трогать**. Трек-статус в дашборде: «#111 ❌ заморожен».
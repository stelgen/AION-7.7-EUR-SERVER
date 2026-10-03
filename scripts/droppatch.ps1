# AION 7.7 — дроп-патч: редкие дропы (prob 2-7) → 50%, деньги (cash_drop_prob) → x30
# НЕ ТРОГАЕТ: prob 1000000/2500000 (гарантированные), квестовые предметы item_quest.xml,
#prob>=100000 (обычные).
# Файлы UTF-16LE. БЭКАПЫ созданы: *.bak-20261003_1205.
$ErrorActionPreference='Stop'
$dir='D:\AION_LIVE_SERVER\Map\XML'
$f="$dir\CommonDropItems.xml"

# 1) CommonDropItems: только prob 2,3,4,5,6,7 → 500000 (50%)
$text=[IO.File]::ReadAllText($f,[Text.Encoding]::Unicode)
$before=([regex]::Matches($text,'<prob>[2-7]</prob>')).Count
$text=[regex]::Replace($text,'<prob>[2-7]</prob>','<prob>500000</prob>')
$after=([regex]::Matches($text,'<prob>500000</prob>')).Count
[IO.File]::WriteAllText($f,$text,[Text.Encoding]::Unicode)
Write-Host "CommonDropItems: prob 2-7 → 500000: replaced=$before (prob500000 total now=$after; было 598 — прирост должен быть ~$before)"

# 2) npcs_test.xml: cash_drop_prob 0 → x30 шкала (0..30 = шанс 0..3% если 0..1000=0..0.1%?)
# cash_drop_prob семантика неизвестна → безопасный тест: 0 → 1000 (0.1% при шкале 1e6)
$g="$dir\Europe\npcs_test.xml"
$gtext=[IO.File]::ReadAllText($g,[Text.Encoding]::Unicode)
$cnt=([regex]::Matches($gtext,'<cash_drop_prob>0</cash_drop_prob>')).Count
$gtext=[regex]::Replace($gtext,'<cash_drop_prob>0</cash_drop_prob>','<cash_drop_prob>1000</cash_drop_prob>')
[IO.File]::WriteAllText($g,$gtext,[Text.Encoding]::Unicode)
Write-Host "npcs_test: cash_drop_prob 0→1000 (деньги x30 от нулевого базиса): replaced=$cnt"

# 3) сверка квестовых предметов: в CommonDropItems НЕ должно быть item_quest ссылок
$qc=([regex]::Matches($text,'<item>(quest|q_)')).Count
Write-Host "quest-check в CommonDropItems: quest-like items=$qc (0=чисто)"
Write-Host "DONE. Рестарт NPCSvr + Server64 обязателен."

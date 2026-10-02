# ================================================================
# AION 7.7 PTS EU — Патч конфигов сервера
# IP-замены, country=2 (EU), region=2 в таблице Server, hardlink Map\XML\Europe.
# ================================================================
$OutputEncoding = [Console]::OutputEncoding = [Text.Encoding]::UTF8
$A = 'D:\AION_LIVE_SERVER'
if (!(Test-Path $A)) { Write-Host "НЕ НАЙДЕНО $A — сначала распакуй AION_LIVE_SERVER!" -ForegroundColor Red; exit 1 }

$vmIp = Read-Host "Введи IP VM (напр. 192.168.0.125)"
$stamp = 'bak_' + (Get-Date -Format 'yyyyMMdd')

$targets = @(
  @{ f = "$A\AccountCacheServer\common.xml"; s = "<mailServer>192.168.200.131</mailServer>"; r = "<mailServer>$vmIp</mailServer>" },
  @{ f = "$A\AuthD\etc\config.txt";          s = 'logdip = "192.168.200.131"';              r = "logdip = `"$vmIp`"" },
  @{ f = "$A\MainServer\config.xml";         s = "<clientAcceptAddr>192.168.200.131</clientAcceptAddr>"; r = "<clientAcceptAddr>$vmIp</clientAcceptAddr>" },
  @{ f = "$A\LogServer\common.xml";          s = "<country>5</country>";                    r = "<country>2</country>" },
  @{ f = "$A\LogServer\config.xml";          s = "<country>5</country>";                    r = "<country>2</country>" },
  @{ f = "$A\Map\NcGuard\ncgsvrcfg.ini";     s = 'countryid=410';                           r = 'countryid=2' }
)

Write-Host "=== Патч конфигов (с бэкапом .$stamp) ===" -ForegroundColor Cyan
foreach ($t in $targets) {
  $f = $t.f
  if (!(Test-Path $f)) { Write-Host "MISSING: $f" -ForegroundColor Red; continue }
  Copy-Item $f "$f.$stamp" -Force
  $bytes = [IO.File]::ReadAllBytes($f)
  $isU16 = ($bytes.Length -ge 2 -and $bytes[0] -eq 0xFF -and $bytes[1] -eq 0xFE)
  if ($isU16) { $enc = [Text.Encoding]::Unicode } else { $enc = [Text.Encoding]::GetEncoding(1251) }
  $txt = [IO.File]::ReadAllText($f, $enc)
  if ($txt.Contains($t.s)) {
    $txt = $txt.Replace($t.s, $t.r)
    [IO.File]::WriteAllText($f, $txt, $enc)
    Write-Host "PATCHED: $($f.Replace($A,''))" -ForegroundColor Green
  } else {
    Write-Host "NOT-FOUND: $($f.Replace($A,'')) (может уже пропатчен)" -ForegroundColor DarkYellow
  }
}

# Hardlink Map\XML\Europe (Server64 ищет L10N-подмножества в этой папке — region=2)
Write-Host ""
Write-Host "=== Map\XML\Europe hardlinks (605 файлов, ~0 байт) ===" -ForegroundColor Cyan
$src = "$A\Map\XML"
$dst = "$src\Europe"
if (!(Test-Path $dst)) { New-Item -ItemType Directory -Force -Path $dst | Out-Null }
$linked = 0; $copied = 0; $failed = 0
Get-ChildItem $src -File -EA SilentlyContinue | ForEach-Object {
  $target = "$dst\$($_.Name)"
  if (Test-Path $target) { return }
  try {
    New-Item -ItemType HardLink -Path $target -Target $_.FullName -EA Stop | Out-Null
    $linked++
  } catch {
    try { Copy-Item $_.FullName $target -Force -EA Stop; $copied++ } catch { $failed++ }
  }
}
Write-Host "HARDLINKED: $linked  COPIED: $copied  FAILED: $failed"

Write-Host ""
Write-Host "=== Удаление старой Russia-папки (если создавалась раньше — это неверное имя) ===" -ForegroundColor DarkCyan
if (Test-Path "$src\Russia") { Remove-Item "$src\Russia" -Recurse -Force; Write-Host "removed Russia" }

Write-Host ""
Write-Host "=== CONFIG-PATCH DONE ===" -ForegroundColor Green

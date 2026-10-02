# ================================================================
# AION 7.7 PTS EU — Пререквизиты (тихая установка)
# VC++ (2010/2012/2013/2015-2022 x64+x86) + 7-Zip + SQL Native Client 11.0 + SSMS (опц.)
# Запускать PowerShell от админа.
# Все ссылки официальные (Microsoft) — см. docs/links.md.
# ================================================================
$OutputEncoding = [Console]::OutputEncoding = [Text.Encoding]::UTF8
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
$dst = 'C:\Temp'
New-Item -ItemType Directory -Force -Path $dst | Out-Null

$downloads = @(
  @{ n='vc2010x64'; u='https://download.microsoft.com/download/1/6/5/165255E7-1014-4D0A-B094-B6A430A6BFFC/vcredist_x64.exe'; a=@('/q','/norestart') },
  @{ n='vc2010x86'; u='https://download.microsoft.com/download/1/6/5/165255E7-1014-4D0A-B094-B6A430A6BFFC/vcredist_x86.exe'; a=@('/q','/norestart') },
  @{ n='vc2012x64'; u='https://download.microsoft.com/download/1/6/B/16B06F60-3B20-4FF2-B699-5E9B7962F9AE/VSU_4/vcredist_x64.exe'; a=@('/quiet','/norestart') },
  @{ n='vc2012x86'; u='https://download.microsoft.com/download/1/6/B/16B06F60-3B20-4FF2-B699-5E9B7962F9AE/VSU_4/vcredist_x86.exe'; a=@('/quiet','/norestart') },
  @{ n='vc2013x64'; u='https://download.microsoft.com/download/5/D/8/5D8C65CB-C849-4025-8E95-C3966CAFD8AE/vcredist_x64.exe'; a=@('/quiet','/norestart') },
  @{ n='vc2013x86'; u='https://download.microsoft.com/download/5/D/8/5D8C65CB-C849-4025-8E95-C3966CAFD8AE/vcredist_x86.exe'; a=@('/quiet','/norestart') },
  @{ n='vc2022x64'; u='https://aka.ms/vs/17/release/vc_redist.x64.exe'; a=@('/quiet','/norestart') },
  @{ n='vc2022x86'; u='https://aka.ms/vs/17/release/vc_redist.x86.exe'; a=@('/quiet','/norestart') },
  @{ n='7zip';      u='https://www.7-zip.org/a/7z2501-x64.exe'; a=@('/S') },
  @{ n='sqlncli';   u='https://download.microsoft.com/download/b/e/d/bed73aac-3c8a-43f5-af4f-eb4fea6c8f3a/ENU/x64/sqlncli.msi'; a=@('IACCEPTSQLNCLILICENSETERMS=YES','/qn') }  # CRITICAL: IACCEPTSQLNCLILICENSETERMS с большой S в SQL
)

Write-Host "=== Загрузка и установка пререквизитов ===" -ForegroundColor Cyan
foreach ($d in $downloads) {
  $f = "$dst\$($d.n).exe"
  if ($d.n -eq 'sqlncli') { $f = "$dst\sqlncli_x64.msi" }
  try {
    if (!(Test-Path $f)) {
      Invoke-WebRequest $d.u -OutFile $f -UseBasicParsing -TimeoutSec 600
      Write-Host "DL-OK   $($d.n)" -ForegroundColor Green
    } else { Write-Host "SKIP    $($d.n) (файл есть)" -ForegroundColor DarkGray }
  } catch { Write-Host "DL-FAIL $($d.n): $($_.Exception.Message)" -ForegroundColor Red; continue }

  $args = if ($f -match '\.msi$') { $d.a + @('/l*v',"$dst\$($d.n).log") } else { $d.a }
  $p = Start-Process (if ($f -match '\.msi$') { 'msiexec' } else { $f }) -ArgumentList $args -Wait -PassThru
  $rc = $p.ExitCode
  $ok = if ($f -match '\.msi$') { $rc -eq 0 } else { $rc -in 0,3010 }
  Write-Host ("INST-{0} {1} (rc={2})" -f ($(if($ok){'OK  '}else{'FAIL'})), $d.n, $rc) -ForegroundColor $(if($ok){'Green'}else{'Red'})
}

Write-Host ""
Write-Host "=== SSMS (опционально, для SQL GUI) ===" -ForegroundColor Cyan
$f = "$dst\ssms21.exe"
if (!(Test-Path $f)) {
  try {
    Invoke-WebRequest 'https://aka.ms/ssmsfullsetup' -OutFile $f -UseBasicParsing -TimeoutSec 1800
    Write-Host "DL-OK ssms21" -ForegroundColor Green
  } catch { Write-Host "DL-FAIL ssms21: $($_.Exception.Message)" -ForegroundColor Red }
}
if (Test-Path $f) {
  $p = Start-Process $f -ArgumentList @('/install','/quiet','/norestart') -Wait -PassThru
  Write-Host "INST ssms21 rc=$($p.ExitCode)"
}

Write-Host ""
Write-Host "=== Верификация ===" -ForegroundColor Cyan
Get-OdbcDriver | Where-Object { $_.Name -match 'Native Client' } | Select-Object Name
Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*','HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*' -EA SilentlyContinue |
  Where-Object { $_.DisplayName -like '*Visual C++ 20*' -or $_.DisplayName -like '*7-Zip*' -or $_.DisplayName -like '*Native Client*' } |
  Select-Object DisplayName, DisplayVersion | Sort-Object DisplayName | Format-Table -AutoSize
"7-Zip exe: " + (Test-Path "$env:ProgramFiles\7-Zip\7z.exe")

Write-Host ""
Write-Host "=== ПРЕРЕКВИЗИТЫ DONE ===" -ForegroundColor Green

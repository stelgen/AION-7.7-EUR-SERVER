# ================================================================
# AION 7.7 PTS EU — Твики VM (запускать PowerShell от админа)
# Отвечает за: zram (Memory Compression), NCSI off, тёмную тему,
# деблоат (телеметрия/WER/DiagTrack/Siuf), DisableCAD, autologin-info.
# Всё обратимо. Не влияет на функционал системы.
# ================================================================
$OutputEncoding=[Console]::OutputEncoding=[Text.Encoding]::UTF8
Write-Host "=== [1/6] zram: Memory Compression + PageCombining ===" -ForegroundColor Cyan
Get-Service sysmain | Set-Service -StartupType Automatic
Start-Service sysmain -EA SilentlyContinue
Enable-MMAgent -MemoryCompression
Enable-MMAgent -PageCombining
Get-MMAgent | Format-List MemoryCompression, PageCombining
Write-Host "    (активируется после ребута; pagefile должен остаться включённым)" -ForegroundColor Yellow

Write-Host "=== [2/6] Тёмная тема (Apps + System + Console) ===" -ForegroundColor Cyan
Set-ItemProperty 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Themes\Personalize' -Name AppsUseLightTheme -Value 0 -Type DWord
Set-ItemProperty 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Themes\Personalize' -Name SystemUsesLightTheme -Value 0 -Type DWord
Set-ItemProperty 'HKCU:\Console' -Name DefaultBackground -Value 0 -Type DWord -EA SilentlyContinue
Set-ItemProperty 'HKCU:\Console' -Name DefaultForeground -Value 15 -Type DWord -EA SilentlyContinue

Write-Host "=== [3/6] Деблоат: телеметрия/CEIP/DiagTrack/WER/Siuf ===" -ForegroundColor Cyan
# DiagTrack (Connected User Experiences and Telemetry) — полностью off
foreach ($s in 'DiagTrack','dmwappushservice') {
  Stop-Service $s -Force -EA SilentlyContinue
  Set-Service $s -StartupType Disabled -EA SilentlyContinue
}
# Очистить ETL-лог авто-сборщика телеметрии
Set-ItemProperty 'HKLM:\SYSTEM\CurrentControlSet\Control\WMI\Autologger\AutoLogger-Diagtrack-Listener' -Name Start -Value 0 -EA SilentlyContinue
Remove-Item 'C:\ProgramData\Microsoft\Diagnosis\ETLLogs\AutoLogger\AutoLogger-Diagtrack-Listener.etl' -Force -EA SilentlyContinue
Remove-Item 'C:\ProgramData\Microsoft\Diagnosis\ETLLogs\ShutdownLogger\ShutdownLogger.etl' -Force -EA SilentlyContinue
# Windows Error Reporting off
Stop-Service WerSvc -Force -EA SilentlyContinue
Set-Service WerSvc -StartupType Disabled -EA SilentlyContinue
New-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows\Windows Error Reporting' -Name Disabled -Value 1 -PropertyType DWord -Force | Out-Null
# Feedback (Siuf) scheduled tasks
Get-ScheduledTask -EA SilentlyContinue | Where-Object { $_.TaskPath -like '*\Feedback\Siuf\*' } | Disable-ScheduledTask -EA SilentlyContinue | Out-Null
# CEIP / Edge update / Consolidator / UsbCeip / Compatibility Appraiser / ProgramDataUpdater / Proxy / DiskDiagnostic
$tasks = @(
  '\Microsoft\Windows\Customer Experience Improvement Program\Consolidator',
  '\Microsoft\Windows\Customer Experience Improvement Program\UsbCeip',
  '\Microsoft\Windows\Application Experience\Microsoft Compatibility Appraiser',
  '\Microsoft\Windows\Application Experience\ProgramDataUpdater',
  '\Microsoft\Windows\Autochk\Proxy',
  '\Microsoft\Windows\DiskDiagnostic\Microsoft-Windows-DiskDiagnosticDataCollector'
)
foreach ($t in $tasks) { try { schtasks /Change /TN $t /Disable | Out-Null } catch {} }
Get-ScheduledTask -EA SilentlyContinue | Where-Object { $_.TaskName -like '*EdgeUpdate*' -or $_.TaskName -like '*Consolidator*' -or $_.TaskName -like '*UsbCeip*' } | Disable-ScheduledTask -EA SilentlyContinue | Out-Null
# Telemetry = Basic (минимум, чтобы Windows Update работал корректно)
New-Item 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\DataCollection' -Force -EA SilentlyContinue | Out-Null
Set-ItemProperty 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\DataCollection' -Name AllowTelemetry -Value 1 -Type DWord
# Delivery Optimization → Manual (Windows Update работает, качает меньше в фоне)
Set-Service DoSvc -StartupType Manual -EA SilentlyContinue
# Defender: телеметрия (MAPS/sample submission) off, защита остаётся ON
Set-MpPreference -MAPSReporting 0 -SubmitSamplesConsent 2 -EA SilentlyContinue

Write-Host "=== [4/6] NCSI off (не пингует msftconnecttest.com) ===" -ForegroundColor Cyan
New-Item 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\NetworkConnectivityStatusIndicator' -Force -EA SilentlyContinue | Out-Null
Set-ItemProperty 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\NetworkConnectivityStatusIndicator' -Name NoActiveProbe -Value 1 -Type DWord
Set-ItemProperty 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\NetworkConnectivityStatusIndicator' -Name NoPassiveProbe -Value 1 -Type DWord

Write-Host "=== [5/6] Disable Ctrl+Alt+Del при входе ===" -ForegroundColor Cyan
Set-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon' -Name DisableCAD -Value 1 -Type DWord

Write-Host "=== [6/6] Автологин (ВНИМАНИЕ: пароль хранится в реестре открытым текстом!) ===" -ForegroundColor Cyan
Write-Host "Если хочешь автологин — запусти вручную:" -ForegroundColor Yellow
Write-Host '  $pw = Read-Host "Пароль Администратора" -AsSecureString  # НЕ ЗАПИСЫВАТЬ В ФАЙЛ!'
Write-Host '  Set-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon" -Name AutoAdminLogon -Value 1'
Write-Host '  Set-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon" -Name DefaultUserName -Value "Администратор"'
Write-Host '  # безопасный вариант: Sysinternals Autologon.exe (LSA-шифрование пароля в реестре)'

Restart-Service spice-agent -EA SilentlyContinue
Stop-Process -Name explorer -Force -EA SilentlyContinue
Write-Host ""
Write-Host "=== ПРЕП-VM DONE (ребут нужен для активации Memory Compression и LogServer ACP-фикса) ===" -ForegroundColor Green

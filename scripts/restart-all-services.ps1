# =============================================================
# AION 7.7 PTS EU — health-check / подъем недостающих сервисов
# VM 109 (Windows Server 2022). ВАЖНО: запуск через schtasks-задачи
# (батники в C:\Temp), а НЕ Start-Process из SSH — процессы, запущенные
# в SSH-сессии, умирают при её закрытии (моделирование Services 0).
# Идемпотентно: запускает только то, чего нет.
# Server64 через RunAsDate НЕ трогает (интерактивный RUN юзером).
# Запуск: powershell -ExecutionPolicy Bypass -File restart-all-services.ps1
# =============================================================
$ErrorActionPreference = 'SilentlyContinue'
$SRV = 'D:\AION_LIVE_SERVER'

# Пак: имя процесса -> задача schtasks (батники C:\Temp)
$pack = @(
  @{ Proc='AccountCacheServer'; Task='AionAcc' }
  @{ Proc='L2Authd';            Task='AionAuth' }
  @{ Proc='AuthGateD';          Task='AionGate' }
  @{ Proc='LogServer64';        Task='AionLog' }
  @{ Proc='CacheD64';           Task='AionCache' }
  @{ Proc='NPCSvr64';           Task='AionNPC' }
  @{ Proc='ICServer';           Task='AionICSrv' }
  @{ Proc='CAPTCHAImageServer'; Task='AionCAPTCHA' }
)

foreach ($c in $pack) {
  $p = Get-Process -Name $c.Proc
  if ($p) { Write-Host ("[UP  ] {0} (PID {1})" -f $c.Proc, ($p.Id -join ',')) }
  else {
    Write-Host ("[DOWN] {0} -> schtasks /Run /TN {1}" -f $c.Proc, $c.Task)
    schtasks /Change /TN $c.Task /ENABLE | Out-Null
    schtasks /Run /TN $c.Task | Out-Null
    schtasks /Change /TN $c.Task /DISABLE | Out-Null
    Start-Sleep -Seconds 8
    if (Get-Process -Name $c.Proc) { Write-Host ("[OK  ] {0} started" -f $c.Proc) }
    else { Write-Host ("[FAIL] {0} не поднялся (см. log)" -f $c.Proc) }
  }
}

# Проверка портов (ожидаемые слушатели)
$expect = @{ 2220='AccountCacheServer'; 2104='L2Authd'; 2106='AuthGateD'; 2051='LogServer64';
             2006='CacheD64'; 22206='CAPTCHAImageServer'; 2005='ICServer'; 2305='ICServer' }
$listen = (netstat -ano) -match 'LISTENING'
foreach ($port in $expect.Keys) {
  if ($listen -match (':{0} ' -f $port)) { Write-Host ("[PORT] {0,5} OK ({1})" -f $port, $expect[$port]) }
  else { Write-Host ("[PORT] {0,5} NOT LISTENING ({1})" -f $port, $expect[$port]) }
}

# Server64 — только напоминание (RunAsDate интерактивный)
if (Get-Process -Name Server64) { Write-Host '[UP  ] Server64 (RunAsDate)' }
else {
  Write-Host '[MANUAL] Server64 не запущен: открываю RunAsDate (AionRAD)...'
  schtasks /Change /TN AionRAD /ENABLE | Out-Null
  schtasks /Run /TN AionRAD | Out-Null
  schtasks /Change /TN AionRAD /DISABLE | Out-Null
  Write-Host '         -> юзер нажимает RUN (DateTime=04-06-2020 16:28:23)'
}

# RAM-контроль (28 ГБ: Server64 8.6 + NPCSvr 15 + SQL 2 — впритык)
$os = Get-CimInstance Win32_OperatingSystem
Write-Host ('[RAM  ] CommitFree: {0} GB / PhysFree: {1} GB' -f [math]::Round($os.FreeVirtualMemory/1MB,1), [math]::Round($os.FreePhysicalMemory/1MB,1))
# =============================================================
# AION 7.7 PTS EU — health-check / подъем недостающих сервисов
# VM 109 (Windows Server 2022). Идемпотентно: запускает только то,
# чего нет. Server64 через RunAsDate НЕ трогает (интерактивный RUN).
# Запуск: powershell -ExecutionPolicy Bypass -File restart-all-services.ps1
# =============================================================
$ErrorActionPreference = 'SilentlyContinue'
$SRV = 'D:\AION_LIVE_SERVER'

# компонент -> процесс, рабочий каталог, exe
$components = @(
  @{ Name='AccountCacheServer'; Proc='AccountCacheServer'; Dir="$SRV\AccountCacheServer"; Exe='AccountCacheServer.exe' }
  @{ Name='L2Authd';            Proc='L2Authd';            Dir="$SRV\AuthD";              Exe='L2Authd.exe' }
  @{ Name='AuthGateD';          Proc='AuthGateD';          Dir="$SRV\AuthGateD";          Exe='AuthGateD.exe' }
  @{ Name='LogServer64';        Proc='LogServer64';        Dir="$SRV\LogServer";          Exe='LogServer64.exe' }
  @{ Name='CacheD64';           Proc='CacheD64';           Dir="$SRV\CacheServer";        Exe='CacheD64.exe' }
  @{ Name='CAPTCHAImageServer'; Proc='CAPTCHAImageServer'; Dir="$SRV\CAPTCHAImageServer"; Exe='CAPTCHAImageServer.exe' }
  @{ Name='ICServer';           Proc='ICServer';           Dir="$SRV\ICServer";           Exe='ICServer.exe' }
  @{ Name='NPCSvr64';           Proc='NPCSvr64';           Dir="$SRV\NPCServer";          Exe='NPCSvr64.exe' }
)

foreach ($c in $components) {
  $p = Get-Process -Name $c.Proc
  if ($p) { Write-Host ("[UP  ] {0} (PID {1})" -f $c.Name, ($p.Id -join ',')) }
  else {
    Write-Host ("[DOWN] {0} -> starting" -f $c.Name)
    Start-Process -FilePath (Join-Path $c.Dir $c.Exe) -WorkingDirectory $c.Dir
    Start-Sleep -Seconds 5
    if (Get-Process -Name $c.Proc) { Write-Host ("[OK  ] {0} started" -f $c.Name) }
    else { Write-Host ("[FAIL] {0} did not start (см. log в {1}\log\)" -f $c.Name, $c.Dir) }
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
else { Write-Host '[MANUAL] Server64 не запущен: MainServer\runasdate-x64\RunAsDate.exe -> RUN (DateTime=04-06-2020 16:28:23)' }

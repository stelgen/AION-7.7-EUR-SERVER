$ErrorActionPreference = 'SilentlyContinue'
Write-Output ('conns2002=' + (netstat -ano | Select-String ':2002' | Select-String 'ESTABLISHED').Count)
$names = @('Server64','NPCSvr64','AuthGateD','L2Authd','CacheD64','LogServer64','AccountCacheServer','ICServer','CAPTCHAImageServer','01-PAServer7.7')
$cores = [Environment]::ProcessorCount
function Snap {
    $h = @{}
    foreach ($n in $names) {
        $p = Get-Process -Name $n -ErrorAction SilentlyContinue
        if ($p) { $h[$n] = @{ cpu = $p.TotalProcessorTime.TotalSeconds; ws = [math]::Round($p.WorkingSet64/1MB); priv = [math]::Round($p.PrivateMemorySize64/1MB); id = $p.Id } }
    }
    return $h
}
$a = Snap
Start-Sleep -Seconds 10
$b = Snap
Write-Output ("cores=" + $cores)
foreach ($n in $names) {
    if ($b.ContainsKey($n)) {
        if ($a.ContainsKey($n)) {
            $pct = [math]::Round((($b[$n].cpu - $a[$n].cpu) / 10 / $cores) * 100, 1)
            Write-Output ($n + ' pid=' + $b[$n].id + ' cpu%=' + $pct + ' ws=' + $b[$n].ws + 'MB priv=' + $b[$n].priv + 'MB')
        } else { Write-Output ($n + ' pid=' + $b[$n].id + ' NEW ws=' + $b[$n].ws + 'MB') }
    } else { Write-Output ($n + ' : NOT RUNNING') }
}
Write-Output '######## AuthD dirs'
Get-ChildItem 'D:\AION_LIVE_SERVER\AuthD' -Directory | ForEach-Object { Write-Output ('  ' + $_.Name) }
$wl = Get-ChildItem 'D:\AION_LIVE_SERVER\AuthD' -Recurse -Filter 'winlog*' -ErrorAction SilentlyContinue | Sort-Object LastWriteTime | Select-Object -Last 2
foreach ($w in $wl) { Write-Output ('  winlog: ' + $w.FullName + ' ' + $w.LastWriteTime) }

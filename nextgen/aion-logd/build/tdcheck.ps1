foreach ($f in @('D:\AION_LIVE_SERVER\NPCServer\log\2026-10-05.err','D:\AION_LIVE_SERVER\MainServer\log\2026-10-05.err','D:\AION_LIVE_SERVER\CacheServer\log\2026-10-05.err')) {
  $hits = Select-String -Path $f -Pattern 'Time difference' -SimpleMatch -ErrorAction SilentlyContinue | Select-Object -Last 5
  foreach ($h in $hits) { Write-Output ($f.Substring(20,10) + ' | ' + $h.Line.Substring(0, [Math]::Min(120, $h.Line.Length))) }
}

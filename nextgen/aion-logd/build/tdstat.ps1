foreach ($f in @('D:\AION_LIVE_SERVER\NPCServer\log\2026-10-05.err','D:\AION_LIVE_SERVER\MainServer\log\2026-10-05.err')) {
  Write-Output ('=== ' + $f)
  $lines = Get-Content $f -ErrorAction SilentlyContinue
  $hits = $lines | Select-String 'Time difference' -SimpleMatch
  Write-Output ('total TD: ' + $hits.Count)
  $hits | ForEach-Object { $_.Line.Substring(0,[Math]::Min(60,$_.Line.Length)) } | Select-Object -First 3
  $hits | ForEach-Object { $_.Line.Substring(0,[Math]::Min(60,$_.Line.Length)) } | Select-Object -Last 3
  $rec = $lines | Select-String 'LogClientSocket' -SimpleMatch | Measure-Object
  Write-Output ('LogClientSocket lines: ' + $rec.Count)
}

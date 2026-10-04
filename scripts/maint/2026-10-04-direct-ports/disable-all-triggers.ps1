$ErrorActionPreference = 'Continue'
$tasks = 'AionKickMain','AionKickMain2','AionKickNPC','AionKickNPC2','AionStopHeavy','AionFullRestart','AionMainR','AionNPCR','AionRADTest'
$wl = Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon'
$pw = $wl.DefaultPassword
foreach ($n in $tasks) {
    try {
        $t = Get-ScheduledTask -TaskName $n -ErrorAction Stop
        $user = $t.Principal.UserId
        $t.Triggers | ForEach-Object { $_.Enabled = $false }
        Set-ScheduledTask -InputObject $t -User $user -Password $pw | Out-Null
        $t2 = Get-ScheduledTask -TaskName $n
        $i2 = Get-ScheduledTaskInfo -TaskName $n
        $en = ($t2.Triggers | ForEach-Object { $_.Enabled }) -join ','
        Write-Output ($n + ' OK trig_en=' + $en + ' state=' + $t2.State)
    } catch { Write-Output ($n + ' ERR: ' + $_.Exception.Message) }
}
Write-Output '===== FINAL (schtasks csv) ====='
schtasks /query /fo csv | Select-String 'AionKickMain|AionKickNPC|AionStopHeavy|AionFullRestart|AionMainR|AionNPCR|AionRADTest'

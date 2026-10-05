# READ-ONLY: login timeline + char save contents (no writes)
$con = New-Object System.Data.SqlClient.SqlConnection("Server=.;Integrated Security=True")
$con.Open()
function Q($q) {
  $c=$con.CreateCommand(); $c.CommandTimeout=60; $c.CommandText=$q
  $r=$c.ExecuteReader(); $out=@()
  while($r.Read()){ $o=@(); for($i=0;$i -lt $r.FieldCount;$i++){ $o += [string]$r.GetValue($i) }; $out += ($o -join ' | ') }
  $r.Close(); $out
}
'== all login attempts today (LoadFameInfo calls) =='
Get-Content 'D:\AION_LIVE_SERVER\CacheServer\log\DBReport_Alert.DBErr' | Select-String 'LoadFameInfo' | ForEach-Object { $_.Line.Substring(0,[Math]::Min(60,$_.Line.Length)) }
'== authd login packets today (account stelgen) =='
Get-ChildItem 'D:\AION_LIVE_SERVER\AuthD\etc\log' -Filter '2026-10-05.*.packet' | Sort Name | ForEach-Object {
  $f=$_.Name
  (Select-String -Path $_.FullName -Pattern '7374656c67656e' | ForEach-Object { $f + ' :: ' + $_.Line.Substring(0,[Math]::Min(70,$_.Line.Length)) })
}
'== user_item columns =='
Q "SELECT COLUMN_NAME, DATA_TYPE FROM [_AionWorldNew114_rc].INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME='user_item' ORDER BY ORDINAL_POSITION"
'== user_skill columns =='
Q "SELECT COLUMN_NAME, DATA_TYPE FROM [_AionWorldNew114_rc].INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME='user_skill' ORDER BY ORDINAL_POSITION"

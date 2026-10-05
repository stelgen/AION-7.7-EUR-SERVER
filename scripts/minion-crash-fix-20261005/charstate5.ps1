# READ-ONLY: char 1002 save contents audit (no writes)
$con = New-Object System.Data.SqlClient.SqlConnection("Server=.;Integrated Security=True")
$con.Open()
function Q($q) {
  $c=$con.CreateCommand(); $c.CommandTimeout=60; $c.CommandText=$q
  $r=$c.ExecuteReader(); $out=@()
  while($r.Read()){ $o=@(); for($i=0;$i -lt $r.FieldCount;$i++){ $o += [string]$r.GetValue($i) }; $out += ($o -join ' | ') }
  $r.Close(); $out
}
'== SetLastTransformId timeline (saves) =='
Get-Content 'D:\AION_LIVE_SERVER\CacheServer\log\DBReport_Alert.DBErr' | Select-String 'SetLastTransformId' | ForEach-Object { $_.Line.Substring(0,[Math]::Min(50,$_.Line.Length)) }
'== items by warehouse =='
Q "SELECT warehouse, COUNT(*), CONVERT(varchar,MIN(update_date),120), CONVERT(varchar,MAX(update_date),120) FROM [_AionWorldNew114_rc].dbo.user_item WHERE char_id=1002 GROUP BY warehouse ORDER BY warehouse"
'== items updated after 04.10 23:00 =='
Q "SELECT id, name_id, slot_id, amount, slot, warehouse, CONVERT(varchar,create_date,120), CONVERT(varchar,update_date,120) FROM [_AionWorldNew114_rc].dbo.user_item WHERE char_id=1002 AND update_date >= '2026-10-04 23:00' ORDER BY update_date"
'== suspicious items (name_id=0 or amount<=0) =='
Q "SELECT id, name_id, slot_id, amount, slot, warehouse, tid FROM [_AionWorldNew114_rc].dbo.user_item WHERE char_id=1002 AND (name_id=0 OR amount<=0)"
'== skills =='
Q "SELECT COUNT(*) FROM [_AionWorldNew114_rc].dbo.user_skill WHERE char_id=1002"
Q "SELECT TOP 50 skill_id, skill_data1, skill_data2 FROM [_AionWorldNew114_rc].dbo.user_skill WHERE char_id=1002 ORDER BY skill_id"
'== quests =='
Q "SELECT COLUMN_NAME FROM [_AionWorldNew114_rc].INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME='user_quest'"
Q "SELECT COUNT(*) FROM [_AionWorldNew114_rc].dbo.user_quest WHERE char_id=1002"

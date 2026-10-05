# ROLLBACK cube: rows 195/196/198 back to slots 0/1/2
$con = New-Object System.Data.SqlClient.SqlConnection("Server=.;Integrated Security=True")
$con.Open()
function Q($q) {
  $c=$con.CreateCommand(); $c.CommandTimeout=60; $c.CommandText=$q
  $r=$c.ExecuteReader(); $out=@()
  while($r.Read()){ $o=@(); for($i=0;$i -lt $r.FieldCount;$i++){ $o += [string]$r.GetValue($i) }; $out += ($o -join ' | ') }
  $r.Close(); $out
}
$c=$con.CreateCommand(); $c.CommandTimeout=60
$c.CommandText="UPDATE [_AionWorldNew114_rc].dbo.user_item SET slot_id=0 WHERE id=195; UPDATE [_AionWorldNew114_rc].dbo.user_item SET slot_id=1 WHERE id=196; UPDATE [_AionWorldNew114_rc].dbo.user_item SET slot_id=2 WHERE id=198;"
'rows restored: ' + $c.ExecuteNonQuery()
'== cube slots 0-2 (should be duplicated again, as before) =='
Q "SELECT slot_id, name_id, amount FROM [_AionWorldNew114_rc].dbo.user_item WHERE char_id=1002 AND warehouse=0 AND slot_id IN (0,1,2) ORDER BY slot_id, create_date"
'== full DB backup exists check =='
Q "SELECT TOP 3 name, CONVERT(varchar,backup_finish_date,120), DATEDIFF(s, backup_start_date, backup_finish_date) FROM msdb.dbo.backupset WHERE database_name='_AionWorldNew114_rc' ORDER BY backup_finish_date DESC"

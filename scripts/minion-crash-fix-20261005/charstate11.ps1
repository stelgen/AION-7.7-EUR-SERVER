# READ-ONLY: last probes (no writes)
$con = New-Object System.Data.SqlClient.SqlConnection("Server=.;Integrated Security=True")
$con.Open()
function Q($q) {
  $c=$con.CreateCommand(); $c.CommandTimeout=60; $c.CommandText=$q
  $r=$c.ExecuteReader(); $out=@()
  while($r.Read()){ $o=@(); for($i=0;$i -lt $r.FieldCount;$i++){ $o += [string]$r.GetValue($i) }; $out += ($o -join ' | ') }
  $r.Close(); $out
}
'== client settings row =='
Q "SELECT char_id, data_size FROM [_AionWorldNew114_rc].dbo.user_client_settings WHERE char_id=1002"
'== quickbar row =='
Q "SELECT char_id, data_size FROM [_AionWorldNew114_rc].dbo.user_client_quickbar WHERE char_id=1002"
'== equipped + bag items (warehouse 0) all =='
Q "SELECT slot, slot_id, name_id, amount, tid, CONVERT(varchar,update_date,120) FROM [_AionWorldNew114_rc].dbo.user_item WHERE char_id=1002 AND warehouse=0 ORDER BY slot"
'== skill_skin / cooltime / stat counts =='
Q "SELECT (SELECT COUNT(*) FROM [_AionWorldNew114_rc].dbo.user_skill_skin WHERE char_id=1002), (SELECT COUNT(*) FROM [_AionWorldNew114_rc].dbo.user_skill_cooltime WHERE char_id=1002), (SELECT COUNT(*) FROM [_AionWorldNew114_rc].dbo.user_stat WHERE char_id=1002), (SELECT COUNT(*) FROM [_AionWorldNew114_rc].dbo.user_title WHERE char_id=1002), (SELECT COUNT(*) FROM [_AionWorldNew114_rc].dbo.user_emotion WHERE char_id=1002)"
'== user_punishment char 1002 =='
Q "SELECT TOP 5 * FROM [_AionWorldNew114_rc].dbo.user_punishment WHERE char_id=1002"

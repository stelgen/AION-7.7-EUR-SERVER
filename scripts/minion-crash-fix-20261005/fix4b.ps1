# FIX: restore 5 quests (no identity nonsense) + zero last_summon_familiar
$con = New-Object System.Data.SqlClient.SqlConnection("Server=.;Integrated Security=True")
$con.Open()
function Q($q) {
  $c=$con.CreateCommand(); $c.CommandTimeout=60; $c.CommandText=$q
  $r=$c.ExecuteReader(); $out=@()
  while($r.Read()){ $o=@(); for($i=0;$i -lt $r.FieldCount;$i++){ $o += [string]$r.GetValue($i) }; $out += ($o -join ' | ') }
  $r.Close(); $out
}
'== quest restore =='
$c=$con.CreateCommand(); $c.CommandTimeout=60
$c.CommandText="IF NOT EXISTS (SELECT 1 FROM [_AionWorldNew114_rc].dbo.user_quest WHERE char_id=1002 AND quest_id IN (61603,63965,63970,63980,63985)) BEGIN INSERT INTO [_AionWorldNew114_rc].dbo.user_quest (char_id,quest_id,quest_status,quest_progress,quest_branch) SELECT char_id,quest_id,quest_status,quest_progress,quest_branch FROM [_AionWorldNew114_rc].dbo.z_backup_userquest_cubefix WHERE quest_id IN (61603,63965,63970,63980,63985) END"
'restored rows: ' + $c.ExecuteNonQuery()
'== quests now (should be 15) =='
Q "SELECT COUNT(*) FROM [_AionWorldNew114_rc].dbo.user_quest WHERE char_id=1002"
'== zero minion auto-summon =='
$c2=$con.CreateCommand(); $c2.CommandTimeout=60
$c2.CommandText="UPDATE [_AionWorldNew114_rc].dbo.user_data_ext SET last_summon_familiar=0 WHERE char_id=1002"
'zeroed: ' + $c2.ExecuteNonQuery()
'== final state =='
Q "SELECT char_id, last_summon_familiar FROM [_AionWorldNew114_rc].dbo.user_data_ext WHERE char_id=1002"
Q "SELECT COUNT(*) FROM [_AionWorldNew114_rc].dbo.user_familiar WHERE char_id=1002"
Q "SELECT quest_id, quest_status FROM [_AionWorldNew114_rc].dbo.user_quest WHERE char_id=1002 AND quest_status<6"

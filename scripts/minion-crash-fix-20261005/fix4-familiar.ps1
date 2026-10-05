# STEP 4: delete minion rows (id 3=980010, id=4=980020) + restore quests from backup table
$con = New-Object System.Data.SqlClient.SqlConnection("Server=.;Integrated Security=True")
$con.Open()
function Q($q) {
  $c=$con.CreateCommand(); $c.CommandTimeout=60; $c.CommandText=$q
  $r=$c.ExecuteReader(); $out=@()
  while($r.Read()){ $o=@(); for($i=0;$i -lt $r.FieldCount;$i++){ $o += [string]$r.GetValue($i) }; $out += ($o -join ' | ') }
  $r.Close(); $out
}
'== minion rows before =='
Q "SELECT id, base_name_id, cur_name_id, deleted FROM [_AionWorldNew114_rc].dbo.user_familiar WHERE char_id=1002"
$c=$con.CreateCommand(); $c.CommandTimeout=60
$c.CommandText="DELETE FROM [_AionWorldNew114_rc].dbo.user_familiar WHERE char_id=1002 AND id IN (3,4)"
'deleted minion rows: ' + $c.ExecuteNonQuery()
'== user_familiar after =='
Q "SELECT COUNT(*) FROM [_AionWorldNew114_rc].dbo.user_familiar WHERE char_id=1002"
'== restore quests =='
$c2=$con.CreateCommand(); $c2.CommandTimeout=60
$c2.CommandText="DELETE FROM [_AionWorldNew114_rc].dbo.user_quest WHERE char_id=1002 AND quest_id IN (61603,63965,63970,63980,63985); SET IDENTITY_INSERT off; INSERT INTO [_AionWorldNew114_rc].dbo.user_quest (char_id,quest_id,quest_status,quest_progress,quest_branch) SELECT char_id,quest_id,quest_status,quest_progress,quest_branch FROM [_AionWorldNew114_rc].dbo.z_backup_userquest_cubefix WHERE quest_id IN (61603,63965,63970,63980,63985)"
'restored quest rows: ' + $c2.ExecuteNonQuery()
'== quests after (should be 15 again) =='
Q "SELECT quest_id, quest_status, quest_progress FROM [_AionWorldNew114_rc].dbo.user_quest WHERE char_id=1002 ORDER BY quest_id"
'== minion auto-summon still off =='
Q "SELECT char_id, last_summon_familiar FROM [_AionWorldNew114_rc].dbo.user_data_ext WHERE char_id=1002"
'BACKUP: full .bak + minions were (id 3: 980010, growth 0, created 04:12; id 4: 980020, growth 65765, created 04:14)'

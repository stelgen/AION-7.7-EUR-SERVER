# STEP 2: delete active (non-story) quests of char 1002
# keep completed (status 6): 14043, 60110, 60201-60208
# delete active: 61603, 63965, 63970, 63980, 63985
$con = New-Object System.Data.SqlClient.SqlConnection("Server=.;Integrated Security=True")
$con.Open()
function Q($q) {
  $c=$con.CreateCommand(); $c.CommandTimeout=60; $c.CommandText=$q
  $r=$c.ExecuteReader(); $out=@()
  while($r.Read()){ $o=@(); for($i=0;$i -lt $r.FieldCount;$i++){ $o += [string]$r.GetValue($i) }; $out += ($o -join ' | ') }
  $r.Close(); $out
}
'== BEFORE: all quests =='
Q "SELECT quest_id, quest_status, quest_progress FROM [_AionWorldNew114_rc].dbo.user_quest WHERE char_id=1002 ORDER BY quest_id"
$c=$con.CreateCommand(); $c.CommandTimeout=60
$c.CommandText="DELETE FROM [_AionWorldNew114_rc].dbo.user_quest WHERE char_id=1002 AND quest_id IN (61603,63965,63970,63980,63985)"
'rows deleted: ' + $c.ExecuteNonQuery()
'== AFTER: remaining (should be only status 6) =='
Q "SELECT quest_id, quest_status, quest_progress FROM [_AionWorldNew114_rc].dbo.user_quest WHERE char_id=1002 ORDER BY quest_id"
'BACKUP NOTE: full row backup in z_backup_userquest_cubefix (15 rows) + full DB .bak AionWorld-20261005-pre-cubefix.bak'

# READ-ONLY: char 1002 full state (no writes)
$con = New-Object System.Data.SqlClient.SqlConnection("Server=.;Integrated Security=True")
$con.Open()
function Q($q) {
  $c=$con.CreateCommand(); $c.CommandTimeout=60; $c.CommandText=$q
  $r=$c.ExecuteReader(); $out=@()
  while($r.Read()){ $o=@(); for($i=0;$i -lt $r.FieldCount;$i++){ $o += [string]$r.GetValue($i) }; $out += ($o -join ' | ') }
  $r.Close(); $out
}
'== chars of account 1010 =='
Q "SELECT char_id, user_id, account_id, race, class, lev, is_banned, cur_server, org_server, world, world_map_number, xlocation, ylocation, zlocation, dir, last_normal_world, last_normal_xlocation, last_normal_ylocation, last_normal_zlocation, now_flight, is_freefly, is_jumping_character, last_login_time, last_logout_time, playtime, housing_id, account_punishment FROM [_AionWorldNew114_rc].dbo.user_data WHERE account_id=1010"
'== abnormal_status char 1002 =='
Q "SELECT char_id, skill_id, skill_level, effect_remain1, effect_remain2, interval_value1 FROM [_AionWorldNew114_rc].dbo.user_abnormal_status WHERE char_id=1002"
'== instances char 1002 =='
Q "SELECT TOP 10 * FROM [_AionWorldNew114_rc].dbo.user_instance WHERE char_id=1002"
'== user_transform schema =='
Q "SELECT COLUMN_NAME, DATA_TYPE FROM [AionAccountCacheD_rc].INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME='user_transform' ORDER BY ORDINAL_POSITION"
'== user_transform rows =='
Q "SELECT TOP 10 * FROM [AionAccountCacheD_rc].dbo.user_transform WHERE char_id=1002 OR account_id=1010"
'== proc aion_SetLastTransformId text (first 25 lines) =='
$c=$con.CreateCommand(); $c.CommandTimeout=60; $c.CommandText="EXEC AionAccountCacheD_rc.sys.sp_helptext 'dbo.aion_SetLastTransformId'"
$r=$c.ExecuteReader(); $n=0
while($r.Read()){ if($n -lt 25){ ([string]$r.GetValue(0)) }; $n++ }
'PROC LINES TOTAL: '+$n
$r.Close()
'== proc params =='
Q "SELECT p.name, t.name, p.max_length FROM AionAccountCacheD_rc.sys.parameters p WHERE p.object_id=OBJECT_ID('AionAccountCacheD_rc.dbo.aion_SetLastTransformId') ORDER BY p.parameter_id"

# READ-ONLY: user_data_ext row + familiars details (no writes)
$con = New-Object System.Data.SqlClient.SqlConnection("Server=.;Integrated Security=True")
$con.Open()
function Q($q) {
  $c=$con.CreateCommand(); $c.CommandTimeout=60; $c.CommandText=$q
  $r=$c.ExecuteReader(); $out=@()
  while($r.Read()){ $o=@(); for($i=0;$i -lt $r.FieldCount;$i++){ $o += [string]$r.GetValue($i) }; $out += ($o -join ' | ') }
  $r.Close(); $out
}
'== now =='
Q "SELECT CONVERT(varchar,GETDATE(),120)"
'== user_data_ext row char 1002 =='
Q "SELECT char_id, exps_login_reward_time, exps_npckill_reward_num, creativity_point, usecp_resetcount, next_usecp_resetcount_dec_time, global_tnmt_apply_seq, local_tnmt_apply_seq, familiar_func_expireTime, familiar_energy, familiar_energy_autocharge, familiar_func_autocharge, last_transform_id, last_transform_scroll_id, last_summon_familiar, last_collection_id, last_collection_id2, last_collection_id3, last_collection_id4, last_collection_id5, last_collection_id6 FROM [_AionWorldNew114_rc].dbo.user_data_ext WHERE char_id=1002"
'== user_familiar rows (named cols) =='
Q "SELECT id, char_id, name_id, slot_id, name, function_data1, function_data2, CONVERT(varchar,create_date,120), visual_data_size, DATALENGTH(visual_data), CONVERT(varchar,change_info_time,120), function_data1_ex1, function_data1_ex2, function_data1_ex3, function_data2_ex1, function_data2_ex2, function_data2_ex3, expired_time FROM [_AionWorldNew114_rc].dbo.user_familiar WHERE char_id=1002"
'== user_familiar data types =='
Q "SELECT COLUMN_NAME, DATA_TYPE FROM [_AionWorldNew114_rc].INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME='user_familiar' ORDER BY ORDINAL_POSITION"
'== familiars of OTHER chars (compare) =='
Q "SELECT id, char_id, name_id, slot_id, function_data1, function_data2, CONVERT(varchar,create_date,120) FROM [_AionWorldNew114_rc].dbo.user_familiar WHERE char_id<>1002"

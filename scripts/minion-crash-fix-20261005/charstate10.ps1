# READ-ONLY: fixes + client data + char 1000 (no writes)
$con = New-Object System.Data.SqlClient.SqlConnection("Server=.;Integrated Security=True")
$con.Open()
function Q($q) {
  $c=$con.CreateCommand(); $c.CommandTimeout=60; $c.CommandText=$q
  $r=$c.ExecuteReader(); $out=@()
  while($r.Read()){ $o=@(); for($i=0;$i -lt $r.FieldCount;$i++){ $o += [string]$r.GetValue($i) }; $out += ($o -join ' | ') }
  $r.Close(); $out
}
'== AionAccounts.user_data rows (account stelgen) =='
Q "SELECT uid, user_char_num, world, char_name, char_id, account_name, Lev, CONVERT(varchar,create_date,120), use_time, subjob0_class, subjob1_class, subjob2_class, subjob3_class FROM [AionAccounts].dbo.user_data WHERE account_name='stelgen'"
'== _AionWorld.user_data login/builder for 1000+1002 =='
Q "SELECT char_id, cur_server, org_server, login_server, builder, account_punishment, delete_date, delete_type FROM [_AionWorldNew114_rc].dbo.user_data WHERE char_id IN (1000,1002)"
'== client settings =='
Q "SELECT COLUMN_NAME, DATA_TYPE FROM [_AionWorldNew114_rc].INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME='user_client_settings' ORDER BY ORDINAL_POSITION"
Q "SELECT char_id, DATALENGTH(settings) FROM [_AionWorldNew114_rc].dbo.user_client_settings WHERE char_id=1002"
'== quickbar count =='
Q "SELECT COLUMN_NAME FROM [_AionWorldNew114_rc].INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME='user_client_quickbar' ORDER BY ORDINAL_POSITION"
Q "SELECT COUNT(*) FROM [_AionWorldNew114_rc].dbo.user_client_quickbar WHERE char_id=1002"
'== char 1000 =='
Q "SELECT char_id, user_id, account_id, race, class, lev, world, xlocation, ylocation, zlocation, CONVERT(varchar,last_logout_time,120), cur_server, org_server FROM [_AionWorldNew114_rc].dbo.user_data WHERE char_id=1000"
'== all chars =='
Q "SELECT char_id, user_id, account_id, lev, CONVERT(varchar,last_logout_time,120) FROM [_AionWorldNew114_rc].dbo.user_data ORDER BY char_id"

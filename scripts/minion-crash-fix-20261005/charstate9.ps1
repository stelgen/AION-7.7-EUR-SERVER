# READ-ONLY: account tables + client data (no writes)
$con = New-Object System.Data.SqlClient.SqlConnection("Server=.;Integrated Security=True")
$con.Open()
function Q($q) {
  $c=$con.CreateCommand(); $c.CommandTimeout=60; $c.CommandText=$q
  $r=$c.ExecuteReader(); $out=@()
  while($r.Read()){ $o=@(); for($i=0;$i -lt $r.FieldCount;$i++){ $o += [string]$r.GetValue($i) }; $out += ($o -join ' | ') }
  $r.Close(); $out
}
'== AionAccounts.user_data schema =='
Q "SELECT COLUMN_NAME, DATA_TYPE FROM [AionAccounts].INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME='user_data' ORDER BY ORDINAL_POSITION"
'== AionAccounts.user_data rows account 1010 =='
Q "SELECT TOP 5 * FROM [AionAccounts].dbo.user_data WHERE account_id=1010"
'== AionAccounts.user_info account 1010 =='
Q "SELECT COLUMN_NAME FROM [AionAccounts].INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME='user_info'"
Q "SELECT TOP 3 * FROM [AionAccounts].dbo.user_info WHERE account_id=1010"
'== _AionWorld.user_data login/builder for 1002 =='
Q "SELECT char_id, cur_server, org_server, login_server, builder, account_punishment, delete_date, delete_type FROM [_AionWorldNew114_rc].dbo.user_data WHERE char_id IN (1000,1002)"
'== client settings + quickbar sizes =='
Q "SELECT DATALENGTH(settings) FROM [_AionWorldNew114_rc].dbo.user_client_settings WHERE char_id=1002"
Q "SELECT COLUMN_NAME FROM [_AionWorldNew114_rc].INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME='user_client_settings'"
Q "SELECT COUNT(*) FROM [_AionWorldNew114_rc].dbo.user_client_quickbar WHERE char_id=1002"
'== char 1000 world state =='
Q "SELECT char_id, user_id, account_id, race, class, lev, world, xlocation, ylocation, zlocation, last_logout_time, cur_server, org_server FROM [_AionWorldNew114_rc].dbo.user_data WHERE char_id=1000"

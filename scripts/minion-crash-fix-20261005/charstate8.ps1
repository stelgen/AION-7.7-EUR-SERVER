# READ-ONLY: user_familiar proper schema + rows (no writes)
$con = New-Object System.Data.SqlClient.SqlConnection("Server=.;Integrated Security=True")
$con.Open()
function Q($q) {
  $c=$con.CreateCommand(); $c.CommandTimeout=60; $c.CommandText=$q
  $r=$c.ExecuteReader(); $out=@()
  while($r.Read()){ $o=@(); for($i=0;$i -lt $r.FieldCount;$i++){ $o += [string]$r.GetValue($i) }; $out += ($o -join ' | ') }
  $r.Close(); $out
}
'== user_familiar schema =='
Q "SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE FROM [_AionWorldNew114_rc].INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME='user_familiar' ORDER BY ORDINAL_POSITION"
'== user_familiar rows char 1002 =='
Q "SELECT TOP 5 * FROM [_AionWorldNew114_rc].dbo.user_familiar WHERE char_id=1002"
'== familiars other chars =='
Q "SELECT TOP 10 * FROM [_AionWorldNew114_rc].dbo.user_familiar WHERE char_id<>1002"
'== REF58_AionWorld user_familiar rows =='
Q "SELECT TOP 10 * FROM [REF58_AionWorld].dbo.user_familiar"
'== REF58L too =='
Q "SELECT TOP 5 * FROM [REF58L_AionAccountDB].dbo.user_familiar"

# STEP 3: disable minion auto-summon on login (char 1002)
# backup value printed first, then update, then verify
$con = New-Object System.Data.SqlClient.SqlConnection("Server=.;Integrated Security=True")
$con.Open()
function Q($q) {
  $c=$con.CreateCommand(); $c.CommandTimeout=60; $c.CommandText=$q
  $r=$c.ExecuteReader(); $out=@()
  while($r.Read()){ $o=@(); for($i=0;$i -lt $r.FieldCount;$i++){ $o += [string]$r.GetValue($i) }; $out += ($o -join ' | ') }
  $r.Close(); $out
}
'== BEFORE =='
Q "SELECT char_id, last_summon_familiar FROM [_AionWorldNew114_rc].dbo.user_data_ext WHERE char_id=1002"
$c=$con.CreateCommand(); $c.CommandTimeout=60
$c.CommandText="UPDATE [_AionWorldNew114_rc].dbo.user_data_ext SET last_summon_familiar=0 WHERE char_id=1002"
'n rows updated: ' + $c.ExecuteNonQuery()
'== AFTER =='
Q "SELECT char_id, last_summon_familiar FROM [_AionWorldNew114_rc].dbo.user_data_ext WHERE char_id=1002"
'== familiar still in list (untouched) =='
Q "SELECT id, char_id, base_name_id FROM [_AionWorldNew114_rc].dbo.user_familiar WHERE char_id=1002"
'BACKUP NOTE: previous value last_summon_familiar=980020 (char 1002)'

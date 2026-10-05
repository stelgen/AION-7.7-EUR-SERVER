# READ-ONLY: quests/pets/ext for char 1002 (no writes)
$con = New-Object System.Data.SqlClient.SqlConnection("Server=.;Integrated Security=True")
$con.Open()
function Q($q) {
  $c=$con.CreateCommand(); $c.CommandTimeout=60; $c.CommandText=$q
  $r=$c.ExecuteReader(); $out=@()
  while($r.Read()){ $o=@(); for($i=0;$i -lt $r.FieldCount;$i++){ $o += [string]$r.GetValue($i) }; $out += ($o -join ' | ') }
  $r.Close(); $out
}
'== quests char 1002 =='
Q "SELECT char_id, quest_id, quest_status, quest_progress, quest_branch FROM [_AionWorldNew114_rc].dbo.user_quest WHERE char_id=1002"
'== user_pet =='
Q "SELECT COLUMN_NAME FROM [_AionWorldNew114_rc].INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME='user_pet'"
Q "SELECT TOP 10 * FROM [_AionWorldNew114_rc].dbo.user_pet WHERE char_id=1002"
'== user_familiar =='
Q "SELECT TOP 10 * FROM [_AionWorldNew114_rc].dbo.user_familiar WHERE char_id=1002"
'== user_extra_info columns+row =='
Q "SELECT COLUMN_NAME, DATA_TYPE FROM [_AionWorldNew114_rc].INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME='user_extra_info' ORDER BY ORDINAL_POSITION"
'== user_data_ext columns =='
Q "SELECT COLUMN_NAME, DATA_TYPE FROM [_AionWorldNew114_rc].INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME='user_data_ext' ORDER BY ORDINAL_POSITION"
'== wardrobe =='
Q "SELECT COLUMN_NAME FROM [_AionWorldNew114_rc].INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME='user_wardrobe'"
Q "SELECT TOP 10 * FROM [_AionWorldNew114_rc].dbo.user_wardrobe WHERE char_id=1002"
